package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/langgenius/dify-go/internal/state"
)

type builtinToolCatalog struct {
	Provider         string
	Author           string
	Label            string
	Description      string
	Icon             string
	Labels           []string
	Tools            []state.WorkspaceTool
	CredentialSchema []map[string]any
}

type parsedAPISchema struct {
	SchemaType string
	Operations []parsedAPIOperation
}

type parsedAPIOperation struct {
	Name       string
	Summary    string
	Method     string
	Path       string
	ServerURL  string
	Parameters []apiParameterSpec
}

type apiParameterSpec struct {
	In        string
	Parameter state.WorkspaceToolParameter
}

func (s *server) mountWorkspaceToolRoutes(r chi.Router) {
	r.Get("/workspaces/current/tool-providers", s.handleToolProviderList)
	r.Get("/workspaces/current/agent-providers", s.handleAgentProviderList)
	r.Get("/workspaces/current/agent-provider/*", s.handleAgentProviderDetail)
	r.Get("/workspaces/current/tools/builtin", s.handleAllBuiltinTools)
	r.Get("/workspaces/current/tools/api", s.handleAllAPITools)
	r.Get("/workspaces/current/tools/workflow", s.handleAllWorkflowTools)
	r.Get("/workspaces/current/tools/mcp", s.handleAllMCPTools)

	r.Get("/workspaces/current/tool-provider/builtin/{provider}/tools", s.handleBuiltinToolList)
	r.Get("/workspaces/current/tool-provider/builtin/{provider}/credentials_schema", s.handleBuiltinToolCredentialSchema)
	r.Get("/workspaces/current/tool-provider/builtin/{provider}/credentials", s.handleBuiltinToolCredentialGet)
	r.Post("/workspaces/current/tool-provider/builtin/{provider}/update", s.handleBuiltinToolCredentialSave)
	r.Post("/workspaces/current/tool-provider/builtin/{provider}/delete", s.handleBuiltinToolCredentialDelete)

	r.Post("/workspaces/current/tool-provider/api/add", s.handleAPIToolCreate)
	r.Get("/workspaces/current/tool-provider/api/remote", s.handleAPIToolRemoteSchema)
	r.Get("/workspaces/current/tool-provider/api/tools", s.handleAPIToolList)
	r.Post("/workspaces/current/tool-provider/api/update", s.handleAPIToolUpdate)
	r.Post("/workspaces/current/tool-provider/api/delete", s.handleAPIToolDelete)
	r.Get("/workspaces/current/tool-provider/api/get", s.handleAPIToolGet)
	r.Post("/workspaces/current/tool-provider/api/schema", s.handleAPIToolSchemaParse)
	r.Post("/workspaces/current/tool-provider/api/test/pre", s.handleAPIToolTest)

	r.Post("/workspaces/current/tool-provider/workflow/create", s.handleWorkflowToolCreate)
	r.Post("/workspaces/current/tool-provider/workflow/update", s.handleWorkflowToolUpdate)
	r.Post("/workspaces/current/tool-provider/workflow/delete", s.handleWorkflowToolDelete)
	r.Get("/workspaces/current/tool-provider/workflow/get", s.handleWorkflowToolGet)
	r.Get("/workspaces/current/tool-provider/workflow/tools", s.handleWorkflowToolList)

	r.Post("/workspaces/current/tool-provider/mcp", s.handleMCPCreate)
	r.Put("/workspaces/current/tool-provider/mcp", s.handleMCPUpdate)
	r.Delete("/workspaces/current/tool-provider/mcp", s.handleMCPDelete)
	r.Post("/workspaces/current/tool-provider/mcp/auth", s.handleMCPAuth)
	r.Get("/workspaces/current/tool-provider/mcp/tools/{providerID}", s.handleMCPDetail)
	r.Get("/workspaces/current/tool-provider/mcp/update/{providerID}", s.handleMCPUpdateTools)
}

func (s *server) handleAgentProviderList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{s.defaultAgentProviderPayload()})
}

func (s *server) handleAgentProviderDetail(w http.ResponseWriter, r *http.Request) {
	providerName := strings.Trim(strings.TrimPrefix(chi.URLParam(r, "*"), "/"), " ")
	provider := s.defaultAgentProviderPayload()
	if providerName != "" && providerName != stringValue(provider["provider"], "") {
		writeError(w, http.StatusNotFound, "provider_not_found", "Agent provider not found.")
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (s *server) handleToolProviderList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	filter := strings.TrimSpace(r.URL.Query().Get("type"))
	items := make([]map[string]any, 0)
	if filter == "" || filter == "builtin" {
		for _, catalog := range builtinToolCatalogs() {
			items = append(items, s.builtinProviderResponse(workspace.ID, catalog))
		}
	}
	if filter == "" || filter == "api" {
		items = append(items, s.apiProviderListPayload(workspace.ID)...)
	}
	if filter == "" || filter == "workflow" {
		items = append(items, s.workflowProviderListPayload(workspace.ID)...)
	}
	if filter == "" || filter == "mcp" {
		items = append(items, s.mcpProviderListPayload(workspace.ID)...)
	}

	writeJSON(w, http.StatusOK, items)
}

func (s *server) handleAllBuiltinTools(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	items := make([]map[string]any, 0, len(builtinToolCatalogs()))
	for _, catalog := range builtinToolCatalogs() {
		items = append(items, s.builtinProviderResponse(workspace.ID, catalog))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) handleAllAPITools(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.apiProviderListPayload(workspace.ID))
}

func (s *server) handleAllWorkflowTools(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.workflowProviderListPayload(workspace.ID))
}

func (s *server) handleAllMCPTools(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.mcpProviderListPayload(workspace.ID))
}

func (s *server) handleBuiltinToolList(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	catalog, ok := builtinToolCatalogByProvider(provider)
	if !ok {
		writeError(w, http.StatusNotFound, "provider_not_found", "Built-in tool provider not found.")
		return
	}

	tools := make([]map[string]any, 0, len(catalog.Tools))
	for _, tool := range catalog.Tools {
		tools = append(tools, s.toolPayload(tool))
	}
	writeJSON(w, http.StatusOK, tools)
}

func (s *server) handleBuiltinToolCredentialSchema(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(chi.URLParam(r, "provider"))
	catalog, ok := builtinToolCatalogByProvider(provider)
	if !ok {
		writeError(w, http.StatusNotFound, "provider_not_found", "Built-in tool provider not found.")
		return
	}
	writeJSON(w, http.StatusOK, catalog.CredentialSchema)
}

func (s *server) handleBuiltinToolCredentialGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	credential, found := s.store.GetBuiltinToolCredential(workspace.ID, chi.URLParam(r, "provider"))
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	writeJSON(w, http.StatusOK, credential.Credentials)
}

