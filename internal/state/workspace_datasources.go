package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type WorkspaceDatasourceOAuthClient struct {
	Enabled   bool              `json:"enabled"`
	Params    map[string]string `json:"params,omitempty"`
	UpdatedAt int64             `json:"updated_at"`
}

type WorkspaceDatasourceCredential struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	AvatarURL  string         `json:"avatar_url"`
	Credential map[string]any `json:"credential,omitempty"`
	CreatedBy  string         `json:"created_by"`
	CreatedAt  int64          `json:"created_at"`
	UpdatedAt  int64          `json:"updated_at"`
}

type WorkspaceDatasourceProviderState struct {
	PluginID               string                          `json:"plugin_id"`
	Provider               string                          `json:"provider"`
	PluginUniqueIdentifier string                          `json:"plugin_unique_identifier,omitempty"`
	Credentials            []WorkspaceDatasourceCredential `json:"credentials,omitempty"`
	DefaultCredentialID    string                          `json:"default_credential_id,omitempty"`
	OAuthClient            WorkspaceDatasourceOAuthClient  `json:"oauth_client,omitempty"`
}

type CreateWorkspaceDatasourceCredentialInput struct {
	PluginID               string
	Provider               string
	PluginUniqueIdentifier string
	Name                   string
	Type                   string
	AvatarURL              string
	Credential             map[string]any
}

type UpdateWorkspaceDatasourceCredentialInput struct {
	PluginID     string
	Provider     string
	CredentialID string
	Name         string
	Type         string
	AvatarURL    string
	Credential   map[string]any
}

func normalizeWorkspaceDatasourceProviderState(provider *WorkspaceDatasourceProviderState) {
	if provider == nil {
		return
	}
	provider.PluginID = strings.TrimSpace(provider.PluginID)
	provider.Provider = strings.TrimSpace(provider.Provider)
	provider.PluginUniqueIdentifier = strings.TrimSpace(provider.PluginUniqueIdentifier)
	provider.DefaultCredentialID = strings.TrimSpace(provider.DefaultCredentialID)
	if provider.Credentials == nil {
		provider.Credentials = []WorkspaceDatasourceCredential{}
	}
	for i := range provider.Credentials {
		normalizeWorkspaceDatasourceCredential(&provider.Credentials[i])
	}
	if provider.OAuthClient.Params == nil {
		provider.OAuthClient.Params = map[string]string{}
	}
	if provider.DefaultCredentialID == "" && len(provider.Credentials) > 0 {
		provider.DefaultCredentialID = provider.Credentials[0].ID
	}
	if provider.DefaultCredentialID != "" && !slices.ContainsFunc(provider.Credentials, func(item WorkspaceDatasourceCredential) bool {
		return item.ID == provider.DefaultCredentialID
	}) {
		if len(provider.Credentials) > 0 {
			provider.DefaultCredentialID = provider.Credentials[0].ID
		} else {
			provider.DefaultCredentialID = ""
		}
	}
}

func normalizeWorkspaceDatasourceCredential(credential *WorkspaceDatasourceCredential) {
	if credential == nil {
		return
	}
	credential.ID = strings.TrimSpace(credential.ID)
	credential.Name = strings.TrimSpace(credential.Name)
	credential.Type = strings.TrimSpace(credential.Type)
	credential.AvatarURL = strings.TrimSpace(credential.AvatarURL)
	if credential.Credential == nil {
		credential.Credential = map[string]any{}
	}
}

func cloneWorkspaceDatasourceProviderState(src WorkspaceDatasourceProviderState) WorkspaceDatasourceProviderState {
	dst := WorkspaceDatasourceProviderState{
		PluginID:               src.PluginID,
		Provider:               src.Provider,
		PluginUniqueIdentifier: src.PluginUniqueIdentifier,
		Credentials:            make([]WorkspaceDatasourceCredential, len(src.Credentials)),
		DefaultCredentialID:    src.DefaultCredentialID,
		OAuthClient: WorkspaceDatasourceOAuthClient{
			Enabled:   src.OAuthClient.Enabled,
			Params:    cloneStringMap(src.OAuthClient.Params),
			UpdatedAt: src.OAuthClient.UpdatedAt,
		},
	}
	for i, item := range src.Credentials {
		dst.Credentials[i] = cloneWorkspaceDatasourceCredential(item)
	}
	return dst
}

