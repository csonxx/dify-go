package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	TriggerCredentialTypeAPIKey       = "api-key"
	TriggerCredentialTypeOAuth2       = "oauth2"
	TriggerCredentialTypeUnauthorized = "unauthorized"
)

type WorkspaceTriggerProviderState struct {
	Provider             string                                `json:"provider"`
	OAuthClient          WorkspaceTriggerOAuthClient           `json:"oauth_client"`
	SubscriptionBuilders []WorkspaceTriggerSubscriptionBuilder `json:"subscription_builders,omitempty"`
	Subscriptions        []WorkspaceTriggerSubscription        `json:"subscriptions,omitempty"`
}

type WorkspaceTriggerOAuthClient struct {
	Enabled   bool              `json:"enabled"`
	Params    map[string]string `json:"params"`
	UpdatedAt int64             `json:"updated_at"`
}

type WorkspaceTriggerSubscriptionBuilder struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Provider       string                `json:"provider"`
	CredentialType string                `json:"credential_type"`
	Credentials    map[string]any        `json:"credentials"`
	Endpoint       string                `json:"endpoint"`
	Parameters     map[string]any        `json:"parameters"`
	Properties     map[string]any        `json:"properties"`
	Verified       bool                  `json:"verified"`
	Logs           []WorkspaceTriggerLog `json:"logs,omitempty"`
	CreatedBy      string                `json:"created_by"`
	CreatedAt      int64                 `json:"created_at"`
	UpdatedAt      int64                 `json:"updated_at"`
}

type WorkspaceTriggerSubscription struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	Provider       string                `json:"provider"`
	CredentialType string                `json:"credential_type"`
	Credentials    map[string]any        `json:"credentials"`
	Endpoint       string                `json:"endpoint"`
	Parameters     map[string]any        `json:"parameters"`
	Properties     map[string]any        `json:"properties"`
	WorkflowsInUse int                   `json:"workflows_in_use"`
	Logs           []WorkspaceTriggerLog `json:"logs,omitempty"`
	CreatedBy      string                `json:"created_by"`
	CreatedAt      int64                 `json:"created_at"`
	UpdatedAt      int64                 `json:"updated_at"`
}

type WorkspaceTriggerLog struct {
	ID        string                      `json:"id"`
	Endpoint  string                      `json:"endpoint"`
	Request   WorkspaceTriggerLogRequest  `json:"request"`
	Response  WorkspaceTriggerLogResponse `json:"response"`
	CreatedAt int64                       `json:"created_at"`
}

type WorkspaceTriggerLogRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Data    string            `json:"data"`
}

type WorkspaceTriggerLogResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Data       string            `json:"data"`
}

type UpdateWorkspaceTriggerSubscriptionBuilderInput struct {
	Name        *string
	Credentials map[string]any
	Parameters  map[string]any
	Properties  map[string]any
	Endpoint    *string
	Verified    *bool
}

type CreateWorkspaceTriggerSubscriptionInput struct {
	Name           string
	Provider       string
	CredentialType string
	Credentials    map[string]any
	Endpoint       string
	Parameters     map[string]any
	Properties     map[string]any
}

type UpdateWorkspaceTriggerSubscriptionInput struct {
	Name        *string
	Credentials map[string]any
	Parameters  map[string]any
	Properties  map[string]any
	Endpoint    *string
}

func normalizeWorkspaceTriggerProviderState(provider *WorkspaceTriggerProviderState) {
	if provider == nil {
		return
	}
	if provider.OAuthClient.Params == nil {
		provider.OAuthClient.Params = map[string]string{}
	}
	if provider.SubscriptionBuilders == nil {
		provider.SubscriptionBuilders = []WorkspaceTriggerSubscriptionBuilder{}
	}
	if provider.Subscriptions == nil {
		provider.Subscriptions = []WorkspaceTriggerSubscription{}
	}
	for i := range provider.SubscriptionBuilders {
		normalizeWorkspaceTriggerSubscriptionBuilder(&provider.SubscriptionBuilders[i])
	}
	for i := range provider.Subscriptions {
		normalizeWorkspaceTriggerSubscription(&provider.Subscriptions[i])
	}
}

