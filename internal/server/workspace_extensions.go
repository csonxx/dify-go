package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

const hiddenSecretValue = "[__HIDDEN__]"

type triggerProviderCatalog struct {
	Provider               string
	PluginID               string
	PluginUniqueIdentifier string
	Author                 string
	Label                  string
	Description            string
	Icon                   string
	IconDark               string
	Tags                   []string
	SupportedMethods       []string
	CredentialsSchema      []map[string]any
	OAuthClientSchema      []map[string]any
	OAuthCredentialSchema  []map[string]any
	SubscriptionParameters []map[string]any
	SubscriptionSchema     []map[string]any
	Events                 []map[string]any
	SystemOAuth            bool
}

func (s *server) mountWorkspaceExtensionRoutes(r chi.Router) {
	r.Post("/workspaces/current/endpoints/create", s.handleWorkspaceEndpointCreate)
	r.Get("/workspaces/current/endpoints/list", s.handleWorkspaceEndpointList)
	r.Get("/workspaces/current/endpoints/list/plugin", s.handleWorkspaceEndpointListByPlugin)
	r.Post("/workspaces/current/endpoints/delete", s.handleWorkspaceEndpointDelete)
	r.Post("/workspaces/current/endpoints/update", s.handleWorkspaceEndpointUpdate)
	r.Post("/workspaces/current/endpoints/enable", s.handleWorkspaceEndpointEnable)
	r.Post("/workspaces/current/endpoints/disable", s.handleWorkspaceEndpointDisable)

	r.Get("/workspaces/current/triggers", s.handleTriggerProviderList)
	r.Get("/workspaces/current/trigger-provider/*", s.handleTriggerProviderGet)
	r.Post("/workspaces/current/trigger-provider/*", s.handleTriggerProviderPost)
	r.Delete("/workspaces/current/trigger-provider/*", s.handleTriggerProviderDelete)
}

func (s *server) triggerRoutes() http.Handler {
	r := chi.NewRouter()
	r.HandleFunc("/builders/{builderID}", s.handleTriggerBuilderIngress)
	r.HandleFunc("/subscriptions/{subscriptionID}", s.handleTriggerSubscriptionIngress)
	r.HandleFunc("/endpoints/{hookID}", s.handleWorkspaceEndpointIngress)
	r.HandleFunc("/endpoints/{hookID}/*", s.handleWorkspaceEndpointIngress)
	r.NotFound(s.compatFallback)
	return r
}

func (s *server) handleWorkspaceEndpointCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginUniqueIdentifier string         `json:"plugin_unique_identifier"`
		Settings               map[string]any `json:"settings"`
		Name                   string         `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	pluginID := pluginIDFromUniqueIdentifier(payload.PluginUniqueIdentifier)
	if pluginID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifier is required.")
		return
	}

	if _, err := s.store.CreateWorkspaceEndpoint(workspace.ID, currentUser(r), state.CreateWorkspaceEndpointInput{
		PluginID:               pluginID,
		PluginUniqueIdentifier: strings.TrimSpace(payload.PluginUniqueIdentifier),
		Name:                   strings.TrimSpace(payload.Name),
		Settings:               payload.Settings,
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handleWorkspaceEndpointList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	page, pageSize := pagingFromRequest(r)
	items := s.store.ListWorkspaceEndpoints(workspace.ID)
	writeJSON(w, http.StatusOK, s.endpointListPayload(r, workspace.ID, items, page, pageSize))
}

func (s *server) handleWorkspaceEndpointListByPlugin(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	pluginID := strings.TrimSpace(r.URL.Query().Get("plugin_id"))
	page, pageSize := pagingFromRequest(r)
	items := s.store.ListWorkspaceEndpointsByPlugin(workspace.ID, pluginID)
	writeJSON(w, http.StatusOK, s.endpointListPayload(r, workspace.ID, items, page, pageSize))
}

func (s *server) handleWorkspaceEndpointDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		EndpointID string `json:"endpoint_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.DeleteWorkspaceEndpoint(workspace.ID, payload.EndpointID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handleWorkspaceEndpointUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		EndpointID string         `json:"endpoint_id"`
		Settings   map[string]any `json:"settings"`
		Name       string         `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateWorkspaceEndpoint(workspace.ID, currentUser(r), state.UpdateWorkspaceEndpointInput{
		EndpointID: payload.EndpointID,
		Name:       payload.Name,
		Settings:   payload.Settings,
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handleWorkspaceEndpointEnable(w http.ResponseWriter, r *http.Request) {
	s.handleWorkspaceEndpointEnabled(w, r, true)
}

func (s *server) handleWorkspaceEndpointDisable(w http.ResponseWriter, r *http.Request) {
	s.handleWorkspaceEndpointEnabled(w, r, false)
}

func (s *server) handleWorkspaceEndpointEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		EndpointID string `json:"endpoint_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.SetWorkspaceEndpointEnabled(workspace.ID, payload.EndpointID, enabled, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handleTriggerProviderList(w http.ResponseWriter, r *http.Request) {
	items := make([]map[string]any, 0, len(triggerProviderCatalogs(s.cfg.AppVersion)))
	for _, catalog := range triggerProviderCatalogs(s.cfg.AppVersion) {
		items = append(items, s.triggerCatalogPayload(catalog))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *server) handleTriggerProviderGet(w http.ResponseWriter, r *http.Request) {
	raw := strings.Trim(strings.TrimSpace(chi.URLParam(r, "*")), "/")
	switch {
	case strings.HasSuffix(raw, "/info"):
		s.handleTriggerProviderInfo(w, r, strings.TrimSuffix(raw, "/info"))
	case strings.HasSuffix(raw, "/subscriptions/list"):
		s.handleTriggerSubscriptionList(w, r, strings.TrimSuffix(raw, "/subscriptions/list"))
	case strings.HasSuffix(raw, "/oauth/client"):
		s.handleTriggerOAuthConfig(w, r, strings.TrimSuffix(raw, "/oauth/client"))
	case strings.HasSuffix(raw, "/subscriptions/oauth/authorize"):
		s.handleTriggerOAuthAuthorize(w, r, strings.TrimSuffix(raw, "/subscriptions/oauth/authorize"))
	case strings.HasSuffix(raw, "/icon"):
		s.handleTriggerProviderIcon(w, r, strings.TrimSuffix(raw, "/icon"))
	default:
		if provider, builderID, ok := splitPathSuffix(raw, "/subscriptions/builder/logs/"); ok {
			s.handleTriggerSubscriptionBuilderLogs(w, r, provider, builderID)
			return
		}
		if provider, builderID, ok := splitPathSuffix(raw, "/subscriptions/builder/"); ok {
			s.handleTriggerSubscriptionBuilderGet(w, r, provider, builderID)
			return
		}
		s.compatFallback(w, r)
	}
}

