package state

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	workspaceRoleOwner           = "owner"
	workspaceRoleAdmin           = "admin"
	workspaceRoleEditor          = "editor"
	workspaceRoleNormal          = "normal"
	workspaceRoleDatasetOperator = "dataset_operator"
	workspaceInvitationPending   = "pending"
)

var (
	ErrWorkspaceMemberNotFound         = errors.New("workspace member not found")
	ErrWorkspaceInvitationNotFound     = errors.New("workspace invitation not found")
	ErrWorkspaceRoleInvalid            = errors.New("workspace role is invalid")
	ErrWorkspaceOwnerRemovalForbidden  = errors.New("workspace owner cannot be removed")
	ErrWorkspaceOwnershipTransferError = errors.New("workspace ownership transfer is invalid")
	ErrWorkspaceMemberAlreadyExists    = errors.New("workspace member already exists")
)

type WorkspaceInvitation struct {
	ID              string `json:"id"`
	WorkspaceID     string `json:"workspace_id"`
	Email           string `json:"email"`
	Role            string `json:"role"`
	Status          string `json:"status"`
	InvitedBy       string `json:"invited_by"`
	CreatedAt       int64  `json:"created_at"`
	ActivationToken string `json:"activation_token"`
}

type WorkspaceInviteResult struct {
	Email      string
	Status     string
	Invitation WorkspaceInvitation
	Message    string
}

func normalizeWorkspaceInvitation(invitation *WorkspaceInvitation) {
	invitation.Email = strings.ToLower(strings.TrimSpace(invitation.Email))
	invitation.Role = normalizeWorkspaceRole(invitation.Role, false)
	if invitation.Role == "" {
		invitation.Role = workspaceRoleNormal
	}
	if strings.TrimSpace(invitation.Status) == "" {
		invitation.Status = workspaceInvitationPending
	}
}

func (s *Store) GetWorkspace(workspaceID string) (Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, workspace := range s.state.Workspaces {
		if workspace.ID == workspaceID {
			return workspace, true
		}
	}
	return Workspace{}, false
}

func (s *Store) ListWorkspaceMembers(workspaceID string) ([]User, []WorkspaceInvitation) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0)
	for _, user := range s.state.Users {
		if user.WorkspaceID != workspaceID {
			continue
		}
		users = append(users, user)
	}

	invitations := make([]WorkspaceInvitation, 0)
	for _, invitation := range s.state.WorkspaceInvitations {
		if invitation.WorkspaceID != workspaceID {
			continue
		}
		invitations = append(invitations, invitation)
	}

	return users, invitations
}

func (s *Store) CountWorkspaceMembers(workspaceID string) int {
	users, invitations := s.ListWorkspaceMembers(workspaceID)
	return len(users) + len(invitations)
}