func normalizeWorkspaceTriggerSubscriptionBuilder(builder *WorkspaceTriggerSubscriptionBuilder) {
	if builder == nil {
		return
	}
	builder.CredentialType = normalizeTriggerCredentialType(builder.CredentialType)
	if builder.Credentials == nil {
		builder.Credentials = map[string]any{}
	}
	if builder.Parameters == nil {
		builder.Parameters = map[string]any{}
	}
	if builder.Properties == nil {
		builder.Properties = map[string]any{}
	}
	if builder.Logs == nil {
		builder.Logs = []WorkspaceTriggerLog{}
	}
	for i := range builder.Logs {
		normalizeWorkspaceTriggerLog(&builder.Logs[i])
	}
}

func normalizeWorkspaceTriggerSubscription(subscription *WorkspaceTriggerSubscription) {
	if subscription == nil {
		return
	}
	subscription.CredentialType = normalizeTriggerCredentialType(subscription.CredentialType)
	if subscription.Credentials == nil {
		subscription.Credentials = map[string]any{}
	}
	if subscription.Parameters == nil {
		subscription.Parameters = map[string]any{}
	}
	if subscription.Properties == nil {
		subscription.Properties = map[string]any{}
	}
	if subscription.Logs == nil {
		subscription.Logs = []WorkspaceTriggerLog{}
	}
	for i := range subscription.Logs {
		normalizeWorkspaceTriggerLog(&subscription.Logs[i])
	}
}

func normalizeWorkspaceTriggerLog(log *WorkspaceTriggerLog) {
	if log == nil {
		return
	}
	if log.Request.Headers == nil {
		log.Request.Headers = map[string]string{}
	}
	if log.Response.Headers == nil {
		log.Response.Headers = map[string]string{}
	}
}

func cloneWorkspaceTriggerProviderState(src WorkspaceTriggerProviderState) WorkspaceTriggerProviderState {
	out := WorkspaceTriggerProviderState{
		Provider:             src.Provider,
		OAuthClient:          cloneWorkspaceTriggerOAuthClient(src.OAuthClient),
		SubscriptionBuilders: make([]WorkspaceTriggerSubscriptionBuilder, len(src.SubscriptionBuilders)),
		Subscriptions:        make([]WorkspaceTriggerSubscription, len(src.Subscriptions)),
	}
	for i, item := range src.SubscriptionBuilders {
		out.SubscriptionBuilders[i] = cloneWorkspaceTriggerSubscriptionBuilder(item)
	}
	for i, item := range src.Subscriptions {
		out.Subscriptions[i] = cloneWorkspaceTriggerSubscription(item)
	}
	normalizeWorkspaceTriggerProviderState(&out)
	return out
}

func cloneWorkspaceTriggerOAuthClient(src WorkspaceTriggerOAuthClient) WorkspaceTriggerOAuthClient {
	return WorkspaceTriggerOAuthClient{
		Enabled:   src.Enabled,
		Params:    cloneStringMap(src.Params),
		UpdatedAt: src.UpdatedAt,
	}
}

func cloneWorkspaceTriggerSubscriptionBuilder(src WorkspaceTriggerSubscriptionBuilder) WorkspaceTriggerSubscriptionBuilder {
	out := WorkspaceTriggerSubscriptionBuilder{
		ID:             src.ID,
		Name:           src.Name,
		Provider:       src.Provider,
		CredentialType: src.CredentialType,
		Credentials:    cloneMap(src.Credentials),
		Endpoint:       src.Endpoint,
		Parameters:     cloneMap(src.Parameters),
		Properties:     cloneMap(src.Properties),
		Verified:       src.Verified,
		Logs:           cloneWorkspaceTriggerLogs(src.Logs),
		CreatedBy:      src.CreatedBy,
		CreatedAt:      src.CreatedAt,
		UpdatedAt:      src.UpdatedAt,
	}
	normalizeWorkspaceTriggerSubscriptionBuilder(&out)
	return out
}

