package server

import (
	"strings"
	"sync"
	"time"
)

const authFlowTTL = 30 * time.Minute

const (
	authFlowRegisterPending   = "register_pending"
	authFlowRegisterVerified  = "register_verified"
	authFlowForgotPending     = "forgot_password_pending"
	authFlowForgotVerified    = "forgot_password_verified"
	authFlowChangeOldPending  = "change_email_old_pending"
	authFlowChangeOldVerified = "change_email_old_verified"
	authFlowChangeNewPending  = "change_email_new_pending"
	authFlowChangeReady       = "change_email_ready"
)

type authFlowManager struct {
	mu      sync.Mutex
	records map[string]authFlowRecord
}

type authFlowRecord struct {
	Token     string
	Kind      string
	Email     string
	UserID    string
	NewEmail  string
	ExpiresAt time.Time
}

func newAuthFlowManager() *authFlowManager {
	return &authFlowManager{
		records: make(map[string]authFlowRecord),
	}
}

func (m *authFlowManager) Issue(kind, email, userID, newEmail string, now time.Time) authFlowRecord {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := authFlowRecord{
		Token:     generateRuntimeID("auth"),
		Kind:      strings.TrimSpace(kind),
		Email:     strings.ToLower(strings.TrimSpace(email)),
		UserID:    strings.TrimSpace(userID),
		NewEmail:  strings.ToLower(strings.TrimSpace(newEmail)),
		ExpiresAt: now.Add(authFlowTTL),
	}
	m.records[record.Token] = record
	return record
}

func (m *authFlowManager) Promote(token, expectKind, nextKind, email, userID, newEmail, code string, now time.Time) (authFlowRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.records[strings.TrimSpace(token)]
	if !ok {
		return authFlowRecord{}, false
	}
	if now.After(current.ExpiresAt) {
		delete(m.records, strings.TrimSpace(token))
		return authFlowRecord{}, false
	}
	if current.Kind != strings.TrimSpace(expectKind) {
		return authFlowRecord{}, false
	}
	if email != "" && !strings.EqualFold(current.Email, strings.TrimSpace(email)) {
		return authFlowRecord{}, false
	}
	if userID != "" && current.UserID != strings.TrimSpace(userID) {
		return authFlowRecord{}, false
	}
	if newEmail != "" && !strings.EqualFold(current.NewEmail, strings.TrimSpace(newEmail)) {
		return authFlowRecord{}, false
	}
	if len(strings.TrimSpace(code)) != 6 {
		return authFlowRecord{}, false
	}

	delete(m.records, current.Token)
	next := authFlowRecord{
		Token:     generateRuntimeID("auth"),
		Kind:      strings.TrimSpace(nextKind),
		Email:     current.Email,
		UserID:    current.UserID,
		NewEmail:  firstNonEmpty(strings.ToLower(strings.TrimSpace(newEmail)), current.NewEmail),
		ExpiresAt: now.Add(authFlowTTL),
	}
	m.records[next.Token] = next
	return next, true
}

func (m *authFlowManager) Get(token, expectKind string, now time.Time) (authFlowRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[strings.TrimSpace(token)]
	if !ok {
		return authFlowRecord{}, false
	}
	if now.After(record.ExpiresAt) {
		delete(m.records, strings.TrimSpace(token))
		return authFlowRecord{}, false
	}
	if record.Kind != strings.TrimSpace(expectKind) {
		return authFlowRecord{}, false
	}
	return record, true
}

func (m *authFlowManager) Consume(token, expectKind string, now time.Time) (authFlowRecord, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.records[strings.TrimSpace(token)]
	if !ok {
		return authFlowRecord{}, false
	}
	if now.After(record.ExpiresAt) {
		delete(m.records, strings.TrimSpace(token))
		return authFlowRecord{}, false
	}
	if record.Kind != strings.TrimSpace(expectKind) {
		return authFlowRecord{}, false
	}
	delete(m.records, strings.TrimSpace(token))
	return record, true
}
