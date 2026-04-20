package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type WorkspaceEndpoint struct {
	ID                     string         `json:"id"`
	PluginID               string         `json:"plugin_id"`
	PluginUniqueIdentifier string         `json:"plugin_unique_identifier"`
	Name                   string         `json:"name"`
	Settings               map[string]any `json:"settings"`
	Enabled                bool           `json:"enabled"`
	HookID                 string         `json:"hook_id"`
	CreatedBy              string         `json:"created_by"`
	CreatedAt              int64          `json:"created_at"`
	UpdatedAt              int64          `json:"updated_at"`
}

type CreateWorkspaceEndpointInput struct {
	PluginID               string
	PluginUniqueIdentifier string
	Name                   string
	Settings               map[string]any
}

type UpdateWorkspaceEndpointInput struct {
	EndpointID string
	Name       string
	Settings   map[string]any
}

func normalizeWorkspaceEndpoint(endpoint *WorkspaceEndpoint) {
	if endpoint == nil {
		return
	}
	if endpoint.Settings == nil {
		endpoint.Settings = map[string]any{}
	}
}

func cloneWorkspaceEndpoint(src WorkspaceEndpoint) WorkspaceEndpoint {
	out := WorkspaceEndpoint{
		ID:                     src.ID,
		PluginID:               src.PluginID,
		PluginUniqueIdentifier: src.PluginUniqueIdentifier,
		Name:                   src.Name,
		Settings:               cloneMap(src.Settings),
		Enabled:                src.Enabled,
		HookID:                 src.HookID,
		CreatedBy:              src.CreatedBy,
		CreatedAt:              src.CreatedAt,
		UpdatedAt:              src.UpdatedAt,
	}
	normalizeWorkspaceEndpoint(&out)
	return out
}

func (s *Store) ListWorkspaceEndpoints(workspaceID string) []WorkspaceEndpoint {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceEndpoint{}
	}
	items := make([]WorkspaceEndpoint, len(settings.Endpoints))
	for i, item := range settings.Endpoints {
		items[i] = cloneWorkspaceEndpoint(item)
	}
	slices.SortFunc(items, func(a, b WorkspaceEndpoint) int {
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

func (s *Store) ListWorkspaceEndpointsByPlugin(workspaceID, pluginID string) []WorkspaceEndpoint {
	items := s.ListWorkspaceEndpoints(workspaceID)
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return items
	}
	filtered := make([]WorkspaceEndpoint, 0, len(items))
	for _, item := range items {
		if item.PluginID == pluginID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *Store) GetWorkspaceEndpoint(workspaceID, endpointID string) (WorkspaceEndpoint, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceEndpoint{}, false
	}
	for _, item := range settings.Endpoints {
		if item.ID == strings.TrimSpace(endpointID) {
			return cloneWorkspaceEndpoint(item), true
		}
	}
	return WorkspaceEndpoint{}, false
}

func (s *Store) FindWorkspaceEndpointByHookID(hookID string) (Workspace, WorkspaceEndpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hookID = strings.TrimSpace(hookID)
	if hookID == "" {
		return Workspace{}, WorkspaceEndpoint{}, false
	}

	for _, workspace := range s.state.Workspaces {
		settings := cloneWorkspaceToolSettings(workspace.ToolSettings)
		normalizeWorkspaceToolSettings(&settings)
		for _, item := range settings.Endpoints {
			if item.HookID == hookID {
				return workspace, cloneWorkspaceEndpoint(item), true
			}
		}
	}

	return Workspace{}, WorkspaceEndpoint{}, false
}

func (s *Store) CreateWorkspaceEndpoint(workspaceID string, user User, input CreateWorkspaceEndpointInput, now time.Time) (WorkspaceEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceEndpoint{}, fmt.Errorf("workspace not found")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return WorkspaceEndpoint{}, fmt.Errorf("name is required")
	}

	pluginID := strings.TrimSpace(input.PluginID)
	if pluginID == "" {
		return WorkspaceEndpoint{}, fmt.Errorf("plugin_id is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	timestamp := now.UTC().Unix()
	endpoint := WorkspaceEndpoint{
		ID:                     generateID("endpoint"),
		PluginID:               pluginID,
		PluginUniqueIdentifier: strings.TrimSpace(input.PluginUniqueIdentifier),
		Name:                   name,
		Settings:               cloneMap(input.Settings),
		Enabled:                true,
		HookID:                 generateID("hook"),
		CreatedBy:              user.ID,
		CreatedAt:              timestamp,
		UpdatedAt:              timestamp,
	}
	settings.Endpoints = append(settings.Endpoints, endpoint)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceEndpoint{}, err
	}
	return cloneWorkspaceEndpoint(endpoint), nil
}

func (s *Store) UpdateWorkspaceEndpoint(workspaceID string, user User, input UpdateWorkspaceEndpointInput, now time.Time) (WorkspaceEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceEndpoint{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	index := slices.IndexFunc(settings.Endpoints, func(item WorkspaceEndpoint) bool {
		return item.ID == strings.TrimSpace(input.EndpointID)
	})
	if index < 0 {
		return WorkspaceEndpoint{}, fmt.Errorf("endpoint %s not found", input.EndpointID)
	}

	endpoint := cloneWorkspaceEndpoint(settings.Endpoints[index])
	if name := strings.TrimSpace(input.Name); name != "" {
		endpoint.Name = name
	}
	endpoint.Settings = cloneMap(input.Settings)
	endpoint.UpdatedAt = now.UTC().Unix()
	endpoint.CreatedBy = firstNonEmpty(endpoint.CreatedBy, user.ID)
	settings.Endpoints[index] = endpoint

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceEndpoint{}, err
	}
	return cloneWorkspaceEndpoint(endpoint), nil
}

func (s *Store) DeleteWorkspaceEndpoint(workspaceID, endpointID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	index := slices.IndexFunc(settings.Endpoints, func(item WorkspaceEndpoint) bool {
		return item.ID == strings.TrimSpace(endpointID)
	})
	if index < 0 {
		return fmt.Errorf("endpoint %s not found", endpointID)
	}

	settings.Endpoints = append(settings.Endpoints[:index], settings.Endpoints[index+1:]...)
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) SetWorkspaceEndpointEnabled(workspaceID, endpointID string, enabled bool, now time.Time) (WorkspaceEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceEndpoint{}, fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	index := slices.IndexFunc(settings.Endpoints, func(item WorkspaceEndpoint) bool {
		return item.ID == strings.TrimSpace(endpointID)
	})
	if index < 0 {
		return WorkspaceEndpoint{}, fmt.Errorf("endpoint %s not found", endpointID)
	}

	endpoint := cloneWorkspaceEndpoint(settings.Endpoints[index])
	endpoint.Enabled = enabled
	endpoint.UpdatedAt = now.UTC().Unix()
	settings.Endpoints[index] = endpoint

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceEndpoint{}, err
	}
	return cloneWorkspaceEndpoint(endpoint), nil
}
