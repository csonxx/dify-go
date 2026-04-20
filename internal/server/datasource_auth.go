package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) mountDatasourceAuthRoutes(r chi.Router) {
	r.Get("/auth/plugin/datasource/list", s.handleDatasourceAuthList)
	r.Get("/auth/plugin/datasource/default-list", s.handleDatasourceAuthDefaultList)
	r.Get("/auth/plugin/datasource/*", s.handleDatasourceAuthGet)
	r.Post("/auth/plugin/datasource/*", s.handleDatasourceAuthPost)
	r.Delete("/auth/plugin/datasource/*", s.handleDatasourceAuthDelete)
	r.Get("/oauth/plugin/*", s.handleDatasourceOAuthGet)
}

func (s *server) handleDatasourceAuthList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": s.datasourceAuthCatalogPayload(r, workspace.ID, false),
	})
}

func (s *server) handleDatasourceAuthDefaultList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": s.datasourceAuthCatalogPayload(r, workspace.ID, true),
	})
}

func (s *server) handleDatasourceAuthGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerID, action := datasourceAuthRouteParts(r)
	if action != "" {
		writeError(w, http.StatusNotFound, "not_found", "Datasource auth route not found.")
		return
	}

	spec, err := s.datasourceSpecFromProviderID(providerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider_not_found", err.Error())
		return
	}

	providerState, _ := s.store.GetWorkspaceDatasourceProviderState(workspace.ID, spec.PluginID, spec.Provider)
	writeJSON(w, http.StatusOK, map[string]any{
		"result": s.datasourceCredentialListPayload(providerState),
	})
}

func (s *server) handleDatasourceAuthPost(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerID, action := datasourceAuthRouteParts(r)
	spec, providerState, err := s.datasourceSpecFromProviderIDForWorkspace(workspace.ID, providerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider_not_found", err.Error())
		return
	}

	switch action {
	case "":
		s.handleCreateDatasourceCredential(w, r, workspace, spec)
	case "update":
		s.handleUpdateDatasourceCredential(w, r, workspace, spec, providerState)
	case "update-name":
		s.handleUpdateDatasourceCredential(w, r, workspace, spec, providerState)
	case "delete":
		s.handleDeleteDatasourceCredential(w, r, workspace, spec)
	case "default":
		s.handleSetDefaultDatasourceCredential(w, r, workspace, spec)
	case "custom-client":
		s.handleDatasourceOAuthCustomClientUpsert(w, r, workspace, spec, providerState)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Datasource auth route not found.")
	}
}

func (s *server) handleDatasourceAuthDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerID, action := datasourceAuthRouteParts(r)
	if action != "custom-client" {
		writeError(w, http.StatusNotFound, "not_found", "Datasource auth route not found.")
		return
	}

	spec, _, err := s.datasourceSpecFromProviderIDForWorkspace(workspace.ID, providerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider_not_found", err.Error())
		return
	}

	if _, err := s.store.DeleteWorkspaceDatasourceOAuthClient(workspace.ID, spec.PluginID, spec.Provider, spec.PluginUniqueIdentifier, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasourceOAuthGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerID, action := datasourceOAuthRouteParts(r)
	spec, providerState, err := s.datasourceSpecFromProviderIDForWorkspace(workspace.ID, providerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "provider_not_found", err.Error())
		return
	}

	switch action {
	case "get-authorization-url":
		s.handleDatasourceOAuthAuthorizationURL(w, r, workspace, spec, providerState)
	case "callback":
		s.handleDatasourceOAuthCallback(w, r, workspace, spec, providerState)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Datasource oauth route not found.")
	}
}

