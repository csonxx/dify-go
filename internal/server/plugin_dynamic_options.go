package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/langgenius/dify-go/internal/state"
)

type pluginDynamicOptionsRequest struct {
	PluginID     string
	Provider     string
	Action       string
	Parameter    string
	CredentialID string
	ProviderType string
	Credentials  map[string]any
	Extra        map[string]any
}

func pluginDynamicOptionsRequestFromQuery(r *http.Request) pluginDynamicOptionsRequest {
	query := r.URL.Query()
	extra := map[string]any{}
	known := map[string]struct{}{
		"plugin_id":     {},
		"provider":      {},
		"action":        {},
		"parameter":     {},
		"credential_id": {},
		"provider_type": {},
	}
	for key, values := range query {
		if _, ok := known[key]; ok || len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			extra[key] = values[0]
			continue
		}
		copied := append([]string{}, values...)
		extra[key] = copied
	}
	return pluginDynamicOptionsRequest{
		PluginID:     strings.TrimSpace(query.Get("plugin_id")),
		Provider:     strings.TrimSpace(query.Get("provider")),
		Action:       strings.TrimSpace(query.Get("action")),
		Parameter:    strings.TrimSpace(query.Get("parameter")),
		CredentialID: strings.TrimSpace(query.Get("credential_id")),
		ProviderType: strings.TrimSpace(query.Get("provider_type")),
		Extra:        extra,
	}
}

func decodePluginDynamicOptionsRequest(r *http.Request) (pluginDynamicOptionsRequest, error) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if err == io.EOF {
			return pluginDynamicOptionsRequest{}, nil
		}
		return pluginDynamicOptionsRequest{}, err
	}
	return pluginDynamicOptionsRequest{
		PluginID:     strings.TrimSpace(stringFromAny(payload["plugin_id"])),
		Provider:     strings.TrimSpace(stringFromAny(payload["provider"])),
		Action:       strings.TrimSpace(stringFromAny(payload["action"])),
		Parameter:    strings.TrimSpace(stringFromAny(payload["parameter"])),
		CredentialID: strings.TrimSpace(stringFromAny(payload["credential_id"])),
		ProviderType: strings.TrimSpace(stringFromAny(payload["provider_type"])),
		Credentials:  mapFromAny(payload["credentials"]),
		Extra:        mapFromAny(payload["extra"]),
	}, nil
}

func (s *server) pluginDynamicOptions(workspace state.Workspace, request pluginDynamicOptionsRequest) []map[string]any {
	request = normalizePluginDynamicOptionsRequest(request)
	switch s.inferPluginDynamicOptionsProviderType(workspace.ID, request) {
	case "trigger":
		return s.triggerPluginDynamicOptions(workspace.ID, request)
	case "datasource":
		return s.datasourcePluginDynamicOptions(workspace, request)
	default:
		return s.toolPluginDynamicOptions(workspace.ID, request)
	}
}

func normalizePluginDynamicOptionsRequest(request pluginDynamicOptionsRequest) pluginDynamicOptionsRequest {
	request.PluginID = strings.TrimSpace(request.PluginID)
	request.Provider = strings.TrimSpace(request.Provider)
	request.Action = strings.TrimSpace(request.Action)
	request.Parameter = strings.TrimSpace(request.Parameter)
	request.CredentialID = strings.TrimSpace(request.CredentialID)
	request.ProviderType = strings.ToLower(strings.TrimSpace(request.ProviderType))
	if request.Credentials == nil {
		request.Credentials = map[string]any{}
	}
	if request.Extra == nil {
		request.Extra = map[string]any{}
	}
	return request
}

func (s *server) inferPluginDynamicOptionsProviderType(workspaceID string, request pluginDynamicOptionsRequest) string {
	switch request.ProviderType {
	case "trigger", "tool", "datasource":
		return request.ProviderType
	}
	if _, ok := triggerCatalogByProvider(request.Provider, s.cfg.AppVersion); ok {
		return "trigger"
	}
	if _, ok := s.store.GetWorkspaceTriggerProviderState(workspaceID, normalizeTriggerProviderParam(request.Provider)); ok {
		return "trigger"
	}
	if _, ok := ragPipelineDatasourceProviderSpecByProvider(request.PluginID, request.Provider); ok {
		return "datasource"
	}
	if _, ok := s.store.GetWorkspaceDatasourceProviderState(workspaceID, request.PluginID, request.Provider); ok {
		return "datasource"
	}
	for _, plugin := range s.store.ListWorkspaceInstalledPlugins(workspaceID) {
		if request.PluginID != "" && plugin.PluginID != request.PluginID {
			continue
		}
		switch strings.TrimSpace(plugin.Category) {
		case "trigger":
			return "trigger"
		case "datasource":
			return "datasource"
		case "tool", "model":
			return "tool"
		}
	}
	return "tool"
}

