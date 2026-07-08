package logto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

type ManagementUser struct {
	ID           string         `json:"id"`
	Username     string         `json:"username"`
	PrimaryEmail string         `json:"primaryEmail"`
	Name         string         `json:"name"`
	Avatar       string         `json:"avatar"`
	Profile      map[string]any `json:"profile"`
	CustomData   map[string]any `json:"customData"`
}

type ManagementApplication struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Type                 string                 `json:"type"`
	OidcClientMetadata   map[string]any         `json:"oidcClientMetadata"`
	CustomClientMetadata map[string]any         `json:"customClientMetadata"`
	ProtectedAppMetadata map[string]any         `json:"protectedAppMetadata"`
	CustomData           map[string]any         `json:"customData"`
	IsThirdParty         bool                   `json:"isThirdParty"`
	CreatedAt            int64                  `json:"createdAt"`
	UpdatedAt            int64                  `json:"updatedAt"`
	AdditionalProperties map[string]interface{} `json:"-"`
}

type CreateApplicationRequest struct {
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`
	Description          string         `json:"description,omitempty"`
	OidcClientMetadata   map[string]any `json:"oidcClientMetadata,omitempty"`
	CustomClientMetadata map[string]any `json:"customClientMetadata,omitempty"`
	CustomData           map[string]any `json:"customData,omitempty"`
}

type ManagementRole struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

type ManagementOrganization struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	OrganizationRoles []ManagementRoleSummary `json:"organizationRoles"`
}

type ManagementRoleSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

func (c *ManagementClient) GetUser(ctx context.Context, userID string) (ManagementUser, error) {
	token, err := c.token(ctx)
	if err != nil {
		return ManagementUser{}, err
	}

	data, err := c.doRawJSON(ctx, http.MethodGet, "/api/users/"+url.PathEscape(userID), token, nil)
	if err != nil {
		return ManagementUser{}, err
	}

	var user ManagementUser
	if err := json.Unmarshal(data, &user); err != nil {
		return ManagementUser{}, err
	}
	return user, nil
}

func (c *ManagementClient) UpdateUser(ctx context.Context, userID string, update ManagementProfileUpdate) (map[string]any, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	return c.doJSON(ctx, http.MethodPatch, "/api/users/"+url.PathEscape(userID), token, update)
}

func (c *ManagementClient) CreateApplication(ctx context.Context, request CreateApplicationRequest) (map[string]any, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}
	return c.doJSON(ctx, http.MethodPost, "/api/applications", token, request)
}

func (c *ManagementClient) ListApplications(ctx context.Context) ([]ManagementApplication, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}

	const pageSize = 100
	apps := []ManagementApplication{}
	for page := 1; ; page++ {
		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("page_size", strconv.Itoa(pageSize))

		data, err := c.doRawJSON(ctx, http.MethodGet, "/api/applications?"+values.Encode(), token, nil)
		if err != nil {
			return nil, err
		}

		pageApps, err := decodeApplications(data)
		if err != nil {
			return nil, err
		}
		apps = append(apps, pageApps...)
		if len(pageApps) < pageSize {
			break
		}
	}

	return apps, nil
}

func (c *ManagementClient) ListUserRoles(ctx context.Context, userID string) ([]ManagementRole, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}

	data, err := c.doRawJSON(ctx, http.MethodGet, "/api/users/"+url.PathEscape(userID)+"/roles", token, nil)
	if err != nil {
		return nil, err
	}
	return decodeList[ManagementRole](data)
}

func (c *ManagementClient) ListUserOrganizations(ctx context.Context, userID string) ([]ManagementOrganization, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}

	data, err := c.doRawJSON(ctx, http.MethodGet, "/api/users/"+url.PathEscape(userID)+"/organizations", token, nil)
	if err != nil {
		return nil, err
	}
	return decodeList[ManagementOrganization](data)
}

func (c *ManagementClient) ListRoles(ctx context.Context) ([]ManagementRole, error) {
	token, err := c.token(ctx)
	if err != nil {
		return nil, err
	}

	data, err := c.doRawJSON(ctx, http.MethodGet, "/api/roles", token, nil)
	if err != nil {
		return nil, err
	}

	return decodeList[ManagementRole](data)
}

func (c *ManagementClient) AssignRolesToUser(ctx context.Context, userID string, roleIDs []string) error {
	token, err := c.token(ctx)
	if err != nil {
		return err
	}
	_, err = c.doJSON(ctx, http.MethodPost, "/api/users/"+url.PathEscape(userID)+"/roles", token, map[string]any{
		"roleIds": roleIDs,
	})
	return err
}

func (c *ManagementClient) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second)) {
		return c.accessToken, nil
	}

	values := url.Values{}
	values.Set("grant_type", "client_credentials")
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
	req.SetBasicAuth(c.clientID, c.clientSecret)

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
	data, err := c.doRawJSON(ctx, method, path, accessToken, body)
	if err != nil {
		return nil, err
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

func (c *ManagementClient) doRawJSON(ctx context.Context, method, path, accessToken string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiBaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
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
	return data, nil
}

func decodeApplications(data []byte) ([]ManagementApplication, error) {
	return decodeList[ManagementApplication](data)
}

func decodeList[T any](data []byte) ([]T, error) {
	var direct []T
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}

	var wrapped struct {
		Data []T `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}