func (s *server) handleBuiltinToolCredentialSave(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpsertBuiltinToolCredential(workspace.ID, chi.URLParam(r, "provider"), payload.Credentials, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleBuiltinToolCredentialDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if err := s.store.DeleteBuiltinToolCredential(workspace.ID, chi.URLParam(r, "provider")); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAPIToolCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Provider         string         `json:"provider"`
		Credentials      map[string]any `json:"credentials"`
		SchemaType       string         `json:"schema_type"`
		Schema           string         `json:"schema"`
		Icon             map[string]any `json:"icon"`
		PrivacyPolicy    string         `json:"privacy_policy"`
		CustomDisclaimer string         `json:"custom_disclaimer"`
		Labels           []string       `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := parseAPISchemaDocument(payload.Schema); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	created, err := s.store.CreateAPIToolProvider(workspace.ID, currentUser(r), state.CreateAPIToolProviderInput{
		Provider:         payload.Provider,
		Icon:             emojiFromPayload(payload.Icon),
		Credentials:      payload.Credentials,
		SchemaType:       payload.SchemaType,
		Schema:           payload.Schema,
		PrivacyPolicy:    payload.PrivacyPolicy,
		CustomDisclaimer: payload.CustomDisclaimer,
		Labels:           payload.Labels,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                created.ID,
		"provider":          created.Provider,
		"credentials":       created.Credentials,
		"icon":              emojiPayload(created.Icon),
		"schema_type":       created.SchemaType,
		"schema":            created.Schema,
		"privacy_policy":    created.PrivacyPolicy,
		"custom_disclaimer": created.CustomDisclaimer,
		"labels":            created.Labels,
	})
}

func (s *server) handleAPIToolRemoteSchema(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "url is required.")
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid schema url.")
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "remote_fetch_failed", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadGateway, "remote_fetch_failed", "Failed to read remote schema.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schema": string(body)})
}

func (s *server) handleAPIToolList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerName := strings.TrimSpace(r.URL.Query().Get("provider"))
	provider, found := s.store.GetAPIToolProviderByName(workspace.ID, providerName)
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "API tool provider not found.")
		return
	}

	parsed, err := parseAPISchemaDocument(provider.Schema)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	tools := make([]map[string]any, 0, len(parsed.Operations))
	for _, operation := range parsed.Operations {
		tools = append(tools, s.apiOperationToolPayload(provider, operation))
	}
	writeJSON(w, http.StatusOK, tools)
}

func (s *server) handleAPIToolUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Provider         string         `json:"provider"`
		OriginalProvider string         `json:"original_provider"`
		Credentials      map[string]any `json:"credentials"`
		SchemaType       string         `json:"schema_type"`
		Schema           string         `json:"schema"`
		Icon             map[string]any `json:"icon"`
		PrivacyPolicy    string         `json:"privacy_policy"`
		CustomDisclaimer string         `json:"custom_disclaimer"`
		Labels           []string       `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := parseAPISchemaDocument(payload.Schema); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updated, err := s.store.UpdateAPIToolProvider(workspace.ID, currentUser(r), state.UpdateAPIToolProviderInput{
		Provider:         payload.Provider,
		OriginalProvider: payload.OriginalProvider,
		Icon:             emojiFromPayload(payload.Icon),
		Credentials:      payload.Credentials,
		SchemaType:       payload.SchemaType,
		Schema:           payload.Schema,
		PrivacyPolicy:    payload.PrivacyPolicy,
		CustomDisclaimer: payload.CustomDisclaimer,
		Labels:           payload.Labels,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                updated.ID,
		"provider":          updated.Provider,
		"credentials":       updated.Credentials,
		"icon":              emojiPayload(updated.Icon),
		"schema_type":       updated.SchemaType,
		"schema":            updated.Schema,
		"privacy_policy":    updated.PrivacyPolicy,
		"custom_disclaimer": updated.CustomDisclaimer,
		"labels":            updated.Labels,
	})
}

func (s *server) handleAPIToolDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.DeleteAPIToolProvider(workspace.ID, payload.Provider); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAPIToolGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerName := strings.TrimSpace(r.URL.Query().Get("provider"))
	provider, found := s.store.GetAPIToolProviderByName(workspace.ID, providerName)
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "API tool provider not found.")
		return
	}

	parsed, err := parseAPISchemaDocument(provider.Schema)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                provider.ID,
		"provider":          provider.Provider,
		"credentials":       provider.Credentials,
		"icon":              emojiPayload(provider.Icon),
		"schema_type":       provider.SchemaType,
		"schema":            provider.Schema,
		"privacy_policy":    provider.PrivacyPolicy,
		"custom_disclaimer": provider.CustomDisclaimer,
		"tools":             s.apiSchemaPreview(parsed.Operations),
		"labels":            provider.Labels,
	})
}

func (s *server) handleAPIToolSchemaParse(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Schema string `json:"schema"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	parsed, err := parseAPISchemaDocument(payload.Schema)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"parameters_schema": s.apiSchemaPreview(parsed.Operations),
		"schema_type":       parsed.SchemaType,
	})
}

func (s *server) handleAPIToolTest(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ToolName     string         `json:"tool_name"`
		ProviderName string         `json:"provider_name"`
		Credentials  map[string]any `json:"credentials"`
		Parameters   map[string]any `json:"parameters"`
		SchemaType   string         `json:"schema_type"`
		Schema       string         `json:"schema"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	result := previewAPIToolCall(r.Context(), payload.Schema, payload.ToolName, payload.Credentials, payload.Parameters)
	writeJSON(w, http.StatusOK, result)
}