func (s *server) triggerPluginDynamicOptions(workspaceID string, request pluginDynamicOptionsRequest) []map[string]any {
	provider := normalizeTriggerProviderParam(firstNonEmpty(request.Provider, triggerProviderFromPluginID(request.PluginID)))
	catalog, ok := triggerCatalogByProvider(provider, s.cfg.AppVersion)
	if !ok {
		catalog = fallbackTriggerCatalog(provider, s.cfg.AppVersion)
	}

	collector := newDynamicOptionCollector()
	parameter := normalizeDynamicOptionParameter(request.Parameter)
	switch parameter {
	case "provider", "provider_id", "trigger_provider", "trigger_provider_id":
		for _, item := range triggerProviderCatalogs(s.cfg.AppVersion) {
			collector.add(item.Provider, item.Label, map[string]any{"icon": item.Icon})
		}
		return collector.items()
	case "event", "event_type", "event_types":
		appendTriggerEventOptions(collector, catalog)
	case "subscription", "subscription_id", "subscriptions":
		for _, item := range s.store.ListWorkspaceTriggerSubscriptions(workspaceID, provider) {
			collector.add(item.ID, firstNonEmpty(item.Name, item.Provider, item.ID), nil)
		}
	case "subscription_builder", "subscription_builder_id", "builder", "builder_id":
		if providerState, found := s.store.GetWorkspaceTriggerProviderState(workspaceID, provider); found {
			for _, item := range providerState.SubscriptionBuilders {
				collector.add(item.ID, firstNonEmpty(item.Name, item.Provider, item.ID), nil)
			}
		}
	default:
		appendOptionsFromSchemas(collector, request.Parameter, catalog.SubscriptionParameters)
		appendOptionsFromSchemas(collector, request.Parameter, catalog.SubscriptionSchema)
		for _, event := range catalog.Events {
			if request.Action != "" && request.Action != stringFromAny(event["name"]) {
				continue
			}
			appendOptionsFromSchemas(collector, request.Parameter, objectListFromAny(event["parameters"]))
		}
		appendTriggerRuntimeParameterValues(collector, s.store, workspaceID, provider, request.Parameter)
	}
	return collector.items()
}

func (s *server) toolPluginDynamicOptions(workspaceID string, request pluginDynamicOptionsRequest) []map[string]any {
	collector := newDynamicOptionCollector()
	parameter := normalizeDynamicOptionParameter(request.Parameter)
	switch parameter {
	case "provider", "provider_id", "tool_provider", "tool_provider_id":
		appendToolProviderOptions(collector, s, workspaceID)
		return collector.items()
	case "tool", "tool_name", "action":
		appendToolActionOptions(collector, s, workspaceID, request.Provider)
		return collector.items()
	case "timezone", "time_zone":
		if request.Provider == "current_time" || request.Action == "get_current_time" {
			appendCommonTimezoneOptions(collector)
		}
	}

	if catalog, ok := builtinToolCatalogByProvider(request.Provider); ok {
		appendOptionsFromWorkspaceTools(collector, request.Action, request.Parameter, catalog.Tools)
	}
	if provider, ok := s.store.GetAPIToolProviderByName(workspaceID, request.Provider); ok {
		if parsed, err := parseAPISchemaDocument(provider.Schema); err == nil {
			for _, operation := range parsed.Operations {
				if request.Action != "" && operation.Name != request.Action {
					continue
				}
				parameters := make([]state.WorkspaceToolParameter, 0, len(operation.Parameters))
				for _, item := range operation.Parameters {
					parameters = append(parameters, item.Parameter)
				}
				appendOptionsFromWorkspaceTools(collector, operation.Name, request.Parameter, []state.WorkspaceTool{{Name: operation.Name, Parameters: parameters}})
			}
		}
	}
	for _, provider := range s.store.ListMCPToolProviders(workspaceID) {
		if provider.ID != request.Provider && provider.Name != request.Provider && provider.ServerIdentifier != request.Provider {
			continue
		}
		appendOptionsFromWorkspaceTools(collector, request.Action, request.Parameter, provider.Tools)
		appendOptionsFromWorkspaceTools(collector, request.Action, request.Parameter, generatedMCPTools(provider, "dify-go"))
	}
	return collector.items()
}

