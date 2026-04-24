package server

import (
	"encoding/json"
	"errors"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) mountWorkspaceMemberRoutes(r chi.Router) {
	r.Get("/workspaces/current/members", s.handleWorkspaceMembers)
	r.Post("/workspaces/current/members/invite-email", s.handleWorkspaceInviteMembers)
	r.Put("/workspaces/current/members/{memberID}/update-role", s.handleWorkspaceMemberRoleUpdate)
	r.Delete("/workspaces/current/members/{memberID}", s.handleWorkspaceMemberDelete)
	r.Post("/workspaces/current/members/send-owner-transfer-confirm-email", s.handleWorkspaceOwnerTransferConfirmEmail)
	r.Post("/workspaces/current/members/owner-transfer-check", s.handleWorkspaceOwnerTransferCheck)
	r.Post("/workspaces/current/members/{memberID}/owner-transfer", s.handleWorkspaceOwnerTransfer)
}

func (s *server) handleFeatures(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	memberCount := s.store.CountWorkspaceMembers(workspace.ID)
	appPage := s.store.ListApps(workspace.ID, state.AppListFilters{Page: 1, Limit: 1})
	datasetPage := s.store.ListDatasets(workspace.ID, state.DatasetListFilters{Page: 1, Limit: 1})
	education, _ := s.store.UserEducationStatus(user.ID, time.Now())

	writeJSON(w, http.StatusOK, map[string]any{
		"billing": map[string]any{
			"enabled": false,
			"subscription": map[string]any{
				"plan": workspace.Plan,
			},
		},
		"members": map[string]any{
			"size":  memberCount,
			"limit": 0,
		},
		"apps": map[string]any{
			"size":  appPage.Total,
			"limit": 0,
		},
		"vector_space": map[string]any{
			"size":  datasetPage.Total,
			"limit": 0,
		},
		"annotation_quota_limit": map[string]any{
			"size":  0,
			"limit": 0,
		},
		"documents_upload_quota": map[string]any{
			"size":  0,
			"limit": 0,
		},
		"docs_processing":                    "standard",
		"can_replace_logo":                   false,
		"model_load_balancing_enabled":       false,
		"dataset_operator_enabled":           true,
		"education":                          map[string]any{"enabled": true, "activated": education.IsStudent},
		"webapp_copyright_enabled":           false,
		"workspace_members":                  map[string]any{"size": memberCount, "limit": 0},
		"is_allow_transfer_workspace":        false,
		"knowledge_pipeline":                 map[string]any{"publish_enabled": true},
		"human_input_email_delivery_enabled": false,
		"user_role":                          user.Role,
	})
}

func (s *server) handleAccountIntegrates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": []any{},
	})
}

func (s *server) handleAccountEducationStatus(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	education, _ := s.store.UserEducationStatus(user.ID, time.Now())

	expireAt := any(nil)
	if education.ExpireAt > 0 {
		expireAt = education.ExpireAt
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"is_student":    education.IsStudent,
		"allow_refresh": education.AllowRefresh,
		"expire_at":     expireAt,
	})
}

func (s *server) handleWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	users, invitations := s.store.ListWorkspaceMembers(workspace.ID)
	slices.SortFunc(users, func(a, b state.User) int {
		if weight := workspaceRoleWeight(a.Role) - workspaceRoleWeight(b.Role); weight != 0 {
			return weight
		}
		if a.LastActiveAt != b.LastActiveAt {
			if parseRFC3339Unix(a.LastActiveAt) > parseRFC3339Unix(b.LastActiveAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Email, b.Email)
	})
	slices.SortFunc(invitations, func(a, b state.WorkspaceInvitation) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(a.Email, b.Email)
		}
		if a.CreatedAt > b.CreatedAt {
			return -1
		}
		return 1
	})

	accounts := make([]map[string]any, 0, len(users)+len(invitations))
	for _, user := range users {
		accounts = append(accounts, activeWorkspaceMemberPayload(user))
	}
	for _, invitation := range invitations {
		accounts = append(accounts, pendingWorkspaceInvitationPayload(invitation))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
	})
}