func (s *Store) InviteWorkspaceMembers(workspaceID string, inviter User, emails []string, role string, now time.Time) ([]WorkspaceInviteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedRole := normalizeWorkspaceRole(role, false)
	if normalizedRole == "" {
		return nil, ErrWorkspaceRoleInvalid
	}

	results := make([]WorkspaceInviteResult, 0, len(emails))
	seen := make(map[string]struct{}, len(emails))
	changed := false

	for _, raw := range emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if _, duplicated := seen[email]; duplicated {
			results = append(results, WorkspaceInviteResult{
				Email:   email,
				Status:  "failed",
				Message: "Invitation email is duplicated in this batch.",
			})
			continue
		}
		seen[email] = struct{}{}

		if userIndex := s.workspaceUserIndexByEmailLocked(email); userIndex >= 0 {
			member := s.state.Users[userIndex]
			message := "This email is already a member of the current workspace."
			if member.WorkspaceID != workspaceID {
				message = "This email already belongs to another workspace."
			}
			results = append(results, WorkspaceInviteResult{
				Email:   email,
				Status:  "failed",
				Message: message,
			})
			continue
		}
		if invitationIndex := s.workspaceInvitationIndexByEmailLocked(workspaceID, email); invitationIndex >= 0 {
			results = append(results, WorkspaceInviteResult{
				Email:   email,
				Status:  "failed",
				Message: "An invitation for this email already exists.",
			})
			continue
		}

		invitation := WorkspaceInvitation{
			ID:              generateID("inv"),
			WorkspaceID:     workspaceID,
			Email:           email,
			Role:            normalizedRole,
			Status:          workspaceInvitationPending,
			InvitedBy:       inviter.ID,
			CreatedAt:       now.UTC().Unix(),
			ActivationToken: generateID("invite"),
		}
		s.state.WorkspaceInvitations = append(s.state.WorkspaceInvitations, invitation)
		results = append(results, WorkspaceInviteResult{
			Email:      email,
			Status:     "success",
			Invitation: invitation,
		})
		changed = true
	}

	if changed {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (s *Store) UpdateWorkspaceMemberRole(workspaceID, memberID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedRole := normalizeWorkspaceRole(role, false)
	if normalizedRole == "" {
		return ErrWorkspaceRoleInvalid
	}

	for i := range s.state.Users {
		if s.state.Users[i].WorkspaceID != workspaceID || s.state.Users[i].ID != memberID {
			continue
		}
		if s.state.Users[i].Role == workspaceRoleOwner {
			return ErrWorkspaceOwnershipTransferError
		}
		s.state.Users[i].Role = normalizedRole
		return s.saveLocked()
	}

	for i := range s.state.WorkspaceInvitations {
		if s.state.WorkspaceInvitations[i].WorkspaceID != workspaceID || s.state.WorkspaceInvitations[i].ID != memberID {
			continue
		}
		s.state.WorkspaceInvitations[i].Role = normalizedRole
		return s.saveLocked()
	}

	return ErrWorkspaceMemberNotFound
}

func (s *Store) DeleteWorkspaceMemberOrInvitation(workspaceID, memberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.state.WorkspaceInvitations {
		if s.state.WorkspaceInvitations[i].WorkspaceID != workspaceID || s.state.WorkspaceInvitations[i].ID != memberID {
			continue
		}
		s.state.WorkspaceInvitations = append(s.state.WorkspaceInvitations[:i], s.state.WorkspaceInvitations[i+1:]...)
		return s.saveLocked()
	}

	for i := range s.state.Users {
		if s.state.Users[i].WorkspaceID != workspaceID || s.state.Users[i].ID != memberID {
			continue
		}
		if s.state.Users[i].Role == workspaceRoleOwner {
			return ErrWorkspaceOwnerRemovalForbidden
		}
		s.state.Users = append(s.state.Users[:i], s.state.Users[i+1:]...)
		return s.saveLocked()
	}

	return ErrWorkspaceMemberNotFound
}

func (s *Store) GetWorkspaceInvitationByToken(token string) (WorkspaceInvitation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, invitation := range s.state.WorkspaceInvitations {
		if invitation.ActivationToken == strings.TrimSpace(token) {
			return invitation, true
		}
	}
	return WorkspaceInvitation{}, false
}

func (s *Store) ActivateWorkspaceInvitation(token, name, language, timezone string, now time.Time) (User, Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token = strings.TrimSpace(token)
	invitationIndex := -1
	for i, invitation := range s.state.WorkspaceInvitations {
		if invitation.ActivationToken == token {
			invitationIndex = i
			break
		}
	}
	if invitationIndex < 0 {
		return User{}, Workspace{}, ErrWorkspaceInvitationNotFound
	}

	invitation := s.state.WorkspaceInvitations[invitationIndex]
	if s.workspaceUserIndexByEmailLocked(invitation.Email) >= 0 {
		return User{}, Workspace{}, ErrWorkspaceMemberAlreadyExists
	}

	workspaceIndex := -1
	for i, workspace := range s.state.Workspaces {
		if workspace.ID == invitation.WorkspaceID {
			workspaceIndex = i
			break
		}
	}
	if workspaceIndex < 0 {
		return User{}, Workspace{}, fmt.Errorf("workspace %s not found", invitation.WorkspaceID)
	}

	user := User{
		ID:                generateID("usr"),
		Email:             invitation.Email,
		Name:              firstNonEmpty(strings.TrimSpace(name), invitationDisplayName(invitation.Email)),
		PasswordHash:      "",
		Avatar:            "",
		AvatarURL:         "",
		Role:              invitation.Role,
		WorkspaceID:       invitation.WorkspaceID,
		InterfaceLanguage: firstNonEmpty(strings.TrimSpace(language), "en-US"),
		InterfaceTheme:    "light",
		Timezone:          firstNonEmpty(strings.TrimSpace(timezone), "UTC"),
		CreatedAt:         now.UTC().Format(time.RFC3339),
		LastLoginAt:       now.UTC().Format(time.RFC3339),
		LastActiveAt:      now.UTC().Format(time.RFC3339),
	}

	s.state.Users = append(s.state.Users, user)
	s.state.WorkspaceInvitations = append(s.state.WorkspaceInvitations[:invitationIndex], s.state.WorkspaceInvitations[invitationIndex+1:]...)
	if err := s.saveLocked(); err != nil {
		return User{}, Workspace{}, err
	}

	return user, s.state.Workspaces[workspaceIndex], nil
}

func (s *Store) TransferWorkspaceOwnership(workspaceID, currentOwnerID, newOwnerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentOwnerIndex := -1
	newOwnerIndex := -1
	for i, user := range s.state.Users {
		if user.WorkspaceID != workspaceID {
			continue
		}
		if user.ID == currentOwnerID {
			currentOwnerIndex = i
		}
		if user.ID == newOwnerID {
			newOwnerIndex = i
		}
	}
	if currentOwnerIndex < 0 || newOwnerIndex < 0 || currentOwnerIndex == newOwnerIndex {
		return ErrWorkspaceOwnershipTransferError
	}
	if s.state.Users[currentOwnerIndex].Role != workspaceRoleOwner {
		return ErrWorkspaceOwnershipTransferError
	}

	s.state.Users[currentOwnerIndex].Role = workspaceRoleAdmin
	s.state.Users[newOwnerIndex].Role = workspaceRoleOwner
	return s.saveLocked()
}

func (s *Store) workspaceUserIndexByEmailLocked(email string) int {
	email = strings.ToLower(strings.TrimSpace(email))
	for i, user := range s.state.Users {
		if user.Email == email {
			return i
		}
	}
	return -1
}

func (s *Store) workspaceInvitationIndexByEmailLocked(workspaceID, email string) int {
	email = strings.ToLower(strings.TrimSpace(email))
	for i, invitation := range s.state.WorkspaceInvitations {
		if invitation.WorkspaceID == workspaceID && invitation.Email == email {
			return i
		}
	}
	return -1
}

func normalizeWorkspaceRole(role string, allowOwner bool) string {
	switch strings.TrimSpace(role) {
	case workspaceRoleOwner:
		if allowOwner {
			return workspaceRoleOwner
		}
		return ""
	case workspaceRoleAdmin:
		return workspaceRoleAdmin
	case workspaceRoleEditor:
		return workspaceRoleEditor
	case workspaceRoleDatasetOperator:
		return workspaceRoleDatasetOperator
	case "", workspaceRoleNormal:
		return workspaceRoleNormal
	default:
		return ""
	}
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
