package auth

import (
	"sync"
	"time"
)

type LoginState struct {
	Nonce     string
	ExpiresAt time.Time
}

type StateStore struct {
	mu     sync.Mutex
	ttl    time.Duration
	states map[string]LoginState
}

func NewStateStore(ttl time.Duration) *StateStore {
	return &StateStore{
		ttl:    ttl,
		states: make(map[string]LoginState),
	}
}

func (s *StateStore) Create() (string, string, error) {
	state, err := randomID()
	if err != nil {
		return "", "", err
	}
	nonce, err := randomID()
	if err != nil {
		return "", "", err
	}

	s.mu.Lock()
	s.states[state] = LoginState{
		Nonce:     nonce,
		ExpiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()
	return state, nonce, nil
}

func (s *StateStore) Consume(state string) (LoginState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	loginState, ok := s.states[state]
	if !ok {
		return LoginState{}, false
	}
	delete(s.states, state)
	if time.Now().After(loginState.ExpiresAt) {
		return LoginState{}, false
	}
	return loginState, true
}