func (s *server) handleWorkspaceInviteMembers(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !canManageWorkspaceMembers(user.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners or admins can invite members.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Emails   []string `json:"emails"`
		Role     string   `json:"role"`
		Language string   `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if len(payload.Emails) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "At least one email is required.")
		return
	}
	if !canAssignWorkspaceRole(user.Role, "", payload.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "The current workspace role cannot assign the requested member role.")
		return
	}

	results, err := s.store.InviteWorkspaceMembers(workspace.ID, user, payload.Emails, payload.Role, time.Now())
	if err != nil {
		if errors.Is(err, state.ErrWorkspaceRoleInvalid) {
			writeError(w, http.StatusBadRequest, "invalid_request", "Workspace role is invalid.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to invite workspace members.")
		return
	}

	invitationResults := make([]map[string]any, 0, len(results))
	for _, item := range results {
		if item.Status == "success" {
			invitationResults = append(invitationResults, map[string]any{
				"status": "success",
				"email":  item.Email,
				"url":    workspaceInvitationURL(item.Invitation),
			})
			continue
		}
		invitationResults = append(invitationResults, map[string]any{
			"status":  "failed",
			"email":   item.Email,
			"message": item.Message,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":             "success",
		"invitation_results": invitationResults,
	})
}

func (s *server) handleWorkspaceMemberRoleUpdate(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !canManageWorkspaceMembers(user.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners or admins can update workspace members.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	memberID := chi.URLParam(r, "memberID")
	targetRole, found, active := s.workspaceMemberRole(workspace.ID, memberID)
	if !found {
		writeError(w, http.StatusNotFound, "member_not_found", "Workspace member not found.")
		return
	}
	if memberID == user.ID {
		writeError(w, http.StatusBadRequest, "invalid_request", "You cannot change your own workspace role.")
		return
	}
	if !active && user.Role != "owner" {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners can update pending invitations.")
		return
	}
	if !canAssignWorkspaceRole(user.Role, targetRole, payload.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "The current workspace role cannot assign the requested member role.")
		return
	}

	if err := s.store.UpdateWorkspaceMemberRole(workspace.ID, memberID, payload.Role); err != nil {
		switch {
		case errors.Is(err, state.ErrWorkspaceRoleInvalid):
			writeError(w, http.StatusBadRequest, "invalid_request", "Workspace role is invalid.")
		case errors.Is(err, state.ErrWorkspaceMemberNotFound):
			writeError(w, http.StatusNotFound, "member_not_found", "Workspace member not found.")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update workspace member role.")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkspaceMemberDelete(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !canManageWorkspaceMembers(user.Role) {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners or admins can remove workspace members.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	memberID := chi.URLParam(r, "memberID")
	targetRole, found, _ := s.workspaceMemberRole(workspace.ID, memberID)
	if !found {
		writeError(w, http.StatusNotFound, "member_not_found", "Workspace member not found.")
		return
	}
	if memberID == user.ID {
		writeError(w, http.StatusBadRequest, "invalid_request", "You cannot remove yourself from the current workspace.")
		return
	}
	if !canDeleteWorkspaceMember(user.Role, targetRole) {
		writeError(w, http.StatusForbidden, "forbidden", "The current workspace role cannot remove this member.")
		return
	}

	if err := s.store.DeleteWorkspaceMemberOrInvitation(workspace.ID, memberID); err != nil {
		switch {
		case errors.Is(err, state.ErrWorkspaceOwnerRemovalForbidden):
			writeError(w, http.StatusForbidden, "forbidden", "Workspace owners cannot be removed.")
		case errors.Is(err, state.ErrWorkspaceMemberNotFound):
			writeError(w, http.StatusNotFound, "member_not_found", "Workspace member not found.")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to remove workspace member.")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkspaceOwnerTransferConfirmEmail(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user.Role != "owner" {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners can transfer workspace ownership.")
		return
	}

	challenge, err := s.authFlows.Issue(authFlowOwnerTransferPending, user.Email, user.ID, "", time.Now())
	if err != nil {
		writeAuthFlowError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"data":   challenge.Token,
	})
}

func (s *server) handleWorkspaceOwnerTransferCheck(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user.Role != "owner" {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners can transfer workspace ownership.")
		return
	}

	var payload struct {
		Code  string `json:"code"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	challenge, ok, err := s.authFlows.Promote(payload.Token, authFlowOwnerTransferPending, authFlowOwnerTransferVerified, user.Email, user.ID, "", normalizedVerificationCode(payload.Code), time.Now())
	if err != nil {
		writeAuthFlowError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result":   "success",
		"is_valid": ok,
		"email":    user.Email,
		"token":    firstNonEmpty(challenge.Token),
	})
}

func (s *server) handleWorkspaceOwnerTransfer(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user.Role != "owner" {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners can transfer workspace ownership.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	memberID := chi.URLParam(r, "memberID")
	if memberID == user.ID {
		writeError(w, http.StatusBadRequest, "invalid_request", "The current owner cannot transfer ownership to themselves.")
		return
	}
	targetRole, found, active := s.workspaceMemberRole(workspace.ID, memberID)
	if !found || !active || targetRole == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Ownership can only be transferred to an active workspace member.")
		return
	}
	challenge, valid, err := s.authFlows.Consume(payload.Token, authFlowOwnerTransferVerified, time.Now())
	if err != nil {
		writeAuthFlowError(w)
		return
	}
	if !valid || challenge.UserID != user.ID {
		writeError(w, http.StatusBadRequest, "invalid_request", "The ownership transfer verification token is invalid.")
		return
	}

	if err := s.store.TransferWorkspaceOwnership(workspace.ID, user.ID, memberID); err != nil {
		if errors.Is(err, state.ErrWorkspaceOwnershipTransferError) {
			writeError(w, http.StatusBadRequest, "invalid_request", "Workspace ownership transfer is invalid.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to transfer workspace ownership.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":   "success",
		"is_valid": true,
		"email":    challenge.Email,
		"token":    challenge.Token,
	})
}

func (s *server) handleInvitationCheck(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	invitation, ok := s.store.GetWorkspaceInvitationByToken(token)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": false,
		})
		return
	}

	workspace, ok := s.store.GetWorkspace(invitation.WorkspaceID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": false,
		})
		return
	}

	if workspaceID := strings.TrimSpace(r.URL.Query().Get("workspace_id")); workspaceID != "" && workspaceID != invitation.WorkspaceID {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": false,
		})
		return
	}
	if email := strings.TrimSpace(r.URL.Query().Get("email")); email != "" && !strings.EqualFold(email, invitation.Email) {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":   "success",
			"is_valid": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":   "success",
		"is_valid": true,
		"data": map[string]any{
			"workspace_name": workspace.Name,
			"email":          invitation.Email,
			"workspace_id":   invitation.WorkspaceID,
		},
	})
}

