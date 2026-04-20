package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type WorkspaceToolSettings struct {
	BuiltinProviders  []WorkspaceBuiltinToolProvider  `json:"builtin_providers,omitempty"`
	APIProviders      []WorkspaceAPIToolProvider      `json:"api_providers,omitempty"`
	WorkflowProviders []WorkspaceWorkflowToolProvider `json:"workflow_providers,omitempty"`
	MCPProviders      []WorkspaceMCPToolProvider      `json:"mcp_providers,omitempty"`
	Endpoints         []WorkspaceEndpoint             `json:"endpoints,omitempty"`
	TriggerProviders  []WorkspaceTriggerProviderState `json:"trigger_providers,omitempty"`
	Plugins           []WorkspacePluginInstallation   `json:"plugins,omitempty"`
	PluginPreferences WorkspacePluginPreferences      `json:"plugin_preferences,omitempty"`
	PluginTasks       []WorkspacePluginTask           `json:"plugin_tasks,omitempty"`
}

type WorkspaceBuiltinToolProvider struct {
	Provider     string         `json:"provider"`
	CredentialID string         `json:"credential_id"`
	Credentials  map[string]any `json:"credentials"`
	UpdatedAt    int64          `json:"updated_at"`
}

type WorkspaceEmoji struct {
	Background string `json:"background"`
	Content    string `json:"content"`
}

type WorkspaceToolOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type WorkspaceToolParameter struct {
	Name             string                `json:"name"`
	Label            string                `json:"label"`
	HumanDescription string                `json:"human_description"`
	Type             string                `json:"type"`
	Form             string                `json:"form"`
	LLMDescription   string                `json:"llm_description"`
	Required         bool                  `json:"required"`
	Multiple         bool                  `json:"multiple"`
	Default          string                `json:"default"`
	Options          []WorkspaceToolOption `json:"options,omitempty"`
	Min              *float64              `json:"min,omitempty"`
	Max              *float64              `json:"max,omitempty"`
}

type WorkspaceTool struct {
	Name         string                   `json:"name"`
	Author       string                   `json:"author"`
	Label        string                   `json:"label"`
	Description  string                   `json:"description"`
	Parameters   []WorkspaceToolParameter `json:"parameters,omitempty"`
	Labels       []string                 `json:"labels,omitempty"`
	OutputSchema map[string]any           `json:"output_schema"`
}

type WorkspaceAPIToolProvider struct {
	ID               string         `json:"id"`
	Provider         string         `json:"provider"`
	Icon             WorkspaceEmoji `json:"icon"`
	Credentials      map[string]any `json:"credentials"`
	SchemaType       string         `json:"schema_type"`
	Schema           string         `json:"schema"`
	PrivacyPolicy    string         `json:"privacy_policy"`
	CustomDisclaimer string         `json:"custom_disclaimer"`
	Labels           []string       `json:"labels,omitempty"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
}

type WorkspaceWorkflowToolParameter struct {
	Name        string `json:"name"`
	Form        string `json:"form"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}

type WorkspaceWorkflowToolProvider struct {
	ID            string                           `json:"id"`
	AppID         string                           `json:"app_id"`
	Name          string                           `json:"name"`
	Label         string                           `json:"label"`
	Icon          WorkspaceEmoji                   `json:"icon"`
	Description   string                           `json:"description"`
	Parameters    []WorkspaceWorkflowToolParameter `json:"parameters,omitempty"`
	PrivacyPolicy string                           `json:"privacy_policy"`
	Labels        []string                         `json:"labels,omitempty"`
	Version       string                           `json:"version"`
	CreatedBy     string                           `json:"created_by"`
	CreatedAt     int64                            `json:"created_at"`
	UpdatedAt     int64                            `json:"updated_at"`
}