func (s *server) handleWorkflowToolCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		WorkflowAppID string         `json:"workflow_app_id"`
		Name          string         `json:"name"`
		Label         string         `json:"label"`
		Description   string         `json:"description"`
		Icon          map[string]any `json:"icon"`
		Parameters    []struct {
			Name        string `json:"name"`
			Form        string `json:"form"`
			Description string `json:"description"`
			Required    bool   `json:"required"`
			Type        string `json:"type"`
		} `json:"parameters"`
		PrivacyPolicy string   `json:"privacy_policy"`
		Labels        []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	app, ok := s.store.GetApp(strings.TrimSpace(payload.WorkflowAppID), workspace.ID)
	if !ok || app.Workflow == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow app not found.")
		return
	}

	_, err := s.store.CreateWorkflowToolProvider(workspace.ID, currentUser(r), state.CreateWorkflowToolProviderInput{
		AppID:         payload.WorkflowAppID,
		Name:          normalizeIdentifier(payload.Name),
		Label:         payload.Label,
		Icon:          emojiFromPayload(payload.Icon),
		Description:   payload.Description,
		Parameters:    workflowToolParametersFromPayload(payload.Parameters),
		PrivacyPolicy: payload.PrivacyPolicy,
		Labels:        payload.Labels,
		Version:       workflowToolVersion(app),
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowToolUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		WorkflowToolID string         `json:"workflow_tool_id"`
		Name           string         `json:"name"`
		Label          string         `json:"label"`
		Description    string         `json:"description"`
		Icon           map[string]any `json:"icon"`
		Parameters     []struct {
			Name        string `json:"name"`
			Form        string `json:"form"`
			Description string `json:"description"`
			Required    bool   `json:"required"`
			Type        string `json:"type"`
		} `json:"parameters"`
		PrivacyPolicy string   `json:"privacy_policy"`
		Labels        []string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider, found := s.store.GetWorkflowToolProviderByID(workspace.ID, payload.WorkflowToolID)
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "Workflow tool provider not found.")
		return
	}
	app, ok := s.store.GetApp(provider.AppID, workspace.ID)
	if !ok || app.Workflow == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow app not found.")
		return
	}

	_, err := s.store.UpdateWorkflowToolProvider(workspace.ID, currentUser(r), state.UpdateWorkflowToolProviderInput{
		ID:            payload.WorkflowToolID,
		Name:          normalizeIdentifier(payload.Name),
		Label:         payload.Label,
		Icon:          emojiFromPayload(payload.Icon),
		Description:   payload.Description,
		Parameters:    workflowToolParametersFromPayload(payload.Parameters),
		PrivacyPolicy: payload.PrivacyPolicy,
		Labels:        payload.Labels,
		Version:       workflowToolVersion(app),
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowToolDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		WorkflowToolID string `json:"workflow_tool_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.DeleteWorkflowToolProvider(workspace.ID, payload.WorkflowToolID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowToolGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	workflowToolID := strings.TrimSpace(r.URL.Query().Get("workflow_tool_id"))
	workflowAppID := strings.TrimSpace(r.URL.Query().Get("workflow_app_id"))

	if workflowToolID == "" && workflowAppID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "workflow_tool_id or workflow_app_id is required.")
		return
	}

	var provider state.WorkspaceWorkflowToolProvider
	var found bool
	if workflowToolID != "" {
		provider, found = s.store.GetWorkflowToolProviderByID(workspace.ID, workflowToolID)
		if !found {
			writeError(w, http.StatusNotFound, "provider_not_found", "Workflow tool provider not found.")
			return
		}
	} else {
		provider, found = s.store.GetWorkflowToolProviderByAppID(workspace.ID, workflowAppID)
		if !found {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
	}

	app, ok := s.store.GetApp(provider.AppID, workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.workflowToolDetailPayload(provider, app))
}

func (s *server) handleWorkflowToolList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	provider, found := s.store.GetWorkflowToolProviderByID(workspace.ID, strings.TrimSpace(r.URL.Query().Get("workflow_tool_id")))
	if !found {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	app, ok := s.store.GetApp(provider.AppID, workspace.ID)
	if !ok {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, []map[string]any{s.workflowToolPayload(provider, app)})
}

func (s *server) handleMCPCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	payload, err := decodeMCPPayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	provider, err := s.store.CreateMCPToolProvider(workspace.ID, currentUser(r), payload.createInput(), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.mcpProviderPayload(provider))
}

func (s *server) handleMCPUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	payload, err := decodeMCPPayload(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if payload.ProviderID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "provider_id is required.")
		return
	}

	currentProvider, found := s.store.GetMCPToolProvider(workspace.ID, payload.ProviderID)
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "MCP provider not found.")
		return
	}
	if payload.ServerURL == "" || payload.ServerURL == "[__HIDDEN__]" {
		payload.ServerURL = currentProvider.ServerURL
	}

	if _, err := s.store.UpdateMCPToolProvider(workspace.ID, currentUser(r), payload.updateInput(), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleMCPDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.DeleteMCPToolProvider(workspace.ID, payload.ProviderID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleMCPAuth(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		ProviderID string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateMCPToolProviderAuthorization(workspace.ID, payload.ProviderID, true, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleMCPDetail(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	provider, found := s.store.GetMCPToolProvider(workspace.ID, chi.URLParam(r, "providerID"))
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "MCP provider not found.")
		return
	}

	tools := make([]map[string]any, 0, len(provider.Tools))
	for _, tool := range provider.Tools {
		tools = append(tools, s.toolPayload(tool))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tools": tools})
}

func (s *server) handleMCPUpdateTools(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	provider, found := s.store.GetMCPToolProvider(workspace.ID, chi.URLParam(r, "providerID"))
	if !found {
		writeError(w, http.StatusNotFound, "provider_not_found", "MCP provider not found.")
		return
	}

	tools := generatedMCPTools(provider, s.userDisplayName(provider.CreatedBy))
	updated, err := s.store.UpdateMCPToolProviderTools(workspace.ID, provider.ID, tools, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	payload := make([]map[string]any, 0, len(updated.Tools))
	for _, tool := range updated.Tools {
		payload = append(payload, s.toolPayload(tool))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tools": payload})
}

func (s *server) apiProviderListPayload(workspaceID string) []map[string]any {
	providers := s.store.ListAPIToolProviders(workspaceID)
	items := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		parsed, err := parseAPISchemaDocument(provider.Schema)
		if err != nil {
			continue
		}
		tools := make([]map[string]any, 0, len(parsed.Operations))
		for _, operation := range parsed.Operations {
			tools = append(tools, s.apiOperationToolPayload(provider, operation))
		}
		items = append(items, map[string]any{
			"id":                    provider.ID,
			"name":                  provider.Provider,
			"author":                s.userDisplayName(provider.CreatedBy),
			"description":           localizedText(firstNonEmpty(provider.CustomDisclaimer, provider.PrivacyPolicy, provider.Provider+" API tools")),
			"icon":                  emojiPayload(provider.Icon),
			"label":                 localizedText(provider.Provider),
			"type":                  "api",
			"team_credentials":      map[string]any{},
			"is_team_authorization": true,
			"allow_delete":          true,
			"labels":                provider.Labels,
			"tools":                 tools,
			"meta":                  map[string]any{"version": s.cfg.AppVersion},
		})
	}
	return items
}

func (s *server) workflowProviderListPayload(workspaceID string) []map[string]any {
	providers := s.store.ListWorkflowToolProviders(workspaceID)
	items := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		app, ok := s.store.GetApp(provider.AppID, workspaceID)
		if !ok {
			continue
		}
		items = append(items, map[string]any{
			"id":                    provider.ID,
			"name":                  provider.Name,
			"author":                app.AuthorName,
			"description":           localizedText(firstNonEmpty(provider.Description, provider.Label)),
			"icon":                  emojiPayload(provider.Icon),
			"label":                 localizedText(firstNonEmpty(provider.Label, provider.Name)),
			"type":                  "workflow",
			"team_credentials":      map[string]any{},
			"is_team_authorization": true,
			"allow_delete":          true,
			"labels":                provider.Labels,
			"workflow_app_id":       provider.AppID,
			"tools":                 []map[string]any{s.workflowToolPayload(provider, app)},
			"meta":                  map[string]any{"version": s.cfg.AppVersion},
		})
	}
	return items
}

func (s *server) mcpProviderListPayload(workspaceID string) []map[string]any {
	providers := s.store.ListMCPToolProviders(workspaceID)
	items := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		items = append(items, s.mcpProviderPayload(provider))
	}
	return items
}

func (s *server) defaultAgentProviderPayload() map[string]any {
	providerName := "langgenius/default"
	icon := providerIconDataURI("A", "#EDE9FE", "#5B21B6")
	return map[string]any{
		"provider":                 providerName,
		"plugin_unique_identifier": providerName + "@" + s.cfg.AppVersion,
		"plugin_id":                providerName,
		"declaration": map[string]any{
			"identity": map[string]any{
				"author":      "langgenius",
				"name":        providerName,
				"label":       localizedText("Built-in Agent Strategies"),
				"description": localizedText("Default agent strategies bundled with dify-go."),
				"icon":        icon,
				"icon_dark":   "",
				"tags":        []any{},
			},
			"plugin_id": providerName,
			"strategies": []map[string]any{
				{
					"identity": map[string]any{
						"author":   "langgenius",
						"name":     "react",
						"icon":     providerIconDataURI("R", "#DCFCE7", "#166534"),
						"label":    localizedText("ReAct"),
						"provider": providerName,
					},
					"parameters":    []any{},
					"description":   localizedText("Reason and act with iterative tool usage."),
					"output_schema": map[string]any{},
					"features":      []any{},
				},
				{
					"identity": map[string]any{
						"author":   "langgenius",
						"name":     "function_call",
						"icon":     providerIconDataURI("F", "#DBEAFE", "#1D4ED8"),
						"label":    localizedText("Function Call"),
						"provider": providerName,
					},
					"parameters":    []any{},
					"description":   localizedText("Plan tool calls in a function-calling style loop."),
					"output_schema": map[string]any{},
					"features":      []any{},
				},
			},
		},
		"meta": map[string]any{
			"version": s.cfg.AppVersion,
		},
	}
}

func (s *server) builtinProviderResponse(workspaceID string, catalog builtinToolCatalog) map[string]any {
	credential, hasCredential := s.store.GetBuiltinToolCredential(workspaceID, catalog.Provider)
	needsCredential := len(catalog.CredentialSchema) > 0
	tools := make([]map[string]any, 0, len(catalog.Tools))
	for _, tool := range catalog.Tools {
		tools = append(tools, s.toolPayload(tool))
	}

	return map[string]any{
		"id":                    catalog.Provider,
		"name":                  catalog.Provider,
		"author":                catalog.Author,
		"description":           localizedText(catalog.Description),
		"icon":                  catalog.Icon,
		"label":                 localizedText(catalog.Label),
		"type":                  "builtin",
		"team_credentials":      map[string]any{},
		"is_team_authorization": !needsCredential || hasCredential,
		"allow_delete":          needsCredential,
		"labels":                catalog.Labels,
		"tools":                 tools,
		"meta":                  map[string]any{"version": s.cfg.AppVersion},
		"credential_id":         credential.CredentialID,
	}
}

func (s *server) apiOperationToolPayload(provider state.WorkspaceAPIToolProvider, operation parsedAPIOperation) map[string]any {
	tool := state.WorkspaceTool{
		Name:         operation.Name,
		Author:       s.userDisplayName(provider.CreatedBy),
		Label:        operation.Name,
		Description:  firstNonEmpty(operation.Summary, operation.Name),
		Parameters:   toolParametersFromAPIOperation(operation),
		Labels:       provider.Labels,
		OutputSchema: map[string]any{},
	}
	return s.toolPayload(tool)
}

func (s *server) workflowToolPayload(provider state.WorkspaceWorkflowToolProvider, app state.App) map[string]any {
	parameters := make([]state.WorkspaceToolParameter, 0, len(provider.Parameters))
	for _, item := range provider.Parameters {
		parameters = append(parameters, state.WorkspaceToolParameter{
			Name:             item.Name,
			Label:            item.Name,
			HumanDescription: item.Description,
			Type:             normalizeParameterType(item.Type),
			Form:             firstNonEmpty(item.Form, "llm"),
			LLMDescription:   item.Description,
			Required:         item.Required,
			Multiple:         false,
			Default:          "",
			Options:          []state.WorkspaceToolOption{},
		})
	}
	return s.toolPayload(state.WorkspaceTool{
		Name:         provider.Name,
		Author:       app.AuthorName,
		Label:        firstNonEmpty(provider.Label, provider.Name),
		Description:  firstNonEmpty(provider.Description, provider.Label),
		Parameters:   parameters,
		Labels:       provider.Labels,
		OutputSchema: workflowToolOutputSchema(app),
	})
}

func (s *server) workflowToolDetailPayload(provider state.WorkspaceWorkflowToolProvider, app state.App) map[string]any {
	tool := s.workflowToolPayload(provider, app)
	return map[string]any{
		"workflow_app_id":  provider.AppID,
		"workflow_tool_id": provider.ID,
		"label":            firstNonEmpty(provider.Label, provider.Name),
		"name":             provider.Name,
		"icon":             emojiPayload(provider.Icon),
		"description":      provider.Description,
		"synced":           provider.Version == workflowToolVersion(app),
		"tool":             tool,
		"privacy_policy":   provider.PrivacyPolicy,
	}
}

func (s *server) mcpProviderPayload(provider state.WorkspaceMCPToolProvider) map[string]any {
	tools := make([]map[string]any, 0, len(provider.Tools))
	for _, tool := range provider.Tools {
		tools = append(tools, s.toolPayload(tool))
	}

	icon := any(provider.Icon)
	if provider.IconType == "emoji" || provider.IconType == "" {
		icon = map[string]any{
			"background": firstNonEmpty(provider.IconBackground, "#E5E7EB"),
			"content":    firstNonEmpty(provider.Icon, "🔗"),
		}
	}

	return map[string]any{
		"id":                      provider.ID,
		"name":                    provider.Name,
		"author":                  s.userDisplayName(provider.CreatedBy),
		"description":             localizedText(firstNonEmpty(provider.Name, provider.ServerIdentifier)),
		"icon":                    icon,
		"label":                   localizedText(firstNonEmpty(provider.Name, provider.ServerIdentifier)),
		"type":                    "mcp",
		"team_credentials":        map[string]any{},
		"is_team_authorization":   provider.IsAuthorized,
		"allow_delete":            true,
		"labels":                  []string{"mcp"},
		"server_url":              provider.ServerURL,
		"updated_at":              provider.UpdatedAt,
		"server_identifier":       provider.ServerIdentifier,
		"timeout":                 provider.Configuration.Timeout,
		"sse_read_timeout":        provider.Configuration.SSEReadTimeout,
		"headers":                 provider.Headers,
		"masked_headers":          provider.Headers,
		"is_authorized":           provider.IsAuthorized,
		"provider":                provider.Name,
		"is_dynamic_registration": provider.IsDynamicRegistration,
		"authentication": map[string]any{
			"client_id":     provider.Authentication.ClientID,
			"client_secret": provider.Authentication.ClientSecret,
		},
		"configuration": map[string]any{
			"timeout":          provider.Configuration.Timeout,
			"sse_read_timeout": provider.Configuration.SSEReadTimeout,
		},
		"tools": tools,
		"meta":  map[string]any{"version": s.cfg.AppVersion},
	}
}

func (s *server) toolPayload(tool state.WorkspaceTool) map[string]any {
	return map[string]any{
		"name":          tool.Name,
		"author":        tool.Author,
		"label":         localizedText(firstNonEmpty(tool.Label, tool.Name)),
		"description":   localizedText(firstNonEmpty(tool.Description, tool.Label, tool.Name)),
		"parameters":    s.toolParameterPayloads(tool.Parameters),
		"labels":        tool.Labels,
		"output_schema": tool.OutputSchema,
	}
}

func (s *server) toolParameterPayloads(parameters []state.WorkspaceToolParameter) []map[string]any {
	items := make([]map[string]any, 0, len(parameters))
	for _, parameter := range parameters {
		item := map[string]any{
			"name":              parameter.Name,
			"label":             localizedText(firstNonEmpty(parameter.Label, parameter.Name)),
			"human_description": localizedText(parameter.HumanDescription),
			"type":              normalizeParameterType(parameter.Type),
			"form":              firstNonEmpty(parameter.Form, "llm"),
			"llm_description":   parameter.LLMDescription,
			"required":          parameter.Required,
			"multiple":          parameter.Multiple,
			"default":           parameter.Default,
		}
		if len(parameter.Options) > 0 {
			options := make([]map[string]any, 0, len(parameter.Options))
			for _, option := range parameter.Options {
				options = append(options, map[string]any{
					"label": localizedText(firstNonEmpty(option.Label, option.Value)),
					"value": option.Value,
				})
			}
			item["options"] = options
		}
		if parameter.Min != nil {
			item["min"] = *parameter.Min
		}
		if parameter.Max != nil {
			item["max"] = *parameter.Max
		}
		items = append(items, item)
	}
	return items
}

func (s *server) apiSchemaPreview(operations []parsedAPIOperation) []map[string]any {
	items := make([]map[string]any, 0, len(operations))
	for _, operation := range operations {
		items = append(items, map[string]any{
			"operation_id": operation.Name,
			"summary":      operation.Summary,
			"server_url":   joinURLPath(operation.ServerURL, operation.Path),
			"method":       operation.Method,
			"parameters":   s.toolParameterPayloads(toolParametersFromAPIOperation(operation)),
		})
	}
	return items
}

func toolParametersFromAPIOperation(operation parsedAPIOperation) []state.WorkspaceToolParameter {
	parameters := make([]state.WorkspaceToolParameter, 0, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		parameters = append(parameters, cloneToolParameter(parameter.Parameter))
	}
	return parameters
}

func cloneToolParameter(parameter state.WorkspaceToolParameter) state.WorkspaceToolParameter {
	out := state.WorkspaceToolParameter{
		Name:             parameter.Name,
		Label:            parameter.Label,
		HumanDescription: parameter.HumanDescription,
		Type:             parameter.Type,
		Form:             parameter.Form,
		LLMDescription:   parameter.LLMDescription,
		Required:         parameter.Required,
		Multiple:         parameter.Multiple,
		Default:          parameter.Default,
		Options:          make([]state.WorkspaceToolOption, len(parameter.Options)),
	}
	if parameter.Min != nil {
		value := *parameter.Min
		out.Min = &value
	}
	if parameter.Max != nil {
		value := *parameter.Max
		out.Max = &value
	}
	copy(out.Options, parameter.Options)
	return out
}

func workflowToolParametersFromPayload(items []struct {
	Name        string `json:"name"`
	Form        string `json:"form"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
}) []state.WorkspaceWorkflowToolParameter {
	out := make([]state.WorkspaceWorkflowToolParameter, 0, len(items))
	for _, item := range items {
		out = append(out, state.WorkspaceWorkflowToolParameter{
			Name:        strings.TrimSpace(item.Name),
			Form:        firstNonEmpty(item.Form, "llm"),
			Description: strings.TrimSpace(item.Description),
			Required:    item.Required,
			Type:        normalizeParameterType(item.Type),
		})
	}
	return out
}