func cloneWorkspaceTriggerSubscription(src WorkspaceTriggerSubscription) WorkspaceTriggerSubscription {
	out := WorkspaceTriggerSubscription{
		ID:             src.ID,
		Name:           src.Name,
		Provider:       src.Provider,
		CredentialType: src.CredentialType,
		Credentials:    cloneMap(src.Credentials),
		Endpoint:       src.Endpoint,
		Parameters:     cloneMap(src.Parameters),
		Properties:     cloneMap(src.Properties),
		WorkflowsInUse: src.WorkflowsInUse,
		Logs:           cloneWorkspaceTriggerLogs(src.Logs),
		CreatedBy:      src.CreatedBy,
		CreatedAt:      src.CreatedAt,
		UpdatedAt:      src.UpdatedAt,
	}
	normalizeWorkspaceTriggerSubscription(&out)
	return out
}

func cloneWorkspaceTriggerLogs(src []WorkspaceTriggerLog) []WorkspaceTriggerLog {
	if src == nil {
		return []WorkspaceTriggerLog{}
	}
	out := make([]WorkspaceTriggerLog, len(src))
	for i, item := range src {
		out[i] = WorkspaceTriggerLog{
			ID:       item.ID,
			Endpoint: item.Endpoint,
			Request: WorkspaceTriggerLogRequest{
				Method:  item.Request.Method,
				URL:     item.Request.URL,
				Headers: cloneStringMap(item.Request.Headers),
				Data:    item.Request.Data,
			},
			Response: WorkspaceTriggerLogResponse{
				StatusCode: item.Response.StatusCode,
				Headers:    cloneStringMap(item.Response.Headers),
				Data:       item.Response.Data,
			},
			CreatedAt: item.CreatedAt,
		}
		normalizeWorkspaceTriggerLog(&out[i])
	}
	return out
}

func normalizeTriggerCredentialType(value string) string {
	switch strings.TrimSpace(value) {
	case TriggerCredentialTypeAPIKey:
		return TriggerCredentialTypeAPIKey
	case TriggerCredentialTypeOAuth2:
		return TriggerCredentialTypeOAuth2
	case TriggerCredentialTypeUnauthorized:
		return TriggerCredentialTypeUnauthorized
	default:
		return TriggerCredentialTypeUnauthorized
	}
}

func normalizeTriggerProviderKey(value string) string {
	return strings.TrimSpace(value)
}

func (s *Store) GetWorkspaceTriggerProviderState(workspaceID, provider string) (WorkspaceTriggerProviderState, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceTriggerProviderState{}, false
	}
	provider = normalizeTriggerProviderKey(provider)
	for _, item := range settings.TriggerProviders {
		if item.Provider == provider {
			return cloneWorkspaceTriggerProviderState(item), true
		}
	}
	return WorkspaceTriggerProviderState{}, false
}

func (s *Store) UpsertWorkspaceTriggerOAuthClient(workspaceID, provider string, params map[string]string, enabled bool, now time.Time) (WorkspaceTriggerProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerProviderState{}, fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	if provider == "" {
		return WorkspaceTriggerProviderState{}, fmt.Errorf("provider is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)
	state.OAuthClient = WorkspaceTriggerOAuthClient{
		Enabled:   enabled,
		Params:    cloneStringMap(params),
		UpdatedAt: now.UTC().Unix(),
	}
	normalizeWorkspaceTriggerProviderState(state)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceTriggerProviderState{}, err
	}
	return cloneWorkspaceTriggerProviderState(*state), nil
}

func (s *Store) DeleteWorkspaceTriggerOAuthClient(workspaceID, provider string, now time.Time) (WorkspaceTriggerProviderState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerProviderState{}, fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)
	state.OAuthClient = WorkspaceTriggerOAuthClient{
		Enabled:   false,
		Params:    map[string]string{},
		UpdatedAt: now.UTC().Unix(),
	}

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceTriggerProviderState{}, err
	}
	return cloneWorkspaceTriggerProviderState(*state), nil
}