func (s *server) handleTriggerProviderPost(w http.ResponseWriter, r *http.Request) {
	raw := strings.Trim(strings.TrimSpace(chi.URLParam(r, "*")), "/")
	switch {
	case strings.HasSuffix(raw, "/subscriptions/builder/create"):
		s.handleTriggerSubscriptionBuilderCreate(w, r, strings.TrimSuffix(raw, "/subscriptions/builder/create"))
	case strings.HasSuffix(raw, "/oauth/client"):
		s.handleTriggerOAuthConfigure(w, r, strings.TrimSuffix(raw, "/oauth/client"))
	case strings.HasSuffix(raw, "/subscriptions/delete"):
		s.handleTriggerSubscriptionDelete(w, r, strings.TrimSuffix(raw, "/subscriptions/delete"))
	case strings.HasSuffix(raw, "/subscriptions/update"):
		s.handleTriggerSubscriptionUpdate(w, r, strings.TrimSuffix(raw, "/subscriptions/update"))
	default:
		if provider, builderID, ok := splitPathSuffix(raw, "/subscriptions/builder/update/"); ok {
			s.handleTriggerSubscriptionBuilderUpdate(w, r, provider, builderID)
			return
		}
		if provider, builderID, ok := splitPathSuffix(raw, "/subscriptions/builder/verify-and-update/"); ok {
			s.handleTriggerSubscriptionBuilderVerifyUpdate(w, r, provider, builderID)
			return
		}
		if provider, builderID, ok := splitPathSuffix(raw, "/subscriptions/builder/build/"); ok {
			s.handleTriggerSubscriptionBuilderBuild(w, r, provider, builderID)
			return
		}
		if provider, subscriptionID, ok := splitPathSuffix(raw, "/subscriptions/verify/"); ok {
			s.handleTriggerSubscriptionVerify(w, r, provider, subscriptionID)
			return
		}
		s.compatFallback(w, r)
	}
}

func (s *server) handleTriggerProviderDelete(w http.ResponseWriter, r *http.Request) {
	raw := strings.Trim(strings.TrimSpace(chi.URLParam(r, "*")), "/")
	if strings.HasSuffix(raw, "/oauth/client") {
		s.handleTriggerOAuthDelete(w, r, strings.TrimSuffix(raw, "/oauth/client"))
		return
	}
	s.compatFallback(w, r)
}

func (s *server) handleTriggerProviderInfo(w http.ResponseWriter, r *http.Request, provider string) {
	catalog, ok := triggerCatalogByProvider(provider, s.cfg.AppVersion)
	if !ok {
		catalog = fallbackTriggerCatalog(normalizeTriggerProviderParam(provider), s.cfg.AppVersion)
	}
	writeJSON(w, http.StatusOK, s.triggerCatalogPayload(catalog))
}