func workflowToolOutputSchema(app state.App) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func workflowToolVersion(app state.App) string {
	if app.WorkflowPublished != nil {
		return firstNonEmpty(app.WorkflowPublished.Version, app.WorkflowPublished.Hash, fmt.Sprintf("%d", app.WorkflowPublished.UpdatedAt))
	}
	if app.WorkflowDraft != nil {
		return firstNonEmpty(app.WorkflowDraft.Version, app.WorkflowDraft.Hash, fmt.Sprintf("%d", app.WorkflowDraft.UpdatedAt))
	}
	if app.Workflow != nil {
		return fmt.Sprintf("%d", app.Workflow.UpdatedAt)
	}
	return fmt.Sprintf("%d", app.UpdatedAt)
}

func localizedText(value string) map[string]string {
	value = strings.TrimSpace(value)
	return map[string]string{
		"en_US":   value,
		"zh_Hans": value,
	}
}

func emojiPayload(icon state.WorkspaceEmoji) map[string]any {
	icon = state.WorkspaceEmoji{
		Background: firstNonEmpty(icon.Background, "#E5E7EB"),
		Content:    firstNonEmpty(icon.Content, "🧩"),
	}
	return map[string]any{
		"background": icon.Background,
		"content":    icon.Content,
	}
}

func emojiFromPayload(payload map[string]any) state.WorkspaceEmoji {
	return state.WorkspaceEmoji{
		Background: stringValue(payload["background"], "#E5E7EB"),
		Content:    stringValue(payload["content"], "🧩"),
	}
}

