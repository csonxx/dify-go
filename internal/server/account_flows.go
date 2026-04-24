package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/langgenius/dify-go/internal/state"
)

const authFlowTTL = 30 * time.Minute

const (
	authFlowRegisterPending       = "register_pending"
	authFlowRegisterVerified      = "register_verified"
	authFlowEmailLoginPending     = "email_login_pending"
	authFlowEmailLoginReady       = "email_login_ready"
	authFlowForgotPending         = "forgot_password_pending"
	authFlowForgotVerified        = "forgot_password_verified"
	authFlowEducationVerify       = "education_verify"
	authFlowChangeOldPending      = "change_email_old_pending"
	authFlowChangeOldVerified     = "change_email_old_verified"
	authFlowChangeNewPending      = "change_email_new_pending"
	authFlowChangeReady           = "change_email_ready"
	authFlowOwnerTransferPending  = "owner_transfer_pending"
	authFlowOwnerTransferVerified = "owner_transfer_verified"
)

type authFlowManager struct {
	store *state.Store
}

type authFlowRecord struct {
	Token     string
	Kind      string
	Email     string
	UserID    string
	NewEmail  string
	ExpiresAt time.Time
}

func newAuthFlowManager(store *state.Store) *authFlowManager {
	return &authFlowManager{
		store: store,
	}
}

func writeAuthFlowError(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist authentication flow.")
}

func (m *authFlowManager) Issue(kind, email, userID, newEmail string, now time.Time) (authFlowRecord, error) {
	flow, err := m.store.IssueAuthFlow(state.AuthFlowInput{
		Kind:      kind,
		Email:     email,
		UserID:    userID,
		NewEmail:  newEmail,
		ExpiresAt: now.Add(authFlowTTL),
	}, now)
	if err != nil {
		return authFlowRecord{}, err
	}
	return authFlowRecordFromState(flow), nil
}

func (m *authFlowManager) Promote(token, expectKind, nextKind, email, userID, newEmail, code string, now time.Time) (authFlowRecord, bool, error) {
	current, ok, err := m.Get(token, expectKind, now)
	if err != nil {
		return authFlowRecord{}, false, err
	}
	if !ok {
		return authFlowRecord{}, false, nil
	}
	if email != "" && !strings.EqualFold(current.Email, strings.TrimSpace(email)) {
		return authFlowRecord{}, false, nil
	}
	if userID != "" && current.UserID != strings.TrimSpace(userID) {
		return authFlowRecord{}, false, nil
	}
	if newEmail != "" && !strings.EqualFold(current.NewEmail, strings.TrimSpace(newEmail)) {
		return authFlowRecord{}, false, nil
	}
	if len(strings.TrimSpace(code)) != 6 {
		return authFlowRecord{}, false, nil
	}

	next, promoted, err := m.store.PromoteAuthFlow(current.Token, expectKind, nextKind, newEmail, now.Add(authFlowTTL), now)
	if err != nil {
		return authFlowRecord{}, false, err
	}
	if !promoted {
		return authFlowRecord{}, false, nil
	}
	return authFlowRecordFromState(next), true, nil
}

func (m *authFlowManager) Get(token, expectKind string, now time.Time) (authFlowRecord, bool, error) {
	flow, ok, err := m.store.GetAuthFlow(token, expectKind, now)
	if err != nil {
		return authFlowRecord{}, false, err
	}
	if !ok {
		return authFlowRecord{}, false, nil
	}
	return authFlowRecordFromState(flow), true, nil
}

func (m *authFlowManager) Consume(token, expectKind string, now time.Time) (authFlowRecord, bool, error) {
	flow, ok, err := m.store.ConsumeAuthFlow(token, expectKind, now)
	if err != nil {
		return authFlowRecord{}, false, err
	}
	if !ok {
		return authFlowRecord{}, false, nil
	}
	return authFlowRecordFromState(flow), true, nil
}

func authFlowRecordFromState(flow state.AuthFlow) authFlowRecord {
	return authFlowRecord{
		Token:     flow.Token,
		Kind:      flow.Kind,
		Email:     flow.Email,
		UserID:    flow.UserID,
		NewEmail:  flow.NewEmail,
		ExpiresAt: time.Unix(flow.ExpiresAt, 0),
	}
}
