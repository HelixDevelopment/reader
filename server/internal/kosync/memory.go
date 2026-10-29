package kosync

import (
	"context"
	"sync"
)

// Memory is a deterministic test and embedded-development store. Production
// deployments use Redis so account and progress state survive restarts.
type Memory struct {
	mu       sync.RWMutex
	users    map[string]string
	progress map[string]map[string]Progress
}

func NewMemory() *Memory {
	return &Memory{users: map[string]string{}, progress: map[string]map[string]Progress{}}
}

func (m *Memory) CreateUser(_ context.Context, username, verifier string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[username]; ok {
		return ErrUserExists
	}
	m.users[username] = verifier
	return nil
}

func (m *Memory) UserVerifier(_ context.Context, username string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	verifier, ok := m.users[username]
	if !ok {
		return "", ErrUserNotFound
	}
	return verifier, nil
}

func (m *Memory) PutProgress(_ context.Context, username string, progress Progress) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress[username] == nil {
		m.progress[username] = map[string]Progress{}
	}
	existing, ok := m.progress[username][progress.Document]
	if !ok || progress.Percentage >= existing.Percentage {
		m.progress[username][progress.Document] = progress
	}
	return nil
}

func (m *Memory) GetProgress(_ context.Context, username, document string) (Progress, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	progress, ok := m.progress[username][document]
	if !ok {
		return Progress{}, ErrProgressNotFound
	}
	return progress, nil
}

func (m *Memory) Close() error { return nil }