func builtinToolCatalogs() []builtinToolCatalog {
	return []builtinToolCatalog{
		{
			Provider:    "serpapi",
			Author:      "dify-go",
			Label:       "SerpApi",
			Description: "Search the public web with SerpApi-backed Google results.",
			Icon:        providerIconDataURI("S", "#FFF2CC", "#7A4D00"),
			Labels:      []string{"search", "web"},
			Tools: []state.WorkspaceTool{
				{
					Name:        "google_search",
					Author:      "dify-go",
					Label:       "Google Search",
					Description: "Search Google results with a natural-language query.",
					Parameters: []state.WorkspaceToolParameter{
						{Name: "query", Label: "Query", HumanDescription: "Search query", Type: "string", Form: "llm", LLMDescription: "Search query", Required: true, Options: []state.WorkspaceToolOption{}},
					},
					Labels:       []string{"search"},
					OutputSchema: map[string]any{},
				},
			},
			CredentialSchema: []map[string]any{
				{
					"name":        "api_key",
					"label":       localizedText("API Key"),
					"help":        localizedText("Create a SerpApi API key in your dashboard."),
					"placeholder": localizedText("Paste your SerpApi API key"),
					"type":        "secret-input",
					"required":    true,
					"default":     "",
				},
			},
		},
		{
			Provider:    "wikipedia",
			Author:      "dify-go",
			Label:       "Wikipedia",
			Description: "Find concise reference information from Wikipedia.",
			Icon:        providerIconDataURI("W", "#E5F2FF", "#144A75"),
			Labels:      []string{"search", "knowledge"},
			Tools: []state.WorkspaceTool{
				{
					Name:        "search_wikipedia",
					Author:      "dify-go",
					Label:       "Search Wikipedia",
					Description: "Look up a topic in Wikipedia and return a summary.",
					Parameters: []state.WorkspaceToolParameter{
						{Name: "query", Label: "Query", HumanDescription: "Topic to search", Type: "string", Form: "llm", LLMDescription: "Topic to search", Required: true, Options: []state.WorkspaceToolOption{}},
					},
					Labels:       []string{"knowledge"},
					OutputSchema: map[string]any{},
				},
			},
			CredentialSchema: []map[string]any{},
		},
		{
			Provider:    "webscraper",
			Author:      "dify-go",
			Label:       "Web Scraper",
			Description: "Fetch a webpage and turn the content into model-friendly text.",
			Icon:        providerIconDataURI("R", "#E8F7EC", "#166534"),
			Labels:      []string{"web", "extract"},
			Tools: []state.WorkspaceTool{
				{
					Name:        "scrape_webpage",
					Author:      "dify-go",
					Label:       "Scrape Webpage",
					Description: "Download a webpage and extract readable content.",
					Parameters: []state.WorkspaceToolParameter{
						{Name: "url", Label: "URL", HumanDescription: "Page URL", Type: "string", Form: "llm", LLMDescription: "Page URL", Required: true, Options: []state.WorkspaceToolOption{}},
					},
					Labels:       []string{"web"},
					OutputSchema: map[string]any{},
				},
			},
			CredentialSchema: []map[string]any{},
		},
		{
			Provider:    "current_time",
			Author:      "dify-go",
			Label:       "Current Time",
			Description: "Return the current time in the requested timezone.",
			Icon:        providerIconDataURI("T", "#FDECEC", "#991B1B"),
			Labels:      []string{"utility"},
			Tools: []state.WorkspaceTool{
				{
					Name:        "get_current_time",
					Author:      "dify-go",
					Label:       "Get Current Time",
					Description: "Get the current time for a timezone offset or city.",
					Parameters: []state.WorkspaceToolParameter{
						{Name: "timezone", Label: "Timezone", HumanDescription: "Timezone offset or city", Type: "string", Form: "llm", LLMDescription: "Timezone offset or city", Required: false, Options: []state.WorkspaceToolOption{}},
					},
					Labels:       []string{"utility"},
					OutputSchema: map[string]any{},
				},
			},
			CredentialSchema: []map[string]any{},
		},
	}
}