type WorkspaceMCPAuthentication struct {
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

type WorkspaceMCPConfiguration struct {
	Timeout        int `json:"timeout"`
	SSEReadTimeout int `json:"sse_read_timeout"`
}

type WorkspaceMCPToolProvider struct {
	ID                    string                     `json:"id"`
	Name                  string                     `json:"name"`
	IconType              string                     `json:"icon_type"`
	Icon                  string                     `json:"icon"`
	IconBackground        string                     `json:"icon_background"`
	ServerURL             string                     `json:"server_url"`
	ServerIdentifier      string                     `json:"server_identifier"`
	Headers               map[string]string          `json:"headers"`
	Authentication        WorkspaceMCPAuthentication `json:"authentication"`
	Configuration         WorkspaceMCPConfiguration  `json:"configuration"`
	IsDynamicRegistration bool                       `json:"is_dynamic_registration"`
	IsAuthorized          bool                       `json:"is_authorized"`
	Tools                 []WorkspaceTool            `json:"tools,omitempty"`
	CreatedBy             string                     `json:"created_by"`
	CreatedAt             int64                      `json:"created_at"`
	UpdatedAt             int64                      `json:"updated_at"`
}

type CreateAPIToolProviderInput struct {
	Provider         string
	Icon             WorkspaceEmoji
	Credentials      map[string]any
	SchemaType       string
	Schema           string
	PrivacyPolicy    string
	CustomDisclaimer string
	Labels           []string
}

type UpdateAPIToolProviderInput struct {
	Provider         string
	OriginalProvider string
	Icon             WorkspaceEmoji
	Credentials      map[string]any
	SchemaType       string
	Schema           string
	PrivacyPolicy    string
	CustomDisclaimer string
	Labels           []string
}

type CreateWorkflowToolProviderInput struct {
	AppID         string
	Name          string
	Label         string
	Icon          WorkspaceEmoji
	Description   string
	Parameters    []WorkspaceWorkflowToolParameter
	PrivacyPolicy string
	Labels        []string
	Version       string
}

type UpdateWorkflowToolProviderInput struct {
	ID            string
	Name          string
	Label         string
	Icon          WorkspaceEmoji
	Description   string
	Parameters    []WorkspaceWorkflowToolParameter
	PrivacyPolicy string
	Labels        []string
	Version       string
}

type CreateMCPToolProviderInput struct {
	Name                  string
	IconType              string
	Icon                  string
	IconBackground        string
	ServerURL             string
	ServerIdentifier      string
	Headers               map[string]string
	Authentication        WorkspaceMCPAuthentication
	Configuration         WorkspaceMCPConfiguration
	IsDynamicRegistration bool
}

type UpdateMCPToolProviderInput struct {
	ProviderID            string
	Name                  string
	IconType              string
	Icon                  string
	IconBackground        string
	ServerURL             string
	ServerIdentifier      string
	Headers               map[string]string
	Authentication        WorkspaceMCPAuthentication
	Configuration         WorkspaceMCPConfiguration
	IsDynamicRegistration bool
}

func normalizeWorkspaceToolSettings(settings *WorkspaceToolSettings) {
	if settings == nil {
		return
	}
	if settings.BuiltinProviders == nil {
		settings.BuiltinProviders = []WorkspaceBuiltinToolProvider{}
	}
	if settings.APIProviders == nil {
		settings.APIProviders = []WorkspaceAPIToolProvider{}
	}
	if settings.WorkflowProviders == nil {
		settings.WorkflowProviders = []WorkspaceWorkflowToolProvider{}
	}
	if settings.MCPProviders == nil {
		settings.MCPProviders = []WorkspaceMCPToolProvider{}
	}
	if settings.Endpoints == nil {
		settings.Endpoints = []WorkspaceEndpoint{}
	}
	if settings.TriggerProviders == nil {
		settings.TriggerProviders = []WorkspaceTriggerProviderState{}
	}
	if settings.Plugins == nil {
		settings.Plugins = []WorkspacePluginInstallation{}
	}
	if settings.PluginTasks == nil {
		settings.PluginTasks = []WorkspacePluginTask{}
	}
	for i := range settings.BuiltinProviders {
		if settings.BuiltinProviders[i].Credentials == nil {
			settings.BuiltinProviders[i].Credentials = map[string]any{}
		}
	}
	for i := range settings.APIProviders {
		if settings.APIProviders[i].Credentials == nil {
			settings.APIProviders[i].Credentials = map[string]any{}
		}
		if settings.APIProviders[i].Labels == nil {
			settings.APIProviders[i].Labels = []string{}
		}
	}
	for i := range settings.WorkflowProviders {
		if settings.WorkflowProviders[i].Parameters == nil {
			settings.WorkflowProviders[i].Parameters = []WorkspaceWorkflowToolParameter{}
		}
		if settings.WorkflowProviders[i].Labels == nil {
			settings.WorkflowProviders[i].Labels = []string{}
		}
	}
	for i := range settings.MCPProviders {
		normalizeWorkspaceMCPProvider(&settings.MCPProviders[i])
	}
	for i := range settings.Endpoints {
		normalizeWorkspaceEndpoint(&settings.Endpoints[i])
	}
	for i := range settings.TriggerProviders {
		normalizeWorkspaceTriggerProviderState(&settings.TriggerProviders[i])
	}
	for i := range settings.Plugins {
		normalizeWorkspacePluginInstallation(&settings.Plugins[i])
	}
	normalizeWorkspacePluginPreferences(&settings.PluginPreferences)
	for i := range settings.PluginTasks {
		normalizeWorkspacePluginTask(&settings.PluginTasks[i])
	}
}

func normalizeWorkspaceMCPProvider(provider *WorkspaceMCPToolProvider) {
	if provider == nil {
		return
	}
	if provider.Headers == nil {
		provider.Headers = map[string]string{}
	}
	if provider.Configuration.Timeout <= 0 {
		provider.Configuration.Timeout = 30
	}
	if provider.Configuration.SSEReadTimeout <= 0 {
		provider.Configuration.SSEReadTimeout = 300
	}
	if provider.Tools == nil {
		provider.Tools = []WorkspaceTool{}
	}
	for i := range provider.Tools {
		normalizeWorkspaceTool(&provider.Tools[i])
	}
}

func normalizeWorkspaceTool(tool *WorkspaceTool) {
	if tool == nil {
		return
	}
	if tool.Parameters == nil {
		tool.Parameters = []WorkspaceToolParameter{}
	}
	if tool.Labels == nil {
		tool.Labels = []string{}
	}
	if tool.OutputSchema == nil {
		tool.OutputSchema = map[string]any{}
	}
	for i := range tool.Parameters {
		if tool.Parameters[i].Options == nil {
			tool.Parameters[i].Options = []WorkspaceToolOption{}
		}
	}
}

func (s *Store) GetWorkspaceToolSettings(workspaceID string) (WorkspaceToolSettings, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, workspace := range s.state.Workspaces {
		if workspace.ID != workspaceID {
			continue
		}
		normalizeWorkspaceToolSettings(&workspace.ToolSettings)
		return cloneWorkspaceToolSettings(workspace.ToolSettings), true
	}
	return WorkspaceToolSettings{}, false
}

func (s *Store) ListBuiltinToolCredentials(workspaceID string) []WorkspaceBuiltinToolProvider {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceBuiltinToolProvider{}
	}
	items := make([]WorkspaceBuiltinToolProvider, len(settings.BuiltinProviders))
	for i, item := range settings.BuiltinProviders {
		items[i] = cloneWorkspaceBuiltinToolProvider(item)
	}
	return items
}