func (s *Store) CreateWorkspaceTriggerSubscriptionBuilder(workspaceID string, user User, provider, credentialType, endpoint string, credentials map[string]any, verified bool, now time.Time) (WorkspaceTriggerSubscriptionBuilder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerSubscriptionBuilder{}, fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	if provider == "" {
		return WorkspaceTriggerSubscriptionBuilder{}, fmt.Errorf("provider is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)

	timestamp := now.UTC().Unix()
	builder := WorkspaceTriggerSubscriptionBuilder{
		ID:             generateID("trb"),
		Name:           fmt.Sprintf("Subscription %d", len(state.Subscriptions)+len(state.SubscriptionBuilders)+1),
		Provider:       provider,
		CredentialType: normalizeTriggerCredentialType(credentialType),
		Credentials:    cloneMap(credentials),
		Endpoint:       strings.TrimSpace(endpoint),
		Parameters:     map[string]any{},
		Properties:     map[string]any{},
		Verified:       verified,
		Logs:           []WorkspaceTriggerLog{},
		CreatedBy:      user.ID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}
	state.SubscriptionBuilders = append(state.SubscriptionBuilders, builder)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceTriggerSubscriptionBuilder{}, err
	}
	return cloneWorkspaceTriggerSubscriptionBuilder(builder), nil
}

func (s *Store) GetWorkspaceTriggerSubscriptionBuilder(workspaceID, provider, builderID string) (WorkspaceTriggerSubscriptionBuilder, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceTriggerSubscriptionBuilder{}, false
	}
	provider = normalizeTriggerProviderKey(provider)
	builderID = strings.TrimSpace(builderID)
	for _, item := range settings.TriggerProviders {
		if item.Provider != provider {
			continue
		}
		for _, builder := range item.SubscriptionBuilders {
			if builder.ID == builderID {
				return cloneWorkspaceTriggerSubscriptionBuilder(builder), true
			}
		}
	}
	return WorkspaceTriggerSubscriptionBuilder{}, false
}

func (s *Store) FindWorkspaceTriggerSubscriptionBuilderByID(builderID string) (Workspace, WorkspaceTriggerSubscriptionBuilder, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	builderID = strings.TrimSpace(builderID)
	if builderID == "" {
		return Workspace{}, WorkspaceTriggerSubscriptionBuilder{}, false
	}

	for _, workspace := range s.state.Workspaces {
		settings := cloneWorkspaceToolSettings(workspace.ToolSettings)
		normalizeWorkspaceToolSettings(&settings)
		for _, provider := range settings.TriggerProviders {
			for _, builder := range provider.SubscriptionBuilders {
				if builder.ID == builderID {
					return workspace, cloneWorkspaceTriggerSubscriptionBuilder(builder), true
				}
			}
		}
	}

	return Workspace{}, WorkspaceTriggerSubscriptionBuilder{}, false
}

