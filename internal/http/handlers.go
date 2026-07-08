package http

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/liteyuki/yuki-id-portal/internal/auth"
	"github.com/liteyuki/yuki-id-portal/internal/logto"
	"github.com/liteyuki/yuki-id-portal/internal/portal"
)

const sessionKey = "portal_session"

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) supportInfo(c *gin.Context) {
	accountBaseURL := s.deps.Config.LogtoAccountBaseURL
	c.JSON(http.StatusOK, gin.H{
		"email": s.deps.Config.SupportEmail,
		"accountCenter": gin.H{
			"profileUrl":          accountBaseURL + "/profile",
			"securityUrl":         accountBaseURL + "/security",
			"sessionsUrl":         accountBaseURL + "/sessions",
			"emailUrl":            accountBaseURL + "/email",
			"phoneUrl":            accountBaseURL + "/phone",
			"usernameUrl":         accountBaseURL + "/username",
			"passwordUrl":         accountBaseURL + "/password",
			"passkeyAddUrl":       accountBaseURL + "/passkey/add",
			"passkeyManageUrl":    accountBaseURL + "/passkey/manage",
			"authenticatorAppUrl": accountBaseURL + "/authenticator-app",
			"backupCodesUrl":      accountBaseURL + "/backup-codes/manage",
		},
		"topics": []string{
			"登录异常帮助",
			"账号迁移说明",
			"常见问题",
			"联系支持入口",
		},
	})
}

func (s *Server) login(c *gin.Context) {
	if s.deps.OIDC == nil {
		if s.deps.Config.DevAuthEnabled {
			if err := s.deps.Session.Create(c.Writer, auth.Session{
				User: auth.User{
					Sub:               "dev-yuki-user",
					Email:             "dev@liteyuki.org",
					EmailVerified:     true,
					Name:              "Yuki Developer",
					PreferredUsername: "yuki-dev",
					Roles: []string{
						"liteyuki-account-user",
						"liteyuki-grafana-user",
						"liteyuki-openlist-user",
						"liteyuki-beta-tester",
					},
					Organizations:     []string{"liteyuki"},
					OrganizationRoles: []string{"liteyuki:member", "liteyuki:beta-tester"},
				},
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "session_create_failed"})
				return
			}
			c.Redirect(http.StatusFound, "/")
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "oidc_not_configured",
			"message": "LOGTO_CLIENT_ID and LOGTO_CLIENT_SECRET are required.",
		})
		return
	}
	state, nonce, err := s.deps.State.Create()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login_state_failed"})
		return
	}
	c.Redirect(http.StatusFound, s.deps.OIDC.AuthCodeURL(state, nonce))
}

func (s *Server) callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_code_or_state"})
		return
	}

	loginState, ok := s.deps.State.Consume(state)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state"})
		return
	}

	session, err := s.deps.OIDC.Exchange(c.Request.Context(), code, loginState.Nonce)
	if err != nil {
		s.deps.Logger.Warn("oidc callback failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "oidc_exchange_failed"})
		return
	}
	if err := s.deps.Session.Create(c.Writer, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session_create_failed"})
		return
	}
	c.Redirect(http.StatusFound, "/")
}

func (s *Server) me(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (s *Server) permissions(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"roles":             user.Roles,
		"organizations":     user.Organizations,
		"organizationRoles": user.OrganizationRoles,
		"isAdmin":           hasRole(user, "liteyuki-account-admin"),
	})
}

func (s *Server) appCatalog(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	if s.deps.Management == nil {
		c.JSON(http.StatusOK, gin.H{
			"applications": []portal.AppCatalogResponseItem{},
			"source":       "management_not_configured",
		})
		return
	}

	managedApps, err := s.deps.Management.ListApplications(c.Request.Context())
	if err != nil {
		s.deps.Logger.Warn("management applications fetch failed", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"applications": []portal.AppCatalogResponseItem{},
			"source":       "management_error",
		})
		return
	}

	overrides, err := s.loadAppCatalog()
	if err != nil {
		s.deps.Logger.Warn("app catalog overlay load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "app_catalog_load_failed"})
		return
	}

	catalog := portal.MergeManagedApps(managedApps, overrides)
	c.JSON(http.StatusOK, gin.H{
		"applications": portal.FilterApps(catalog, user),
		"source":       "management",
	})
}

func (s *Server) announcements(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	announcements, err := s.loadAnnouncements()
	if err != nil {
		s.deps.Logger.Warn("announcements load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "announcements_load_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"announcements": portal.FilterAnnouncements(announcements, user, time.Now()),
	})
}

