package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/langgenius/dify-go/internal/state"
)

type appDSLDocument struct {
	Version string `yaml:"version"`
	Kind    string `yaml:"kind"`
	App     struct {
		Name           string         `yaml:"name"`
		Description    string         `yaml:"description,omitempty"`
		Mode           string         `yaml:"mode"`
		IconType       string         `yaml:"icon_type,omitempty"`
		Icon           string         `yaml:"icon,omitempty"`
		IconBackground string         `yaml:"icon_background,omitempty"`
		ModelConfig    map[string]any `yaml:"model_config,omitempty"`
		Site           map[string]any `yaml:"site,omitempty"`
	} `yaml:"app"`
}

func (s *server) handleAppAPIKeyList(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	keys, err := s.store.ListAppAPIKeys(app.ID, app.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	data := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		item := map[string]any{
			"id":         key.ID,
			"type":       key.Type,
			"token":      key.Token,
			"created_at": key.CreatedAt,
		}
		if key.LastUsedAt != nil {
			item["last_used_at"] = *key.LastUsedAt
		} else {
			item["last_used_at"] = nil
		}
		data = append(data, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleAppAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	key, err := s.store.CreateAppAPIKey(app.ID, app.WorkspaceID, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         key.ID,
		"type":       key.Type,
		"token":      key.Token,
		"created_at": key.CreatedAt,
	})
}

func (s *server) handleAppAPIKeyDelete(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	apiKeyID := chi.URLParam(r, "apiKeyID")
	if apiKeyID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "API key ID is required.")
		return
	}

	if err := s.store.DeleteAppAPIKey(app.ID, app.WorkspaceID, apiKeyID); err != nil {
		writeError(w, http.StatusNotFound, "api_key_not_found", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAppExport(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	includeSecret := boolQuery(r, "include_secret")
	document := appDSLDocument{
		Version: "0.1.0",
		Kind:    "dify-go/app",
	}
	document.App.Name = app.Name
	document.App.Description = app.Description
	document.App.Mode = app.Mode
	document.App.IconType = app.IconType
	document.App.Icon = app.Icon
	document.App.IconBackground = app.IconBackground
	document.App.ModelConfig = cloneJSONObject(app.ModelConfig)
	document.App.Site = map[string]any{
		"title":                     app.Site.Title,
		"description":               app.Site.Description,
		"chat_color_theme":          app.Site.ChatColorTheme,
		"chat_color_theme_inverted": app.Site.ChatColorThemeInverted,
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
		"icon_background":           app.Site.IconBackground,
		"show_workflow_steps":       app.Site.ShowWorkflowSteps,
		"use_icon_as_answer_icon":   app.Site.UseIconAsAnswerIcon,
	}
	if includeSecret {
		document.App.Site["access_token"] = app.Site.AccessToken
	}

	payload, err := yaml.Marshal(document)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to export app DSL.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": string(payload),
	})
}

func (s *server) handleAppImport(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Mode           string `json:"mode"`
		YAMLContent    string `json:"yaml_content"`
		YAMLURL        string `json:"yaml_url"`
		AppID          string `json:"app_id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		IconType       string `json:"icon_type"`
		Icon           string `json:"icon"`
		IconBackground string `json:"icon_background"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	content, err := resolveImportContent(payload.Mode, payload.YAMLContent, payload.YAMLURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	document, err := decodeAppDSL(content)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if strings.TrimSpace(payload.Name) != "" {
		document.App.Name = strings.TrimSpace(payload.Name)
	}
	if strings.TrimSpace(payload.Description) != "" {
		document.App.Description = strings.TrimSpace(payload.Description)
	}
	if strings.TrimSpace(payload.IconType) != "" {
		document.App.IconType = strings.TrimSpace(payload.IconType)
	}
	if strings.TrimSpace(payload.Icon) != "" {
		document.App.Icon = strings.TrimSpace(payload.Icon)
	}
	if strings.TrimSpace(payload.IconBackground) != "" {
		document.App.IconBackground = strings.TrimSpace(payload.IconBackground)
	}

	now := time.Now()
	var app state.App
	if strings.TrimSpace(payload.AppID) != "" {
		existing, ok := s.store.GetApp(payload.AppID, workspace.ID)
		if !ok {
			writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
			return
		}
		if mode := normalizeImportMode(document.App.Mode, existing.Mode); mode != existing.Mode {
			writeError(w, http.StatusBadRequest, "invalid_request", "Imported DSL mode does not match the existing app.")
			return
		}

		updated, err := s.store.UpdateApp(existing.ID, existing.WorkspaceID, state.UpdateAppInput{
			Name:                firstImportValue(document.App.Name, existing.Name),
			Description:         firstImportValue(document.App.Description, existing.Description),
			IconType:            firstImportValue(document.App.IconType, existing.IconType),
			Icon:                firstImportValue(document.App.Icon, existing.Icon),
			IconBackground:      firstImportValue(document.App.IconBackground, existing.IconBackground),
			UseIconAsAnswerIcon: boolPtr(existing.UseIconAsAnswerIcon),
			MaxActiveRequests:   existing.MaxActiveRequests,
		}, user, now)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		app = updated
	} else {
		created, err := s.store.CreateApp(workspace.ID, user, state.CreateAppInput{
			Name:           firstImportValue(document.App.Name, "Imported App"),
			Description:    document.App.Description,
			Mode:           normalizeImportMode(document.App.Mode, "workflow"),
			IconType:       document.App.IconType,
			Icon:           document.App.Icon,
			IconBackground: document.App.IconBackground,
		}, now)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		app = created
	}

	if len(document.App.ModelConfig) > 0 {
		app, err = s.store.UpdateModelConfig(app.ID, app.WorkspaceID, document.App.ModelConfig, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to import model configuration.")
			return
		}
	}
	if len(document.App.Site) > 0 {
		app, err = s.store.UpdateAppSite(app.ID, app.WorkspaceID, document.App.Site, now)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Failed to import site configuration.")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                   generateImportID(),
		"status":               "completed",
		"app_mode":             app.Mode,
		"app_id":               app.ID,
		"current_dsl_version":  "0.1.0",
		"imported_dsl_version": document.Version,
		"error":                "",
		"leaked_dependencies":  []any{},
	})
}