func (s *server) handleInvitationActivate(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token             string `json:"token"`
		Name              string `json:"name"`
		InterfaceLanguage string `json:"interface_language"`
		Timezone          string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	user, _, err := s.store.ActivateWorkspaceInvitation(payload.Token, payload.Name, payload.InterfaceLanguage, payload.Timezone, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, state.ErrWorkspaceInvitationNotFound):
			writeError(w, http.StatusBadRequest, "invalid_invitation", "The invitation token is invalid.")
		case errors.Is(err, state.ErrWorkspaceMemberAlreadyExists):
			writeError(w, http.StatusConflict, "member_already_exists", "The invited email is already active.")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to activate workspace invitation.")
		}
		return
	}

	session := s.sessions.Issue(user.ID)
	s.setAuthCookies(w, session)
	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
	})
}

func (s *server) workspaceMemberRole(workspaceID, memberID string) (string, bool, bool) {
	users, invitations := s.store.ListWorkspaceMembers(workspaceID)
	for _, user := range users {
		if user.ID == memberID {
			return user.Role, true, true
		}
	}
	for _, invitation := range invitations {
		if invitation.ID == memberID {
			return invitation.Role, true, false
		}
	}
	return "", false, false
}

func activeWorkspaceMemberPayload(user state.User) map[string]any {
	return map[string]any{
		"id":             user.ID,
		"name":           firstNonEmpty(strings.TrimSpace(user.Name), invitationDisplayName(user.Email)),
		"email":          user.Email,
		"avatar":         user.Avatar,
		"avatar_url":     nilIfEmpty(user.AvatarURL),
		"last_login_at":  parseRFC3339Unix(user.LastLoginAt),
		"last_active_at": parseRFC3339Unix(user.LastActiveAt),
		"created_at":     parseRFC3339Unix(user.CreatedAt),
		"status":         "active",
		"role":           firstNonEmpty(user.Role, "normal"),
	}
}

func pendingWorkspaceInvitationPayload(invitation state.WorkspaceInvitation) map[string]any {
	return map[string]any{
		"id":             invitation.ID,
		"name":           invitationDisplayName(invitation.Email),
		"email":          invitation.Email,
		"avatar":         "",
		"avatar_url":     nil,
		"last_login_at":  nil,
		"last_active_at": invitation.CreatedAt,
		"created_at":     invitation.CreatedAt,
		"status":         "pending",
		"role":           firstNonEmpty(invitation.Role, "normal"),
	}
}

func workspaceInvitationURL(invitation state.WorkspaceInvitation) string {
	query := neturl.Values{}
	query.Set("workspace_id", invitation.WorkspaceID)
	query.Set("email", invitation.Email)
	query.Set("token", invitation.ActivationToken)
	return "/activate?" + query.Encode()
}

func canManageWorkspaceMembers(role string) bool {
	return role == "owner" || role == "admin"
}

func canAssignWorkspaceRole(actorRole, currentRole, targetRole string) bool {
	switch actorRole {
	case "owner":
		return isWorkspaceAssignableRole(targetRole) && currentRole != "owner"
	case "admin":
		if currentRole == "owner" || currentRole == "admin" {
			return false
		}
		return targetRole == "editor" || targetRole == "normal" || targetRole == "dataset_operator"
	default:
		return false
	}
}

func canDeleteWorkspaceMember(actorRole, currentRole string) bool {
	if currentRole == "owner" {
		return false
	}
	if actorRole == "owner" {
		return true
	}
	return currentRole != "admin"
}

func isWorkspaceAssignableRole(role string) bool {
	switch strings.TrimSpace(role) {
	case "admin", "editor", "normal", "dataset_operator":
		return true
	default:
		return false
	}
}

func workspaceRoleWeight(role string) int {
	switch role {
	case "owner":
		return 0
	case "admin":
		return 1
	case "editor":
		return 2
	case "dataset_operator":
		return 3
	default:
		return 4
	}
}

func parseRFC3339Unix(value string) int64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}

func invitationDisplayName(email string) string {
	local := strings.TrimSpace(strings.Split(strings.ToLower(email), "@")[0])
	if local == "" {
		return "Invited Member"
	}
	local = strings.ReplaceAll(local, ".", " ")
	local = strings.ReplaceAll(local, "_", " ")
	parts := strings.Fields(local)
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return firstNonEmpty(strings.Join(parts, " "), "Invited Member")
}

func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
