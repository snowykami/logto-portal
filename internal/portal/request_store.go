package portal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

var ErrRequestNotPending = errors.New("request is not pending")

const (
	RequestStatusPending  = "pending"
	RequestStatusApproved = "approved"
	RequestStatusRejected = "rejected"
)

type AppRequest struct {
	ID                     string     `json:"id"`
	RequesterSub           string     `json:"requesterSub"`
	RequesterEmail         string     `json:"requesterEmail"`
	Name                   string     `json:"name"`
	Type                   string     `json:"type"`
	Description            string     `json:"description"`
	RedirectURIs           []string   `json:"redirectUris"`
	PostLogoutRedirectURIs []string   `json:"postLogoutRedirectUris"`
	CORSAllowedOrigins     []string   `json:"corsAllowedOrigins"`
	PortalURL              string     `json:"portalUrl"`
	Reason                 string     `json:"reason"`
	Status                 string     `json:"status"`
	LogtoApplicationID     string     `json:"logtoApplicationId,omitempty"`
	ReviewerSub            string     `json:"reviewerSub,omitempty"`
	ReviewNote             string     `json:"reviewNote,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	ReviewedAt             *time.Time `json:"reviewedAt,omitempty"`
}

type PermissionRequest struct {
	ID             string     `json:"id"`
	RequesterSub   string     `json:"requesterSub"`
	RequesterEmail string     `json:"requesterEmail"`
	Kind           string     `json:"kind"`
	RoleID         string     `json:"roleId,omitempty"`
	RoleName       string     `json:"roleName,omitempty"`
	ApplicationID  string     `json:"applicationId,omitempty"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	ReviewerSub    string     `json:"reviewerSub,omitempty"`
	ReviewNote     string     `json:"reviewNote,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
}

type AuditLog struct {
	ID         string         `json:"id"`
	ActorSub   string         `json:"actorSub"`
	Action     string         `json:"action"`
	TargetType string         `json:"targetType"`
	TargetID   string         `json:"targetId"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type requestStoreData struct {
	AppRequests        []AppRequest        `json:"appRequests"`
	PermissionRequests []PermissionRequest `json:"permissionRequests"`
	AuditLogs          []AuditLog          `json:"auditLogs"`
}

type RequestStore struct {
	mu   sync.Mutex
	path string
	data requestStoreData
}

func NewRequestStore(path string) (*RequestStore, error) {
	store := &RequestStore{path: path}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *RequestStore) CreateAppRequest(request AppRequest) (AppRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request.ID = newRequestID("app")
	request.Status = RequestStatusPending
	request.CreatedAt = time.Now()
	s.data.AppRequests = append(s.data.AppRequests, request)
	return request, s.saveLocked()
}

func (s *RequestStore) ListAppRequests(requesterSub string) []AppRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := []AppRequest{}
	for _, request := range s.data.AppRequests {
		if requesterSub == "" || request.RequesterSub == requesterSub {
			result = append(result, request)
		}
	}
	slices.SortFunc(result, func(a, b AppRequest) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return result
}

func (s *RequestStore) GetAppRequest(id string) (AppRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, request := range s.data.AppRequests {
		if request.ID == id {
			return request, true
		}
	}
	return AppRequest{}, false
}

func (s *RequestStore) ReviewAppRequest(id string, status string, reviewerSub string, note string, logtoApplicationID string) (AppRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.data.AppRequests {
		if s.data.AppRequests[index].ID != id {
			continue
		}
		if s.data.AppRequests[index].Status != RequestStatusPending {
			return AppRequest{}, ErrRequestNotPending
		}
		now := time.Now()
		s.data.AppRequests[index].Status = status
		s.data.AppRequests[index].ReviewerSub = reviewerSub
		s.data.AppRequests[index].ReviewNote = note
		s.data.AppRequests[index].LogtoApplicationID = logtoApplicationID
		s.data.AppRequests[index].ReviewedAt = &now
		return s.data.AppRequests[index], s.saveLocked()
	}
	return AppRequest{}, os.ErrNotExist
}

func (s *RequestStore) CreatePermissionRequest(request PermissionRequest) (PermissionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request.ID = newRequestID("perm")
	request.Status = RequestStatusPending
	request.CreatedAt = time.Now()
	s.data.PermissionRequests = append(s.data.PermissionRequests, request)
	return request, s.saveLocked()
}

func (s *RequestStore) ListPermissionRequests(requesterSub string) []PermissionRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := []PermissionRequest{}
	for _, request := range s.data.PermissionRequests {
		if requesterSub == "" || request.RequesterSub == requesterSub {
			result = append(result, request)
		}
	}
	slices.SortFunc(result, func(a, b PermissionRequest) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return result
}

func (s *RequestStore) GetPermissionRequest(id string) (PermissionRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, request := range s.data.PermissionRequests {
		if request.ID == id {
			return request, true
		}
	}
	return PermissionRequest{}, false
}

func (s *RequestStore) ReviewPermissionRequest(id string, status string, reviewerSub string, note string) (PermissionRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for index := range s.data.PermissionRequests {
		if s.data.PermissionRequests[index].ID != id {
			continue
		}
		if s.data.PermissionRequests[index].Status != RequestStatusPending {
			return PermissionRequest{}, ErrRequestNotPending
		}
		now := time.Now()
		s.data.PermissionRequests[index].Status = status
		s.data.PermissionRequests[index].ReviewerSub = reviewerSub
		s.data.PermissionRequests[index].ReviewNote = note
		s.data.PermissionRequests[index].ReviewedAt = &now
		return s.data.PermissionRequests[index], s.saveLocked()
	}
	return PermissionRequest{}, os.ErrNotExist
}

func (s *RequestStore) AppendAuditLog(log AuditLog) (AuditLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.ID = newRequestID("audit")
	log.CreatedAt = time.Now()
	s.data.AuditLogs = append(s.data.AuditLogs, log)
	return log, s.saveLocked()
}

func (s *RequestStore) ListAuditLogs() []AuditLog {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := slices.Clone(s.data.AuditLogs)
	slices.SortFunc(result, func(a, b AuditLog) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return result
}

func (s *RequestStore) load() error {
	if s.path == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.data)
}

func (s *RequestStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func newRequestID(prefix string) string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return prefix + "-" + time.Now().Format("20060102150405")
	}
	return prefix + "-" + hex.EncodeToString(bytes[:])
}