func builtinToolCatalogByProvider(provider string) (builtinToolCatalog, bool) {
	for _, catalog := range builtinToolCatalogs() {
		if catalog.Provider == strings.TrimSpace(provider) {
			return catalog, true
		}
	}
	return builtinToolCatalog{}, false
}

func parseAPISchemaDocument(schema string) (parsedAPISchema, error) {
	schema = strings.TrimSpace(schema)
	if schema == "" {
		return parsedAPISchema{}, fmt.Errorf("schema is required")
	}

	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(schema), &root); err != nil {
		return parsedAPISchema{}, fmt.Errorf("parse schema: %w", err)
	}

	result := parsedAPISchema{
		SchemaType: detectAPISchemaType(root),
		Operations: []parsedAPIOperation{},
	}

	baseURL := baseURLFromSchema(root)
	paths := mapValue(root["paths"])
	if len(paths) == 0 {
		return parsedAPISchema{}, fmt.Errorf("schema paths are required")
	}

	pathKeys := make([]string, 0, len(paths))
	for path := range paths {
		pathKeys = append(pathKeys, path)
	}
	slices.Sort(pathKeys)

	for _, path := range pathKeys {
		pathItem := mapValue(paths[path])
		if len(pathItem) == 0 {
			continue
		}
		pathParameters := arrayValue(pathItem["parameters"])
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			operationMap := mapValue(pathItem[strings.ToLower(method)])
			if len(operationMap) == 0 {
				continue
			}
			name := normalizeIdentifier(firstNonEmpty(stringValue(operationMap["operationId"], ""), method+"_"+path))
			summary := firstNonEmpty(stringValue(operationMap["summary"], ""), stringValue(operationMap["description"], ""), name)
			parameters := parseAPIParameters(root, append(slices.Clone(pathParameters), arrayValue(operationMap["parameters"])...))
			requestParameters := parseRequestBodyParameters(root, operationMap)
			parameters = append(parameters, requestParameters...)
			result.Operations = append(result.Operations, parsedAPIOperation{
				Name:       name,
				Summary:    summary,
				Method:     strings.ToUpper(method),
				Path:       path,
				ServerURL:  baseURL,
				Parameters: parameters,
			})
		}
	}

	if len(result.Operations) == 0 {
		return parsedAPISchema{}, fmt.Errorf("schema does not define any operations")
	}

	return result, nil
}

func detectAPISchemaType(root map[string]any) string {
	if strings.TrimSpace(stringValue(root["swagger"], "")) != "" {
		return "swagger"
	}
	return "openapi"
}

func baseURLFromSchema(root map[string]any) string {
	if servers := arrayValue(root["servers"]); len(servers) > 0 {
		server := mapValue(servers[0])
		if url := strings.TrimSpace(stringValue(server["url"], "")); url != "" {
			return url
		}
	}

	host := strings.TrimSpace(stringValue(root["host"], ""))
	if host == "" {
		return ""
	}
	schemes := arrayValue(root["schemes"])
	scheme := "https"
	if len(schemes) > 0 {
		scheme = firstNonEmpty(anyToString(schemes[0]), scheme)
	}
	basePath := strings.TrimSpace(stringValue(root["basePath"], ""))
	return strings.TrimRight(fmt.Sprintf("%s://%s%s", scheme, host, basePath), "/")
}

func parseAPIParameters(root map[string]any, raw []any) []apiParameterSpec {
	items := make([]apiParameterSpec, 0, len(raw))
	for _, entry := range raw {
		parameterMap := resolveSchemaRef(root, mapValue(entry))
		if len(parameterMap) == 0 {
			continue
		}
		name := strings.TrimSpace(stringValue(parameterMap["name"], ""))
		if name == "" {
			continue
		}
		location := strings.TrimSpace(stringValue(parameterMap["in"], "query"))
		schema := resolveSchemaRef(root, mapValue(parameterMap["schema"]))
		parameter := buildToolParameterFromSchema(name, firstNonEmpty(stringValue(parameterMap["description"], ""), stringValue(schema["description"], "")), schema)
		parameter.Required = boolValue(parameterMap["required"], false)
		items = append(items, apiParameterSpec{In: location, Parameter: parameter})
	}
	return items
}