func (s *Store) GetBuiltinToolCredential(workspaceID, provider string) (WorkspaceBuiltinToolProvider, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceBuiltinToolProvider{}, false
	}
	provider = strings.TrimSpace(provider)
	for _, item := range settings.BuiltinProviders {
		if item.Provider == provider {
			return cloneWorkspaceBuiltinToolProvider(item), true
		}
	}
	return WorkspaceBuiltinToolProvider{}, false
}

func (s *Store) UpsertBuiltinToolCredential(workspaceID, provider string, credentials map[string]any, now time.Time) (WorkspaceBuiltinToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceBuiltinToolProvider{}, fmt.Errorf("workspace not found")
	}

	provider = strings.TrimSpace(provider)
	if provider == "" {
		return WorkspaceBuiltinToolProvider{}, fmt.Errorf("provider is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	index := slices.IndexFunc(settings.BuiltinProviders, func(item WorkspaceBuiltinToolProvider) bool {
		return item.Provider == provider
	})
	credential := WorkspaceBuiltinToolProvider{
		Provider:     provider,
		CredentialID: generateID("tool_cred"),
		Credentials:  cloneMap(credentials),
		UpdatedAt:    now.UTC().Unix(),
	}
	if index >= 0 {
		credential = cloneWorkspaceBuiltinToolProvider(settings.BuiltinProviders[index])
		if strings.TrimSpace(credential.CredentialID) == "" {
			credential.CredentialID = generateID("tool_cred")
		}
		credential.Credentials = cloneMap(credentials)
		credential.UpdatedAt = now.UTC().Unix()
		settings.BuiltinProviders[index] = credential
	} else {
		settings.BuiltinProviders = append(settings.BuiltinProviders, credential)
	}

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceBuiltinToolProvider{}, err
	}
	return cloneWorkspaceBuiltinToolProvider(credential), nil
}

func (s *Store) DeleteBuiltinToolCredential(workspaceID, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.BuiltinProviders, func(item WorkspaceBuiltinToolProvider) bool {
		return item.Provider == strings.TrimSpace(provider)
	})
	if index >= 0 {
		settings.BuiltinProviders = append(settings.BuiltinProviders[:index], settings.BuiltinProviders[index+1:]...)
	}

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListAPIToolProviders(workspaceID string) []WorkspaceAPIToolProvider {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceAPIToolProvider{}
	}
	items := make([]WorkspaceAPIToolProvider, len(settings.APIProviders))
	for i, item := range settings.APIProviders {
		items[i] = cloneWorkspaceAPIToolProvider(item)
	}
	slices.SortFunc(items, func(a, b WorkspaceAPIToolProvider) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.Provider, b.Provider)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})
	return items
}

func (s *Store) GetAPIToolProviderByName(workspaceID, provider string) (WorkspaceAPIToolProvider, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceAPIToolProvider{}, false
	}
	provider = strings.TrimSpace(provider)
	for _, item := range settings.APIProviders {
		if item.Provider == provider {
			return cloneWorkspaceAPIToolProvider(item), true
		}
	}
	return WorkspaceAPIToolProvider{}, false
}

