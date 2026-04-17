package server

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type sessionTokens struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	CSRFToken    string
}

type sessionRecord struct {
	sessionTokens
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type sessionManager struct {
	mu           sync.RWMutex
	accessTTL    time.Duration
	refreshTTL   time.Duration
	accessIndex  map[string]sessionRecord
	refreshIndex map[string]sessionRecord
}

func newSessionManager(accessTTL, refreshTTL time.Duration) *sessionManager {
	return &sessionManager{
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
		accessIndex:  make(map[string]sessionRecord),
		refreshIndex: make(map[string]sessionRecord),
	}
}

func (m *sessionManager) Issue(userID string) sessionTokens {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := sessionRecord{
		sessionTokens: sessionTokens{
			UserID:       userID,
			AccessToken:  randomToken(),
			RefreshToken: randomToken(),
			CSRFToken:    randomToken(),
		},
		AccessExpiresAt:  time.Now().Add(m.accessTTL),
		RefreshExpiresAt: time.Now().Add(m.refreshTTL),
	}

	m.accessIndex[record.AccessToken] = record
	m.refreshIndex[record.RefreshToken] = record
	return record.sessionTokens
}

func (m *sessionManager) Get(accessToken string) (sessionTokens, bool) {
	m.mu.RLock()
	record, ok := m.accessIndex[accessToken]
	m.mu.RUnlock()
	if !ok {
		return sessionTokens{}, false
	}
	if time.Now().After(record.AccessExpiresAt) {
		m.Delete(record.AccessToken, record.RefreshToken)
		return sessionTokens{}, false
	}
	return record.sessionTokens, true
}

func (m *sessionManager) Refresh(refreshToken string) (sessionTokens, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.refreshIndex[refreshToken]
	if !ok {
		return sessionTokens{}, false
	}
	if time.Now().After(record.RefreshExpiresAt) {
		delete(m.refreshIndex, record.RefreshToken)
		delete(m.accessIndex, record.AccessToken)
		return sessionTokens{}, false
	}

	delete(m.refreshIndex, record.RefreshToken)
	delete(m.accessIndex, record.AccessToken)

	record.sessionTokens = sessionTokens{
		UserID:       record.UserID,
		AccessToken:  randomToken(),
		RefreshToken: randomToken(),
		CSRFToken:    randomToken(),
	}
	record.AccessExpiresAt = time.Now().Add(m.accessTTL)
	record.RefreshExpiresAt = time.Now().Add(m.refreshTTL)

	m.accessIndex[record.AccessToken] = record
	m.refreshIndex[record.RefreshToken] = record
	return record.sessionTokens, true
}

func (m *sessionManager) Delete(accessToken, refreshToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if accessToken != "" {
		delete(m.accessIndex, accessToken)
	}
	if refreshToken != "" {
		delete(m.refreshIndex, refreshToken)
	}
}

func randomToken() string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