func (s *server) datasourcePluginDynamicOptions(workspace state.Workspace, request pluginDynamicOptionsRequest) []map[string]any {
	spec, ok := s.ragPipelineDatasourceAvailableSpecByProvider(workspace.ID, request.PluginID, request.Provider)
	if !ok {
		return []map[string]any{}
	}

	collector := newDynamicOptionCollector()
	parameter := normalizeDynamicOptionParameter(request.Parameter)
	providerState, _ := s.store.GetWorkspaceDatasourceProviderState(workspace.ID, spec.PluginID, spec.Provider)
	switch parameter {
	case "provider", "provider_id", "datasource_provider":
		for _, item := range s.ragPipelineDatasourceAvailableSpecs(workspace.ID) {
			collector.add(item.Provider, item.Label, map[string]any{"icon": item.Icon})
		}
	case "credential", "credential_id", "credentials":
		appendDatasourceCredentialOptions(collector, providerState)
	case "workspace", "workspace_id":
		credential, found := datasourceCredentialForDynamicOptions(providerState, request.CredentialID)
		if found && spec.ProviderType == "online_document" {
			for _, item := range s.datasourceNotionWorkspaces(datasourceNodeRunContext{Workspace: workspace, Spec: spec, Credential: credential, Inputs: request.Extra}) {
				collector.add(stringFromAny(item["workspace_id"]), stringFromAny(item["workspace_name"]), nil)
			}
		}
	case "page", "page_id":
		credential, found := datasourceCredentialForDynamicOptions(providerState, request.CredentialID)
		if found && spec.ProviderType == "online_document" {
			for _, workspaceItem := range s.datasourceNotionWorkspaces(datasourceNodeRunContext{Workspace: workspace, Spec: spec, Credential: credential, Inputs: request.Extra}) {
				for _, page := range objectListFromAny(workspaceItem["pages"]) {
					collector.add(stringFromAny(page["page_id"]), stringFromAny(page["page_name"]), nil)
				}
			}
		}
	case "source_url", "url", "page_url":
		credential, found := datasourceCredentialForDynamicOptions(providerState, request.CredentialID)
		if found && spec.ProviderType == "website_crawl" {
			for _, item := range s.datasourceWebsiteResults(datasourceNodeRunContext{Workspace: workspace, Spec: spec, Credential: credential, Inputs: request.Extra}) {
				collector.add(stringFromAny(item["source_url"]), stringFromAny(item["title"]), nil)
			}
		}
	case "bucket", "bucket_id":
		credential, found := datasourceCredentialForDynamicOptions(providerState, request.CredentialID)
		if found && spec.ProviderType == "online_drive" {
			for _, item := range s.datasourceOnlineDriveData(datasourceNodeRunContext{Workspace: workspace, Spec: spec, Credential: credential, Inputs: request.Extra}) {
				collector.add(stringFromAny(item["bucket"]), stringFromAny(item["bucket"]), nil)
			}
		}
	case "file", "file_id", "item", "item_id":
		credential, found := datasourceCredentialForDynamicOptions(providerState, request.CredentialID)
		if found && spec.ProviderType == "online_drive" {
			for _, bucket := range s.datasourceOnlineDriveData(datasourceNodeRunContext{Workspace: workspace, Spec: spec, Credential: credential, Inputs: request.Extra}) {
				for _, file := range objectListFromAny(bucket["files"]) {
					collector.add(stringFromAny(file["id"]), stringFromAny(file["name"]), nil)
				}
			}
		}
	}
	return collector.items()
}

type dynamicOptionCollector struct {
	seen    map[string]struct{}
	options []map[string]any
}

func newDynamicOptionCollector() *dynamicOptionCollector {
	return &dynamicOptionCollector{
		seen:    map[string]struct{}{},
		options: []map[string]any{},
	}
}

func (c *dynamicOptionCollector) add(value, label string, extra map[string]any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := c.seen[value]; ok {
		return
	}
	c.seen[value] = struct{}{}
	if strings.TrimSpace(label) == "" {
		label = value
	}
	item := map[string]any{
		"label": localizedText(label),
		"value": value,
	}
	for key, val := range extra {
		if key == "label" || key == "value" || val == nil {
			continue
		}
		item[key] = val
	}
	c.options = append(c.options, item)
}

