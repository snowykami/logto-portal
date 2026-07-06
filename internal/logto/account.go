package logto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AccountClient struct {
	baseURL string
	client  *http.Client
}

func NewAccountClient(baseURL string) *AccountClient {
	return &AccountClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *AccountClient) ListSessions(ctx context.Context, accessToken string) (map[string]any, error) {
	return c.doJSON(ctx, http.MethodGet, "/api/my-account/sessions", accessToken, nil)
}

func (c *AccountClient) DeleteSession(ctx context.Context, accessToken string, sessionID string) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/api/my-account/sessions/"+sessionID, accessToken, nil)
	return err
}

func (c *AccountClient) ListApplications(ctx context.Context, accessToken string) (map[string]any, error) {
	return c.doJSON(ctx, http.MethodGet, "/api/my-account/applications", accessToken, nil)
}

func (c *AccountClient) doJSON(ctx context.Context, method, path, accessToken string, body any) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
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
		return nil, fmt.Errorf("logto account api returned %s", resp.Status)
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