func (s *Store) CreateAPIToolProvider(workspaceID string, user User, input CreateAPIToolProviderInput, now time.Time) (WorkspaceAPIToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("workspace not found")
	}

	providerName := strings.TrimSpace(input.Provider)
	if providerName == "" {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("provider is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	if slices.IndexFunc(settings.APIProviders, func(item WorkspaceAPIToolProvider) bool {
		return item.Provider == providerName
	}) >= 0 {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("provider %s already exists", providerName)
	}

	timestamp := now.UTC().Unix()
	provider := WorkspaceAPIToolProvider{
		ID:               generateID("api_tool"),
		Provider:         providerName,
		Icon:             normalizeWorkspaceEmoji(input.Icon),
		Credentials:      cloneMap(input.Credentials),
		SchemaType:       normalizeAPISchemaType(input.SchemaType),
		Schema:           strings.TrimSpace(input.Schema),
		PrivacyPolicy:    strings.TrimSpace(input.PrivacyPolicy),
		CustomDisclaimer: strings.TrimSpace(input.CustomDisclaimer),
		Labels:           cloneStringSlice(input.Labels),
		CreatedBy:        user.ID,
		CreatedAt:        timestamp,
		UpdatedAt:        timestamp,
	}
	settings.APIProviders = append(settings.APIProviders, provider)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceAPIToolProvider{}, err
	}
	return cloneWorkspaceAPIToolProvider(provider), nil
}

func (s *Store) UpdateAPIToolProvider(workspaceID string, user User, input UpdateAPIToolProviderInput, now time.Time) (WorkspaceAPIToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	originalProvider := firstNonEmpty(input.OriginalProvider, input.Provider)
	index := slices.IndexFunc(settings.APIProviders, func(item WorkspaceAPIToolProvider) bool {
		return item.Provider == strings.TrimSpace(originalProvider)
	})
	if index < 0 {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("provider %s not found", originalProvider)
	}

	providerName := strings.TrimSpace(input.Provider)
	if providerName == "" {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("provider is required")
	}
	if slices.IndexFunc(settings.APIProviders, func(item WorkspaceAPIToolProvider) bool {
		return item.Provider == providerName && item.Provider != strings.TrimSpace(originalProvider)
	}) >= 0 {
		return WorkspaceAPIToolProvider{}, fmt.Errorf("provider %s already exists", providerName)
	}

	provider := cloneWorkspaceAPIToolProvider(settings.APIProviders[index])
	provider.Provider = providerName
	provider.Icon = normalizeWorkspaceEmoji(input.Icon)
	provider.Credentials = cloneMap(input.Credentials)
	provider.SchemaType = normalizeAPISchemaType(input.SchemaType)
	provider.Schema = strings.TrimSpace(input.Schema)
	provider.PrivacyPolicy = strings.TrimSpace(input.PrivacyPolicy)
	provider.CustomDisclaimer = strings.TrimSpace(input.CustomDisclaimer)
	provider.Labels = cloneStringSlice(input.Labels)
	provider.UpdatedAt = now.UTC().Unix()
	provider.CreatedBy = firstNonEmpty(provider.CreatedBy, user.ID)
	settings.APIProviders[index] = provider

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceAPIToolProvider{}, err
	}
	return cloneWorkspaceAPIToolProvider(provider), nil
}

func (s *Store) DeleteAPIToolProvider(workspaceID, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.APIProviders, func(item WorkspaceAPIToolProvider) bool {
		return item.Provider == strings.TrimSpace(provider)
	})
	if index < 0 {
		return fmt.Errorf("provider %s not found", provider)
	}
	settings.APIProviders = append(settings.APIProviders[:index], settings.APIProviders[index+1:]...)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListWorkflowToolProviders(workspaceID string) []WorkspaceWorkflowToolProvider {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceWorkflowToolProvider{}
	}
	items := make([]WorkspaceWorkflowToolProvider, len(settings.WorkflowProviders))
	for i, item := range settings.WorkflowProviders {
		items[i] = cloneWorkspaceWorkflowToolProvider(item)
	}
	slices.SortFunc(items, func(a, b WorkspaceWorkflowToolProvider) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.Name, b.Name)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})
	return items
}

func (s *Store) GetWorkflowToolProviderByID(workspaceID, id string) (WorkspaceWorkflowToolProvider, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceWorkflowToolProvider{}, false
	}
	for _, item := range settings.WorkflowProviders {
		if item.ID == strings.TrimSpace(id) {
			return cloneWorkspaceWorkflowToolProvider(item), true
		}
	}
	return WorkspaceWorkflowToolProvider{}, false
}

