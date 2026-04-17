package state

import (
	"fmt"
	"strings"
)

type WorkspaceModelSettings struct {
	DefaultModels []WorkspaceDefaultModel       `json:"default_models,omitempty"`
	Providers     []WorkspaceModelProviderState `json:"providers,omitempty"`
}

type WorkspaceDefaultModel struct {
	ModelType string `json:"model_type"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
}

type WorkspaceModelProviderState struct {
	Provider                   string                  `json:"provider"`
	ProviderCredentials        []WorkspaceCredential   `json:"provider_credentials,omitempty"`
	ActiveProviderCredentialID string                  `json:"active_provider_credential_id,omitempty"`
	ModelSettings              []WorkspaceModelSetting `json:"model_settings,omitempty"`
}

type WorkspaceCredential struct {
	CredentialID string         `json:"credential_id"`
	Name         string         `json:"name"`
	Credentials  map[string]any `json:"credentials"`
}

type WorkspaceModelSetting struct {
	Model              string                       `json:"model"`
	ModelType          string                       `json:"model_type"`
	Enabled            *bool                        `json:"enabled,omitempty"`
	Credentials        []WorkspaceCredential        `json:"credentials,omitempty"`
	ActiveCredentialID string                       `json:"active_credential_id,omitempty"`
	LoadBalancing      WorkspaceLoadBalancingConfig `json:"load_balancing"`
}

type WorkspaceLoadBalancingConfig struct {
	Enabled bool                          `json:"enabled"`
	Configs []WorkspaceLoadBalancingEntry `json:"configs,omitempty"`
}

type WorkspaceLoadBalancingEntry struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name"`
	Enabled      bool           `json:"enabled"`
	Credentials  map[string]any `json:"credentials"`
	CredentialID string         `json:"credential_id,omitempty"`
}

func normalizeWorkspaceModelSettings(settings *WorkspaceModelSettings) {
	if settings == nil {
		return
	}
	if settings.DefaultModels == nil {
		settings.DefaultModels = []WorkspaceDefaultModel{}
	}
	if settings.Providers == nil {
		settings.Providers = []WorkspaceModelProviderState{}
	}
	for i := range settings.Providers {
		if settings.Providers[i].ProviderCredentials == nil {
			settings.Providers[i].ProviderCredentials = []WorkspaceCredential{}
		}
		if settings.Providers[i].ModelSettings == nil {
			settings.Providers[i].ModelSettings = []WorkspaceModelSetting{}
		}
		for j := range settings.Providers[i].ProviderCredentials {
			if settings.Providers[i].ProviderCredentials[j].Credentials == nil {
				settings.Providers[i].ProviderCredentials[j].Credentials = map[string]any{}
			}
		}
		for j := range settings.Providers[i].ModelSettings {
			if settings.Providers[i].ModelSettings[j].Credentials == nil {
				settings.Providers[i].ModelSettings[j].Credentials = []WorkspaceCredential{}
			}
			normalizeLoadBalancing(&settings.Providers[i].ModelSettings[j].LoadBalancing)
			for k := range settings.Providers[i].ModelSettings[j].Credentials {
				if settings.Providers[i].ModelSettings[j].Credentials[k].Credentials == nil {
					settings.Providers[i].ModelSettings[j].Credentials[k].Credentials = map[string]any{}
				}
			}
		}
	}
}

func normalizeLoadBalancing(config *WorkspaceLoadBalancingConfig) {
	if config == nil {
		return
	}
	if config.Configs == nil {
		config.Configs = []WorkspaceLoadBalancingEntry{}
	}
	for i := range config.Configs {
		if config.Configs[i].Credentials == nil {
			config.Configs[i].Credentials = map[string]any{}
		}
	}
}

func (s *Store) GetWorkspaceModelSettings(workspaceID string) (WorkspaceModelSettings, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, workspace := range s.state.Workspaces {
		if workspace.ID != workspaceID {
			continue
		}
		normalizeWorkspaceModelSettings(&workspace.ModelSettings)
		return cloneWorkspaceModelSettings(workspace.ModelSettings), true
	}
	return WorkspaceModelSettings{}, false
}

func (s *Store) MutateWorkspaceModelSettings(workspaceID string, apply func(settings *WorkspaceModelSettings) error) (WorkspaceModelSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.state.Workspaces {
		if s.state.Workspaces[i].ID != workspaceID {
			continue
		}
		normalizeWorkspaceModelSettings(&s.state.Workspaces[i].ModelSettings)
		settings := cloneWorkspaceModelSettings(s.state.Workspaces[i].ModelSettings)
		if err := apply(&settings); err != nil {
			return WorkspaceModelSettings{}, err
		}
		normalizeWorkspaceModelSettings(&settings)
		s.state.Workspaces[i].ModelSettings = settings
		if err := s.saveLocked(); err != nil {
			return WorkspaceModelSettings{}, err
		}
		return cloneWorkspaceModelSettings(settings), nil
	}
	return WorkspaceModelSettings{}, fmt.Errorf("workspace not found")
}