func (s *server) handleTriggerProviderIcon(w http.ResponseWriter, r *http.Request, provider string) {
	catalog, ok := triggerCatalogByProvider(provider, s.cfg.AppVersion)
	if !ok {
		writeError(w, http.StatusNotFound, "provider_not_found", "Trigger provider not found.")
		return
	}
	if strings.HasPrefix(catalog.Icon, "data:image/svg+xml;base64,") {
		encoded := strings.TrimPrefix(catalog.Icon, "data:image/svg+xml;base64,")
		data, err := io.ReadAll(base64Reader(encoded))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "icon_decode_failed", "Failed to decode icon.")
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}
	http.Redirect(w, r, catalog.Icon, http.StatusFound)
}

func (s *server) handleTriggerSubscriptionList(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	items := s.store.ListWorkspaceTriggerSubscriptions(workspace.ID, normalizeTriggerProviderParam(provider))
	payload := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payload = append(payload, triggerSubscriptionPayload(item))
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *server) handleTriggerSubscriptionBuilderCreate(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialType string `json:"credential_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	normalizedProvider := normalizeTriggerProviderParam(provider)
	builder, err := s.store.CreateWorkspaceTriggerSubscriptionBuilder(
		workspace.ID,
		currentUser(r),
		normalizedProvider,
		firstNonEmpty(payload.CredentialType, state.TriggerCredentialTypeUnauthorized),
		"",
		nil,
		false,
		time.Now(),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/trigger/builders/%s", requestBaseURL(r), builder.ID)
	updated, err := s.store.UpdateWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizedProvider, builder.ID, state.UpdateWorkspaceTriggerSubscriptionBuilderInput{
		Endpoint: &endpoint,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "builder_update_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subscription_builder": triggerSubscriptionBuilderPayload(updated),
	})
}

func (s *server) handleTriggerSubscriptionBuilderGet(w http.ResponseWriter, r *http.Request, provider, builderID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	builder, found := s.store.GetWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID)
	if !found {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}
	writeJSON(w, http.StatusOK, triggerSubscriptionBuilderPayload(builder))
}

func (s *server) handleTriggerSubscriptionBuilderUpdate(w http.ResponseWriter, r *http.Request, provider, builderID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Name        *string        `json:"name"`
		Properties  map[string]any `json:"properties"`
		Parameters  map[string]any `json:"parameters"`
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	builder, found := s.store.GetWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID)
	if !found {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}

	credentials := builder.Credentials
	if payload.Credentials != nil {
		credentials = mergeSecretMap(builder.Credentials, payload.Credentials)
	}

	updated, err := s.store.UpdateWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID, state.UpdateWorkspaceTriggerSubscriptionBuilderInput{
		Name:        payload.Name,
		Credentials: credentials,
		Parameters:  payload.Parameters,
		Properties:  payload.Properties,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, triggerSubscriptionBuilderPayload(updated))
}

func (s *server) handleTriggerSubscriptionBuilderVerifyUpdate(w http.ResponseWriter, r *http.Request, provider, builderID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	builder, found := s.store.GetWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID)
	if !found {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}

	credentials := builder.Credentials
	if payload.Credentials != nil {
		credentials = mergeSecretMap(builder.Credentials, payload.Credentials)
	}
	verified := builder.Verified || hasMeaningfulCredentials(credentials)
	if !verified {
		writeError(w, http.StatusBadRequest, "verification_failed", "Please fill in all required credentials before verifying.")
		return
	}

	if _, err := s.store.UpdateWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID, state.UpdateWorkspaceTriggerSubscriptionBuilderInput{
		Credentials: credentials,
		Verified:    boolPtr(true),
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (s *server) handleTriggerSubscriptionVerify(w http.ResponseWriter, r *http.Request, provider, subscriptionID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	subscription, found := s.store.GetWorkspaceTriggerSubscription(workspace.ID, subscriptionID)
	if !found || subscription.Provider != normalizeTriggerProviderParam(provider) {
		writeError(w, http.StatusNotFound, "subscription_not_found", "Subscription not found.")
		return
	}

	credentials := mergeSecretMap(subscription.Credentials, payload.Credentials)
	if !hasMeaningfulCredentials(credentials) {
		writeError(w, http.StatusBadRequest, "verification_failed", "Please fill in all required credentials before verifying.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (s *server) handleTriggerSubscriptionBuilderBuild(w http.ResponseWriter, r *http.Request, provider, builderID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Name       string         `json:"name"`
		Parameters map[string]any `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	normalizedProvider := normalizeTriggerProviderParam(provider)
	builder, found := s.store.GetWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizedProvider, builderID)
	if !found {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}

	parameters := builder.Parameters
	if payload.Parameters != nil {
		parameters = cloneJSONObject(payload.Parameters)
	}
	name := firstNonEmpty(strings.TrimSpace(payload.Name), builder.Name)

	subscription, err := s.store.CreateWorkspaceTriggerSubscription(workspace.ID, currentUser(r), state.CreateWorkspaceTriggerSubscriptionInput{
		Name:           name,
		Provider:       normalizedProvider,
		CredentialType: builder.CredentialType,
		Credentials:    builder.Credentials,
		Endpoint:       "",
		Parameters:     parameters,
		Properties:     builder.Properties,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/trigger/subscriptions/%s", requestBaseURL(r), subscription.ID)
	subscription, err = s.store.UpdateWorkspaceTriggerSubscription(workspace.ID, subscription.ID, state.UpdateWorkspaceTriggerSubscriptionInput{
		Endpoint: &endpoint,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "subscription_update_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"id":     subscription.ID,
	})
}

func (s *server) handleTriggerSubscriptionDelete(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if err := s.store.DeleteWorkspaceTriggerSubscription(workspace.ID, subscriptionID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleTriggerSubscriptionUpdate(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Name        *string        `json:"name"`
		Properties  map[string]any `json:"properties"`
		Parameters  map[string]any `json:"parameters"`
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	subscription, found := s.store.GetWorkspaceTriggerSubscription(workspace.ID, subscriptionID)
	if !found {
		writeError(w, http.StatusNotFound, "subscription_not_found", "Subscription not found.")
		return
	}

	credentials := subscription.Credentials
	if payload.Credentials != nil {
		credentials = mergeSecretMap(subscription.Credentials, payload.Credentials)
	}

	updated, err := s.store.UpdateWorkspaceTriggerSubscription(workspace.ID, subscriptionID, state.UpdateWorkspaceTriggerSubscriptionInput{
		Name:        payload.Name,
		Credentials: credentials,
		Parameters:  payload.Parameters,
		Properties:  payload.Properties,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": "success",
		"id":     updated.ID,
	})
}

func (s *server) handleTriggerSubscriptionBuilderLogs(w http.ResponseWriter, r *http.Request, provider, builderID string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	builder, found := s.store.GetWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizeTriggerProviderParam(provider), builderID)
	if !found {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}

	logs := make([]map[string]any, 0, len(builder.Logs))
	for _, item := range builder.Logs {
		logs = append(logs, triggerLogPayload(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (s *server) handleTriggerOAuthConfig(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	normalizedProvider := normalizeTriggerProviderParam(provider)
	catalog, found := triggerCatalogByProvider(normalizedProvider, s.cfg.AppVersion)
	if !found {
		catalog = fallbackTriggerCatalog(normalizedProvider, s.cfg.AppVersion)
	}

	stateValue, _ := s.store.GetWorkspaceTriggerProviderState(workspace.ID, normalizedProvider)
	redirectURI := fmt.Sprintf("%s/oauth-callback", frontendOriginURL(r))
	customConfigured := len(stateValue.OAuthClient.Params) > 0

	writeJSON(w, http.StatusOK, map[string]any{
		"configured":          customConfigured || catalog.SystemOAuth,
		"system_configured":   catalog.SystemOAuth,
		"custom_configured":   customConfigured,
		"custom_enabled":      stateValue.OAuthClient.Enabled,
		"redirect_uri":        redirectURI,
		"oauth_client_schema": catalog.OAuthClientSchema,
		"params":              stateValue.OAuthClient.Params,
	})
}

func (s *server) handleTriggerOAuthConfigure(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	normalizedProvider := normalizeTriggerProviderParam(provider)
	currentState, _ := s.store.GetWorkspaceTriggerProviderState(workspace.ID, normalizedProvider)

	var payload struct {
		ClientParams map[string]any `json:"client_params"`
		Enabled      bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	params := mergeSecretStringMap(currentState.OAuthClient.Params, stringifyMap(payload.ClientParams))
	if _, err := s.store.UpsertWorkspaceTriggerOAuthClient(workspace.ID, normalizedProvider, params, payload.Enabled, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleTriggerOAuthDelete(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if _, err := s.store.DeleteWorkspaceTriggerOAuthClient(workspace.ID, normalizeTriggerProviderParam(provider), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleTriggerOAuthAuthorize(w http.ResponseWriter, r *http.Request, provider string) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	normalizedProvider := normalizeTriggerProviderParam(provider)
	builder, err := s.store.CreateWorkspaceTriggerSubscriptionBuilder(
		workspace.ID,
		currentUser(r),
		normalizedProvider,
		state.TriggerCredentialTypeOAuth2,
		"",
		map[string]any{
			"access_token":  "oauth_" + strings.ReplaceAll(generateOAuthTokenSeed(), "-", ""),
			"refresh_token": "refresh_" + strings.ReplaceAll(generateOAuthTokenSeed(), "-", ""),
		},
		true,
		time.Now(),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/trigger/builders/%s", requestBaseURL(r), builder.ID)
	builder, err = s.store.UpdateWorkspaceTriggerSubscriptionBuilder(workspace.ID, normalizedProvider, builder.ID, state.UpdateWorkspaceTriggerSubscriptionBuilderInput{
		Endpoint: &endpoint,
		Verified: boolPtr(true),
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "builder_update_failed", err.Error())
		return
	}

	callbackURL := fmt.Sprintf("%s/oauth-callback?subscription_id=%s", frontendOriginURL(r), neturl.QueryEscape(builder.ID))
	writeJSON(w, http.StatusOK, map[string]any{
		"authorization_url":       callbackURL,
		"subscription_builder_id": builder.ID,
		"subscription_builder":    triggerSubscriptionBuilderPayload(builder),
	})
}

func (s *server) handleTriggerBuilderIngress(w http.ResponseWriter, r *http.Request) {
	builderID := strings.TrimSpace(chi.URLParam(r, "builderID"))
	workspace, builder, ok := s.store.FindWorkspaceTriggerSubscriptionBuilderByID(builderID)
	if !ok {
		writeError(w, http.StatusNotFound, "builder_not_found", "Subscription builder not found.")
		return
	}

	response := map[string]any{
		"result":                  "received",
		"subscription_builder_id": builder.ID,
		"provider":                builder.Provider,
	}
	body := readLimitedBody(r)
	if err := s.store.AppendWorkspaceTriggerBuilderLog(workspace.ID, builder.Provider, builder.ID, buildTriggerLog(r, builder.Endpoint, body, http.StatusOK, response), time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "log_write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleTriggerSubscriptionIngress(w http.ResponseWriter, r *http.Request) {
	subscriptionID := strings.TrimSpace(chi.URLParam(r, "subscriptionID"))
	workspace, subscription, ok := s.store.FindWorkspaceTriggerSubscriptionByID(subscriptionID)
	if !ok {
		writeError(w, http.StatusNotFound, "subscription_not_found", "Subscription not found.")
		return
	}

	response := map[string]any{
		"result":          "received",
		"subscription_id": subscription.ID,
		"provider":        subscription.Provider,
	}
	body := readLimitedBody(r)
	if err := s.store.AppendWorkspaceTriggerSubscriptionLog(workspace.ID, subscription.ID, buildTriggerLog(r, subscription.Endpoint, body, http.StatusOK, response), time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "log_write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleWorkspaceEndpointIngress(w http.ResponseWriter, r *http.Request) {
	hookID := strings.TrimSpace(chi.URLParam(r, "hookID"))
	workspace, endpoint, ok := s.store.FindWorkspaceEndpointByHookID(hookID)
	if !ok || !endpoint.Enabled {
		writeError(w, http.StatusNotFound, "endpoint_not_found", "Endpoint not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":       "received",
		"workspace_id": workspace.ID,
		"endpoint_id":  endpoint.ID,
		"plugin_id":    endpoint.PluginID,
		"settings":     endpoint.Settings,
		"path":         strings.TrimPrefix(r.URL.Path, "/trigger/endpoints/"+hookID),
	})
}

func (s *server) endpointListPayload(r *http.Request, workspaceID string, items []state.WorkspaceEndpoint, page, pageSize int) map[string]any {
	total := len(items)
	start, end := pageBounds(page, pageSize, total)
	selected := items[start:end]

	payload := make([]map[string]any, 0, len(selected))
	for _, item := range selected {
		payload = append(payload, s.endpointPayload(r, workspaceID, item))
	}

	return map[string]any{
		"endpoints": payload,
		"has_more":  end < total,
		"limit":     pageSize,
		"total":     total,
		"page":      page,
	}
}

func (s *server) endpointPayload(r *http.Request, workspaceID string, endpoint state.WorkspaceEndpoint) map[string]any {
	return map[string]any{
		"id":         endpoint.ID,
		"created_at": unixToRFC3339(endpoint.CreatedAt),
		"updated_at": unixToRFC3339(endpoint.UpdatedAt),
		"settings":   endpoint.Settings,
		"tenant_id":  workspaceID,
		"plugin_id":  endpoint.PluginID,
		"expired_at": "",
		"declaration": map[string]any{
			"settings":  endpointSettingsSchema(endpoint.Settings),
			"endpoints": []map[string]any{{"path": "", "method": "POST"}},
		},
		"name":    endpoint.Name,
		"enabled": endpoint.Enabled,
		"url":     fmt.Sprintf("%s/trigger/endpoints/%s", requestBaseURL(r), endpoint.HookID),
		"hook_id": endpoint.HookID,
	}
}

func endpointSettingsSchema(settings map[string]any) []map[string]any {
	if len(settings) == 0 {
		return []map[string]any{}
	}
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		value := settings[key]
		schemaType := "text-input"
		switch value.(type) {
		case bool:
			schemaType = "boolean"
		case int, int32, int64, float32, float64:
			schemaType = "text-number"
		}
		items = append(items, map[string]any{
			"name":        key,
			"label":       localizedText(humanizeIdentifier(key)),
			"placeholder": localizedText(humanizeIdentifier(key)),
			"type":        schemaType,
			"required":    false,
			"default":     value,
		})
	}
	return items
}

func (s *server) triggerCatalogPayload(catalog triggerProviderCatalog) map[string]any {
	return map[string]any{
		"author":                     catalog.Author,
		"name":                       catalog.Provider,
		"label":                      localizedText(catalog.Label),
		"description":                localizedText(catalog.Description),
		"icon":                       catalog.Icon,
		"icon_dark":                  catalog.IconDark,
		"tags":                       catalog.Tags,
		"plugin_id":                  catalog.PluginID,
		"plugin_unique_identifier":   catalog.PluginUniqueIdentifier,
		"supported_creation_methods": catalog.SupportedMethods,
		"credentials_schema":         catalog.CredentialsSchema,
		"subscription_constructor": map[string]any{
			"credentials_schema": catalog.CredentialsSchema,
			"oauth_schema": map[string]any{
				"client_schema":      catalog.OAuthClientSchema,
				"credentials_schema": catalog.OAuthCredentialSchema,
			},
			"parameters": catalog.SubscriptionParameters,
		},
		"subscription_schema": catalog.SubscriptionSchema,
		"events":              catalog.Events,
	}
}

func triggerSubscriptionPayload(item state.WorkspaceTriggerSubscription) map[string]any {
	return map[string]any{
		"id":               item.ID,
		"name":             item.Name,
		"provider":         item.Provider,
		"credential_type":  item.CredentialType,
		"credentials":      maskSecrets(item.Credentials),
		"endpoint":         item.Endpoint,
		"parameters":       item.Parameters,
		"properties":       item.Properties,
		"workflows_in_use": item.WorkflowsInUse,
	}
}

func triggerSubscriptionBuilderPayload(item state.WorkspaceTriggerSubscriptionBuilder) map[string]any {
	return map[string]any{
		"id":               item.ID,
		"name":             item.Name,
		"provider":         item.Provider,
		"credential_type":  item.CredentialType,
		"credentials":      maskSecrets(item.Credentials),
		"endpoint":         item.Endpoint,
		"parameters":       item.Parameters,
		"properties":       item.Properties,
		"workflows_in_use": 0,
	}
}

func triggerLogPayload(item state.WorkspaceTriggerLog) map[string]any {
	return map[string]any{
		"id":         item.ID,
		"endpoint":   item.Endpoint,
		"created_at": unixToRFC3339(item.CreatedAt),
		"request": map[string]any{
			"method":  item.Request.Method,
			"url":     item.Request.URL,
			"headers": item.Request.Headers,
			"data":    item.Request.Data,
		},
		"response": map[string]any{
			"status_code": item.Response.StatusCode,
			"headers":     item.Response.Headers,
			"data":        item.Response.Data,
		},
	}
}

func triggerProviderCatalogs(version string) []triggerProviderCatalog {
	return []triggerProviderCatalog{
		{
			Provider:               "langgenius/github/github",
			PluginID:               "langgenius/github",
			PluginUniqueIdentifier: "langgenius/github@" + version,
			Author:                 "langgenius",
			Label:                  "GitHub",
			Description:            "Subscribe workflows to GitHub repository activity with OAuth, API keys, or manual webhooks.",
			Icon:                   providerIconDataURI("G", "#111827", "#FFFFFF"),
			IconDark:               providerIconDataURI("G", "#FFFFFF", "#111827"),
			Tags:                   []string{"code", "git", "webhook"},
			SupportedMethods:       []string{"OAUTH", "APIKEY", "MANUAL"},
			CredentialsSchema: []map[string]any{
				triggerCredentialSchema("token", "API Token", "secret-input", true, "", "Personal access token for webhook verification."),
			},
			OAuthClientSchema: []map[string]any{
				triggerCredentialSchema("client_id", "Client ID", "text-input", true, "", ""),
				triggerCredentialSchema("client_secret", "Client Secret", "secret-input", true, "", ""),
				triggerCredentialSchema("authorization_url", "Authorization URL", "text-input", false, "https://github.com/login/oauth/authorize", ""),
				triggerCredentialSchema("token_url", "Token URL", "text-input", false, "https://github.com/login/oauth/access_token", ""),
				triggerCredentialSchema("scope", "Scope", "text-input", false, "repo read:user", ""),
			},
			OAuthCredentialSchema: []map[string]any{
				triggerCredentialSchema("access_token", "Access Token", "secret-input", true, "", ""),
				triggerCredentialSchema("refresh_token", "Refresh Token", "secret-input", false, "", ""),
			},
			SubscriptionParameters: []map[string]any{
				triggerSelectSchema("event_types", "Event Types", []triggerOption{
					{Value: "push", Label: "Push"},
					{Value: "issues", Label: "Issues"},
					{Value: "issue_comment", Label: "Issue Comment"},
				}, true, true),
				triggerCredentialSchema("repository", "Repository", "text-input", false, "", "owner/repo"),
			},
			SubscriptionSchema: []map[string]any{
				triggerCredentialSchema("webhook_secret", "Webhook Secret", "secret-input", false, "", "Optional secret used by the sender."),
				triggerSelectSchema("event_types", "Event Types", []triggerOption{
					{Value: "push", Label: "Push"},
					{Value: "issues", Label: "Issues"},
					{Value: "issue_comment", Label: "Issue Comment"},
				}, true, true),
			},
			Events: []map[string]any{
				triggerEvent("push", "Push", "Repository push event", []map[string]any{
					triggerCredentialSchema("branch", "Branch", "text-input", false, "", ""),
				}),
				triggerEvent("issues", "Issues", "Issue lifecycle event", []map[string]any{
					triggerSelectSchema("action", "Action", []triggerOption{
						{Value: "opened", Label: "Opened"},
						{Value: "closed", Label: "Closed"},
						{Value: "edited", Label: "Edited"},
					}, false, false),
				}),
				triggerEvent("issue_comment", "Issue Comment", "Issue comment event", []map[string]any{
					triggerCredentialSchema("commenter", "Commenter", "text-input", false, "", ""),
				}),
			},
			SystemOAuth: true,
		},
		{
			Provider:               "langgenius/http/http",
			PluginID:               "langgenius/http",
			PluginUniqueIdentifier: "langgenius/http@" + version,
			Author:                 "langgenius",
			Label:                  "HTTP Webhook",
			Description:            "Accept raw inbound HTTP events with manual configuration.",
			Icon:                   providerIconDataURI("H", "#DBEAFE", "#1D4ED8"),
			IconDark:               providerIconDataURI("H", "#1D4ED8", "#DBEAFE"),
			Tags:                   []string{"http", "webhook"},
			SupportedMethods:       []string{"MANUAL"},
			CredentialsSchema:      []map[string]any{},
			OAuthClientSchema:      []map[string]any{},
			OAuthCredentialSchema:  []map[string]any{},
			SubscriptionParameters: []map[string]any{},
			SubscriptionSchema: []map[string]any{
				triggerCredentialSchema("secret", "Secret", "secret-input", false, "", "Optional signing secret."),
				triggerSelectSchema("method", "Method", []triggerOption{
					{Value: "POST", Label: "POST"},
					{Value: "PUT", Label: "PUT"},
					{Value: "PATCH", Label: "PATCH"},
				}, false, false),
			},
			Events: []map[string]any{
				triggerEvent("incoming_request", "Incoming Request", "Any inbound webhook request.", []map[string]any{}),
			},
			SystemOAuth: false,
		},
	}
}

func triggerCatalogByProvider(provider, version string) (triggerProviderCatalog, bool) {
	normalized := normalizeTriggerProviderParam(provider)
	for _, item := range triggerProviderCatalogs(version) {
		if item.Provider == normalized {
			return item, true
		}
	}
	return triggerProviderCatalog{}, false
}

func fallbackTriggerCatalog(provider, version string) triggerProviderCatalog {
	pluginID := triggerPluginID(provider)
	name := triggerProviderName(provider)
	return triggerProviderCatalog{
		Provider:               provider,
		PluginID:               pluginID,
		PluginUniqueIdentifier: pluginID + "@" + version,
		Author:                 "dify-go",
		Label:                  humanizeIdentifier(name),
		Description:            "Generic trigger provider migrated by dify-go.",
		Icon:                   providerIconDataURI(strings.ToUpper(firstNonEmpty(name, "T"))[:1], "#E5E7EB", "#111827"),
		IconDark:               providerIconDataURI(strings.ToUpper(firstNonEmpty(name, "T"))[:1], "#111827", "#E5E7EB"),
		Tags:                   []string{"trigger"},
		SupportedMethods:       []string{"MANUAL"},
		CredentialsSchema:      []map[string]any{},
		OAuthClientSchema:      []map[string]any{},
		OAuthCredentialSchema:  []map[string]any{},
		SubscriptionParameters: []map[string]any{},
		SubscriptionSchema: []map[string]any{
			triggerCredentialSchema("secret", "Secret", "secret-input", false, "", "Optional shared secret."),
		},
		Events: []map[string]any{
			triggerEvent("incoming_request", "Incoming Request", "Generic inbound webhook event.", []map[string]any{}),
		},
		SystemOAuth: false,
	}
}

type triggerOption struct {
	Value string
	Label string
}

func triggerCredentialSchema(name, label, kind string, required bool, defaultValue any, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"label":       localizedText(label),
		"description": localizedText(description),
		"type":        kind,
		"required":    required,
		"default":     defaultValue,
		"multiple":    false,
	}
}

func triggerSelectSchema(name, label string, options []triggerOption, required, multiple bool) map[string]any {
	items := make([]map[string]any, 0, len(options))
	for _, option := range options {
		items = append(items, map[string]any{
			"value": option.Value,
			"label": localizedText(option.Label),
		})
	}
	return map[string]any{
		"name":        name,
		"label":       localizedText(label),
		"description": localizedText(label),
		"type":        "select",
		"required":    required,
		"default":     "",
		"multiple":    multiple,
		"options":     items,
	}
}

func triggerEvent(name, label, description string, parameters []map[string]any) map[string]any {
	return map[string]any{
		"name": name,
		"identity": map[string]any{
			"author":   "langgenius",
			"name":     name,
			"label":    localizedText(label),
			"provider": "",
		},
		"description": localizedText(description),
		"parameters":  parameters,
		"output_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event":   map[string]any{"type": "string"},
				"payload": map[string]any{"type": "object"},
			},
		},
	}
}

func normalizeTriggerProviderParam(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	switch len(parts) {
	case 1:
		return "langgenius/" + value + "/" + value
	case 2:
		return value + "/" + parts[1]
	default:
		return value
	}
}

func triggerPluginID(provider string) string {
	parts := strings.Split(normalizeTriggerProviderParam(provider), "/")
	if len(parts) < 2 {
		return strings.TrimSpace(provider)
	}
	return parts[0] + "/" + parts[1]
}

func triggerProviderName(provider string) string {
	parts := strings.Split(normalizeTriggerProviderParam(provider), "/")
	if len(parts) == 0 {
		return strings.TrimSpace(provider)
	}
	return parts[len(parts)-1]
}

func pagingFromRequest(r *http.Request) (int, int) {
	page := 1
	pageSize := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		fmt.Sscanf(raw, "%d", &page)
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		fmt.Sscanf(raw, "%d", &pageSize)
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	return page, pageSize
}

func pageBounds(page, pageSize, total int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func splitPathSuffix(value, marker string) (string, string, bool) {
	index := strings.LastIndex(value, marker)
	if index < 0 {
		return "", "", false
	}
	left := strings.Trim(strings.TrimSpace(value[:index]), "/")
	right := strings.Trim(strings.TrimSpace(value[index+len(marker):]), "/")
	if left == "" || right == "" {
		return "", "", false
	}
	return left, right, true
}

func pluginIDFromUniqueIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.LastIndex(value, "@"); index > 0 {
		return value[:index]
	}
	if index := strings.LastIndex(value, ":"); index > 0 && strings.Contains(value[:index], "/") {
		return value[:index]
	}
	return value
}

func humanizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == '/' || r == '.'
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func unixToRFC3339(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func frontendOriginURL(r *http.Request) string {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return strings.TrimRight(origin, "/")
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		if parsed, err := neturl.Parse(referer); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return strings.TrimRight(parsed.Scheme+"://"+parsed.Host, "/")
		}
	}
	return requestBaseURL(r)
}

func readLimitedBody(r *http.Request) string {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))
	return string(body)
}

func buildTriggerLog(r *http.Request, endpoint, body string, statusCode int, response any) state.WorkspaceTriggerLog {
	data, _ := json.Marshal(response)
	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}
	return state.WorkspaceTriggerLog{
		ID:       generateRuntimeID("trlog"),
		Endpoint: endpoint,
		Request: state.WorkspaceTriggerLogRequest{
			Method:  r.Method,
			URL:     requestBaseURL(r) + r.URL.RequestURI(),
			Headers: headers,
			Data:    body,
		},
		Response: state.WorkspaceTriggerLogResponse{
			StatusCode: statusCode,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Data: string(data),
		},
		CreatedAt: time.Now().UTC().Unix(),
	}
}

func mergeSecretMap(existing, updates map[string]any) map[string]any {
	if updates == nil {
		return cloneJSONObject(existing)
	}
	out := cloneJSONObject(existing)
	for key, value := range updates {
		if stringValueAny(value, "") == hiddenSecretValue {
			continue
		}
		out[key] = value
	}
	return out
}

func mergeSecretStringMap(existing, updates map[string]string) map[string]string {
	if updates == nil {
		return cloneStringMapLocal(existing)
	}
	out := cloneStringMapLocal(existing)
	for key, value := range updates {
		if strings.TrimSpace(value) == hiddenSecretValue {
			continue
		}
		out[key] = value
	}
	return out
}

func stringifyMap(value map[string]any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(value))
	for key, item := range value {
		out[key] = stringValueAny(item, "")
	}
	return out
}

func cloneStringMapLocal(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func maskSecrets(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				out[key] = ""
			} else {
				out[key] = hiddenSecretValue
			}
		default:
			out[key] = value
		}
	}
	return out
}

func hasMeaningfulCredentials(value map[string]any) bool {
	for _, item := range value {
		switch typed := item.(type) {
		case string:
			if strings.TrimSpace(typed) != "" && strings.TrimSpace(typed) != hiddenSecretValue {
				return true
			}
		case nil:
		default:
			return true
		}
	}
	return false
}

func base64Reader(encoded string) io.Reader {
	return io.LimitReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded)), 1<<20)
}

func generateOAuthTokenSeed() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

func generateRuntimeID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