func cloneWorkspaceDatasourceCredential(src WorkspaceDatasourceCredential) WorkspaceDatasourceCredential {
	return WorkspaceDatasourceCredential{
		ID:         src.ID,
		Name:       src.Name,
		Type:       src.Type,
		AvatarURL:  src.AvatarURL,
		Credential: cloneMap(src.Credential),
		CreatedBy:  src.CreatedBy,
		CreatedAt:  src.CreatedAt,
		UpdatedAt:  src.UpdatedAt,
	}
}

func (s *Store) GetWorkspaceDatasourceProviderState(workspaceID, pluginID, provider string) (WorkspaceDatasourceProviderState, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceDatasourceProviderState{}, false
	}
	pluginID = strings.TrimSpace(pluginID)
	provider = strings.TrimSpace(provider)
	for _, item := range settings.DatasourceProviders {
		if item.PluginID == pluginID && item.Provider == provider {
			return cloneWorkspaceDatasourceProviderState(item), true
		}
	}
	return WorkspaceDatasourceProviderState{}, false
}

func (s *Store) ListWorkspaceDatasourceCredentials(workspaceID, pluginID, provider string) []WorkspaceDatasourceCredential {
	providerState, ok := s.GetWorkspaceDatasourceProviderState(workspaceID, pluginID, provider)
	if !ok {
		return []WorkspaceDatasourceCredential{}
	}
	items := make([]WorkspaceDatasourceCredential, len(providerState.Credentials))
	for i, item := range providerState.Credentials {
		items[i] = cloneWorkspaceDatasourceCredential(item)
	}
	return items
}

func (s *Store) UpsertWorkspaceDatasourceOAuthClient(workspaceID, pluginID, provider, pluginUniqueIdentifier string, params map[string]string, enabled bool, now time.Time) (WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, pluginID, provider, pluginUniqueIdentifier)
	providerState.OAuthClient = WorkspaceDatasourceOAuthClient{
		Enabled:   enabled,
		Params:    cloneStringMap(params),
		UpdatedAt: now.UTC().Unix(),
	}
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func (s *Store) DeleteWorkspaceDatasourceOAuthClient(workspaceID, pluginID, provider, pluginUniqueIdentifier string, now time.Time) (WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, pluginID, provider, pluginUniqueIdentifier)
	providerState.OAuthClient = WorkspaceDatasourceOAuthClient{
		Enabled:   false,
		Params:    map[string]string{},
		UpdatedAt: now.UTC().Unix(),
	}
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func (s *Store) CreateWorkspaceDatasourceCredential(workspaceID string, user User, input CreateWorkspaceDatasourceCredentialInput, now time.Time) (WorkspaceDatasourceCredential, WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, input.PluginID, input.Provider, input.PluginUniqueIdentifier)

	credentialType := normalizeWorkspaceDatasourceCredentialType(input.Type)
	if credentialType == "" {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, fmt.Errorf("credential type is required")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = defaultWorkspaceDatasourceCredentialName(input.Provider, len(providerState.Credentials)+1)
	}

	credential := WorkspaceDatasourceCredential{
		ID:         generateID("dscred"),
		Name:       name,
		Type:       credentialType,
		AvatarURL:  strings.TrimSpace(input.AvatarURL),
		Credential: cloneMap(input.Credential),
		CreatedBy:  user.ID,
		CreatedAt:  now.UTC().Unix(),
		UpdatedAt:  now.UTC().Unix(),
	}
	providerState.Credentials = append(providerState.Credentials, credential)
	if providerState.DefaultCredentialID == "" {
		providerState.DefaultCredentialID = credential.ID
	}
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceCredential(credential), cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func (s *Store) UpdateWorkspaceDatasourceCredential(workspaceID string, input UpdateWorkspaceDatasourceCredentialInput, now time.Time) (WorkspaceDatasourceCredential, WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, input.PluginID, input.Provider, "")

	credentialIndex := slices.IndexFunc(providerState.Credentials, func(item WorkspaceDatasourceCredential) bool {
		return item.ID == strings.TrimSpace(input.CredentialID)
	})
	if credentialIndex < 0 {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, fmt.Errorf("credential not found")
	}

	credential := cloneWorkspaceDatasourceCredential(providerState.Credentials[credentialIndex])
	if name := strings.TrimSpace(input.Name); name != "" {
		credential.Name = name
	}
	if credentialType := normalizeWorkspaceDatasourceCredentialType(input.Type); credentialType != "" {
		credential.Type = credentialType
	}
	if strings.TrimSpace(input.AvatarURL) != "" {
		credential.AvatarURL = strings.TrimSpace(input.AvatarURL)
	}
	if input.Credential != nil {
		credential.Credential = cloneMap(input.Credential)
	}
	credential.UpdatedAt = now.UTC().Unix()
	providerState.Credentials[credentialIndex] = credential
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceCredential{}, WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceCredential(credential), cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func (s *Store) DeleteWorkspaceDatasourceCredential(workspaceID, pluginID, provider, credentialID string, now time.Time) (WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, pluginID, provider, "")

	credentialID = strings.TrimSpace(credentialID)
	credentialIndex := slices.IndexFunc(providerState.Credentials, func(item WorkspaceDatasourceCredential) bool {
		return item.ID == credentialID
	})
	if credentialIndex < 0 {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("credential not found")
	}

	providerState.Credentials = append(providerState.Credentials[:credentialIndex], providerState.Credentials[credentialIndex+1:]...)
	if providerState.DefaultCredentialID == credentialID {
		providerState.DefaultCredentialID = ""
	}
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func (s *Store) SetWorkspaceDatasourceDefaultCredential(workspaceID, pluginID, provider, credentialID string, now time.Time) (WorkspaceDatasourceProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	providerState := ensureWorkspaceDatasourceProviderState(&settings, pluginID, provider, "")

	credentialID = strings.TrimSpace(credentialID)
	if !slices.ContainsFunc(providerState.Credentials, func(item WorkspaceDatasourceCredential) bool {
		return item.ID == credentialID
	}) {
		return WorkspaceDatasourceProviderState{}, fmt.Errorf("credential not found")
	}
	providerState.DefaultCredentialID = credentialID
	providerState.OAuthClient.UpdatedAt = now.UTC().Unix()
	normalizeWorkspaceDatasourceProviderState(providerState)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceDatasourceProviderState{}, err
	}
	return cloneWorkspaceDatasourceProviderState(*providerState), nil
}

