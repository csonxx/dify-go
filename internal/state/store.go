package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PasswordHash      string `json:"password_hash"`
	Avatar            string `json:"avatar"`
	AvatarURL         string `json:"avatar_url"`
	Role              string `json:"role"`
	WorkspaceID       string `json:"workspace_id"`
	InterfaceLanguage string `json:"interface_language"`
	InterfaceTheme    string `json:"interface_theme"`
	Timezone          string `json:"timezone"`
	CreatedAt         string `json:"created_at"`
	LastLoginAt       string `json:"last_login_at"`
	LastActiveAt      string `json:"last_active_at"`
}

type Workspace struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Plan                string                 `json:"plan"`
	Status              string                 `json:"status"`
	Role                string                 `json:"role"`
	CreatedAt           int64                  `json:"created_at"`
	TrialCredits        int                    `json:"trial_credits"`
	TrialCreditsUsed    int                    `json:"trial_credits_used"`
	NextCreditResetDate int64                  `json:"next_credit_reset_date"`
	ModelSettings       WorkspaceModelSettings `json:"model_settings,omitempty"`
	ToolSettings        WorkspaceToolSettings  `json:"tool_settings,omitempty"`
}

type State struct {
	SetupCompleted        bool                   `json:"setup_completed"`
	SetupAt               string                 `json:"setup_at"`
	Users                 []User                 `json:"users"`
	Workspaces            []Workspace            `json:"workspaces"`
	WorkspaceInvitations  []WorkspaceInvitation  `json:"workspace_invitations"`
	Apps                  []App                  `json:"apps"`
	Datasets              []Dataset              `json:"datasets"`
	PipelineTemplates     []PipelineTemplate     `json:"pipeline_templates"`
	UploadedFiles         []UploadedFile         `json:"uploaded_files"`
	ExternalKnowledgeAPIs []ExternalKnowledgeAPI `json:"external_knowledge_apis"`
	APIKeys               []APIKey               `json:"api_keys"`
}

type Store struct {
	mu    sync.RWMutex
	path  string
	state State
}

func Open(path string) (*Store, error) {
	store := &Store{
		path: path,
		state: State{
			Users:                 []User{},
			Workspaces:            []Workspace{},
			WorkspaceInvitations:  []WorkspaceInvitation{},
			Apps:                  []App{},
			Datasets:              []Dataset{},
			PipelineTemplates:     []PipelineTemplate{},
			UploadedFiles:         []UploadedFile{},
			ExternalKnowledgeAPIs: []ExternalKnowledgeAPI{},
			APIKeys:               []APIKey{},
		},
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}
	if len(data) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(data, &store.state); err != nil {
		return nil, fmt.Errorf("decode state file: %w", err)
	}
	if store.state.Users == nil {
		store.state.Users = []User{}
	}
	if store.state.Workspaces == nil {
		store.state.Workspaces = []Workspace{}
	}
	for i := range store.state.Workspaces {
		normalizeWorkspaceModelSettings(&store.state.Workspaces[i].ModelSettings)
		normalizeWorkspaceToolSettings(&store.state.Workspaces[i].ToolSettings)
	}
	if store.state.WorkspaceInvitations == nil {
		store.state.WorkspaceInvitations = []WorkspaceInvitation{}
	}
	for i := range store.state.WorkspaceInvitations {
		normalizeWorkspaceInvitation(&store.state.WorkspaceInvitations[i])
	}
	if store.state.Apps == nil {
		store.state.Apps = []App{}
	}
	for i := range store.state.Apps {
		normalizeApp(&store.state.Apps[i])
	}
	if store.state.Datasets == nil {
		store.state.Datasets = []Dataset{}
	}
	for i := range store.state.Datasets {
		normalizeDataset(&store.state.Datasets[i])
	}
	if store.state.PipelineTemplates == nil {
		store.state.PipelineTemplates = []PipelineTemplate{}
	}
	for i := range store.state.PipelineTemplates {
		normalizePipelineTemplate(&store.state.PipelineTemplates[i])
	}
	if store.state.UploadedFiles == nil {
		store.state.UploadedFiles = []UploadedFile{}
	}
	for i := range store.state.UploadedFiles {
		normalizeUploadedFile(&store.state.UploadedFiles[i])
	}
	if store.state.ExternalKnowledgeAPIs == nil {
		store.state.ExternalKnowledgeAPIs = []ExternalKnowledgeAPI{}
	}
	for i := range store.state.ExternalKnowledgeAPIs {
		normalizeExternalKnowledgeAPI(&store.state.ExternalKnowledgeAPIs[i])
	}
	if store.state.APIKeys == nil {
		store.state.APIKeys = []APIKey{}
	}
	return store, nil
}