func (s *Server) markAnnouncementRead(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"read": true,
		"mode": "stateless",
	})
}

func (s *Server) listMyAppRequests(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	requests, err := s.deps.Requests.ListAppRequests(user.Sub)
	if err != nil {
		s.deps.Logger.Warn("app requests load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *Server) createAppRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	var request struct {
		Name                   string   `json:"name"`
		Type                   string   `json:"type"`
		Description            string   `json:"description"`
		RedirectURIs           []string `json:"redirectUris"`
		PostLogoutRedirectURIs []string `json:"postLogoutRedirectUris"`
		CORSAllowedOrigins     []string `json:"corsAllowedOrigins"`
		PortalURL              string   `json:"portalUrl"`
		Reason                 string   `json:"reason"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	request.Type = strings.TrimSpace(request.Type)
	request.RedirectURIs = cleanStrings(request.RedirectURIs)
	request.PostLogoutRedirectURIs = cleanStrings(request.PostLogoutRedirectURIs)
	request.CORSAllowedOrigins = cleanStrings(request.CORSAllowedOrigins)
	request.PortalURL = strings.TrimSpace(request.PortalURL)
	if request.Name == "" || !isAllowedUserApplicationType(request.Type) || len(request.RedirectURIs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_app_request"})
		return
	}
	if !allAbsoluteURLs(request.RedirectURIs) || !allAbsoluteURLs(request.PostLogoutRedirectURIs) || !allAbsoluteURLs(request.CORSAllowedOrigins) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_urls"})
		return
	}
	if request.PortalURL != "" && !allAbsoluteURLs([]string{request.PortalURL}) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_portal_url"})
		return
	}

	created, err := s.deps.Requests.CreateAppRequest(portal.AppRequest{
		RequesterSub:           user.Sub,
		RequesterEmail:         user.Email,
		Name:                   request.Name,
		Type:                   request.Type,
		Description:            strings.TrimSpace(request.Description),
		RedirectURIs:           request.RedirectURIs,
		PostLogoutRedirectURIs: request.PostLogoutRedirectURIs,
		CORSAllowedOrigins:     request.CORSAllowedOrigins,
		PortalURL:              request.PortalURL,
		Reason:                 strings.TrimSpace(request.Reason),
	})
	if err != nil {
		s.deps.Logger.Warn("app request create failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"request": created})
}

func (s *Server) listMyPermissionRequests(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	requests, err := s.deps.Requests.ListPermissionRequests(user.Sub)
	if err != nil {
		s.deps.Logger.Warn("permission requests load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *Server) createPermissionRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	var request struct {
		Kind          string `json:"kind"`
		RoleID        string `json:"roleId"`
		RoleName      string `json:"roleName"`
		ApplicationID string `json:"applicationId"`
		Reason        string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	request.Kind = strings.TrimSpace(request.Kind)
	request.RoleID = strings.TrimSpace(request.RoleID)
	request.RoleName = strings.TrimSpace(request.RoleName)
	if request.Kind == "" {
		request.Kind = "global_role"
	}
	if request.Kind != "global_role" || (request.RoleID == "" && request.RoleName == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_permission_request"})
		return
	}

	created, err := s.deps.Requests.CreatePermissionRequest(portal.PermissionRequest{
		RequesterSub:   user.Sub,
		RequesterEmail: user.Email,
		Kind:           request.Kind,
		RoleID:         request.RoleID,
		RoleName:       request.RoleName,
		ApplicationID:  strings.TrimSpace(request.ApplicationID),
		Reason:         strings.TrimSpace(request.Reason),
	})
	if err != nil {
		s.deps.Logger.Warn("permission request create failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"request": created})
}

func (s *Server) updateProfile(c *gin.Context) {
	session := mustSession(c)
	subject := sessionSubject(session)
	if s.deps.Management == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "management_api_not_configured",
			"message": "LOGTO_MANAGEMENT_CLIENT_ID and LOGTO_MANAGEMENT_CLIENT_SECRET are required.",
		})
		return
	}

	var request struct {
		Name              *string `json:"name"`
		Picture           *string `json:"picture"`
		PreferredUsername *string `json:"preferredUsername"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	update := logto.ManagementProfileUpdate{
		Name:     request.Name,
		Username: request.PreferredUsername,
		Avatar:   request.Picture,
	}
	result, err := s.deps.Management.UpdateUser(c.Request.Context(), subject, update)
	if err != nil {
		s.deps.Logger.Warn("profile update failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return
	}

	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": result, "user": user})
}

func (s *Server) authorizedApplications(c *gin.Context) {
	session := mustSession(c)
	result, err := s.deps.Account.ListApplications(c.Request.Context(), session.AccessToken)
	if err != nil {
		s.deps.Logger.Warn("authorized applications fetch failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "account_api_failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) sessions(c *gin.Context) {
	session := mustSession(c)
	result, err := s.deps.Account.ListSessions(c.Request.Context(), session.AccessToken)
	if err != nil {
		s.deps.Logger.Warn("sessions fetch failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "account_api_failed"})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) deleteSession(c *gin.Context) {
	session := mustSession(c)
	if err := s.deps.Account.DeleteSession(c.Request.Context(), session.AccessToken, c.Param("id")); err != nil {
		s.deps.Logger.Warn("session delete failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "account_api_failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) logout(c *gin.Context) {
	s.deps.Session.Destroy(c.Writer, c.Request)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) logoutPage(c *gin.Context) {
	s.deps.Session.Destroy(c.Writer, c.Request)
	c.Redirect(http.StatusFound, "/")
}

func (s *Server) logoutGlobal(c *gin.Context) {
	session := mustSession(c)
	redirectURL := s.globalLogoutURL(session)
	s.deps.Session.Destroy(c.Writer, c.Request)
	c.JSON(http.StatusOK, gin.H{"redirectUrl": redirectURL})
}

func (s *Server) adminAnnouncements(c *gin.Context) {
	announcements, err := s.loadAnnouncements()
	if err != nil {
		s.deps.Logger.Warn("announcements load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "announcements_load_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": announcements})
}

func (s *Server) adminAppCatalog(c *gin.Context) {
	catalog, err := s.loadAppCatalog()
	if err != nil {
		s.deps.Logger.Warn("app catalog overlay load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "app_catalog_load_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"applications": catalog})
}

func (s *Server) auditLogs(c *gin.Context) {
	logs, err := s.deps.Requests.ListAuditLogs()
	if err != nil {
		s.deps.Logger.Warn("audit logs load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auditLogs": logs})
}

func (s *Server) adminListAppRequests(c *gin.Context) {
	requests, err := s.deps.Requests.ListAppRequests("")
	if err != nil {
		s.deps.Logger.Warn("app requests load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *Server) approveAppRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	if s.deps.Management == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management_api_not_configured"})
		return
	}

	request, ok, err := s.deps.Requests.GetAppRequest(c.Param("id"))
	if err != nil {
		s.deps.Logger.Warn("app request load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "request_not_found"})
		return
	}
	if request.Status != portal.RequestStatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "request_is_not_pending"})
		return
	}

	createdApp, err := s.deps.Management.CreateApplication(c.Request.Context(), logto.CreateApplicationRequest{
		Name:        request.Name,
		Type:        request.Type,
		Description: request.Description,
		OidcClientMetadata: map[string]any{
			"redirectUris":           request.RedirectURIs,
			"postLogoutRedirectUris": request.PostLogoutRedirectURIs,
		},
		CustomClientMetadata: map[string]any{
			"corsAllowedOrigins": request.CORSAllowedOrigins,
		},
		CustomData: map[string]any{
			"portalOwnerSub":   request.RequesterSub,
			"portalOwnerEmail": request.RequesterEmail,
			"portalStatus":     "approved",
			"portalUrl":        request.PortalURL,
		},
	})
	if err != nil {
		s.deps.Logger.Warn("app request approve failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return
	}
	appID, _ := createdApp["id"].(string)
	reviewNote := reviewNote(c)
	reviewed, err := s.deps.Requests.ReviewAppRequest(c.Param("id"), portal.RequestStatusApproved, user.Sub, reviewNote, appID)
	if err != nil {
		s.deps.Logger.Warn("app request review store failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	_, _ = s.deps.Requests.AppendAuditLog(portal.AuditLog{
		ActorSub:   user.Sub,
		Action:     "approve_app_request",
		TargetType: "app_request",
		TargetID:   reviewed.ID,
		Metadata: map[string]any{
			"logtoApplicationId": appID,
			"requesterSub":       reviewed.RequesterSub,
		},
	})
	c.JSON(http.StatusOK, gin.H{"request": reviewed, "application": createdApp})
}

func (s *Server) rejectAppRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	reviewed, err := s.deps.Requests.ReviewAppRequest(c.Param("id"), portal.RequestStatusRejected, user.Sub, reviewNote(c), "")
	if errors.Is(err, portal.ErrRequestNotPending) {
		c.JSON(http.StatusConflict, gin.H{"error": "request_is_not_pending"})
		return
	}
	if err != nil {
		c.JSON(statusForStoreError(err), gin.H{"error": "request_review_failed"})
		return
	}
	_, _ = s.deps.Requests.AppendAuditLog(portal.AuditLog{
		ActorSub:   user.Sub,
		Action:     "reject_app_request",
		TargetType: "app_request",
		TargetID:   reviewed.ID,
		Metadata: map[string]any{
			"requesterSub": reviewed.RequesterSub,
		},
	})
	c.JSON(http.StatusOK, gin.H{"request": reviewed})
}

func (s *Server) adminListPermissionRequests(c *gin.Context) {
	requests, err := s.deps.Requests.ListPermissionRequests("")
	if err != nil {
		s.deps.Logger.Warn("permission requests load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (s *Server) approvePermissionRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	if s.deps.Management == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management_api_not_configured"})
		return
	}

	request, ok, err := s.deps.Requests.GetPermissionRequest(c.Param("id"))
	if err != nil {
		s.deps.Logger.Warn("permission request load failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request_store_failed"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "request_not_found"})
		return
	}
	if request.Status != portal.RequestStatusPending {
		c.JSON(http.StatusConflict, gin.H{"error": "request_is_not_pending"})
		return
	}
	roleID := request.RoleID
	if roleID == "" {
		resolvedRoleID, err := s.resolveRoleID(c.Request.Context(), request.RoleName)
		if err != nil {
			s.deps.Logger.Warn("permission request role resolve failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "role_resolve_failed"})
			return
		}
		roleID = resolvedRoleID
	}
	if err := s.deps.Management.AssignRolesToUser(c.Request.Context(), request.RequesterSub, []string{roleID}); err != nil {
		s.deps.Logger.Warn("permission request approve failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return
	}
	reviewed, err := s.deps.Requests.ReviewPermissionRequest(c.Param("id"), portal.RequestStatusApproved, user.Sub, reviewNote(c))
	if err != nil {
		c.JSON(statusForStoreError(err), gin.H{"error": "request_review_failed"})
		return
	}
	_, _ = s.deps.Requests.AppendAuditLog(portal.AuditLog{
		ActorSub:   user.Sub,
		Action:     "approve_permission_request",
		TargetType: "permission_request",
		TargetID:   reviewed.ID,
		Metadata: map[string]any{
			"requesterSub": reviewed.RequesterSub,
			"roleId":       roleID,
			"roleName":     reviewed.RoleName,
		},
	})
	c.JSON(http.StatusOK, gin.H{"request": reviewed})
}

func (s *Server) rejectPermissionRequest(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		return
	}
	reviewed, err := s.deps.Requests.ReviewPermissionRequest(c.Param("id"), portal.RequestStatusRejected, user.Sub, reviewNote(c))
	if err != nil {
		c.JSON(statusForStoreError(err), gin.H{"error": "request_review_failed"})
		return
	}
	_, _ = s.deps.Requests.AppendAuditLog(portal.AuditLog{
		ActorSub:   user.Sub,
		Action:     "reject_permission_request",
		TargetType: "permission_request",
		TargetID:   reviewed.ID,
		Metadata: map[string]any{
			"requesterSub": reviewed.RequesterSub,
			"roleId":       reviewed.RoleID,
			"roleName":     reviewed.RoleName,
		},
	})
	c.JSON(http.StatusOK, gin.H{"request": reviewed})
}

func (s *Server) requireSession(c *gin.Context) {
	session, ok := s.deps.Session.Get(c.Request)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":    "unauthorized",
			"loginUrl": "/auth/login",
		})
		c.Abort()
		return
	}
	c.Set(sessionKey, session)
	c.Next()
}

func (s *Server) requireAdmin(c *gin.Context) {
	user, ok := s.currentUser(c)
	if !ok {
		c.Abort()
		return
	}
	if !hasRole(user, "liteyuki-account-admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		c.Abort()
		return
	}
	c.Next()
}

func (s *Server) globalLogoutURL(session auth.Session) string {
	postLogout := s.deps.Config.PostLogoutRedirectURI()
	if !s.deps.Config.IsAllowedRedirect(postLogout) {
		postLogout = s.deps.Config.AppBaseURL
	}

	values := url.Values{}
	values.Set("post_logout_redirect_uri", postLogout)
	if session.IDToken != "" {
		values.Set("id_token_hint", session.IDToken)
	}
	return strings.TrimRight(s.deps.Config.LogtoIssuer, "/") + "/session/end?" + values.Encode()
}

func mustSession(c *gin.Context) auth.Session {
	value, ok := c.Get(sessionKey)
	if !ok {
		return auth.Session{}
	}
	session, _ := value.(auth.Session)
	return session
}

func (s *Server) currentUser(c *gin.Context) (auth.User, bool) {
	session := mustSession(c)
	subject := sessionSubject(session)
	if s.deps.Management == nil {
		if s.deps.Config.DevAuthEnabled {
			return session.User, true
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management_api_not_configured"})
		return auth.User{}, false
	}

	managedUser, err := s.deps.Management.GetUser(c.Request.Context(), subject)
	if err != nil {
		s.deps.Logger.Warn("management user fetch failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return auth.User{}, false
	}

	roles, err := s.deps.Management.ListUserRoles(c.Request.Context(), subject)
	if err != nil {
		s.deps.Logger.Warn("management user roles fetch failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return auth.User{}, false
	}

	organizations, err := s.deps.Management.ListUserOrganizations(c.Request.Context(), subject)
	if err != nil {
		s.deps.Logger.Warn("management user organizations fetch failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return auth.User{}, false
	}

	return userFromManagement(managedUser, roles, organizations), true
}

func sessionSubject(session auth.Session) string {
	if session.Subject != "" {
		return session.Subject
	}
	return session.User.Sub
}

func (s *Server) loadAppCatalog() ([]portal.AppCatalogItem, error) {
	return portal.LoadAppCatalog(s.deps.Config.AppCatalogPath)
}

func (s *Server) loadAnnouncements() ([]portal.Announcement, error) {
	return portal.LoadAnnouncements(s.deps.Config.AnnouncementsPath)
}

func userFromManagement(user logto.ManagementUser, roles []logto.ManagementRole, organizations []logto.ManagementOrganization) auth.User {
	result := auth.User{
		Sub:               user.ID,
		Email:             user.PrimaryEmail,
		Name:              user.Name,
		PreferredUsername: user.Username,
		Picture:           user.Avatar,
		Roles:             make([]string, 0, len(roles)),
		Organizations:     make([]string, 0, len(organizations)*2),
		OrganizationRoles: []string{},
	}

	if result.PreferredUsername == "" {
		result.PreferredUsername = stringMapValue(user.Profile, "preferredUsername")
	}
	for _, role := range roles {
		if role.Name != "" {
			result.Roles = append(result.Roles, role.Name)
		}
	}
	for _, organization := range organizations {
		if organization.ID != "" {
			result.Organizations = append(result.Organizations, organization.ID)
		}
		if organization.Name != "" && organization.Name != organization.ID {
			result.Organizations = append(result.Organizations, organization.Name)
		}
		for _, role := range organization.OrganizationRoles {
			if role.Name == "" {
				continue
			}
			if organization.ID != "" {
				result.OrganizationRoles = append(result.OrganizationRoles, organization.ID+":"+role.Name)
			}
			if organization.Name != "" && organization.Name != organization.ID {
				result.OrganizationRoles = append(result.OrganizationRoles, organization.Name+":"+role.Name)
			}
		}
	}
	return result
}

func hasRole(user auth.User, role string) bool {
	for _, value := range user.Roles {
		if value == role {
			return true
		}
	}
	return false
}

func reviewNote(c *gin.Context) string {
	var request struct {
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		return ""
	}
	return strings.TrimSpace(request.Note)
}

func (s *Server) resolveRoleID(ctx context.Context, roleName string) (string, error) {
	roles, err := s.deps.Management.ListRoles(ctx)
	if err != nil {
		return "", err
	}
	for _, role := range roles {
		if role.Name == roleName {
			return role.ID, nil
		}
	}
	return "", os.ErrNotExist
}

func statusForStoreError(err error) int {
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound
	}
	if errors.Is(err, portal.ErrRequestNotPending) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func isAllowedUserApplicationType(value string) bool {
	return value == "SPA" || value == "Traditional"
}

func allAbsoluteURLs(values []string) bool {
	for _, value := range cleanStrings(values) {
		parsed, err := url.ParseRequestURI(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return false
		}
	}
	return true
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
