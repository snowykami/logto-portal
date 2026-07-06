package http

import (
	"net/http"
	"net/url"
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
	session := mustSession(c)
	c.JSON(http.StatusOK, gin.H{
		"user": session.User,
	})
}

func (s *Server) permissions(c *gin.Context) {
	user := mustSession(c).User
	c.JSON(http.StatusOK, gin.H{
		"roles":             user.Roles,
		"organizations":     user.Organizations,
		"organizationRoles": user.OrganizationRoles,
		"isAdmin":           hasRole(user, "liteyuki-account-admin"),
	})
}

func (s *Server) appCatalog(c *gin.Context) {
	user := mustSession(c).User
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

	catalog := portal.MergeManagedApps(managedApps, s.deps.Catalog)
	c.JSON(http.StatusOK, gin.H{
		"applications": portal.FilterApps(catalog, user),
		"source":       "management",
	})
}

func (s *Server) announcements(c *gin.Context) {
	user := mustSession(c).User
	c.JSON(http.StatusOK, gin.H{
		"announcements": portal.FilterAnnouncements(s.deps.Announcements, user, time.Now()),
	})
}

func (s *Server) markAnnouncementRead(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"read": true,
		"mode": "stateless",
	})
}

func (s *Server) updateProfile(c *gin.Context) {
	session := mustSession(c)
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
	result, err := s.deps.Management.UpdateUser(c.Request.Context(), session.User.Sub, update)
	if err != nil {
		s.deps.Logger.Warn("profile update failed", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "management_api_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": result})
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
	c.JSON(http.StatusOK, gin.H{"announcements": s.deps.Announcements})
}

func (s *Server) adminAppCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"applications": s.deps.Catalog})
}

func (s *Server) auditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"auditLogs": []gin.H{}})
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
	session := mustSession(c)
	if !hasRole(session.User, "liteyuki-account-admin") {
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

func hasRole(user auth.User, role string) bool {
	for _, value := range user.Roles {
		if value == role {
			return true
		}
	}
	return false
}