func (s *Store) SetupStatus() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.SetupCompleted, s.state.SetupAt
}

func (s *Store) IsSetupComplete() bool {
	completed, _ := s.SetupStatus()
	return completed
}

func (s *Store) CreateInitialSetup(email, name, passwordHash, language, workspaceName string, now time.Time) (User, Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.SetupCompleted {
		return User{}, Workspace{}, fmt.Errorf("already setup")
	}

	userID := generateID("usr")
	workspaceID := generateID("ws")

	workspace := Workspace{
		ID:                  workspaceID,
		Name:                workspaceName,
		Plan:                "sandbox",
		Status:              "normal",
		Role:                "owner",
		CreatedAt:           now.Unix(),
		TrialCredits:        0,
		TrialCreditsUsed:    0,
		NextCreditResetDate: 0,
	}
	user := User{
		ID:                userID,
		Email:             strings.ToLower(strings.TrimSpace(email)),
		Name:              strings.TrimSpace(name),
		PasswordHash:      passwordHash,
		Avatar:            "",
		AvatarURL:         "",
		Role:              "owner",
		WorkspaceID:       workspaceID,
		InterfaceLanguage: language,
		InterfaceTheme:    "light",
		Timezone:          "UTC",
		CreatedAt:         now.UTC().Format(time.RFC3339),
		LastLoginAt:       now.UTC().Format(time.RFC3339),
		LastActiveAt:      now.UTC().Format(time.RFC3339),
	}

	s.state.SetupCompleted = true
	s.state.SetupAt = now.UTC().Format(time.RFC3339)
	s.state.Users = []User{user}
	s.state.Workspaces = []Workspace{workspace}

	if err := s.saveLocked(); err != nil {
		return User{}, Workspace{}, err
	}
	return user, workspace, nil
}

func (s *Store) FindUserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	normalized := strings.ToLower(strings.TrimSpace(email))
	for _, user := range s.state.Users {
		if user.Email == normalized {
			return user, true
		}
	}
	return User{}, false
}

func (s *Store) GetUser(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.state.Users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}

func (s *Store) UserWorkspace(userID string) (Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var workspaceID string
	for _, user := range s.state.Users {
		if user.ID == userID {
			workspaceID = user.WorkspaceID
			break
		}
	}
	if workspaceID == "" {
		return Workspace{}, false
	}

	for _, workspace := range s.state.Workspaces {
		if workspace.ID == workspaceID {
			return workspace, true
		}
	}
	return Workspace{}, false
}

func (s *Store) ListWorkspacesForUser(userID string) []Workspace {
	workspace, ok := s.UserWorkspace(userID)
	if !ok {
		return []Workspace{}
	}
	return []Workspace{workspace}
}

func (s *Store) PrimaryWorkspace() (Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.state.Workspaces) == 0 {
		return Workspace{}, false
	}
	return s.state.Workspaces[0], true
}

func (s *Store) TouchLogin(userID string, now time.Time) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, user := range s.state.Users {
		if user.ID != userID {
			continue
		}
		user.LastLoginAt = now.UTC().Format(time.RFC3339)
		user.LastActiveAt = now.UTC().Format(time.RFC3339)
		s.state.Users[i] = user
		if err := s.saveLocked(); err != nil {
			return User{}, err
		}
		return user, nil
	}

	return User{}, fmt.Errorf("user %s not found", userID)
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp state: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}