func (s *Store) GetWorkflowToolProviderByAppID(workspaceID, appID string) (WorkspaceWorkflowToolProvider, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceWorkflowToolProvider{}, false
	}
	for _, item := range settings.WorkflowProviders {
		if item.AppID == strings.TrimSpace(appID) {
			return cloneWorkspaceWorkflowToolProvider(item), true
		}
	}
	return WorkspaceWorkflowToolProvider{}, false
}

func (s *Store) CreateWorkflowToolProvider(workspaceID string, user User, input CreateWorkflowToolProviderInput, now time.Time) (WorkspaceWorkflowToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("workspace not found")
	}

	if strings.TrimSpace(input.AppID) == "" {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("workflow_app_id is required")
	}
	if strings.TrimSpace(input.Name) == "" {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("name is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	if slices.IndexFunc(settings.WorkflowProviders, func(item WorkspaceWorkflowToolProvider) bool {
		return item.AppID == strings.TrimSpace(input.AppID) || item.Name == strings.TrimSpace(input.Name)
	}) >= 0 {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("workflow tool with same name or workflow_app_id already exists")
	}

	timestamp := now.UTC().Unix()
	provider := WorkspaceWorkflowToolProvider{
		ID:            generateID("workflow_tool"),
		AppID:         strings.TrimSpace(input.AppID),
		Name:          strings.TrimSpace(input.Name),
		Label:         firstNonEmpty(input.Label, input.Name),
		Icon:          normalizeWorkspaceEmoji(input.Icon),
		Description:   strings.TrimSpace(input.Description),
		Parameters:    cloneWorkspaceWorkflowToolParameters(input.Parameters),
		PrivacyPolicy: strings.TrimSpace(input.PrivacyPolicy),
		Labels:        cloneStringSlice(input.Labels),
		Version:       strings.TrimSpace(input.Version),
		CreatedBy:     user.ID,
		CreatedAt:     timestamp,
		UpdatedAt:     timestamp,
	}
	settings.WorkflowProviders = append(settings.WorkflowProviders, provider)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceWorkflowToolProvider{}, err
	}
	return cloneWorkspaceWorkflowToolProvider(provider), nil
}

func (s *Store) UpdateWorkflowToolProvider(workspaceID string, user User, input UpdateWorkflowToolProviderInput, now time.Time) (WorkspaceWorkflowToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.WorkflowProviders, func(item WorkspaceWorkflowToolProvider) bool {
		return item.ID == strings.TrimSpace(input.ID)
	})
	if index < 0 {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("workflow tool %s not found", input.ID)
	}

	if slices.IndexFunc(settings.WorkflowProviders, func(item WorkspaceWorkflowToolProvider) bool {
		return item.Name == strings.TrimSpace(input.Name) && item.ID != strings.TrimSpace(input.ID)
	}) >= 0 {
		return WorkspaceWorkflowToolProvider{}, fmt.Errorf("tool with name %s already exists", input.Name)
	}

	provider := cloneWorkspaceWorkflowToolProvider(settings.WorkflowProviders[index])
	provider.Name = strings.TrimSpace(input.Name)
	provider.Label = firstNonEmpty(input.Label, provider.Name)
	provider.Icon = normalizeWorkspaceEmoji(input.Icon)
	provider.Description = strings.TrimSpace(input.Description)
	provider.Parameters = cloneWorkspaceWorkflowToolParameters(input.Parameters)
	provider.PrivacyPolicy = strings.TrimSpace(input.PrivacyPolicy)
	provider.Labels = cloneStringSlice(input.Labels)
	provider.Version = strings.TrimSpace(input.Version)
	provider.UpdatedAt = now.UTC().Unix()
	provider.CreatedBy = firstNonEmpty(provider.CreatedBy, user.ID)
	settings.WorkflowProviders[index] = provider

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceWorkflowToolProvider{}, err
	}
	return cloneWorkspaceWorkflowToolProvider(provider), nil
}

func (s *Store) DeleteWorkflowToolProvider(workspaceID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.WorkflowProviders, func(item WorkspaceWorkflowToolProvider) bool {
		return item.ID == strings.TrimSpace(id)
	})
	if index < 0 {
		return fmt.Errorf("workflow tool %s not found", id)
	}
	settings.WorkflowProviders = append(settings.WorkflowProviders[:index], settings.WorkflowProviders[index+1:]...)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListMCPToolProviders(workspaceID string) []WorkspaceMCPToolProvider {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceMCPToolProvider{}
	}
	items := make([]WorkspaceMCPToolProvider, len(settings.MCPProviders))
	for i, item := range settings.MCPProviders {
		items[i] = cloneWorkspaceMCPToolProvider(item)
	}
	slices.SortFunc(items, func(a, b WorkspaceMCPToolProvider) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.Name, b.Name)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})
	return items
}