func cloneWorkspaceModelSettings(src WorkspaceModelSettings) WorkspaceModelSettings {
	dst := WorkspaceModelSettings{
		DefaultModels: make([]WorkspaceDefaultModel, len(src.DefaultModels)),
		Providers:     make([]WorkspaceModelProviderState, len(src.Providers)),
	}
	copy(dst.DefaultModels, src.DefaultModels)
	for i, provider := range src.Providers {
		dst.Providers[i] = WorkspaceModelProviderState{
			Provider:                   provider.Provider,
			ProviderCredentials:        make([]WorkspaceCredential, len(provider.ProviderCredentials)),
			ActiveProviderCredentialID: provider.ActiveProviderCredentialID,
			ModelSettings:              make([]WorkspaceModelSetting, len(provider.ModelSettings)),
		}
		for j, credential := range provider.ProviderCredentials {
			dst.Providers[i].ProviderCredentials[j] = cloneWorkspaceCredential(credential)
		}
		for j, model := range provider.ModelSettings {
			dst.Providers[i].ModelSettings[j] = cloneWorkspaceModelSetting(model)
		}
	}
	return dst
}

func cloneWorkspaceCredential(src WorkspaceCredential) WorkspaceCredential {
	return WorkspaceCredential{
		CredentialID: src.CredentialID,
		Name:         src.Name,
		Credentials:  cloneMap(src.Credentials),
	}
}

func cloneWorkspaceModelSetting(src WorkspaceModelSetting) WorkspaceModelSetting {
	var enabled *bool
	if src.Enabled != nil {
		value := *src.Enabled
		enabled = &value
	}
	out := WorkspaceModelSetting{
		Model:              src.Model,
		ModelType:          src.ModelType,
		Enabled:            enabled,
		Credentials:        make([]WorkspaceCredential, len(src.Credentials)),
		ActiveCredentialID: src.ActiveCredentialID,
		LoadBalancing: WorkspaceLoadBalancingConfig{
			Enabled: src.LoadBalancing.Enabled,
			Configs: make([]WorkspaceLoadBalancingEntry, len(src.LoadBalancing.Configs)),
		},
	}
	for i, credential := range src.Credentials {
		out.Credentials[i] = cloneWorkspaceCredential(credential)
	}
	for i, entry := range src.LoadBalancing.Configs {
		out.LoadBalancing.Configs[i] = WorkspaceLoadBalancingEntry{
			ID:           entry.ID,
			Name:         entry.Name,
			Enabled:      entry.Enabled,
			Credentials:  cloneMap(entry.Credentials),
			CredentialID: entry.CredentialID,
		}
	}
	return out
}

func FindWorkspaceProviderState(settings *WorkspaceModelSettings, provider string) *WorkspaceModelProviderState {
	for i := range settings.Providers {
		if settings.Providers[i].Provider == provider {
			return &settings.Providers[i]
		}
	}
	settings.Providers = append(settings.Providers, WorkspaceModelProviderState{
		Provider:            provider,
		ProviderCredentials: []WorkspaceCredential{},
		ModelSettings:       []WorkspaceModelSetting{},
	})
	return &settings.Providers[len(settings.Providers)-1]
}

func FindWorkspaceModelState(provider *WorkspaceModelProviderState, modelType, model string) *WorkspaceModelSetting {
	for i := range provider.ModelSettings {
		if provider.ModelSettings[i].ModelType == modelType && provider.ModelSettings[i].Model == model {
			return &provider.ModelSettings[i]
		}
	}
	provider.ModelSettings = append(provider.ModelSettings, WorkspaceModelSetting{
		Model:       model,
		ModelType:   modelType,
		Credentials: []WorkspaceCredential{},
		LoadBalancing: WorkspaceLoadBalancingConfig{
			Enabled: false,
			Configs: []WorkspaceLoadBalancingEntry{},
		},
	})
	return &provider.ModelSettings[len(provider.ModelSettings)-1]
}

func WorkspaceDefaultModelMap(settings WorkspaceModelSettings) map[string]WorkspaceDefaultModel {
	out := make(map[string]WorkspaceDefaultModel, len(settings.DefaultModels))
	for _, item := range settings.DefaultModels {
		if strings.TrimSpace(item.ModelType) == "" {
			continue
		}
		out[item.ModelType] = item
	}
	return out
}