func ensureWorkspaceDatasourceProviderState(settings *WorkspaceToolSettings, pluginID, provider, pluginUniqueIdentifier string) *WorkspaceDatasourceProviderState {
	pluginID = strings.TrimSpace(pluginID)
	provider = strings.TrimSpace(provider)
	pluginUniqueIdentifier = strings.TrimSpace(pluginUniqueIdentifier)

	index := slices.IndexFunc(settings.DatasourceProviders, func(item WorkspaceDatasourceProviderState) bool {
		return item.PluginID == pluginID && item.Provider == provider
	})
	if index < 0 {
		settings.DatasourceProviders = append(settings.DatasourceProviders, WorkspaceDatasourceProviderState{
			PluginID:               pluginID,
			Provider:               provider,
			PluginUniqueIdentifier: pluginUniqueIdentifier,
			Credentials:            []WorkspaceDatasourceCredential{},
			OAuthClient: WorkspaceDatasourceOAuthClient{
				Params: map[string]string{},
			},
		})
		index = len(settings.DatasourceProviders) - 1
	}

	providerState := &settings.DatasourceProviders[index]
	if providerState.PluginID == "" {
		providerState.PluginID = pluginID
	}
	if providerState.Provider == "" {
		providerState.Provider = provider
	}
	if pluginUniqueIdentifier != "" {
		providerState.PluginUniqueIdentifier = pluginUniqueIdentifier
	}
	normalizeWorkspaceDatasourceProviderState(providerState)
	return providerState
}

func defaultWorkspaceDatasourceCredentialName(provider string, position int) string {
	name := strings.TrimSpace(provider)
	if name == "" {
		name = "datasource"
	}
	return fmt.Sprintf("%s connection %d", name, position)
}

func normalizeWorkspaceDatasourceCredentialType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "oauth2":
		return "oauth2"
	case "api-key", "api_key":
		return "api-key"
	default:
		return ""
	}
}