func (s *Store) GetMCPToolProvider(workspaceID, providerID string) (WorkspaceMCPToolProvider, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceMCPToolProvider{}, false
	}
	for _, item := range settings.MCPProviders {
		if item.ID == strings.TrimSpace(providerID) {
			return cloneWorkspaceMCPToolProvider(item), true
		}
	}
	return WorkspaceMCPToolProvider{}, false
}

func (s *Store) CreateMCPToolProvider(workspaceID string, user User, input CreateMCPToolProviderInput, now time.Time) (WorkspaceMCPToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	serverIdentifier := strings.TrimSpace(input.ServerIdentifier)
	if serverIdentifier == "" {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("server_identifier is required")
	}
	if slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ServerIdentifier == serverIdentifier
	}) >= 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("server_identifier %s already exists", serverIdentifier)
	}

	timestamp := now.UTC().Unix()
	provider := WorkspaceMCPToolProvider{
		ID:               generateID("mcp_provider"),
		Name:             strings.TrimSpace(input.Name),
		IconType:         strings.TrimSpace(input.IconType),
		Icon:             strings.TrimSpace(input.Icon),
		IconBackground:   strings.TrimSpace(input.IconBackground),
		ServerURL:        strings.TrimSpace(input.ServerURL),
		ServerIdentifier: serverIdentifier,
		Headers:          cloneStringMap(input.Headers),
		Authentication: WorkspaceMCPAuthentication{
			ClientID:     strings.TrimSpace(input.Authentication.ClientID),
			ClientSecret: strings.TrimSpace(input.Authentication.ClientSecret),
		},
		Configuration: WorkspaceMCPConfiguration{
			Timeout:        normalizePositiveInt(input.Configuration.Timeout, 30),
			SSEReadTimeout: normalizePositiveInt(input.Configuration.SSEReadTimeout, 300),
		},
		IsDynamicRegistration: input.IsDynamicRegistration,
		IsAuthorized:          false,
		Tools:                 []WorkspaceTool{},
		CreatedBy:             user.ID,
		CreatedAt:             timestamp,
		UpdatedAt:             timestamp,
	}
	settings.MCPProviders = append(settings.MCPProviders, provider)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceMCPToolProvider{}, err
	}
	return cloneWorkspaceMCPToolProvider(provider), nil
}

func (s *Store) UpdateMCPToolProvider(workspaceID string, user User, input UpdateMCPToolProviderInput, now time.Time) (WorkspaceMCPToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ID == strings.TrimSpace(input.ProviderID)
	})
	if index < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("provider %s not found", input.ProviderID)
	}

	serverIdentifier := strings.TrimSpace(input.ServerIdentifier)
	if serverIdentifier == "" {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("server_identifier is required")
	}
	if slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ServerIdentifier == serverIdentifier && item.ID != strings.TrimSpace(input.ProviderID)
	}) >= 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("server_identifier %s already exists", serverIdentifier)
	}

	provider := cloneWorkspaceMCPToolProvider(settings.MCPProviders[index])
	provider.Name = strings.TrimSpace(input.Name)
	provider.IconType = strings.TrimSpace(input.IconType)
	provider.Icon = strings.TrimSpace(input.Icon)
	provider.IconBackground = strings.TrimSpace(input.IconBackground)
	provider.ServerURL = strings.TrimSpace(input.ServerURL)
	provider.ServerIdentifier = serverIdentifier
	provider.Headers = cloneStringMap(input.Headers)
	provider.Authentication = WorkspaceMCPAuthentication{
		ClientID:     strings.TrimSpace(input.Authentication.ClientID),
		ClientSecret: strings.TrimSpace(input.Authentication.ClientSecret),
	}
	provider.Configuration = WorkspaceMCPConfiguration{
		Timeout:        normalizePositiveInt(input.Configuration.Timeout, provider.Configuration.Timeout),
		SSEReadTimeout: normalizePositiveInt(input.Configuration.SSEReadTimeout, provider.Configuration.SSEReadTimeout),
	}
	provider.IsDynamicRegistration = input.IsDynamicRegistration
	provider.UpdatedAt = now.UTC().Unix()
	provider.CreatedBy = firstNonEmpty(provider.CreatedBy, user.ID)
	settings.MCPProviders[index] = provider

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceMCPToolProvider{}, err
	}
	return cloneWorkspaceMCPToolProvider(provider), nil
}

func (s *Store) DeleteMCPToolProvider(workspaceID, providerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ID == strings.TrimSpace(providerID)
	})
	if index < 0 {
		return fmt.Errorf("provider %s not found", providerID)
	}
	settings.MCPProviders = append(settings.MCPProviders[:index], settings.MCPProviders[index+1:]...)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) UpdateMCPToolProviderAuthorization(workspaceID, providerID string, authorized bool, now time.Time) (WorkspaceMCPToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ID == strings.TrimSpace(providerID)
	})
	if index < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("provider %s not found", providerID)
	}
	provider := cloneWorkspaceMCPToolProvider(settings.MCPProviders[index])
	provider.IsAuthorized = authorized
	provider.UpdatedAt = now.UTC().Unix()
	settings.MCPProviders[index] = provider

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceMCPToolProvider{}, err
	}
	return cloneWorkspaceMCPToolProvider(provider), nil
}