func parseRequestBodyParameters(root map[string]any, operation map[string]any) []apiParameterSpec {
	requestBody := resolveSchemaRef(root, mapValue(operation["requestBody"]))
	if len(requestBody) == 0 {
		return []apiParameterSpec{}
	}
	content := mapValue(requestBody["content"])
	if len(content) == 0 {
		return []apiParameterSpec{}
	}

	bodySchema := map[string]any{}
	if jsonContent := resolveSchemaRef(root, mapValue(content["application/json"])); len(jsonContent) > 0 {
		bodySchema = resolveSchemaRef(root, mapValue(jsonContent["schema"]))
	} else {
		for _, value := range content {
			bodySchema = resolveSchemaRef(root, mapValue(mapValue(value)["schema"]))
			if len(bodySchema) > 0 {
				break
			}
		}
	}
	if len(bodySchema) == 0 {
		return []apiParameterSpec{}
	}

	requiredMap := make(map[string]bool)
	for _, name := range arrayValue(bodySchema["required"]) {
		requiredMap[anyToString(name)] = true
	}

	properties := mapValue(bodySchema["properties"])
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	items := make([]apiParameterSpec, 0, len(keys))
	for _, key := range keys {
		propertySchema := resolveSchemaRef(root, mapValue(properties[key]))
		parameter := buildToolParameterFromSchema(key, stringValue(propertySchema["description"], ""), propertySchema)
		parameter.Required = requiredMap[key]
		items = append(items, apiParameterSpec{In: "body", Parameter: parameter})
	}
	return items
}

func buildToolParameterFromSchema(name, description string, schema map[string]any) state.WorkspaceToolParameter {
	parameterType, multiple := schemaType(schema)
	parameter := state.WorkspaceToolParameter{
		Name:             name,
		Label:            name,
		HumanDescription: description,
		Type:             normalizeParameterType(parameterType),
		Form:             "llm",
		LLMDescription:   description,
		Required:         false,
		Multiple:         multiple,
		Default:          firstNonEmpty(anyToString(schema["default"]), ""),
		Options:          []state.WorkspaceToolOption{},
	}

	for _, option := range arrayValue(schema["enum"]) {
		value := anyToString(option)
		parameter.Options = append(parameter.Options, state.WorkspaceToolOption{
			Label: value,
			Value: value,
		})
	}
	if value, ok := numberValue(schema["minimum"]); ok {
		parameter.Min = &value
	}
	if value, ok := numberValue(schema["maximum"]); ok {
		parameter.Max = &value
	}
	return parameter
}

func schemaType(schema map[string]any) (string, bool) {
	if ref := strings.TrimSpace(stringValue(schema["$ref"], "")); ref != "" {
		return "string", false
	}
	if typeValue := schema["type"]; typeValue != nil {
		switch typed := typeValue.(type) {
		case string:
			if typed == "array" {
				items := mapValue(schema["items"])
				itemType, _ := schemaType(items)
				return itemType, true
			}
			return typed, false
		case []any:
			for _, item := range typed {
				value := anyToString(item)
				if value != "" && value != "null" {
					return value, false
				}
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		for _, item := range arrayValue(schema[key]) {
			itemType, multiple := schemaType(mapValue(item))
			if itemType != "" {
				return itemType, multiple
			}
		}
	}
	return "string", false
}

func resolveSchemaRef(root, value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	ref := strings.TrimSpace(stringValue(value["$ref"], ""))
	if ref == "" {
		return value
	}
	if !strings.HasPrefix(ref, "#/") {
		return value
	}

	current := any(root)
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		next, ok := current.(map[string]any)
		if !ok {
			return value
		}
		current = next[part]
	}
	resolved, ok := current.(map[string]any)
	if !ok {
		return value
	}
	if nestedRef := strings.TrimSpace(stringValue(resolved["$ref"], "")); nestedRef != "" && nestedRef != ref {
		return resolveSchemaRef(root, resolved)
	}
	return resolved
}

func previewAPIToolCall(ctx any, schema, toolName string, credentials map[string]any, parameters map[string]any) map[string]any {
	parsed, err := parseAPISchemaDocument(schema)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	index := slices.IndexFunc(parsed.Operations, func(operation parsedAPIOperation) bool {
		return operation.Name == strings.TrimSpace(toolName)
	})
	if index < 0 {
		return map[string]any{"error": fmt.Sprintf("tool %s not found", toolName)}
	}
	operation := parsed.Operations[index]

	requestURL := joinURLPath(operation.ServerURL, operation.Path)
	if requestURL == "" {
		return map[string]any{"error": "schema server url is required"}
	}
	parsedURL, err := neturl.Parse(requestURL)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	body := map[string]any{}
	for _, parameter := range operation.Parameters {
		value, ok := parameters[parameter.Parameter.Name]
		if !ok {
			continue
		}
		switch parameter.In {
		case "path":
			parsedURL.Path = strings.ReplaceAll(parsedURL.Path, "{"+parameter.Parameter.Name+"}", neturl.PathEscape(anyToString(value)))
		case "query":
			query := parsedURL.Query()
			query.Set(parameter.Parameter.Name, anyToString(value))
			parsedURL.RawQuery = query.Encode()
		case "body":
			body[parameter.Parameter.Name] = value
		case "header":
		default:
			query := parsedURL.Query()
			query.Set(parameter.Parameter.Name, anyToString(value))
			parsedURL.RawQuery = query.Encode()
		}
	}

	applyCredentialToURL(parsedURL, credentials)

	var bodyReader io.Reader
	if len(body) > 0 {
		payload, err := json.Marshal(body)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(strings.ToUpper(operation.Method), parsedURL.String(), bodyReader)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	applyCredentialToRequest(req, credentials)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	text := strings.TrimSpace(string(responseBody))
	if resp.StatusCode >= 400 {
		return map[string]any{"error": fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, text)}
	}
	if text == "" {
		text = resp.Status
	}
	return map[string]any{"result": text}
}

func applyCredentialToURL(parsedURL *neturl.URL, credentials map[string]any) {
	authType := strings.TrimSpace(stringValue(credentials["auth_type"], ""))
	if authType != "api_key_query" {
		return
	}
	name := firstNonEmpty(stringValue(credentials["api_key_query_param"], ""), "api_key")
	value := strings.TrimSpace(stringValue(credentials["api_key_value"], ""))
	if value == "" {
		return
	}
	query := parsedURL.Query()
	query.Set(name, value)
	parsedURL.RawQuery = query.Encode()
}

func applyCredentialToRequest(req *http.Request, credentials map[string]any) {
	authType := strings.TrimSpace(stringValue(credentials["auth_type"], ""))
	if authType != "api_key" && authType != "api_key_header" {
		return
	}

	headerName := firstNonEmpty(stringValue(credentials["api_key_header"], ""), "Authorization")
	value := strings.TrimSpace(stringValue(credentials["api_key_value"], ""))
	if value == "" {
		return
	}
	switch strings.TrimSpace(stringValue(credentials["api_key_header_prefix"], "")) {
	case "basic":
		value = "Basic " + value
	case "bearer":
		value = "Bearer " + value
	}
	req.Header.Set(headerName, value)
}

func generatedMCPTools(provider state.WorkspaceMCPToolProvider, author string) []state.WorkspaceTool {
	prefix := normalizeIdentifier(firstNonEmpty(provider.ServerIdentifier, provider.Name, "mcp"))
	if prefix == "" {
		prefix = "mcp"
	}
	return []state.WorkspaceTool{
		{
			Name:        prefix + "_call",
			Author:      author,
			Label:       firstNonEmpty(provider.Name, provider.ServerIdentifier) + " Call",
			Description: "Invoke the connected MCP server with a free-form request.",
			Parameters: []state.WorkspaceToolParameter{
				{Name: "request", Label: "Request", HumanDescription: "Instruction sent to the MCP server", Type: "string", Form: "llm", LLMDescription: "Instruction sent to the MCP server", Required: true, Options: []state.WorkspaceToolOption{}},
			},
			Labels:       []string{"mcp"},
			OutputSchema: map[string]any{},
		},
		{
			Name:         prefix + "_status",
			Author:       author,
			Label:        firstNonEmpty(provider.Name, provider.ServerIdentifier) + " Status",
			Description:  "Inspect the current MCP server connection and available capabilities.",
			Parameters:   []state.WorkspaceToolParameter{},
			Labels:       []string{"mcp"},
			OutputSchema: map[string]any{},
		},
	}
}

func normalizeIdentifier(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-' || unicode.IsSpace(r) || r == '/':
			if !lastUnderscore {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(builder.String(), "_")
}

func normalizeParameterType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "paragraph", "text-input", "url":
		return "string"
	case "number", "integer", "int":
		return "number"
	case "checkbox", "boolean", "bool":
		return "boolean"
	case "file", "file-list":
		return "file"
	case "array", "list":
		return "array"
	case "object", "json", "json_object":
		return "object"
	case "":
		return "string"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func joinURLPath(baseURL, path string) string {
	baseURL = strings.TrimSpace(baseURL)
	path = strings.TrimSpace(path)
	if baseURL == "" {
		return path
	}
	if path == "" {
		return baseURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func arrayValue(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = item
		}
		return items
	default:
		return []any{}
	}
}

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[anyToString(key)] = item
		}
		return out
	default:
		return map[string]any{}
	}
}

