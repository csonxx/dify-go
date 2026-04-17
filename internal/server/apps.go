package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) mountAppRoutes(r chi.Router) {
	r.Get("/apps", s.handleAppList)
	r.Post("/apps", s.handleAppCreate)
	r.Post("/apps/imports", s.handleAppImport)
	r.Post("/apps/imports/{importID}/confirm", s.handleAppImportConfirm)
	r.Get("/apps/workflows/online-users", s.handleWorkflowOnlineUsers)
	r.Get("/apps/{appID}", s.handleAppDetail)
	r.Put("/apps/{appID}", s.handleAppUpdate)
	r.Delete("/apps/{appID}", s.handleAppDelete)
	r.Post("/apps/{appID}/copy", s.handleAppCopy)
	r.Get("/apps/{appID}/export", s.handleAppExport)
	r.Get("/apps/{appID}/api-keys", s.handleAppAPIKeyList)
	r.Post("/apps/{appID}/api-keys", s.handleAppAPIKeyCreate)
	r.Delete("/apps/{appID}/api-keys/{apiKeyID}", s.handleAppAPIKeyDelete)
	r.Post("/apps/{appID}/site-enable", s.handleAppSiteEnable)
	r.Post("/apps/{appID}/api-enable", s.handleAppAPIEnable)
	r.Post("/apps/{appID}/site", s.handleAppSiteUpdate)
	r.Post("/apps/{appID}/site/access-token-reset", s.handleAppSiteAccessTokenReset)
	r.Post("/apps/{appID}/convert-to-workflow", s.handleAppConvertToWorkflow)
	r.Get("/apps/{appID}/trace", s.handleAppTraceStatus)
	r.Post("/apps/{appID}/trace", s.handleAppTraceUpdate)
	r.Get("/apps/{appID}/trace-config", s.handleAppTraceConfigGet)
	r.Post("/apps/{appID}/trace-config", s.handleAppTraceConfigSave)
	r.Patch("/apps/{appID}/trace-config", s.handleAppTraceConfigSave)
	r.Delete("/apps/{appID}/trace-config", s.handleAppTraceConfigDelete)
	r.Post("/apps/{appID}/model-config", s.handleAppModelConfigUpdate)
	r.Get("/apps/{appID}/workflows/triggers/webhook", s.handleWorkflowWebhookTrigger)
	r.Get("/apps/{appID}/text-to-audio/voices", s.handleAppVoices)
	r.Get("/apps/{appID}/statistics/{metric}", s.handleAppStatistics)
	r.Get("/apps/{appID}/workflow/statistics/{metric}", s.handleWorkflowStatistics)
	r.Get("/apps/{appID}/workflows/draft", s.handleWorkflowDraftGet)
	r.Post("/apps/{appID}/workflows/draft", s.handleWorkflowDraftSync)
	r.Get("/apps/{appID}/workflows/draft/environment-variables", s.handleWorkflowDraftEnvironmentVariables)
	r.Post("/apps/{appID}/workflows/draft/environment-variables", s.handleWorkflowDraftEnvironmentVariablesUpdate)
	r.Get("/apps/{appID}/workflows/draft/conversation-variables", s.handleWorkflowDraftConversationVariables)
	r.Post("/apps/{appID}/workflows/draft/conversation-variables", s.handleWorkflowDraftConversationVariablesUpdate)
	r.Get("/apps/{appID}/workflows/draft/system-variables", s.handleWorkflowDraftSystemVariables)
	r.Get("/apps/{appID}/workflows/draft/variables", s.handleWorkflowDraftVariables)
	r.Get("/apps/{appID}/workflows/draft/nodes/{nodeID}/variables", s.handleWorkflowDraftNodeVariables)
	r.Post("/apps/{appID}/workflows/draft/features", s.handleWorkflowDraftFeaturesUpdate)
	r.Get("/apps/{appID}/workflows/default-workflow-block-configs", s.handleWorkflowDefaultBlockConfigs)
	r.Get("/apps/{appID}/workflows/default-workflow-block-configs/{blockType}", s.handleWorkflowDefaultBlockConfig)
	r.Get("/apps/{appID}/workflows/publish", s.handleWorkflowPublishedGet)
	r.Post("/apps/{appID}/workflows/publish", s.handleWorkflowPublish)
}

func (s *server) handleAppList(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	page := intQuery(r, "page", 1)
	limit := intQuery(r, "limit", 20)
	appPage := s.store.ListApps(workspace.ID, state.AppListFilters{
		Page:          page,
		Limit:         limit,
		Mode:          strings.TrimSpace(r.URL.Query().Get("mode")),
		Name:          strings.TrimSpace(r.URL.Query().Get("name")),
		IsCreatedByMe: boolQuery(r, "is_created_by_me"),
		CurrentUserID: user.ID,
	})

	data := make([]map[string]any, 0, len(appPage.Data))
	for _, app := range appPage.Data {
		data = append(data, s.appResponse(app))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"page":     appPage.Page,
		"limit":    appPage.Limit,
		"total":    appPage.Total,
		"has_more": appPage.HasMore,
		"data":     data,
	})
}