func (s *Store) UpdateMCPToolProviderTools(workspaceID, providerID string, tools []WorkspaceTool, now time.Time) (WorkspaceMCPToolProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.MCPProviders, func(item WorkspaceMCPToolProvider) bool {
		return item.ID == strings.TrimSpace(providerID)
	})
	if index < 0 {
		return WorkspaceMCPToolProvider{}, fmt.Errorf("provider %s not found", providerID)
	}

	provider := cloneWorkspaceMCPToolProvider(settings.MCPProviders[index])
	provider.Tools = cloneWorkspaceTools(tools)
	provider.IsAuthorized = true
	provider.UpdatedAt = now.UTC().Unix()
	settings.MCPProviders[index] = provider

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceMCPToolProvider{}, err
	}
	return cloneWorkspaceMCPToolProvider(provider), nil
}

func cloneWorkspaceToolSettings(src WorkspaceToolSettings) WorkspaceToolSettings {
	dst := WorkspaceToolSettings{
		BuiltinProviders:  make([]WorkspaceBuiltinToolProvider, len(src.BuiltinProviders)),
		APIProviders:      make([]WorkspaceAPIToolProvider, len(src.APIProviders)),
		WorkflowProviders: make([]WorkspaceWorkflowToolProvider, len(src.WorkflowProviders)),
		MCPProviders:      make([]WorkspaceMCPToolProvider, len(src.MCPProviders)),
		Endpoints:         make([]WorkspaceEndpoint, len(src.Endpoints)),
		TriggerProviders:  make([]WorkspaceTriggerProviderState, len(src.TriggerProviders)),
		Plugins:           make([]WorkspacePluginInstallation, len(src.Plugins)),
		PluginPreferences: cloneWorkspacePluginPreferences(src.PluginPreferences),
		PluginTasks:       make([]WorkspacePluginTask, len(src.PluginTasks)),
	}
	for i, item := range src.BuiltinProviders {
		dst.BuiltinProviders[i] = cloneWorkspaceBuiltinToolProvider(item)
	}
	for i, item := range src.APIProviders {
		dst.APIProviders[i] = cloneWorkspaceAPIToolProvider(item)
	}
	for i, item := range src.WorkflowProviders {
		dst.WorkflowProviders[i] = cloneWorkspaceWorkflowToolProvider(item)
	}
	for i, item := range src.MCPProviders {
		dst.MCPProviders[i] = cloneWorkspaceMCPToolProvider(item)
	}
	for i, item := range src.Endpoints {
		dst.Endpoints[i] = cloneWorkspaceEndpoint(item)
	}
	for i, item := range src.TriggerProviders {
		dst.TriggerProviders[i] = cloneWorkspaceTriggerProviderState(item)
	}
	for i, item := range src.Plugins {
		dst.Plugins[i] = cloneWorkspacePluginInstallation(item)
	}
	for i, item := range src.PluginTasks {
		dst.PluginTasks[i] = cloneWorkspacePluginTask(item)
	}
	return dst
}

func cloneWorkspaceBuiltinToolProvider(src WorkspaceBuiltinToolProvider) WorkspaceBuiltinToolProvider {
	return WorkspaceBuiltinToolProvider{
		Provider:     src.Provider,
		CredentialID: src.CredentialID,
		Credentials:  cloneMap(src.Credentials),
		UpdatedAt:    src.UpdatedAt,
	}
}

func cloneWorkspaceAPIToolProvider(src WorkspaceAPIToolProvider) WorkspaceAPIToolProvider {
	return WorkspaceAPIToolProvider{
		ID:               src.ID,
		Provider:         src.Provider,
		Icon:             normalizeWorkspaceEmoji(src.Icon),
		Credentials:      cloneMap(src.Credentials),
		SchemaType:       src.SchemaType,
		Schema:           src.Schema,
		PrivacyPolicy:    src.PrivacyPolicy,
		CustomDisclaimer: src.CustomDisclaimer,
		Labels:           cloneStringSlice(src.Labels),
		CreatedBy:        src.CreatedBy,
		CreatedAt:        src.CreatedAt,
		UpdatedAt:        src.UpdatedAt,
	}
}