func (s *Store) UpdateWorkspaceTriggerSubscriptionBuilder(workspaceID, provider, builderID string, input UpdateWorkspaceTriggerSubscriptionBuilderInput, now time.Time) (WorkspaceTriggerSubscriptionBuilder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerSubscriptionBuilder{}, fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	builderID = strings.TrimSpace(builderID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)
	index := slices.IndexFunc(state.SubscriptionBuilders, func(item WorkspaceTriggerSubscriptionBuilder) bool {
		return item.ID == builderID
	})
	if index < 0 {
		return WorkspaceTriggerSubscriptionBuilder{}, fmt.Errorf("subscription builder %s not found", builderID)
	}

	builder := cloneWorkspaceTriggerSubscriptionBuilder(state.SubscriptionBuilders[index])
	if input.Name != nil {
		builder.Name = strings.TrimSpace(*input.Name)
	}
	if input.Credentials != nil {
		builder.Credentials = cloneMap(input.Credentials)
	}
	if input.Parameters != nil {
		builder.Parameters = cloneMap(input.Parameters)
	}
	if input.Properties != nil {
		builder.Properties = cloneMap(input.Properties)
	}
	if input.Endpoint != nil {
		builder.Endpoint = strings.TrimSpace(*input.Endpoint)
	}
	if input.Verified != nil {
		builder.Verified = *input.Verified
	}
	builder.UpdatedAt = now.UTC().Unix()
	state.SubscriptionBuilders[index] = builder

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceTriggerSubscriptionBuilder{}, err
	}
	return cloneWorkspaceTriggerSubscriptionBuilder(builder), nil
}

func (s *Store) DeleteWorkspaceTriggerSubscriptionBuilder(workspaceID, provider, builderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	builderID = strings.TrimSpace(builderID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)
	index := slices.IndexFunc(state.SubscriptionBuilders, func(item WorkspaceTriggerSubscriptionBuilder) bool {
		return item.ID == builderID
	})
	if index < 0 {
		return fmt.Errorf("subscription builder %s not found", builderID)
	}
	state.SubscriptionBuilders = append(state.SubscriptionBuilders[:index], state.SubscriptionBuilders[index+1:]...)
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListWorkspaceTriggerSubscriptions(workspaceID, provider string) []WorkspaceTriggerSubscription {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspaceTriggerSubscription{}
	}
	provider = normalizeTriggerProviderKey(provider)
	for _, item := range settings.TriggerProviders {
		if item.Provider != provider {
			continue
		}
		out := make([]WorkspaceTriggerSubscription, len(item.Subscriptions))
		for i, subscription := range item.Subscriptions {
			out[i] = cloneWorkspaceTriggerSubscription(subscription)
		}
		slices.SortFunc(out, func(a, b WorkspaceTriggerSubscription) int {
			if a.UpdatedAt == b.UpdatedAt {
				return strings.Compare(a.Name, b.Name)
			}
			if a.UpdatedAt > b.UpdatedAt {
				return -1
			}
			return 1
		})
		return out
	}
	return []WorkspaceTriggerSubscription{}
}

func (s *Store) GetWorkspaceTriggerSubscription(workspaceID, subscriptionID string) (WorkspaceTriggerSubscription, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspaceTriggerSubscription{}, false
	}
	subscriptionID = strings.TrimSpace(subscriptionID)
	for _, item := range settings.TriggerProviders {
		for _, subscription := range item.Subscriptions {
			if subscription.ID == subscriptionID {
				return cloneWorkspaceTriggerSubscription(subscription), true
			}
		}
	}
	return WorkspaceTriggerSubscription{}, false
}

func (s *Store) FindWorkspaceTriggerSubscriptionByID(subscriptionID string) (Workspace, WorkspaceTriggerSubscription, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return Workspace{}, WorkspaceTriggerSubscription{}, false
	}

	for _, workspace := range s.state.Workspaces {
		settings := cloneWorkspaceToolSettings(workspace.ToolSettings)
		normalizeWorkspaceToolSettings(&settings)
		for _, provider := range settings.TriggerProviders {
			for _, subscription := range provider.Subscriptions {
				if subscription.ID == subscriptionID {
					return workspace, cloneWorkspaceTriggerSubscription(subscription), true
				}
			}
		}
	}
	return Workspace{}, WorkspaceTriggerSubscription{}, false
}

