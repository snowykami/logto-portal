package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/liteyuki/yuki-id-portal/internal/auth"
	"github.com/liteyuki/yuki-id-portal/internal/config"
	portalhttp "github.com/liteyuki/yuki-id-portal/internal/http"
	"github.com/liteyuki/yuki-id-portal/internal/logto"
	"github.com/liteyuki/yuki-id-portal/internal/portal"
	"github.com/liteyuki/yuki-id-portal/internal/static"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	catalog, err := portal.LoadAppCatalog(cfg.AppCatalogPath)
	if err != nil {
		logger.Error("failed to load app catalog", "error", err)
		os.Exit(1)
	}

	announcements, err := portal.LoadAnnouncements(cfg.AnnouncementsPath)
	if err != nil {
		logger.Error("failed to load announcements", "error", err)
		os.Exit(1)
	}

	requestStore, err := portal.NewRequestStore(cfg.PortalRequestsPath)
	if err != nil {
		logger.Error("failed to load portal requests", "error", err)
		os.Exit(1)
	}

	sessionManager := auth.NewSessionManager(cfg.SessionCookieName, cfg.CookieSecure, 12*time.Hour)
	stateStore := auth.NewStateStore(10 * time.Minute)

	var oidcClient *auth.OIDCClient
	if cfg.OIDCEnabled() {
		oidcClient, err = auth.NewOIDCClient(ctx, cfg)
		if err != nil {
			logger.Error("failed to initialize oidc client", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Warn("OIDC is not configured; /auth/login will return a configuration error")
	}

	accountClient := logto.NewAccountClient(cfg.LogtoAPIBaseURL)
	var managementClient *logto.ManagementClient
	if cfg.ManagementAPIEnabled() {
		managementClient = logto.NewManagementClient(logto.ManagementConfig{
			APIBaseURL:   cfg.LogtoAPIBaseURL,
			Issuer:       cfg.LogtoIssuer,
			ClientID:     cfg.ManagementClientID,
			ClientSecret: cfg.ManagementClientSecret,
			Resource:     cfg.ManagementAPIResource,
			Scope:        cfg.ManagementAPIScope,
		})
	} else {
		logger.Warn("Logto Management API is not configured; profile updates are disabled")
	}
	router := portalhttp.NewRouter(portalhttp.Dependencies{
		Config:        cfg,
		Logger:        logger,
		Session:       sessionManager,
		State:         stateStore,
		OIDC:          oidcClient,
		Account:       accountClient,
		Management:    managementClient,
		Catalog:       catalog,
		Announcements: announcements,
		Requests:      requestStore,
		Static:        static.FS(),
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting Yuki ID Portal", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}