func cloneWorkspaceWorkflowToolProvider(src WorkspaceWorkflowToolProvider) WorkspaceWorkflowToolProvider {
	return WorkspaceWorkflowToolProvider{
		ID:            src.ID,
		AppID:         src.AppID,
		Name:          src.Name,
		Label:         src.Label,
		Icon:          normalizeWorkspaceEmoji(src.Icon),
		Description:   src.Description,
		Parameters:    cloneWorkspaceWorkflowToolParameters(src.Parameters),
		PrivacyPolicy: src.PrivacyPolicy,
		Labels:        cloneStringSlice(src.Labels),
		Version:       src.Version,
		CreatedBy:     src.CreatedBy,
		CreatedAt:     src.CreatedAt,
		UpdatedAt:     src.UpdatedAt,
	}
}

func cloneWorkspaceWorkflowToolParameters(src []WorkspaceWorkflowToolParameter) []WorkspaceWorkflowToolParameter {
	if src == nil {
		return []WorkspaceWorkflowToolParameter{}
	}
	dst := make([]WorkspaceWorkflowToolParameter, len(src))
	copy(dst, src)
	return dst
}

func cloneWorkspaceMCPToolProvider(src WorkspaceMCPToolProvider) WorkspaceMCPToolProvider {
	dst := WorkspaceMCPToolProvider{
		ID:               src.ID,
		Name:             src.Name,
		IconType:         src.IconType,
		Icon:             src.Icon,
		IconBackground:   src.IconBackground,
		ServerURL:        src.ServerURL,
		ServerIdentifier: src.ServerIdentifier,
		Headers:          cloneStringMap(src.Headers),
		Authentication: WorkspaceMCPAuthentication{
			ClientID:     src.Authentication.ClientID,
			ClientSecret: src.Authentication.ClientSecret,
		},
		Configuration: WorkspaceMCPConfiguration{
			Timeout:        src.Configuration.Timeout,
			SSEReadTimeout: src.Configuration.SSEReadTimeout,
		},
		IsDynamicRegistration: src.IsDynamicRegistration,
		IsAuthorized:          src.IsAuthorized,
		Tools:                 cloneWorkspaceTools(src.Tools),
		CreatedBy:             src.CreatedBy,
		CreatedAt:             src.CreatedAt,
		UpdatedAt:             src.UpdatedAt,
	}
	normalizeWorkspaceMCPProvider(&dst)
	return dst
}

func cloneWorkspaceTools(src []WorkspaceTool) []WorkspaceTool {
	if src == nil {
		return []WorkspaceTool{}
	}
	dst := make([]WorkspaceTool, len(src))
	for i, item := range src {
		dst[i] = cloneWorkspaceTool(item)
	}
	return dst
}

func cloneWorkspaceTool(src WorkspaceTool) WorkspaceTool {
	out := WorkspaceTool{
		Name:         src.Name,
		Author:       src.Author,
		Label:        src.Label,
		Description:  src.Description,
		Parameters:   make([]WorkspaceToolParameter, len(src.Parameters)),
		Labels:       cloneStringSlice(src.Labels),
		OutputSchema: cloneMap(src.OutputSchema),
	}
	for i, parameter := range src.Parameters {
		out.Parameters[i] = cloneWorkspaceToolParameter(parameter)
	}
	normalizeWorkspaceTool(&out)
	return out
}

func cloneWorkspaceToolParameter(src WorkspaceToolParameter) WorkspaceToolParameter {
	out := WorkspaceToolParameter{
		Name:             src.Name,
		Label:            src.Label,
		HumanDescription: src.HumanDescription,
		Type:             src.Type,
		Form:             src.Form,
		LLMDescription:   src.LLMDescription,
		Required:         src.Required,
		Multiple:         src.Multiple,
		Default:          src.Default,
		Options:          make([]WorkspaceToolOption, len(src.Options)),
	}
	if src.Min != nil {
		value := *src.Min
		out.Min = &value
	}
	if src.Max != nil {
		value := *src.Max
		out.Max = &value
	}
	copy(out.Options, src.Options)
	return out
}

func cloneStringSlice(src []string) []string {
	if src == nil {
		return []string{}
	}
	return slices.Clone(src)
}

func normalizeWorkspaceEmoji(icon WorkspaceEmoji) WorkspaceEmoji {
	out := WorkspaceEmoji{
		Background: strings.TrimSpace(icon.Background),
		Content:    strings.TrimSpace(icon.Content),
	}
	if out.Content == "" {
		out.Content = "🧩"
	}
	if out.Background == "" {
		out.Background = "#E5E7EB"
	}
	return out
}

func normalizePositiveInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func normalizeAPISchemaType(schemaType string) string {
	switch strings.ToLower(strings.TrimSpace(schemaType)) {
	case "swagger":
		return "swagger"
	case "openai_plugin":
		return "openai_plugin"
	case "openai_actions":
		return "openai_actions"
	default:
		return "openapi"
	}
}

func (s *Store) findWorkspaceIndexLocked(workspaceID string) int {
	return slices.IndexFunc(s.state.Workspaces, func(workspace Workspace) bool {
		return workspace.ID == workspaceID
	})
}