func (s *Store) CreateWorkspaceTriggerSubscription(workspaceID string, user User, input CreateWorkspaceTriggerSubscriptionInput, now time.Time) (WorkspaceTriggerSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerSubscription{}, fmt.Errorf("workspace not found")
	}

	provider := normalizeTriggerProviderKey(input.Provider)
	if provider == "" {
		return WorkspaceTriggerSubscription{}, fmt.Errorf("provider is required")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return WorkspaceTriggerSubscription{}, fmt.Errorf("name is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)

	timestamp := now.UTC().Unix()
	subscription := WorkspaceTriggerSubscription{
		ID:             generateID("trs"),
		Name:           name,
		Provider:       provider,
		CredentialType: normalizeTriggerCredentialType(input.CredentialType),
		Credentials:    cloneMap(input.Credentials),
		Endpoint:       strings.TrimSpace(input.Endpoint),
		Parameters:     cloneMap(input.Parameters),
		Properties:     cloneMap(input.Properties),
		WorkflowsInUse: 0,
		Logs:           []WorkspaceTriggerLog{},
		CreatedBy:      user.ID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}
	state.Subscriptions = append(state.Subscriptions, subscription)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspaceTriggerSubscription{}, err
	}
	return cloneWorkspaceTriggerSubscription(subscription), nil
}

func (s *Store) UpdateWorkspaceTriggerSubscription(workspaceID, subscriptionID string, input UpdateWorkspaceTriggerSubscriptionInput, now time.Time) (WorkspaceTriggerSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspaceTriggerSubscription{}, fmt.Errorf("workspace not found")
	}

	subscriptionID = strings.TrimSpace(subscriptionID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	for providerIndex := range settings.TriggerProviders {
		subIndex := slices.IndexFunc(settings.TriggerProviders[providerIndex].Subscriptions, func(item WorkspaceTriggerSubscription) bool {
			return item.ID == subscriptionID
		})
		if subIndex < 0 {
			continue
		}

		subscription := cloneWorkspaceTriggerSubscription(settings.TriggerProviders[providerIndex].Subscriptions[subIndex])
		if input.Name != nil {
			subscription.Name = strings.TrimSpace(*input.Name)
		}
		if input.Credentials != nil {
			subscription.Credentials = cloneMap(input.Credentials)
		}
		if input.Parameters != nil {
			subscription.Parameters = cloneMap(input.Parameters)
		}
		if input.Properties != nil {
			subscription.Properties = cloneMap(input.Properties)
		}
		if input.Endpoint != nil {
			subscription.Endpoint = strings.TrimSpace(*input.Endpoint)
		}
		subscription.UpdatedAt = now.UTC().Unix()
		settings.TriggerProviders[providerIndex].Subscriptions[subIndex] = subscription

		s.state.Workspaces[workspaceIndex].ToolSettings = settings
		if err := s.saveLocked(); err != nil {
			return WorkspaceTriggerSubscription{}, err
		}
		return cloneWorkspaceTriggerSubscription(subscription), nil
	}

	return WorkspaceTriggerSubscription{}, fmt.Errorf("subscription %s not found", subscriptionID)
}