func (c *dynamicOptionCollector) addRaw(value string, label any, extra map[string]any) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := c.seen[value]; ok {
		return
	}
	c.seen[value] = struct{}{}
	item := map[string]any{
		"label": dynamicOptionLabel(label, value),
		"value": value,
	}
	for key, val := range extra {
		if key == "label" || key == "value" || val == nil {
			continue
		}
		item[key] = val
	}
	c.options = append(c.options, item)
}

func (c *dynamicOptionCollector) itemsList() []map[string]any {
	return c.options
}

func (c *dynamicOptionCollector) items() []map[string]any {
	return c.itemsList()
}

func dynamicOptionLabel(label any, fallback string) any {
	switch typed := label.(type) {
	case map[string]any:
		if len(typed) > 0 {
			return typed
		}
	case map[string]string:
		if len(typed) > 0 {
			return typed
		}
	case string:
		if strings.TrimSpace(typed) != "" {
			return localizedText(typed)
		}
	}
	return localizedText(fallback)
}

func normalizeDynamicOptionParameter(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func appendOptionsFromSchemas(collector *dynamicOptionCollector, parameter string, schemas []map[string]any) {
	parameter = strings.TrimSpace(parameter)
	if parameter == "" {
		return
	}
	for _, schema := range schemas {
		name := firstNonEmpty(stringFromAny(schema["name"]), stringFromAny(schema["variable"]))
		if name != parameter {
			continue
		}
		appendOptionsFromAny(collector, schema["options"])
	}
}

func appendOptionsFromAny(collector *dynamicOptionCollector, raw any) {
	items := anySlice(raw)
	switch typed := raw.(type) {
	case []map[string]any:
		items = make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
	case []string:
		items = make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
	}
	for _, item := range items {
		option := mapFromAny(item)
		if len(option) == 0 {
			value := strings.TrimSpace(anyToString(item))
			collector.add(value, value, nil)
			continue
		}
		value := firstNonEmpty(
			stringValue(option["value"], ""),
			stringValue(option["id"], ""),
			stringValue(option["name"], ""),
		)
		label := option["label"]
		if label == nil {
			label = firstNonEmpty(stringValue(option["text"], ""), stringValue(option["name"], ""), value)
		}
		extra := cloneJSONObject(option)
		delete(extra, "label")
		delete(extra, "value")
		delete(extra, "id")
		delete(extra, "name")
		delete(extra, "text")
		collector.addRaw(value, label, extra)
	}
}

func appendTriggerEventOptions(collector *dynamicOptionCollector, catalog triggerProviderCatalog) {
	for _, event := range catalog.Events {
		identity := mapFromAny(event["identity"])
		name := firstNonEmpty(stringFromAny(event["name"]), stringFromAny(identity["name"]))
		label := identity["label"]
		if label == nil {
			label = firstNonEmpty(stringFromAny(event["label"]), name)
		}
		collector.addRaw(name, label, nil)
	}
}

func appendTriggerRuntimeParameterValues(collector *dynamicOptionCollector, store *state.Store, workspaceID, provider, parameter string) {
	providerState, ok := store.GetWorkspaceTriggerProviderState(workspaceID, provider)
	if !ok {
		return
	}
	for _, item := range providerState.SubscriptionBuilders {
		appendRuntimeParameterValue(collector, item.Parameters[parameter])
	}
	for _, item := range providerState.Subscriptions {
		appendRuntimeParameterValue(collector, item.Parameters[parameter])
	}
}

func appendRuntimeParameterValue(collector *dynamicOptionCollector, value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			appendRuntimeParameterValue(collector, item)
		}
	case []string:
		for _, item := range typed {
			collector.add(item, item, nil)
		}
	default:
		text := strings.TrimSpace(anyToString(value))
		collector.add(text, text, nil)
	}
}