func (s *server) handleCreateDatasourceCredential(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec) {
	var payload struct {
		Credentials map[string]any `json:"credentials"`
		Type        string         `json:"type"`
		Name        string         `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if normalizeDatasourceCredentialType(payload.Type) != "api-key" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Only api-key datasource credentials can be created directly.")
		return
	}
	if len(spec.CredentialSchema) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource does not support API key credentials.")
		return
	}

	if _, _, err := s.store.CreateWorkspaceDatasourceCredential(workspace.ID, currentUser(r), state.CreateWorkspaceDatasourceCredentialInput{
		PluginID:               spec.PluginID,
		Provider:               spec.Provider,
		PluginUniqueIdentifier: spec.PluginUniqueIdentifier,
		Name:                   payload.Name,
		Type:                   "api-key",
		AvatarURL:              spec.Icon,
		Credential:             payload.Credentials,
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleUpdateDatasourceCredential(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec, providerState state.WorkspaceDatasourceProviderState) {
	var payload struct {
		CredentialID string         `json:"credential_id"`
		Credentials  map[string]any `json:"credentials"`
		Name         string         `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	credential, ok := datasourceCredentialByID(providerState, payload.CredentialID)
	if !ok {
		writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
		return
	}

	var mergedCredentials map[string]any
	if payload.Credentials != nil {
		mergedCredentials = mergeSecretMap(credential.Credential, payload.Credentials)
	}

	if _, _, err := s.store.UpdateWorkspaceDatasourceCredential(workspace.ID, state.UpdateWorkspaceDatasourceCredentialInput{
		PluginID:     spec.PluginID,
		Provider:     spec.Provider,
		CredentialID: credential.ID,
		Name:         payload.Name,
		Credential:   mergedCredentials,
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDeleteDatasourceCredential(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec) {
	var payload struct {
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.DeleteWorkspaceDatasourceCredential(workspace.ID, spec.PluginID, spec.Provider, payload.CredentialID, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleSetDefaultDatasourceCredential(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.SetWorkspaceDatasourceDefaultCredential(workspace.ID, spec.PluginID, spec.Provider, payload.ID, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasourceOAuthCustomClientUpsert(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec, providerState state.WorkspaceDatasourceProviderState) {
	if len(spec.OAuthClientSchema) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource does not support OAuth client settings.")
		return
	}

	var payload struct {
		ClientParams            map[string]any `json:"client_params"`
		EnableOAuthCustomClient bool           `json:"enable_oauth_custom_client"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	params := mergeSecretStringMap(providerState.OAuthClient.Params, stringifyMap(payload.ClientParams))
	if _, err := s.store.UpsertWorkspaceDatasourceOAuthClient(workspace.ID, spec.PluginID, spec.Provider, spec.PluginUniqueIdentifier, params, payload.EnableOAuthCustomClient, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasourceOAuthAuthorizationURL(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec, providerState state.WorkspaceDatasourceProviderState) {
	if len(spec.OAuthClientSchema) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource does not support OAuth.")
		return
	}
	if !spec.SystemOAuth && !providerState.OAuthClient.Enabled {
		writeError(w, http.StatusBadRequest, "invalid_request", "OAuth client is not configured.")
		return
	}

	credentialID := strings.TrimSpace(r.URL.Query().Get("credential_id"))
	stateToken := generateOAuthTokenSeed()
	callbackURL := fmt.Sprintf(
		"%s/console/api/oauth/plugin/%s/datasource/callback?state=%s&redirect_origin=%s",
		requestBaseURL(r),
		spec.PluginID+"/"+spec.Provider,
		neturl.QueryEscape(stateToken),
		neturl.QueryEscape(datasourceRedirectOrigin("", frontendOriginURL(r))),
	)
	if credentialID != "" {
		callbackURL += "&credential_id=" + neturl.QueryEscape(credentialID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authorization_url": callbackURL,
		"state":             stateToken,
		"context_id":        firstNonEmpty(credentialID, workspace.ID),
	})
}

func (s *server) handleDatasourceOAuthCallback(w http.ResponseWriter, r *http.Request, workspace state.Workspace, spec ragPipelineDatasourceProviderSpec, providerState state.WorkspaceDatasourceProviderState) {
	redirectOrigin := datasourceRedirectOrigin(r.URL.Query().Get("redirect_origin"), frontendOriginURL(r))
	redirectWithError := func(code, description string) {
		values := neturl.Values{}
		values.Set("error", code)
		values.Set("error_description", description)
		http.Redirect(w, r, buildDatasourceOAuthCallbackURL(redirectOrigin, values), http.StatusFound)
	}

	if len(spec.OAuthClientSchema) == 0 {
		redirectWithError("oauth_not_supported", "Datasource does not support OAuth.")
		return
	}

	credentialPayload := map[string]any{
		"access_token":  "oauth_" + strings.ReplaceAll(generateOAuthTokenSeed(), "-", ""),
		"refresh_token": "refresh_" + strings.ReplaceAll(generateOAuthTokenSeed(), "-", ""),
		"provider":      spec.Provider,
		"authorized_at": time.Now().UTC().Format(time.RFC3339),
	}

	credentialID := strings.TrimSpace(r.URL.Query().Get("credential_id"))
	if credentialID != "" {
		existing, ok := datasourceCredentialByID(providerState, credentialID)
		if !ok {
			redirectWithError("credential_not_found", "Credential not found.")
			return
		}
		if _, _, err := s.store.UpdateWorkspaceDatasourceCredential(workspace.ID, state.UpdateWorkspaceDatasourceCredentialInput{
			PluginID:     spec.PluginID,
			Provider:     spec.Provider,
			CredentialID: existing.ID,
			Type:         "oauth2",
			AvatarURL:    spec.Icon,
			Credential:   mergeSecretMap(existing.Credential, credentialPayload),
		}, time.Now()); err != nil {
			redirectWithError("credential_update_failed", err.Error())
			return
		}
	} else {
		if _, _, err := s.store.CreateWorkspaceDatasourceCredential(workspace.ID, currentUser(r), state.CreateWorkspaceDatasourceCredentialInput{
			PluginID:               spec.PluginID,
			Provider:               spec.Provider,
			PluginUniqueIdentifier: spec.PluginUniqueIdentifier,
			Name:                   spec.Label + " OAuth",
			Type:                   "oauth2",
			AvatarURL:              spec.Icon,
			Credential:             credentialPayload,
		}, time.Now()); err != nil {
			redirectWithError("credential_create_failed", err.Error())
			return
		}
	}

	values := neturl.Values{}
	if credentialID != "" {
		values.Set("credential_id", credentialID)
	}
	http.Redirect(w, r, buildDatasourceOAuthCallbackURL(redirectOrigin, values), http.StatusFound)
}

func (s *server) datasourceAuthCatalogPayload(r *http.Request, workspaceID string, defaultOnly bool) []map[string]any {
	specs := ragPipelineDatasourceProviderSpecs()
	items := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		if !spec.IncludeInAuthCatalog {
			continue
		}
		if defaultOnly && !spec.IncludeInAuthCatalog {
			continue
		}
		items = append(items, s.datasourceAuthProviderPayload(r, workspaceID, spec))
	}
	return items
}

func (s *server) datasourceAuthProviderPayload(r *http.Request, workspaceID string, spec ragPipelineDatasourceProviderSpec) map[string]any {
	providerState, _ := s.store.GetWorkspaceDatasourceProviderState(workspaceID, spec.PluginID, spec.Provider)
	payload := map[string]any{
		"author":                   spec.Author,
		"provider":                 spec.Provider,
		"plugin_id":                spec.PluginID,
		"plugin_unique_identifier": spec.PluginUniqueIdentifier,
		"icon":                     spec.Icon,
		"name":                     spec.Provider,
		"label":                    localizedText(spec.DatasourceLabel),
		"description":              localizedText(spec.DatasourceDescription),
		"credential_schema":        cloneSchemaList(spec.CredentialSchema),
		"credentials_list":         s.datasourceCredentialListPayload(providerState),
	}

	if len(spec.OAuthClientSchema) > 0 || len(spec.OAuthCredentialSchema) > 0 || providerState.OAuthClient.Enabled {
		payload["oauth_schema"] = map[string]any{
			"client_schema":                  cloneSchemaList(spec.OAuthClientSchema),
			"credentials_schema":             cloneSchemaList(spec.OAuthCredentialSchema),
			"is_oauth_custom_client_enabled": providerState.OAuthClient.Enabled,
			"is_system_oauth_params_exists":  spec.SystemOAuth,
			"oauth_custom_client_params":     maskOAuthClientParams(providerState.OAuthClient.Params),
			"redirect_uri":                   fmt.Sprintf("%s/oauth-callback", frontendOriginURL(r)),
		}
	}

	return payload
}

func (s *server) datasourceCredentialListPayload(providerState state.WorkspaceDatasourceProviderState) []map[string]any {
	items := make([]map[string]any, 0, len(providerState.Credentials))
	for _, item := range providerState.Credentials {
		items = append(items, map[string]any{
			"credential": maskSecrets(item.Credential),
			"type":       item.Type,
			"name":       item.Name,
			"id":         item.ID,
			"is_default": item.ID == providerState.DefaultCredentialID,
			"avatar_url": item.AvatarURL,
		})
	}
	return items
}

func (s *server) datasourceSpecFromProviderIDForWorkspace(workspaceID, providerID string) (ragPipelineDatasourceProviderSpec, state.WorkspaceDatasourceProviderState, error) {
	spec, err := s.datasourceSpecFromProviderID(providerID)
	if err != nil {
		return ragPipelineDatasourceProviderSpec{}, state.WorkspaceDatasourceProviderState{}, err
	}
	currentState, _ := s.store.GetWorkspaceDatasourceProviderState(workspaceID, spec.PluginID, spec.Provider)
	return spec, currentState, nil
}

func (s *server) datasourceSpecFromProviderID(providerID string) (ragPipelineDatasourceProviderSpec, error) {
	pluginID, provider, err := parseDatasourceProviderID(providerID)
	if err != nil {
		return ragPipelineDatasourceProviderSpec{}, err
	}
	spec, ok := ragPipelineDatasourceProviderSpecByProvider(pluginID, provider)
	if !ok || !spec.IncludeInAuthCatalog {
		return ragPipelineDatasourceProviderSpec{}, fmt.Errorf("datasource provider %s not found", providerID)
	}
	return spec, nil
}

func datasourceAuthRouteParts(r *http.Request) (providerID, action string) {
	tail := strings.Trim(strings.TrimPrefix(chi.URLParam(r, "*"), "/"), "/")
	switch {
	case strings.HasSuffix(tail, "/update-name"):
		return strings.TrimSuffix(tail, "/update-name"), "update-name"
	case strings.HasSuffix(tail, "/update"):
		return strings.TrimSuffix(tail, "/update"), "update"
	case strings.HasSuffix(tail, "/delete"):
		return strings.TrimSuffix(tail, "/delete"), "delete"
	case strings.HasSuffix(tail, "/default"):
		return strings.TrimSuffix(tail, "/default"), "default"
	case strings.HasSuffix(tail, "/custom-client"):
		return strings.TrimSuffix(tail, "/custom-client"), "custom-client"
	default:
		return tail, ""
	}
}

func datasourceOAuthRouteParts(r *http.Request) (providerID, action string) {
	tail := strings.Trim(strings.TrimPrefix(chi.URLParam(r, "*"), "/"), "/")
	switch {
	case strings.HasSuffix(tail, "/datasource/get-authorization-url"):
		return strings.TrimSuffix(tail, "/datasource/get-authorization-url"), "get-authorization-url"
	case strings.HasSuffix(tail, "/datasource/callback"):
		return strings.TrimSuffix(tail, "/datasource/callback"), "callback"
	default:
		return tail, ""
	}
}

func parseDatasourceProviderID(providerID string) (string, string, error) {
	providerID = strings.Trim(strings.TrimSpace(providerID), "/")
	index := strings.LastIndex(providerID, "/")
	if index <= 0 || index >= len(providerID)-1 {
		return "", "", fmt.Errorf("invalid datasource provider id")
	}
	return providerID[:index], providerID[index+1:], nil
}

func datasourceCredentialByID(providerState state.WorkspaceDatasourceProviderState, credentialID string) (state.WorkspaceDatasourceCredential, bool) {
	credentialID = strings.TrimSpace(credentialID)
	for _, item := range providerState.Credentials {
		if item.ID == credentialID {
			return item, true
		}
	}
	return state.WorkspaceDatasourceCredential{}, false
}

func cloneSchemaList(input []map[string]any) []map[string]any {
	if input == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, len(input))
	for i, item := range input {
		out[i] = cloneJSONObject(item)
	}
	return out
}

func stringMapToAnyMap(input map[string]string) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func maskOAuthClientParams(input map[string]string) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if shouldMaskOAuthClientParam(key) && strings.TrimSpace(value) != "" {
			out[key] = hiddenSecretValue
			continue
		}
		out[key] = value
	}
	return out
}

func shouldMaskOAuthClientParam(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(key, "secret") || strings.Contains(key, "token") || strings.HasSuffix(key, "key")
}

func datasourceRedirectOrigin(raw, fallback string) string {
	for _, candidate := range []string{raw, fallback} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		parsed, err := neturl.Parse(candidate)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		return strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
	}
	return strings.TrimRight(fallback, "/")
}

func buildDatasourceOAuthCallbackURL(origin string, values neturl.Values) string {
	base := strings.TrimRight(origin, "/") + "/oauth-callback"
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}

func normalizeDatasourceCredentialType(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "oauth2":
		return "oauth2"
	case "api-key", "api_key":
		return "api-key"
	default:
		return ""
	}
}