func (s *Store) DeleteWorkspaceTriggerSubscription(workspaceID, subscriptionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	subscriptionID = strings.TrimSpace(subscriptionID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	for providerIndex := range settings.TriggerProviders {
		subIndex := slices.IndexFunc(settings.TriggerProviders[providerIndex].Subscriptions, func(item WorkspaceTriggerSubscription) bool {
			return item.ID == subscriptionID
		})
		if subIndex < 0 {
			continue
		}
		settings.TriggerProviders[providerIndex].Subscriptions = append(
			settings.TriggerProviders[providerIndex].Subscriptions[:subIndex],
			settings.TriggerProviders[providerIndex].Subscriptions[subIndex+1:]...,
		)
		s.state.Workspaces[workspaceIndex].ToolSettings = settings
		return s.saveLocked()
	}

	return fmt.Errorf("subscription %s not found", subscriptionID)
}

func (s *Store) AppendWorkspaceTriggerBuilderLog(workspaceID, provider, builderID string, log WorkspaceTriggerLog, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	provider = normalizeTriggerProviderKey(provider)
	builderID = strings.TrimSpace(builderID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	state := ensureWorkspaceTriggerProviderState(&settings, provider)
	index := slices.IndexFunc(state.SubscriptionBuilders, func(item WorkspaceTriggerSubscriptionBuilder) bool {
		return item.ID == builderID
	})
	if index < 0 {
		return fmt.Errorf("subscription builder %s not found", builderID)
	}

	builder := cloneWorkspaceTriggerSubscriptionBuilder(state.SubscriptionBuilders[index])
	builder.Logs = append([]WorkspaceTriggerLog{cloneWorkspaceTriggerLog(log)}, builder.Logs...)
	if len(builder.Logs) > 20 {
		builder.Logs = builder.Logs[:20]
	}
	builder.UpdatedAt = now.UTC().Unix()
	state.SubscriptionBuilders[index] = builder

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) AppendWorkspaceTriggerSubscriptionLog(workspaceID, subscriptionID string, log WorkspaceTriggerLog, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	subscriptionID = strings.TrimSpace(subscriptionID)
	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	for providerIndex := range settings.TriggerProviders {
		subIndex := slices.IndexFunc(settings.TriggerProviders[providerIndex].Subscriptions, func(item WorkspaceTriggerSubscription) bool {
			return item.ID == subscriptionID
		})
		if subIndex < 0 {
			continue
		}

		subscription := cloneWorkspaceTriggerSubscription(settings.TriggerProviders[providerIndex].Subscriptions[subIndex])
		subscription.Logs = append([]WorkspaceTriggerLog{cloneWorkspaceTriggerLog(log)}, subscription.Logs...)
		if len(subscription.Logs) > 20 {
			subscription.Logs = subscription.Logs[:20]
		}
		subscription.UpdatedAt = now.UTC().Unix()
		settings.TriggerProviders[providerIndex].Subscriptions[subIndex] = subscription

		s.state.Workspaces[workspaceIndex].ToolSettings = settings
		return s.saveLocked()
	}

	return fmt.Errorf("subscription %s not found", subscriptionID)
}

func cloneWorkspaceTriggerLog(src WorkspaceTriggerLog) WorkspaceTriggerLog {
	out := WorkspaceTriggerLog{
		ID:       src.ID,
		Endpoint: src.Endpoint,
		Request: WorkspaceTriggerLogRequest{
			Method:  src.Request.Method,
			URL:     src.Request.URL,
			Headers: cloneStringMap(src.Request.Headers),
			Data:    src.Request.Data,
		},
		Response: WorkspaceTriggerLogResponse{
			StatusCode: src.Response.StatusCode,
			Headers:    cloneStringMap(src.Response.Headers),
			Data:       src.Response.Data,
		},
		CreatedAt: src.CreatedAt,
	}
	normalizeWorkspaceTriggerLog(&out)
	return out
}

func ensureWorkspaceTriggerProviderState(settings *WorkspaceToolSettings, provider string) *WorkspaceTriggerProviderState {
	for i := range settings.TriggerProviders {
		if settings.TriggerProviders[i].Provider == provider {
			return &settings.TriggerProviders[i]
		}
	}

	settings.TriggerProviders = append(settings.TriggerProviders, WorkspaceTriggerProviderState{
		Provider:             provider,
		OAuthClient:          WorkspaceTriggerOAuthClient{Params: map[string]string{}},
		SubscriptionBuilders: []WorkspaceTriggerSubscriptionBuilder{},
		Subscriptions:        []WorkspaceTriggerSubscription{},
	})
	return &settings.TriggerProviders[len(settings.TriggerProviders)-1]
}