func (s *server) handleAppImportConfirm(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"id":                  chi.URLParam(r, "importID"),
		"status":              "failed",
		"app_mode":            "",
		"error":               "This import does not require confirmation.",
		"leaked_dependencies": []any{},
	})
}

func (s *server) handleAppConvertToWorkflow(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	targetMode := ""
	switch app.Mode {
	case "chat":
		targetMode = "advanced-chat"
	case "completion":
		targetMode = "workflow"
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "Only chat and completion apps can be converted.")
		return
	}

	user := currentUser(r)
	var payload struct {
		Name           string `json:"name"`
		IconType       string `json:"icon_type"`
		Icon           string `json:"icon"`
		IconBackground string `json:"icon_background"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	converted, err := s.store.CopyApp(app.ID, app.WorkspaceID, user, state.CopyAppInput{
		Name:           payload.Name,
		Description:    app.Description,
		Mode:           targetMode,
		IconType:       payload.IconType,
		Icon:           payload.Icon,
		IconBackground: payload.IconBackground,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"new_app_id": converted.ID})
}

func (s *server) handleWorkflowWebhookTrigger(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if app.Workflow == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Webhook trigger is only available for workflow-based apps.")
		return
	}

	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "node_id is required.")
		return
	}

	baseURL := requestBaseURL(r)
	webhookID := sanitizeTokenPart(app.ID + "-" + nodeID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                "whcfg_" + webhookID,
		"webhook_id":        "wh_" + webhookID,
		"webhook_url":       fmt.Sprintf("%s/trigger/%s/webhook/%s", baseURL, app.ID, webhookID),
		"webhook_debug_url": fmt.Sprintf("%s/trigger/%s/webhook/%s/debug", baseURL, app.ID, webhookID),
		"node_id":           nodeID,
		"created_at":        app.CreatedAt,
	})
}

func (s *server) handleAppVoices(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, []map[string]string{
		{"name": "Alloy", "value": "alloy"},
		{"name": "Echo", "value": "echo"},
		{"name": "Nova", "value": "nova"},
		{"name": "Shimmer", "value": "shimmer"},
	})
}

func (s *server) handleAppStatistics(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func (s *server) handleWorkflowStatistics(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
}

func resolveImportContent(mode, yamlContent, yamlURL string) ([]byte, error) {
	switch strings.TrimSpace(mode) {
	case "yaml-content":
		if strings.TrimSpace(yamlContent) == "" {
			return nil, fmt.Errorf("yaml_content is required")
		}
		return []byte(yamlContent), nil
	case "yaml-url":
		if strings.TrimSpace(yamlURL) == "" {
			return nil, fmt.Errorf("yaml_url is required")
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(yamlURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download DSL: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("failed to download DSL: unexpected status %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return nil, fmt.Errorf("failed to read DSL content: %w", err)
		}
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported import mode")
	}
}

func decodeAppDSL(content []byte) (appDSLDocument, error) {
	var document appDSLDocument
	if err := yaml.Unmarshal(content, &document); err == nil && strings.TrimSpace(document.App.Mode) != "" {
		if strings.TrimSpace(document.Version) == "" {
			document.Version = "0.1.0"
		}
		return document, nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return appDSLDocument{}, fmt.Errorf("invalid DSL content")
	}

	appData := raw
	if nested, ok := raw["app"].(map[string]any); ok {
		appData = nested
	}

	document.Version = stringFromAny(raw["version"])
	if document.Version == "" {
		document.Version = "0.1.0"
	}
	document.Kind = stringFromAny(raw["kind"])
	document.App.Name = stringFromAny(appData["name"])
	document.App.Description = stringFromAny(appData["description"])
	document.App.Mode = stringFromAny(appData["mode"])
	document.App.IconType = stringFromAny(appData["icon_type"])
	document.App.Icon = stringFromAny(appData["icon"])
	document.App.IconBackground = stringFromAny(appData["icon_background"])
	document.App.ModelConfig = mapFromAny(appData["model_config"])
	document.App.Site = mapFromAny(appData["site"])
	if strings.TrimSpace(document.App.Mode) == "" {
		return appDSLDocument{}, fmt.Errorf("dsl mode is required")
	}
	return document, nil
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func sanitizeTokenPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", "/", "-", " ", "-", ".", "-")
	return replacer.Replace(value)
}

func generateImportID() string {
	return fmt.Sprintf("imp_%d", time.Now().UTC().UnixNano())
}

func normalizeImportMode(value, fallback string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "chat", "agent-chat", "advanced-chat", "workflow", "completion":
		return value
	default:
		return fallback
	}
}

func cloneJSONObject(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}

	data, err := json.Marshal(src)
	if err != nil {
		return map[string]any{}
	}

	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONObject(typed)
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[fmt.Sprint(key)] = item
		}
		return out
	default:
		return map[string]any{}
	}
}

func stringFromAny(value any) string {
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func firstImportValue(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return fallback
}

func boolPtr(value bool) *bool {
	return &value
}
