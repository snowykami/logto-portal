package auth

import (
	"context"
	"errors"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/liteyuki/yuki-id-portal/internal/config"
	"golang.org/x/oauth2"
)

type OIDCClient struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth2   oauth2.Config
}

func NewOIDCClient(ctx context.Context, cfg config.Config) (*OIDCClient, error) {
	provider, err := oidc.NewProvider(ctx, cfg.LogtoIssuer)
	if err != nil {
		return nil, err
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.LogtoClientID,
		ClientSecret: cfg.LogtoClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURI(),
		Scopes: []string{
			oidc.ScopeOpenID,
			"profile",
			"email",
			"role",
			"urn:logto:scope:organizations",
			"urn:logto:scope:organization_roles",
		},
	}

	return &OIDCClient{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{
			ClientID: cfg.LogtoClientID,
		}),
		oauth2: oauth2Config,
	}, nil
}

func (c *OIDCClient) AuthCodeURL(state, nonce string) string {
	return c.oauth2.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce))
}

func (c *OIDCClient) Exchange(ctx context.Context, code string, expectedNonce string) (Session, error) {
	token, err := c.oauth2.Exchange(ctx, code)
	if err != nil {
		return Session{}, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return Session{}, errors.New("missing id_token")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Session{}, err
	}
	if idToken.Nonce != expectedNonce {
		return Session{}, errors.New("invalid nonce")
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return Session{}, err
	}

	subject := stringClaim(claims, "sub")
	if subject == "" {
		return Session{}, errors.New("missing sub claim")
	}

	return Session{
		Subject:      subject,
		AccessToken:  token.AccessToken,
		IDToken:      rawIDToken,
		RefreshToken: token.RefreshToken,
	}, nil
}

func stringClaim(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return value
}
