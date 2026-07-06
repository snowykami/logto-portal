package logto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ManagementClient struct {
	apiBaseURL   string
	tokenURL     string
	clientID     string
	clientSecret string
	resource     string
	scope        string
	client       *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

type ManagementConfig struct {
	APIBaseURL   string
	Issuer       string
	ClientID     string
	ClientSecret string
	Resource     string
	Scope        string
}

type ManagementProfileUpdate struct {
	Username *string `json:"username,omitempty"`
	Name     *string `json:"name,omitempty"`
	Avatar   *string `json:"avatar,omitempty"`
}

func NewManagementClient(cfg ManagementConfig) *ManagementClient {
	return &ManagementClient{
		apiBaseURL:   strings.TrimRight(cfg.APIBaseURL, "/"),
		tokenURL:     strings.TrimRight(cfg.Issuer, "/") + "/token",
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		resource:     cfg.Resource,
		scope:        cfg.Scope,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *ManagementClient) UpdateUser(ctx context.Context, userID string, update ManagementProfileUpdate) (map[string]any, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	return c.doJSON(ctx, http.MethodPatch, "/api/users/"+url.PathEscape(userID), token, update)
}

func (c *ManagementClient) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", c.clientID)
	values.Set("client_secret", c.clientSecret)
	values.Set("resource", c.resource)
	if c.scope != "" {
		values.Set("scope", c.scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("logto management token request returned %s", resp.Status)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("logto management token response missing access_token")
	}
	if payload.ExpiresIn <= 0 {
		payload.ExpiresIn = 3600
	}

	c.accessToken = payload.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return c.accessToken, nil
}

func (c *ManagementClient) doJSON(ctx context.Context, method, path, accessToken string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiBaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("logto management api returned %s", resp.Status)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
