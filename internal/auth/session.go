package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	User         User      `json:"user"`
	AccessToken  string    `json:"-"`
	IDToken      string    `json:"-"`
	RefreshToken string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type User struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"emailVerified"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferredUsername"`
	Picture           string   `json:"picture"`
	Roles             []string `json:"roles"`
	Organizations     []string `json:"organizations"`
	OrganizationRoles []string `json:"organizationRoles"`
}

type SessionManager struct {
	mu         sync.RWMutex
	cookieName string
	secure     bool
	ttl        time.Duration
	sessions   map[string]Session
}

func NewSessionManager(cookieName string, secure bool, ttl time.Duration) *SessionManager {
	return &SessionManager{
		cookieName: cookieName,
		secure:     secure,
		ttl:        ttl,
		sessions:   make(map[string]Session),
	}
}

func (m *SessionManager) Create(w http.ResponseWriter, session Session) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	now := time.Now()
	session.CreatedAt = now
	session.ExpiresAt = now.Add(m.ttl)

	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    id,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (m *SessionManager) Get(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		return Session{}, false
	}

	m.mu.RLock()
	session, ok := m.sessions[cookie.Value]
	m.mu.RUnlock()
	if !ok || time.Now().After(session.ExpiresAt) {
		if ok {
			m.delete(cookie.Value)
		}
		return Session{}, false
	}
	return session, true
}

func (m *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(m.cookieName); err == nil {
		m.delete(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *SessionManager) delete(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func randomID() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
