package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleEmailRegisterSend(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email is required.")
		return
	}
	if _, exists := s.store.FindUserByEmail(email); exists {
		writeError(w, http.StatusBadRequest, "email_already_in_use", "Email already in use.")
		return
	}
	if _, ok := s.store.PrimaryWorkspace(); !ok {
		writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
		return
	}

	record := s.authFlows.Issue(authFlowRegisterPending, email, "", "", time.Now())
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"data":   record.Token,
	})
}

func (s *server) handleEmailRegisterValidity(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	next, ok := s.authFlows.Promote(payload.Token, authFlowRegisterPending, authFlowRegisterVerified, payload.Email, "", "", payload.Code, time.Now())
	writeJSON(w, http.StatusOK, map[string]any{
		"is_valid": ok,
		"token":    firstNonEmpty(next.Token),
	})
}

func (s *server) handleEmailRegister(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		PasswordConfirm string `json:"password_confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if payload.NewPassword == "" || payload.NewPassword != payload.PasswordConfirm {
		writeError(w, http.StatusBadRequest, "invalid_request", "Passwords do not match.")
		return
	}

	record, ok := s.authFlows.Consume(payload.Token, authFlowRegisterVerified, time.Now())
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request", "The email registration token is invalid.")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to secure the password.")
		return
	}

	workspace, ok := s.store.PrimaryWorkspace()
	if !ok {
		writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
		return
	}

	user, err := s.store.CreateWorkspaceUser(workspace.ID, state.CreateWorkspaceUserInput{
		Email:        record.Email,
		PasswordHash: string(passwordHash),
		Role:         "normal",
	}, time.Now())
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusBadRequest, "email_already_in_use", "Email already in use.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create account.")
		return
	}

	session := s.sessions.Issue(user.ID)
	s.setAuthCookies(w, session)
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"data":   map[string]any{},
	})
}

func (s *server) handleForgotPasswordSend(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email is required.")
		return
	}
	if _, exists := s.store.FindUserByEmail(email); !exists {
		writeError(w, http.StatusBadRequest, "account_not_found", "Account not found.")
		return
	}

	record := s.authFlows.Issue(authFlowForgotPending, email, "", "", time.Now())
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"data":   record.Token,
	})
}

func (s *server) handleForgotPasswordValidity(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if strings.TrimSpace(payload.Code) != "" || strings.TrimSpace(payload.Email) != "" {
		next, ok := s.authFlows.Promote(payload.Token, authFlowForgotPending, authFlowForgotVerified, payload.Email, "", "", payload.Code, time.Now())
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": ok,
			"token":    firstNonEmpty(next.Token),
		})
		return
	}

	record, ok := s.authFlows.Get(payload.Token, authFlowForgotVerified, time.Now())
	if !ok {
		record, ok = s.authFlows.Get(payload.Token, authFlowForgotPending, time.Now())
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result":   "success",
		"is_valid": ok,
		"email":    firstNonEmpty(record.Email),
	})
}

func (s *server) handleForgotPasswordReset(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token           string `json:"token"`
		NewPassword     string `json:"new_password"`
		PasswordConfirm string `json:"password_confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if payload.NewPassword == "" || payload.NewPassword != payload.PasswordConfirm {
		writeError(w, http.StatusBadRequest, "invalid_request", "Passwords do not match.")
		return
	}

	record, ok := s.authFlows.Consume(payload.Token, authFlowForgotVerified, time.Now())
	if !ok {
		record, ok = s.authFlows.Consume(payload.Token, authFlowForgotPending, time.Now())
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request", "The password reset token is invalid.")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to secure the password.")
		return
	}

	if _, err := s.store.UpdateUserPasswordByEmail(record.Email, string(passwordHash), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "account_not_found", "Account not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
	})
}

func (s *server) handleAccountInit(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var payload struct {
		InvitationCode    string `json:"invitation_code"`
		InterfaceLanguage string `json:"interface_language"`
		Timezone          string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateUserAccountInit(user.ID, payload.InterfaceLanguage, payload.Timezone, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update account setup.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
	})
}

func (s *server) handleAccountChangeEmailSend(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var payload struct {
		Email string `json:"email"`
		Phase string `json:"phase"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	switch strings.TrimSpace(payload.Phase) {
	case "old_email":
		if email == "" || !strings.EqualFold(email, user.Email) {
			writeError(w, http.StatusBadRequest, "invalid_request", "Original email verification is invalid.")
			return
		}
		record := s.authFlows.Issue(authFlowChangeOldPending, user.Email, user.ID, "", time.Now())
		writeJSON(w, http.StatusOK, map[string]any{
			"result": "success",
			"data":   record.Token,
		})
	case "new_email":
		oldRecord, ok := s.authFlows.Get(payload.Token, authFlowChangeOldVerified, time.Now())
		if !ok || oldRecord.UserID != user.ID {
			writeError(w, http.StatusBadRequest, "invalid_request", "Original email verification is required.")
			return
		}
		if email == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "New email is required.")
			return
		}
		if existing, exists := s.store.FindUserByEmail(email); exists && existing.ID != user.ID {
			writeError(w, http.StatusBadRequest, "email_already_in_use", "Email already in use.")
			return
		}
		record := s.authFlows.Issue(authFlowChangeNewPending, user.Email, user.ID, email, time.Now())
		writeJSON(w, http.StatusOK, map[string]any{
			"result": "success",
			"data":   record.Token,
		})
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "Email verification phase is invalid.")
	}
}

func (s *server) handleAccountChangeEmailValidity(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if next, ok := s.authFlows.Promote(payload.Token, authFlowChangeOldPending, authFlowChangeOldVerified, payload.Email, user.ID, "", payload.Code, time.Now()); ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": true,
			"email":    user.Email,
			"token":    next.Token,
		})
		return
	}

	if next, ok := s.authFlows.Promote(payload.Token, authFlowChangeNewPending, authFlowChangeReady, "", user.ID, payload.Email, payload.Code, time.Now()); ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": true,
			"email":    strings.ToLower(strings.TrimSpace(payload.Email)),
			"token":    next.Token,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":   "success",
		"is_valid": false,
		"email":    strings.ToLower(strings.TrimSpace(payload.Email)),
		"token":    "",
	})
}

func (s *server) handleAccountChangeEmailReset(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var payload struct {
		NewEmail string `json:"new_email"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	record, ok := s.authFlows.Consume(payload.Token, authFlowChangeReady, time.Now())
	if !ok || record.UserID != user.ID || !strings.EqualFold(record.NewEmail, strings.TrimSpace(payload.NewEmail)) {
		writeError(w, http.StatusBadRequest, "invalid_request", "The email change token is invalid.")
		return
	}

	if _, err := s.store.UpdateUserEmail(user.ID, payload.NewEmail, time.Now()); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusBadRequest, "email_already_in_use", "Email already in use.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update account email.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
	})
}

func (s *server) handleAccountChangeEmailUnique(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if existing, exists := s.store.FindUserByEmail(email); exists && existing.ID != user.ID {
		writeError(w, http.StatusBadRequest, "email_already_in_use", "Email already in use.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
	})
}

func userFacingAuthError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return "authentication_failed"
	}
	return "internal_error"
}