func (s *server) handleAppCreate(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Mode           string `json:"mode"`
		IconType       string `json:"icon_type"`
		Icon           string `json:"icon"`
		IconBackground string `json:"icon_background"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "App name is required.")
		return
	}

	app, err := s.store.CreateApp(workspace.ID, user, state.CreateAppInput{
		Name:           payload.Name,
		Description:    payload.Description,
		Mode:           payload.Mode,
		IconType:       payload.IconType,
		Icon:           payload.Icon,
		IconBackground: payload.IconBackground,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.appResponse(app))
}

func (s *server) handleAppDetail(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.appResponse(app))
}

func (s *server) handleAppUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		Name                string `json:"name"`
		Description         string `json:"description"`
		IconType            string `json:"icon_type"`
		Icon                string `json:"icon"`
		IconBackground      string `json:"icon_background"`
		UseIconAsAnswerIcon *bool  `json:"use_icon_as_answer_icon"`
		MaxActiveRequests   *int   `json:"max_active_requests"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.UpdateApp(app.ID, app.WorkspaceID, state.UpdateAppInput{
		Name:                payload.Name,
		Description:         payload.Description,
		IconType:            payload.IconType,
		Icon:                payload.Icon,
		IconBackground:      payload.IconBackground,
		UseIconAsAnswerIcon: payload.UseIconAsAnswerIcon,
		MaxActiveRequests:   payload.MaxActiveRequests,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.appResponse(updated))
}

func (s *server) handleAppDelete(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	if err := s.store.DeleteApp(app.ID, app.WorkspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete app.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAppCopy(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	user := currentUser(r)

	var payload struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		Mode           string `json:"mode"`
		IconType       string `json:"icon_type"`
		Icon           string `json:"icon"`
		IconBackground string `json:"icon_background"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	duplicate, err := s.store.CopyApp(app.ID, app.WorkspaceID, user, state.CopyAppInput{
		Name:           payload.Name,
		Description:    payload.Description,
		Mode:           payload.Mode,
		IconType:       payload.IconType,
		Icon:           payload.Icon,
		IconBackground: payload.IconBackground,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.appResponse(duplicate))
}

func (s *server) handleAppSiteEnable(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	var payload struct {
		EnableSite bool `json:"enable_site"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.UpdateAppSiteStatus(app.ID, app.WorkspaceID, payload.EnableSite, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update app site status.")
		return
	}
	writeJSON(w, http.StatusOK, s.appResponse(updated))
}

func (s *server) handleAppAPIEnable(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	var payload struct {
		EnableAPI bool `json:"enable_api"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.UpdateAppAPIStatus(app.ID, app.WorkspaceID, payload.EnableAPI, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update app API status.")
		return
	}
	writeJSON(w, http.StatusOK, s.appResponse(updated))
}

func (s *server) handleAppSiteUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.UpdateAppSite(app.ID, app.WorkspaceID, payload, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update app site.")
		return
	}
	writeJSON(w, http.StatusOK, s.appResponse(updated))
}

func (s *server) handleAppSiteAccessTokenReset(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	updated, err := s.store.ResetAppSiteAccessToken(app.ID, app.WorkspaceID, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to reset site access token.")
		return
	}
	site := updated.Site
	writeJSON(w, http.StatusOK, map[string]any{
		"app_id":                    updated.ID,
		"access_token":              site.AccessToken,
		"title":                     site.Title,
		"description":               site.Description,
		"chat_color_theme":          site.ChatColorTheme,
		"chat_color_theme_inverted": site.ChatColorThemeInverted,
		"author":                    site.Author,
		"support_email":             site.SupportEmail,
		"default_language":          site.DefaultLanguage,
		"customize_domain":          site.CustomizeDomain,
		"theme":                     site.Theme,
		"customize_token_strategy":  site.CustomizeTokenStrategy,
		"prompt_public":             site.PromptPublic,
		"app_base_url":              site.AppBaseURL,
		"copyright":                 site.Copyright,
		"privacy_policy":            site.PrivacyPolicy,
		"custom_disclaimer":         site.CustomDisclaimer,
		"icon_type":                 site.IconType,
		"icon":                      site.Icon,
		"icon_background":           site.IconBackground,
		"icon_url":                  site.IconURL,
		"show_workflow_steps":       site.ShowWorkflowSteps,
		"use_icon_as_answer_icon":   site.UseIconAsAnswerIcon,
	})
}

func (s *server) handleAppTraceStatus(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	tracing, ok := s.store.GetTracingStatus(app.ID, app.WorkspaceID)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	provider := any(nil)
	if tracing.Provider != "" {
		provider = tracing.Provider
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":          tracing.Enabled,
		"tracing_provider": provider,
	})
}

func (s *server) handleAppTraceUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		Enabled         bool   `json:"enabled"`
		TracingProvider string `json:"tracing_provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateTracingStatus(app.ID, app.WorkspaceID, payload.Enabled, payload.TracingProvider, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update tracing status.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAppTraceConfigGet(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	provider := strings.TrimSpace(r.URL.Query().Get("tracing_provider"))
	if provider == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tracing_provider is required.")
		return
	}

	config, configured, ok := s.store.GetTracingConfig(app.ID, app.WorkspaceID, provider)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tracing_provider":   provider,
		"tracing_config":     config,
		"has_not_configured": !configured,
	})
}

func (s *server) handleAppTraceConfigSave(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		TracingProvider string         `json:"tracing_provider"`
		TracingConfig   map[string]any `json:"tracing_config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.TracingProvider) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tracing_provider is required.")
		return
	}

	if _, err := s.store.SaveTracingConfig(app.ID, app.WorkspaceID, payload.TracingProvider, payload.TracingConfig, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save tracing configuration.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAppTraceConfigDelete(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	provider := strings.TrimSpace(r.URL.Query().Get("tracing_provider"))
	if provider == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "tracing_provider is required.")
		return
	}

	if _, err := s.store.RemoveTracingConfig(app.ID, app.WorkspaceID, provider, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete tracing configuration.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAppModelConfigUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if _, err := s.store.UpdateModelConfig(app.ID, app.WorkspaceID, payload, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update model config.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowOnlineUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{},
	})
}

func (s *server) currentUserApp(r *http.Request) (state.App, bool) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		return state.App{}, false
	}

	appID := chi.URLParam(r, "appID")
	if appID == "" {
		return state.App{}, false
	}
	return s.store.GetApp(appID, workspace.ID)
}

func (s *server) appResponse(app state.App) map[string]any {
	iconURL := any(nil)
	if app.Site.IconURL != nil {
		iconURL = *app.Site.IconURL
	}

	response := map[string]any{
		"id":                      app.ID,
		"name":                    app.Name,
		"description":             app.Description,
		"author_name":             app.AuthorName,
		"icon_type":               app.IconType,
		"icon":                    app.Icon,
		"icon_background":         nullIfEmpty(app.IconBackground),
		"icon_url":                iconURL,
		"use_icon_as_answer_icon": app.UseIconAsAnswerIcon,
		"mode":                    app.Mode,
		"enable_site":             app.EnableSite,
		"enable_api":              app.EnableAPI,
		"api_rpm":                 app.APIRPM,
		"api_rph":                 app.APIRPH,
		"is_demo":                 app.IsDemo,
		"model_config":            app.ModelConfig,
		"app_model_config":        app.ModelConfig,
		"created_at":              app.CreatedAt,
		"updated_at":              app.UpdatedAt,
		"site": map[string]any{
			"access_token":              app.Site.AccessToken,
			"title":                     app.Site.Title,
			"description":               app.Site.Description,
			"chat_color_theme":          app.Site.ChatColorTheme,
			"chat_color_theme_inverted": app.Site.ChatColorThemeInverted,
			"author":                    app.Site.Author,
			"support_email":             app.Site.SupportEmail,
			"default_language":          app.Site.DefaultLanguage,
			"customize_domain":          app.Site.CustomizeDomain,
			"theme":                     app.Site.Theme,
			"customize_token_strategy":  app.Site.CustomizeTokenStrategy,
			"prompt_public":             app.Site.PromptPublic,
			"app_base_url":              app.Site.AppBaseURL,
			"copyright":                 app.Site.Copyright,
			"privacy_policy":            app.Site.PrivacyPolicy,
			"custom_disclaimer":         app.Site.CustomDisclaimer,
			"icon_type":                 app.Site.IconType,
			"icon":                      app.Site.Icon,
			"icon_background":           nullIfEmpty(app.Site.IconBackground),
			"icon_url":                  iconURL,
			"show_workflow_steps":       app.Site.ShowWorkflowSteps,
			"use_icon_as_answer_icon":   app.Site.UseIconAsAnswerIcon,
		},
		"api_base_url":      fmt.Sprintf("http://localhost:5001/v1/apps/%s", app.ID),
		"tags":              []any{},
		"deleted_tools":     []any{},
		"access_mode":       app.AccessMode,
		"has_draft_trigger": false,
		"tracing": map[string]any{
			"enabled":          app.Tracing.Enabled,
			"tracing_provider": nullIfEmpty(app.Tracing.Provider),
		},
	}
	if app.Workflow != nil {
		response["workflow"] = map[string]any{
			"id":         app.Workflow.ID,
			"created_by": app.Workflow.CreatedBy,
			"created_at": app.Workflow.CreatedAt,
			"updated_by": app.Workflow.UpdatedBy,
			"updated_at": app.Workflow.UpdatedAt,
		}
	}
	if app.MaxActiveRequests != nil {
		response["max_active_requests"] = *app.MaxActiveRequests
	} else {
		response["max_active_requests"] = nil
	}
	return response
}

func intQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func boolQuery(r *http.Request, key string) bool {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	ok, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return ok
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
