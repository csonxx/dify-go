package state

import (
	"fmt"
	"strings"
	"time"
)

type CreateWorkspaceUserInput struct {
	Email             string
	Name              string
	PasswordHash      string
	Role              string
	InterfaceLanguage string
	InterfaceTheme    string
	Timezone          string
}

func (s *Store) CreateWorkspaceUser(workspaceID string, input CreateWorkspaceUserInput, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return User{}, fmt.Errorf("email is required")
	}
	for _, user := range s.state.Users {
		if user.Email == email {
			return User{}, fmt.Errorf("user %s already exists", email)
		}
	}

	role := normalizeWorkspaceRole(input.Role, false)
	if role == "" {
		role = workspaceRoleNormal
	}

	user := User{
		ID:                generateID("usr"),
		Email:             email,
		Name:              firstNonEmpty(strings.TrimSpace(input.Name), userDisplayName(email)),
		PasswordHash:      strings.TrimSpace(input.PasswordHash),
		Avatar:            "",
		AvatarURL:         "",
		Role:              role,
		WorkspaceID:       workspaceID,
		InterfaceLanguage: firstNonEmpty(strings.TrimSpace(input.InterfaceLanguage), "en-US"),
		InterfaceTheme:    firstNonEmpty(strings.TrimSpace(input.InterfaceTheme), "light"),
		Timezone:          firstNonEmpty(strings.TrimSpace(input.Timezone), "UTC"),
		CreatedAt:         now.UTC().Format(time.RFC3339),
		LastLoginAt:       now.UTC().Format(time.RFC3339),
		LastActiveAt:      now.UTC().Format(time.RFC3339),
	}

	s.state.Users = append(s.state.Users, user)
	if err := s.saveLocked(); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Store) UpdateUserPasswordByEmail(email, passwordHash string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := strings.ToLower(strings.TrimSpace(email))
	for i, user := range s.state.Users {
		if user.Email != normalized {
			continue
		}
		user.PasswordHash = strings.TrimSpace(passwordHash)
		user.LastActiveAt = now.UTC().Format(time.RFC3339)
		s.state.Users[i] = user
		if err := s.saveLocked(); err != nil {
			return User{}, err
		}
		return user, nil
	}

	return User{}, fmt.Errorf("user %s not found", normalized)
}

func (s *Store) UpdateUserEmail(userID, email string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return User{}, fmt.Errorf("email is required")
	}

	for _, user := range s.state.Users {
		if user.ID != userID && user.Email == normalized {
			return User{}, fmt.Errorf("user %s already exists", normalized)
		}
	}

	for i, user := range s.state.Users {
		if user.ID != userID {
			continue
		}
		user.Email = normalized
		user.LastActiveAt = now.UTC().Format(time.RFC3339)
		s.state.Users[i] = user
		if err := s.saveLocked(); err != nil {
			return User{}, err
		}
		return user, nil
	}

	return User{}, fmt.Errorf("user %s not found", userID)
}

func (s *Store) UpdateUserAccountInit(userID, language, timezone string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, user := range s.state.Users {
		if user.ID != userID {
			continue
		}
		user.InterfaceLanguage = firstNonEmpty(strings.TrimSpace(language), user.InterfaceLanguage, "en-US")
		user.Timezone = firstNonEmpty(strings.TrimSpace(timezone), user.Timezone, "UTC")
		user.LastActiveAt = now.UTC().Format(time.RFC3339)
		s.state.Users[i] = user
		if err := s.saveLocked(); err != nil {
			return User{}, err
		}
		return user, nil
	}

	return User{}, fmt.Errorf("user %s not found", userID)
}

func userDisplayName(email string) string {
	local := strings.TrimSpace(strings.Split(strings.ToLower(email), "@")[0])
	if local == "" {
		return "User"
	}
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ")
	parts := strings.Fields(replacer.Replace(local))
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return firstNonEmpty(strings.Join(parts, " "), "User")
}