func stringValue(value any, fallback string) string {
	str := strings.TrimSpace(anyToString(value))
	if str == "" {
		return fallback
	}
	return str
}

func boolValue(value any, fallback bool) bool {
	boolean, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolean
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func (s *server) userDisplayName(userID string) string {
	user, ok := s.store.GetUser(userID)
	if !ok {
		return "Anonymous"
	}
	return firstNonEmpty(user.Name, user.Email, "Anonymous")
}

type mcpPayload struct {
	ProviderID            string
	ServerURL             string
	Name                  string
	Icon                  string
	IconType              string
	IconBackground        string
	ServerIdentifier      string
	Headers               map[string]string
	IsDynamicRegistration bool
	Authentication        state.WorkspaceMCPAuthentication
	Configuration         state.WorkspaceMCPConfiguration
}

func decodeMCPPayload(r *http.Request) (mcpPayload, error) {
	var payload struct {
		ProviderID            string            `json:"provider_id"`
		ServerURL             string            `json:"server_url"`
		Name                  string            `json:"name"`
		Icon                  string            `json:"icon"`
		IconType              string            `json:"icon_type"`
		IconBackground        string            `json:"icon_background"`
		ServerIdentifier      string            `json:"server_identifier"`
		Headers               map[string]string `json:"headers"`
		IsDynamicRegistration *bool             `json:"is_dynamic_registration"`
		Authentication        struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		} `json:"authentication"`
		Configuration struct {
			Timeout        int `json:"timeout"`
			SSEReadTimeout int `json:"sse_read_timeout"`
		} `json:"configuration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return mcpPayload{}, fmt.Errorf("invalid JSON payload")
	}

	isDynamicRegistration := true
	if payload.IsDynamicRegistration != nil {
		isDynamicRegistration = *payload.IsDynamicRegistration
	}

	return mcpPayload{
		ProviderID:            strings.TrimSpace(payload.ProviderID),
		ServerURL:             strings.TrimSpace(payload.ServerURL),
		Name:                  strings.TrimSpace(payload.Name),
		Icon:                  strings.TrimSpace(payload.Icon),
		IconType:              strings.TrimSpace(payload.IconType),
		IconBackground:        strings.TrimSpace(payload.IconBackground),
		ServerIdentifier:      strings.TrimSpace(payload.ServerIdentifier),
		Headers:               payload.Headers,
		IsDynamicRegistration: isDynamicRegistration,
		Authentication: state.WorkspaceMCPAuthentication{
			ClientID:     strings.TrimSpace(payload.Authentication.ClientID),
			ClientSecret: strings.TrimSpace(payload.Authentication.ClientSecret),
		},
		Configuration: state.WorkspaceMCPConfiguration{
			Timeout:        payload.Configuration.Timeout,
			SSEReadTimeout: payload.Configuration.SSEReadTimeout,
		},
	}, nil
}

func (p mcpPayload) createInput() state.CreateMCPToolProviderInput {
	return state.CreateMCPToolProviderInput{
		Name:                  p.Name,
		IconType:              p.IconType,
		Icon:                  p.Icon,
		IconBackground:        p.IconBackground,
		ServerURL:             p.ServerURL,
		ServerIdentifier:      p.ServerIdentifier,
		Headers:               p.Headers,
		Authentication:        p.Authentication,
		Configuration:         p.Configuration,
		IsDynamicRegistration: p.IsDynamicRegistration,
	}
}

func (p mcpPayload) updateInput() state.UpdateMCPToolProviderInput {
	return state.UpdateMCPToolProviderInput{
		ProviderID:            p.ProviderID,
		Name:                  p.Name,
		IconType:              p.IconType,
		Icon:                  p.Icon,
		IconBackground:        p.IconBackground,
		ServerURL:             p.ServerURL,
		ServerIdentifier:      p.ServerIdentifier,
		Headers:               p.Headers,
		Authentication:        p.Authentication,
		Configuration:         p.Configuration,
		IsDynamicRegistration: p.IsDynamicRegistration,
	}
}
