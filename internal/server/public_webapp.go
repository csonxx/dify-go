package server

import (
	"net/http"
	"strings"

	"github.com/langgenius/dify-go/internal/state"
)

const (
	webAppCodeHeader     = "X-App-Code"
	webAppPassportHeader = "X-App-Passport"
)

func (s *server) currentPublicApp(r *http.Request) (state.App, string, bool) {
	appCode := strings.TrimSpace(r.Header.Get(webAppCodeHeader))
	if appCode == "" {
		appCode = strings.TrimSpace(r.URL.Query().Get("app_code"))
	}
	if appCode == "" {
		appCode = strings.TrimSpace(r.URL.Query().Get("appCode"))
	}
	if appCode == "" {
		return state.App{}, "", false
	}

	app, ok := s.store.FindAppBySiteAccessToken(appCode)
	if !ok {
		return state.App{}, appCode, false
	}
	return app, appCode, true
}

func (s *server) handlePublicWebAppAccessMode(w http.ResponseWriter, r *http.Request) {
	app, _, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accessMode": firstImportValue(strings.TrimSpace(app.AccessMode), "public"),
	})
}

func (s *server) handlePublicPassport(w http.ResponseWriter, r *http.Request) {
	app, appCode, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	accessToken := strings.TrimSpace(r.Header.Get(webAppPassportHeader))
	if accessToken == "" {
		accessToken = "passport_" + app.ID
	}
	if strings.TrimSpace(accessToken) == "" {
		accessToken = "passport_" + appCode
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
	})
}

func (s *server) handlePublicAppSite(w http.ResponseWriter, r *http.Request) {
	app, _, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.publicAppInfoResponse(app))
}

func (s *server) handlePublicAppParameters(w http.ResponseWriter, r *http.Request) {
	app, _, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	parameters := cloneJSONObject(app.ModelConfig)
	delete(parameters, "model")
	writeJSON(w, http.StatusOK, parameters)
}

func (s *server) handlePublicAppMeta(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.currentPublicApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tool_icons": map[string]any{},
	})
}

func (s *server) publicAppInfoResponse(app state.App) map[string]any {
	return map[string]any{
		"app_id":           app.ID,
		"can_replace_logo": false,
		"custom_config":    nil,
		"enable_site":      app.EnableSite,
		"end_user_id":      nil,
		"site":             s.publicSiteResponse(app),
	}
}

func (s *server) publicSiteResponse(app state.App) map[string]any {
	iconURL := any(nil)
	if app.Site.IconURL != nil {
		iconURL = *app.Site.IconURL
	}

	return map[string]any{
		"title":                     app.Site.Title,
		"chat_color_theme":          app.Site.ChatColorTheme,
		"chat_color_theme_inverted": app.Site.ChatColorThemeInverted,
		"icon_type":                 app.Site.IconType,
		"icon":                      app.Site.Icon,
		"icon_background":           nullIfEmpty(app.Site.IconBackground),
		"icon_url":                  iconURL,
		"description":               app.Site.Description,
		"default_language":          app.Site.DefaultLanguage,
		"prompt_public":             app.Site.PromptPublic,
		"copyright":                 app.Site.Copyright,
		"privacy_policy":            app.Site.PrivacyPolicy,
		"custom_disclaimer":         app.Site.CustomDisclaimer,
		"show_workflow_steps":       app.Site.ShowWorkflowSteps,
		"use_icon_as_answer_icon":   app.Site.UseIconAsAnswerIcon,
	}
}
