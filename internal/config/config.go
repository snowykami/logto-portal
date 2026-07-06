package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Port                string
	AppBaseURL          string
	LogtoIssuer         string
	LogtoAPIBaseURL     string
	LogtoClientID       string
	LogtoClientSecret   string
	SessionCookieName   string
	CookieSecure        bool
	AppCatalogPath      string
	AnnouncementsPath   string
	SupportEmail        string
	DevAuthEnabled      bool
	AllowedRedirectURIs map[string]struct{}
}

func Load() (Config, error) {
	appBaseURL := env("APP_BASE_URL", "http://localhost:8080")
	logtoIssuer := strings.TrimRight(env("LOGTO_ISSUER", "https://auth.liteyuki.org/oidc"), "/")
	logtoAPIBaseURL := env("LOGTO_API_BASE_URL", strings.TrimSuffix(logtoIssuer, "/oidc"))

	redirectURI := strings.TrimRight(appBaseURL, "/") + "/auth/callback"
	postLogoutURI := strings.TrimRight(appBaseURL, "/") + "/"
	allowedRedirects := map[string]struct{}{
		redirectURI:   {},
		postLogoutURI: {},
	}

	cfg := Config{
		Port:                env("PORT", "8080"),
		AppBaseURL:          appBaseURL,
		LogtoIssuer:         logtoIssuer,
		LogtoAPIBaseURL:     strings.TrimRight(logtoAPIBaseURL, "/"),
		LogtoClientID:       os.Getenv("LOGTO_CLIENT_ID"),
		LogtoClientSecret:   os.Getenv("LOGTO_CLIENT_SECRET"),
		SessionCookieName:   env("SESSION_COOKIE_NAME", "yp_session"),
		CookieSecure:        envBool("COOKIE_SECURE", strings.HasPrefix(appBaseURL, "https://")),
		AppCatalogPath:      env("APP_CATALOG_PATH", "configs/app-catalog.yaml"),
		AnnouncementsPath:   env("ANNOUNCEMENTS_PATH", "configs/announcements.yaml"),
		SupportEmail:        env("SUPPORT_EMAIL", "contact@liteyuki.org"),
		DevAuthEnabled:      envBool("PORTAL_DEV_AUTH", false),
		AllowedRedirectURIs: allowedRedirects,
	}

	if _, err := url.ParseRequestURI(cfg.AppBaseURL); err != nil {
		return Config{}, errors.New("APP_BASE_URL must be an absolute URL")
	}
	if _, err := url.ParseRequestURI(cfg.LogtoIssuer); err != nil {
		return Config{}, errors.New("LOGTO_ISSUER must be an absolute URL")
	}

	return cfg, nil
}

func (c Config) OIDCEnabled() bool {
	return c.LogtoClientID != "" && c.LogtoClientSecret != ""
}

func (c Config) RedirectURI() string {
	return strings.TrimRight(c.AppBaseURL, "/") + "/auth/callback"
}

func (c Config) PostLogoutRedirectURI() string {
	return strings.TrimRight(c.AppBaseURL, "/") + "/"
}

func (c Config) IsAllowedRedirect(uri string) bool {
	_, ok := c.AllowedRedirectURIs[uri]
	return ok
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