func appendToolProviderOptions(collector *dynamicOptionCollector, s *server, workspaceID string) {
	for _, catalog := range builtinToolCatalogs() {
		collector.add(catalog.Provider, catalog.Label, map[string]any{"icon": catalog.Icon})
	}
	for _, provider := range s.store.ListAPIToolProviders(workspaceID) {
		collector.add(provider.Provider, provider.Provider, nil)
	}
	for _, provider := range s.store.ListWorkflowToolProviders(workspaceID) {
		collector.add(provider.Name, firstNonEmpty(provider.Label, provider.Name), nil)
	}
	for _, provider := range s.store.ListMCPToolProviders(workspaceID) {
		collector.add(provider.ID, firstNonEmpty(provider.Name, provider.ServerIdentifier, provider.ID), nil)
	}
}

func appendToolActionOptions(collector *dynamicOptionCollector, s *server, workspaceID, providerName string) {
	if catalog, ok := builtinToolCatalogByProvider(providerName); ok {
		appendWorkspaceToolNameOptions(collector, catalog.Tools)
	}
	if provider, ok := s.store.GetAPIToolProviderByName(workspaceID, providerName); ok {
		if parsed, err := parseAPISchemaDocument(provider.Schema); err == nil {
			for _, operation := range parsed.Operations {
				collector.add(operation.Name, firstNonEmpty(operation.Summary, operation.Name), nil)
			}
		}
	}
	for _, provider := range s.store.ListWorkflowToolProviders(workspaceID) {
		if provider.Name == providerName || provider.ID == providerName {
			collector.add(provider.Name, firstNonEmpty(provider.Label, provider.Name), nil)
		}
	}
	for _, provider := range s.store.ListMCPToolProviders(workspaceID) {
		if provider.ID != providerName && provider.Name != providerName && provider.ServerIdentifier != providerName {
			continue
		}
		appendWorkspaceToolNameOptions(collector, provider.Tools)
		appendWorkspaceToolNameOptions(collector, generatedMCPTools(provider, "dify-go"))
	}
}

func appendWorkspaceToolNameOptions(collector *dynamicOptionCollector, tools []state.WorkspaceTool) {
	for _, tool := range tools {
		collector.add(tool.Name, firstNonEmpty(tool.Label, tool.Name), nil)
	}
}

func appendOptionsFromWorkspaceTools(collector *dynamicOptionCollector, action, parameter string, tools []state.WorkspaceTool) {
	parameter = strings.TrimSpace(parameter)
	if parameter == "" {
		return
	}
	for _, tool := range tools {
		if action != "" && tool.Name != action {
			continue
		}
		for _, toolParameter := range tool.Parameters {
			if toolParameter.Name != parameter {
				continue
			}
			for _, option := range toolParameter.Options {
				collector.add(option.Value, firstNonEmpty(option.Label, option.Value), nil)
			}
		}
	}
}

func appendCommonTimezoneOptions(collector *dynamicOptionCollector) {
	for _, item := range []struct {
		value string
		label string
	}{
		{"UTC", "UTC"},
		{"Asia/Shanghai", "Asia/Shanghai"},
		{"Asia/Tokyo", "Asia/Tokyo"},
		{"Europe/London", "Europe/London"},
		{"Europe/Berlin", "Europe/Berlin"},
		{"America/New_York", "America/New_York"},
		{"America/Los_Angeles", "America/Los_Angeles"},
	} {
		collector.add(item.value, item.label, nil)
	}
}

func appendDatasourceCredentialOptions(collector *dynamicOptionCollector, providerState state.WorkspaceDatasourceProviderState) {
	for _, item := range providerState.Credentials {
		extra := map[string]any{}
		if item.ID == providerState.DefaultCredentialID {
			extra["is_default"] = true
		}
		if item.AvatarURL != "" {
			extra["icon"] = item.AvatarURL
		}
		collector.add(item.ID, firstNonEmpty(item.Name, item.ID), extra)
	}
}

func datasourceCredentialForDynamicOptions(providerState state.WorkspaceDatasourceProviderState, credentialID string) (state.WorkspaceDatasourceCredential, bool) {
	credentialID = firstNonEmpty(strings.TrimSpace(credentialID), providerState.DefaultCredentialID)
	for _, item := range providerState.Credentials {
		if item.ID == credentialID {
			return item, true
		}
	}
	if len(providerState.Credentials) > 0 && credentialID == "" {
		return providerState.Credentials[0], true
	}
	return state.WorkspaceDatasourceCredential{}, false
}

func triggerProviderFromPluginID(pluginID string) string {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return ""
	}
	switch pluginID {
	case "langgenius/github":
		return "langgenius/github/github"
	case "langgenius/http":
		return "langgenius/http/http"
	default:
		return pluginID
	}
}
