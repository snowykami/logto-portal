package http

import (
	"embed"
	"io/fs"
	"log/slog"
	nethttp "net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/liteyuki/yuki-id-portal/internal/auth"
	"github.com/liteyuki/yuki-id-portal/internal/config"
	"github.com/liteyuki/yuki-id-portal/internal/logto"
	"github.com/liteyuki/yuki-id-portal/internal/portal"
)

type Dependencies struct {
	Config        config.Config
	Logger        *slog.Logger
	Session       *auth.SessionManager
	State         *auth.StateStore
	OIDC          *auth.OIDCClient
	Account       *logto.AccountClient
	Management    *logto.ManagementClient
	Catalog       []portal.AppCatalogItem
	Announcements []portal.Announcement
	Static        embed.FS
}

type Server struct {
	deps Dependencies
}

func NewRouter(deps Dependencies) nethttp.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	server := &Server{deps: deps}

	router.GET("/auth/login", server.login)
	router.GET("/auth/callback", server.callback)
	router.GET("/logout", server.logoutPage)
	router.GET("/api/healthz", server.health)
	router.GET("/api/support-info", server.supportInfo)

	api := router.Group("/api")
	api.Use(server.requireSession)
	{
		api.GET("/me", server.me)
		api.GET("/me/permissions", server.permissions)
		api.PATCH("/me/profile", server.updateProfile)
		api.GET("/me/applications", server.authorizedApplications)
		api.GET("/me/sessions", server.sessions)
		api.DELETE("/me/sessions/:id", server.deleteSession)
		api.POST("/me/logout", server.logout)
		api.POST("/me/logout-global", server.logoutGlobal)
		api.GET("/app-catalog", server.appCatalog)
		api.GET("/announcements", server.announcements)
		api.POST("/announcements/:id/read", server.markAnnouncementRead)
	}

	admin := router.Group("/api/admin")
	admin.Use(server.requireSession, server.requireAdmin)
	{
		admin.GET("/announcements", server.adminAnnouncements)
		admin.GET("/app-catalog", server.adminAppCatalog)
		admin.GET("/audit-logs", server.auditLogs)
	}

	router.NoRoute(server.spa)
	return router
}

func (s *Server) spa(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(nethttp.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	dist, err := fs.Sub(s.deps.Static, "dist")
	if err != nil {
		c.String(nethttp.StatusInternalServerError, "static files are unavailable")
		return
	}

	requestPath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
	if requestPath != "" && requestPath != "." {
		if file, err := dist.Open(requestPath); err == nil {
			_ = file.Close()
			nethttp.FileServer(nethttp.FS(dist)).ServeHTTP(c.Writer, c.Request)
			return
		}
	}

	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		c.String(nethttp.StatusInternalServerError, "index file is unavailable")
		return
	}
	c.Data(nethttp.StatusOK, "text/html; charset=utf-8", index)
}
