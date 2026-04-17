package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

type modelProviderCatalog struct {
	Provider              string
	Label                 string
	Description           string
	HelpURL               string
	Icon                  string
	IconDark              string
	Background            string
	SupportedModelTypes   []string
	ConfigurateMethods    []string
	SystemEnabled         bool
	SystemQuotaValid      bool
	ProviderSchema        []map[string]any
	ModelCredentialSchema []map[string]any
	Models                []modelCatalogItem
}

type modelCatalogItem struct {
	Model                string
	Label                string
	ModelType            string
	Features             []string
	FetchFrom            string
	LoadBalancingEnabled bool
	Deprecated           bool
}

func (s *server) mountWorkspaceModelRoutes(r chi.Router) {
	r.Get("/workspaces/current/model-providers", s.handleModelProviders)
	r.Get("/workspaces/current/models/model-types/{modelType}", s.handleModelListByType)
	r.Get("/workspaces/current/default-model", s.handleDefaultModelGet)
	r.Post("/workspaces/current/default-model", s.handleDefaultModelUpdate)
	r.Route("/workspaces/current/model-providers/{provider}", func(r chi.Router) {
		r.Get("/models", s.handleProviderModelList)
		r.Post("/models", s.handleModelProviderModelUpsert)
		r.Delete("/models", s.handleModelProviderModelDelete)
		r.Patch("/models/enable", s.handleModelEnable)
		r.Post("/models/enable", s.handleModelEnable)
		r.Patch("/models/disable", s.handleModelDisable)
		r.Post("/models/disable", s.handleModelDisable)
		r.Get("/models/parameter-rules", s.handleModelParameterRules)
		r.Get("/models/credentials", s.handleModelCredentialGet)
		r.Post("/models/credentials", s.handleModelCredentialCreate)
		r.Put("/models/credentials", s.handleModelCredentialUpdate)
		r.Delete("/models/credentials", s.handleModelCredentialDelete)
		r.Post("/models/credentials/switch", s.handleModelCredentialSwitch)
		r.Post("/models/credentials/validate", s.handleModelProviderValidate)
		r.Post("/models/load-balancing-configs/credentials-validate", s.handleLoadBalancingCredentialValidate)
		r.Post("/models/load-balancing-configs/{configID}/credentials-validate", s.handleLoadBalancingCredentialValidate)
		r.Get("/credentials", s.handleProviderCredentialGet)
		r.Post("/credentials", s.handleProviderCredentialCreate)
		r.Put("/credentials", s.handleProviderCredentialUpdate)
		r.Delete("/credentials", s.handleProviderCredentialDelete)
		r.Post("/credentials/switch", s.handleProviderCredentialSwitch)
		r.Post("/credentials/validate", s.handleModelProviderValidate)
		r.Get("/checkout-url", s.handleModelProviderCheckoutURL)
	})
}

func (s *server) handleModelProviders(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	data := make([]map[string]any, 0, len(modelProviderCatalogs()))
	for _, catalog := range modelProviderCatalogs() {
		data = append(data, s.modelProviderResponse(catalog, settings))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleModelListByType(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	modelType := strings.TrimSpace(chi.URLParam(r, "modelType"))
	items := make([]map[string]any, 0)
	for _, catalog := range modelProviderCatalogs() {
		models := s.modelItemsByType(catalog, settings, modelType)
		if len(models) == 0 {
			continue
		}
		groupStatus := "active"
		if !hasActiveModel(models) {
			groupStatus = "no-configure"
		}
		items = append(items, map[string]any{
			"provider":        catalog.Provider,
			"icon_small":      i18nText(catalog.Icon),
			"icon_small_dark": i18nText(catalog.IconDark),
			"label":           i18nText(catalog.Label),
			"models":          models,
			"status":          groupStatus,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *server) handleDefaultModelGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	modelType := strings.TrimSpace(r.URL.Query().Get("model_type"))
	model := s.defaultModelSelection(settings, modelType)
	if model.Model == "" {
		writeJSON(w, http.StatusOK, map[string]any{"data": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"model":      model.Model,
			"model_type": model.ModelType,
			"provider": map[string]any{
				"provider":   model.Provider,
				"icon_small": i18nText(modelCatalog(model.Provider).Icon),
			},
		},
	})
}

func (s *server) handleDefaultModelUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		ModelSettings []state.WorkspaceDefaultModel `json:"model_settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		items := make([]state.WorkspaceDefaultModel, 0, len(payload.ModelSettings))
		for _, item := range payload.ModelSettings {
			if strings.TrimSpace(item.ModelType) == "" || strings.TrimSpace(item.Provider) == "" || strings.TrimSpace(item.Model) == "" {
				continue
			}
			items = append(items, item)
		}
		settings.DefaultModels = items
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save default model settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleProviderModelList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	catalog := modelCatalog(chi.URLParam(r, "provider"))
	if catalog.Provider == "" {
		writeError(w, http.StatusNotFound, "provider_not_found", "Model provider not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": s.providerModelItems(catalog, settings)})
}

func (s *server) handleProviderCredentialGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	provider := chi.URLParam(r, "provider")
	providerState := s.providerState(settings, provider)
	credentialID := strings.TrimSpace(r.URL.Query().Get("credential_id"))
	credential := activeOrSelectedCredential(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID, credentialID)

	writeJSON(w, http.StatusOK, s.providerCredentialPayload(providerState, credential))
}

func (s *server) handleProviderCredentialCreate(w http.ResponseWriter, r *http.Request) {
	s.handleProviderCredentialUpsert(w, r, false)
}

func (s *server) handleProviderCredentialUpdate(w http.ResponseWriter, r *http.Request) {
	s.handleProviderCredentialUpsert(w, r, true)
}

func (s *server) handleProviderCredentialUpsert(w http.ResponseWriter, r *http.Request, requireExisting bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID string         `json:"credential_id"`
		Name         string         `json:"name"`
		Credentials  map[string]any `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	result, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		index := slices.IndexFunc(providerState.ProviderCredentials, func(item state.WorkspaceCredential) bool {
			return item.CredentialID == strings.TrimSpace(payload.CredentialID)
		})

		if requireExisting && index < 0 {
			return fmt.Errorf("credential_not_found")
		}

		credential := state.WorkspaceCredential{
			CredentialID: firstNonEmpty(strings.TrimSpace(payload.CredentialID), generateRuntimeCredentialID("cred")),
			Name:         firstNonEmpty(strings.TrimSpace(payload.Name), "Default Credential"),
			Credentials:  cloneJSONObject(payload.Credentials),
		}
		if index >= 0 {
			providerState.ProviderCredentials[index] = credential
		} else {
			providerState.ProviderCredentials = append(providerState.ProviderCredentials, credential)
		}
		if providerState.ActiveProviderCredentialID == "" {
			providerState.ActiveProviderCredentialID = credential.CredentialID
		}
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save provider credential.")
		return
	}

	providerState := s.providerState(result, provider)
	credential := activeOrSelectedCredential(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID, strings.TrimSpace(payload.CredentialID))
	response := s.providerCredentialPayload(providerState, credential)
	response["result"] = "success"
	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleProviderCredentialDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		index := slices.IndexFunc(providerState.ProviderCredentials, func(item state.WorkspaceCredential) bool {
			return item.CredentialID == strings.TrimSpace(payload.CredentialID)
		})
		if index < 0 {
			return fmt.Errorf("credential_not_found")
		}
		providerState.ProviderCredentials = append(providerState.ProviderCredentials[:index], providerState.ProviderCredentials[index+1:]...)
		if providerState.ActiveProviderCredentialID == payload.CredentialID {
			providerState.ActiveProviderCredentialID = ""
			if len(providerState.ProviderCredentials) > 0 {
				providerState.ActiveProviderCredentialID = providerState.ProviderCredentials[0].CredentialID
			}
		}
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete provider credential.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleProviderCredentialSwitch(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		if !containsCredential(providerState.ProviderCredentials, payload.CredentialID) {
			return fmt.Errorf("credential_not_found")
		}
		providerState.ActiveProviderCredentialID = payload.CredentialID
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to switch provider credential.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelCredentialGet(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	settings, ok := s.store.GetWorkspaceModelSettings(workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	providerState := s.providerState(settings, chi.URLParam(r, "provider"))
	modelSetting := s.modelState(providerState, strings.TrimSpace(r.URL.Query().Get("model_type")), strings.TrimSpace(r.URL.Query().Get("model")))
	configFrom := strings.TrimSpace(r.URL.Query().Get("config_from"))
	var credential state.WorkspaceCredential
	switch configFrom {
	case "predefined-model":
		credential = activeOrSelectedCredential(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID, strings.TrimSpace(r.URL.Query().Get("credential_id")))
	default:
		credential = activeOrSelectedCredential(modelSetting.Credentials, modelSetting.ActiveCredentialID, strings.TrimSpace(r.URL.Query().Get("credential_id")))
	}
	writeJSON(w, http.StatusOK, s.modelCredentialPayload(providerState, modelSetting, credential, configFrom))
}

func (s *server) handleModelCredentialCreate(w http.ResponseWriter, r *http.Request) {
	s.handleModelCredentialUpsert(w, r, false)
}

func (s *server) handleModelCredentialUpdate(w http.ResponseWriter, r *http.Request) {
	s.handleModelCredentialUpsert(w, r, true)
}

func (s *server) handleModelCredentialUpsert(w http.ResponseWriter, r *http.Request, requireExisting bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID  string                             `json:"credential_id"`
		Model         string                             `json:"model"`
		ModelType     string                             `json:"model_type"`
		Credentials   map[string]any                     `json:"credentials"`
		LoadBalancing state.WorkspaceLoadBalancingConfig `json:"load_balancing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	result, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		modelState := state.FindWorkspaceModelState(providerState, payload.ModelType, payload.Model)
		index := slices.IndexFunc(modelState.Credentials, func(item state.WorkspaceCredential) bool {
			return item.CredentialID == strings.TrimSpace(payload.CredentialID)
		})
		if requireExisting && index < 0 {
			return fmt.Errorf("credential_not_found")
		}

		credential := state.WorkspaceCredential{
			CredentialID: firstNonEmpty(strings.TrimSpace(payload.CredentialID), generateRuntimeCredentialID("mcred")),
			Name:         firstNonEmpty(strings.TrimSpace(payload.Model), "Model Credential"),
			Credentials:  cloneJSONObject(payload.Credentials),
		}
		if index >= 0 {
			modelState.Credentials[index] = credential
		} else {
			modelState.Credentials = append(modelState.Credentials, credential)
		}
		if modelState.ActiveCredentialID == "" {
			modelState.ActiveCredentialID = credential.CredentialID
		}
		normalizeWorkspaceLoadBalancing(&payload.LoadBalancing)
		modelState.LoadBalancing = payload.LoadBalancing
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to save model credential.")
		return
	}

	providerState := s.providerState(result, provider)
	modelState := s.modelState(providerState, payload.ModelType, payload.Model)
	credential := activeOrSelectedCredential(modelState.Credentials, modelState.ActiveCredentialID, strings.TrimSpace(payload.CredentialID))
	response := s.modelCredentialPayload(providerState, modelState, credential, "")
	response["result"] = "success"
	writeJSON(w, http.StatusOK, response)
}

func (s *server) handleModelCredentialDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID string `json:"credential_id"`
		Model        string `json:"model"`
		ModelType    string `json:"model_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		modelState := state.FindWorkspaceModelState(providerState, payload.ModelType, payload.Model)
		index := slices.IndexFunc(modelState.Credentials, func(item state.WorkspaceCredential) bool {
			return item.CredentialID == strings.TrimSpace(payload.CredentialID)
		})
		if index < 0 {
			return fmt.Errorf("credential_not_found")
		}
		modelState.Credentials = append(modelState.Credentials[:index], modelState.Credentials[index+1:]...)
		if modelState.ActiveCredentialID == payload.CredentialID {
			modelState.ActiveCredentialID = ""
			if len(modelState.Credentials) > 0 {
				modelState.ActiveCredentialID = modelState.Credentials[0].CredentialID
			}
		}
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete model credential.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelCredentialSwitch(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		CredentialID string `json:"credential_id"`
		Model        string `json:"model"`
		ModelType    string `json:"model_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		modelState := state.FindWorkspaceModelState(providerState, payload.ModelType, payload.Model)
		if !containsCredential(modelState.Credentials, payload.CredentialID) {
			return fmt.Errorf("credential_not_found")
		}
		modelState.ActiveCredentialID = payload.CredentialID
		return nil
	})
	if err != nil {
		if err.Error() == "credential_not_found" {
			writeError(w, http.StatusNotFound, "credential_not_found", "Credential not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to switch model credential.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelProviderModelUpsert(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		ConfigFrom    string                             `json:"config_from"`
		Model         string                             `json:"model"`
		ModelType     string                             `json:"model_type"`
		Credentials   map[string]any                     `json:"credentials"`
		LoadBalancing state.WorkspaceLoadBalancingConfig `json:"load_balancing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		modelState := state.FindWorkspaceModelState(providerState, payload.ModelType, payload.Model)
		normalizeWorkspaceLoadBalancing(&payload.LoadBalancing)
		modelState.LoadBalancing = payload.LoadBalancing
		if len(payload.Credentials) > 0 {
			credential := state.WorkspaceCredential{
				CredentialID: generateRuntimeCredentialID("mcred"),
				Name:         payload.Model,
				Credentials:  cloneJSONObject(payload.Credentials),
			}
			index := slices.IndexFunc(modelState.Credentials, func(item state.WorkspaceCredential) bool {
				return reflectCredential(item.Credentials) == reflectCredential(credential.Credentials)
			})
			if index < 0 {
				modelState.Credentials = append(modelState.Credentials, credential)
				modelState.ActiveCredentialID = credential.CredentialID
			}
		}
		if modelState.Enabled == nil {
			enabled := true
			modelState.Enabled = &enabled
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update model settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelProviderModelDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Model     string `json:"model"`
		ModelType string `json:"model_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		index := slices.IndexFunc(providerState.ModelSettings, func(item state.WorkspaceModelSetting) bool {
			return item.Model == payload.Model && item.ModelType == payload.ModelType
		})
		if index < 0 {
			return nil
		}
		if modelCatalogHas(provider, payload.ModelType, payload.Model) {
			providerState.ModelSettings[index].Credentials = []state.WorkspaceCredential{}
			providerState.ModelSettings[index].ActiveCredentialID = ""
			providerState.ModelSettings[index].LoadBalancing = state.WorkspaceLoadBalancingConfig{}
			providerState.ModelSettings[index].Enabled = nil
			return nil
		}
		providerState.ModelSettings = append(providerState.ModelSettings[:index], providerState.ModelSettings[index+1:]...)
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to delete model settings.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelEnable(w http.ResponseWriter, r *http.Request) {
	s.handleModelEnabledState(w, r, true)
}

func (s *server) handleModelDisable(w http.ResponseWriter, r *http.Request) {
	s.handleModelEnabledState(w, r, false)
}

func (s *server) handleModelEnabledState(w http.ResponseWriter, r *http.Request, enabled bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Model     string `json:"model"`
		ModelType string `json:"model_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	provider := chi.URLParam(r, "provider")
	_, err := s.store.MutateWorkspaceModelSettings(workspace.ID, func(settings *state.WorkspaceModelSettings) error {
		providerState := state.FindWorkspaceProviderState(settings, provider)
		modelState := state.FindWorkspaceModelState(providerState, payload.ModelType, payload.Model)
		modelState.Enabled = &enabled
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update model status.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelParameterRules(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if modelCatalog(provider).Provider == "" {
		writeError(w, http.StatusNotFound, "provider_not_found", "Model provider not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": parameterRules(strings.TrimSpace(r.URL.Query().Get("model")))})
}

func (s *server) handleModelProviderValidate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleLoadBalancingCredentialValidate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleModelProviderCheckoutURL(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	url := "https://platform.openai.com/settings/organization/billing/overview"
	switch provider {
	case "anthropic":
		url = "https://console.anthropic.com/settings/plans"
	case "cohere":
		url = "https://dashboard.cohere.com/api-keys"
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": url})
}

func (s *server) modelProviderResponse(catalog modelProviderCatalog, settings state.WorkspaceModelSettings) map[string]any {
	providerState := s.providerState(settings, catalog.Provider)
	customCredentials := credentialList(providerState.ProviderCredentials)
	customModels := customModelCredentials(catalog, providerState)
	status := "no-configure"
	if providerState.ActiveProviderCredentialID != "" || providerHasConfiguredModels(providerState) {
		status = "active"
	}

	return map[string]any{
		"provider":              catalog.Provider,
		"label":                 i18nText(catalog.Label),
		"description":           i18nText(catalog.Description),
		"help":                  map[string]any{"title": i18nText("Help"), "url": i18nText(catalog.HelpURL)},
		"icon_small":            i18nText(catalog.Icon),
		"icon_small_dark":       i18nText(catalog.IconDark),
		"background":            catalog.Background,
		"supported_model_types": catalog.SupportedModelTypes,
		"configurate_methods":   catalog.ConfigurateMethods,
		"provider_credential_schema": map[string]any{
			"credential_form_schemas": catalog.ProviderSchema,
		},
		"model_credential_schema": map[string]any{
			"model": map[string]any{
				"label":       i18nText("Model"),
				"placeholder": i18nText("Please enter model name"),
			},
			"credential_form_schemas": catalog.ModelCredentialSchema,
		},
		"preferred_provider_type": "custom",
		"custom_configuration": map[string]any{
			"status":                  status,
			"current_credential_id":   nullIfEmpty(providerState.ActiveProviderCredentialID),
			"current_credential_name": nullIfEmpty(credentialName(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID)),
			"available_credentials":   customCredentials,
			"custom_models":           customModels,
			"can_added_models":        addableCustomModels(catalog),
		},
		"system_configuration": map[string]any{
			"enabled":            catalog.SystemEnabled,
			"current_quota_type": "free",
			"quota_configurations": []map[string]any{
				{
					"quota_type":  "free",
					"quota_unit":  "times",
					"quota_limit": 1000,
					"quota_used":  0,
					"last_used":   0,
					"is_valid":    catalog.SystemQuotaValid,
				},
			},
		},
		"allow_custom_token": true,
	}
}

func (s *server) providerModelItems(catalog modelProviderCatalog, settings state.WorkspaceModelSettings) []map[string]any {
	items := make([]map[string]any, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		items = append(items, s.modelItemResponse(catalog, settings, model))
	}
	providerState := s.providerState(settings, catalog.Provider)
	for _, modelState := range providerState.ModelSettings {
		if modelCatalogHas(catalog.Provider, modelState.ModelType, modelState.Model) {
			continue
		}
		items = append(items, s.customModelItemResponse(catalog, settings, modelState))
	}
	return items
}

func (s *server) modelItemsByType(catalog modelProviderCatalog, settings state.WorkspaceModelSettings, modelType string) []map[string]any {
	items := make([]map[string]any, 0)
	for _, model := range catalog.Models {
		if model.ModelType != modelType {
			continue
		}
		items = append(items, s.modelItemResponse(catalog, settings, model))
	}
	providerState := s.providerState(settings, catalog.Provider)
	for _, modelState := range providerState.ModelSettings {
		if modelState.ModelType != modelType || modelCatalogHas(catalog.Provider, modelState.ModelType, modelState.Model) {
			continue
		}
		items = append(items, s.customModelItemResponse(catalog, settings, modelState))
	}
	return items
}

func (s *server) modelItemResponse(catalog modelProviderCatalog, settings state.WorkspaceModelSettings, model modelCatalogItem) map[string]any {
	providerState := s.providerState(settings, catalog.Provider)
	modelState := s.modelState(providerState, model.ModelType, model.Model)
	status := modelStatus(catalog, providerState, modelState)
	return map[string]any{
		"model":                              model.Model,
		"label":                              i18nText(model.Label),
		"model_type":                         model.ModelType,
		"features":                           model.Features,
		"fetch_from":                         model.FetchFrom,
		"status":                             status,
		"model_properties":                   map[string]any{},
		"load_balancing_enabled":             model.LoadBalancingEnabled,
		"deprecated":                         model.Deprecated,
		"has_invalid_load_balancing_configs": false,
	}
}

func (s *server) customModelItemResponse(catalog modelProviderCatalog, settings state.WorkspaceModelSettings, modelState state.WorkspaceModelSetting) map[string]any {
	providerState := s.providerState(settings, catalog.Provider)
	model := modelCatalogItem{
		Model:                modelState.Model,
		Label:                modelState.Model,
		ModelType:            modelState.ModelType,
		FetchFrom:            "customizable-model",
		LoadBalancingEnabled: false,
	}
	return s.modelItemResponse(catalog, settings, model.withStatus(modelStatus(catalog, providerState, modelState)))
}

func (m modelCatalogItem) withStatus(_ string) modelCatalogItem { return m }

func (s *server) providerCredentialPayload(providerState state.WorkspaceModelProviderState, credential state.WorkspaceCredential) map[string]any {
	return map[string]any{
		"credential_id":           nullIfEmpty(credential.CredentialID),
		"name":                    nullIfEmpty(credential.Name),
		"credentials":             cloneJSONObject(credential.Credentials),
		"load_balancing":          map[string]any{"enabled": false, "configs": []any{}},
		"available_credentials":   credentialList(providerState.ProviderCredentials),
		"current_credential_id":   nullIfEmpty(providerState.ActiveProviderCredentialID),
		"current_credential_name": nullIfEmpty(credentialName(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID)),
	}
}

func (s *server) modelCredentialPayload(providerState state.WorkspaceModelProviderState, modelState state.WorkspaceModelSetting, credential state.WorkspaceCredential, configFrom string) map[string]any {
	availableCredentials := credentialList(modelState.Credentials)
	currentCredentialID := modelState.ActiveCredentialID
	currentCredentialName := credentialName(modelState.Credentials, modelState.ActiveCredentialID)
	if strings.TrimSpace(configFrom) == "predefined-model" {
		availableCredentials = credentialList(providerState.ProviderCredentials)
		currentCredentialID = providerState.ActiveProviderCredentialID
		currentCredentialName = credentialName(providerState.ProviderCredentials, providerState.ActiveProviderCredentialID)
	}
	return map[string]any{
		"credential_id":           nullIfEmpty(credential.CredentialID),
		"name":                    nullIfEmpty(credential.Name),
		"credentials":             cloneJSONObject(credential.Credentials),
		"load_balancing":          workspaceLoadBalancingPayload(modelState.LoadBalancing),
		"available_credentials":   availableCredentials,
		"current_credential_id":   nullIfEmpty(currentCredentialID),
		"current_credential_name": nullIfEmpty(currentCredentialName),
	}
}

func (s *server) defaultModelSelection(settings state.WorkspaceModelSettings, modelType string) state.WorkspaceDefaultModel {
	defaults := state.WorkspaceDefaultModelMap(settings)
	if item, ok := defaults[modelType]; ok {
		return item
	}
	for _, catalog := range modelProviderCatalogs() {
		for _, model := range catalog.Models {
			if model.ModelType == modelType {
				return state.WorkspaceDefaultModel{
					ModelType: modelType,
					Provider:  catalog.Provider,
					Model:     model.Model,
				}
			}
		}
	}
	return state.WorkspaceDefaultModel{}
}

func (s *server) providerState(settings state.WorkspaceModelSettings, provider string) state.WorkspaceModelProviderState {
	for _, item := range settings.Providers {
		if item.Provider == provider {
			return item
		}
	}
	return state.WorkspaceModelProviderState{
		Provider:            provider,
		ProviderCredentials: []state.WorkspaceCredential{},
		ModelSettings:       []state.WorkspaceModelSetting{},
	}
}

func (s *server) modelState(providerState state.WorkspaceModelProviderState, modelType, model string) state.WorkspaceModelSetting {
	for _, item := range providerState.ModelSettings {
		if item.ModelType == modelType && item.Model == model {
			return item
		}
	}
	return state.WorkspaceModelSetting{
		Model:       model,
		ModelType:   modelType,
		Credentials: []state.WorkspaceCredential{},
		LoadBalancing: state.WorkspaceLoadBalancingConfig{
			Enabled: false,
			Configs: []state.WorkspaceLoadBalancingEntry{},
		},
	}
}

func (s *server) currentUserWorkspace(r *http.Request) (state.Workspace, bool) {
	return s.store.UserWorkspace(currentUser(r).ID)
}

func modelProviderCatalogs() []modelProviderCatalog {
	return []modelProviderCatalog{
		{
			Provider:              "openai",
			Label:                 "OpenAI",
			Description:           "OpenAI text, embedding, moderation, speech, and TTS models.",
			HelpURL:               "https://platform.openai.com/docs",
			Icon:                  providerIconDataURI("O", "#0F172A", "#FFFFFF"),
			IconDark:              providerIconDataURI("O", "#FFFFFF", "#0F172A"),
			Background:            "#E7F8EF",
			SupportedModelTypes:   []string{"llm", "text-embedding", "moderation", "speech2text", "tts"},
			ConfigurateMethods:    []string{"predefined-model", "customizable-model"},
			SystemEnabled:         true,
			SystemQuotaValid:      true,
			ProviderSchema:        providerCredentialSchema("OpenAI API Key", "api_key"),
			ModelCredentialSchema: providerCredentialSchema("API Key", "api_key"),
			Models: []modelCatalogItem{
				{Model: "gpt-4o-mini", Label: "GPT-4o Mini", ModelType: "llm", Features: []string{"vision", "audio", "structured-output"}, FetchFrom: "predefined-model"},
				{Model: "gpt-4.1", Label: "GPT-4.1", ModelType: "llm", Features: []string{"vision", "structured-output", "tool-call"}, FetchFrom: "predefined-model"},
				{Model: "gpt-4o", Label: "GPT-4o", ModelType: "llm", Features: []string{"vision", "audio", "tool-call"}, FetchFrom: "predefined-model"},
				{Model: "text-embedding-3-small", Label: "Text Embedding 3 Small", ModelType: "text-embedding", FetchFrom: "predefined-model"},
				{Model: "text-embedding-3-large", Label: "Text Embedding 3 Large", ModelType: "text-embedding", FetchFrom: "predefined-model"},
				{Model: "omni-moderation-latest", Label: "Omni Moderation Latest", ModelType: "moderation", FetchFrom: "predefined-model"},
				{Model: "whisper-1", Label: "Whisper 1", ModelType: "speech2text", FetchFrom: "predefined-model"},
				{Model: "tts-1", Label: "TTS 1", ModelType: "tts", FetchFrom: "predefined-model"},
			},
		},
		{
			Provider:              "anthropic",
			Label:                 "Anthropic",
			Description:           "Anthropic Claude text generation models.",
			HelpURL:               "https://docs.anthropic.com",
			Icon:                  providerIconDataURI("A", "#C96A2B", "#FFFFFF"),
			IconDark:              providerIconDataURI("A", "#F3E7DE", "#5C2E12"),
			Background:            "#F7EFEA",
			SupportedModelTypes:   []string{"llm"},
			ConfigurateMethods:    []string{"predefined-model", "customizable-model"},
			SystemEnabled:         false,
			SystemQuotaValid:      false,
			ProviderSchema:        providerCredentialSchema("Anthropic API Key", "api_key"),
			ModelCredentialSchema: providerCredentialSchema("API Key", "api_key"),
			Models: []modelCatalogItem{
				{Model: "claude-3-5-sonnet", Label: "Claude 3.5 Sonnet", ModelType: "llm", Features: []string{"vision", "tool-call"}, FetchFrom: "predefined-model"},
				{Model: "claude-3-7-sonnet", Label: "Claude 3.7 Sonnet", ModelType: "llm", Features: []string{"vision", "tool-call"}, FetchFrom: "predefined-model"},
			},
		},
		{
			Provider:              "cohere",
			Label:                 "Cohere",
			Description:           "Cohere rerank models.",
			HelpURL:               "https://docs.cohere.com",
			Icon:                  providerIconDataURI("C", "#3656D4", "#FFFFFF"),
			IconDark:              providerIconDataURI("C", "#DCE5FF", "#1B2B6B"),
			Background:            "#EDF2FF",
			SupportedModelTypes:   []string{"rerank"},
			ConfigurateMethods:    []string{"predefined-model"},
			SystemEnabled:         true,
			SystemQuotaValid:      true,
			ProviderSchema:        providerCredentialSchema("Cohere API Key", "api_key"),
			ModelCredentialSchema: providerCredentialSchema("API Key", "api_key"),
			Models: []modelCatalogItem{
				{Model: "rerank-v3.5", Label: "Rerank v3.5", ModelType: "rerank", FetchFrom: "predefined-model"},
			},
		},
	}
}

func modelCatalog(provider string) modelProviderCatalog {
	for _, item := range modelProviderCatalogs() {
		if item.Provider == provider {
			return item
		}
	}
	return modelProviderCatalog{}
}

func modelCatalogHas(provider, modelType, model string) bool {
	catalog := modelCatalog(provider)
	for _, item := range catalog.Models {
		if item.ModelType == modelType && item.Model == model {
			return true
		}
	}
	return false
}

func modelStatus(catalog modelProviderCatalog, providerState state.WorkspaceModelProviderState, modelState state.WorkspaceModelSetting) string {
	if modelState.Enabled != nil && !*modelState.Enabled {
		return "disabled"
	}
	if modelState.ActiveCredentialID != "" || providerState.ActiveProviderCredentialID != "" {
		return "active"
	}
	if catalog.SystemEnabled && catalog.SystemQuotaValid {
		return "active"
	}
	return "no-configure"
}

func parameterRules(model string) []map[string]any {
	if model == "" {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"name":     "temperature",
			"label":    i18nText("Temperature"),
			"type":     "float",
			"required": false,
			"default":  0.7,
			"min":      0,
			"max":      2,
		},
		{
			"name":     "top_p",
			"label":    i18nText("Top P"),
			"type":     "float",
			"required": false,
			"default":  1,
			"min":      0,
			"max":      1,
		},
		{
			"name":     "max_tokens",
			"label":    i18nText("Max Tokens"),
			"type":     "int",
			"required": false,
			"default":  2048,
			"min":      1,
			"max":      16384,
		},
	}
}

func providerCredentialSchema(label, variable string) []map[string]any {
	return []map[string]any{
		{
			"name":        variable,
			"variable":    variable,
			"label":       i18nText(label),
			"type":        "secret-input",
			"required":    true,
			"show_on":     []any{},
			"placeholder": i18nText("Enter value"),
		},
	}
}

func i18nText(value string) map[string]any {
	return map[string]any{
		"en_US":   value,
		"zh_Hans": value,
	}
}

func hasActiveModel(items []map[string]any) bool {
	for _, item := range items {
		if item["status"] == "active" {
			return true
		}
	}
	return false
}

func credentialList(items []state.WorkspaceCredential) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]any{
			"credential_id":      item.CredentialID,
			"credential_name":    item.Name,
			"from_enterprise":    false,
			"not_allowed_to_use": false,
		})
	}
	return result
}

func credentialName(items []state.WorkspaceCredential, credentialID string) string {
	for _, item := range items {
		if item.CredentialID == credentialID {
			return item.Name
		}
	}
	return ""
}

func activeOrSelectedCredential(items []state.WorkspaceCredential, activeID, selectedID string) state.WorkspaceCredential {
	targetID := firstNonEmpty(strings.TrimSpace(selectedID), strings.TrimSpace(activeID))
	for _, item := range items {
		if item.CredentialID == targetID {
			return item
		}
	}
	if len(items) > 0 {
		return items[0]
	}
	return state.WorkspaceCredential{Credentials: map[string]any{}}
}

func containsCredential(items []state.WorkspaceCredential, credentialID string) bool {
	for _, item := range items {
		if item.CredentialID == credentialID {
			return true
		}
	}
	return false
}

func providerHasConfiguredModels(providerState state.WorkspaceModelProviderState) bool {
	for _, item := range providerState.ModelSettings {
		if item.ActiveCredentialID != "" || len(item.Credentials) > 0 || len(item.LoadBalancing.Configs) > 0 {
			return true
		}
	}
	return false
}

func customModelCredentials(catalog modelProviderCatalog, providerState state.WorkspaceModelProviderState) []map[string]any {
	items := make([]map[string]any, 0)
	for _, item := range providerState.ModelSettings {
		if modelCatalogHas(catalog.Provider, item.ModelType, item.Model) {
			continue
		}
		items = append(items, map[string]any{
			"model":                       item.Model,
			"model_type":                  item.ModelType,
			"credentials":                 map[string]any{},
			"available_model_credentials": credentialList(item.Credentials),
			"current_credential_id":       nullIfEmpty(item.ActiveCredentialID),
			"current_credential_name":     nullIfEmpty(credentialName(item.Credentials, item.ActiveCredentialID)),
		})
	}
	return items
}

func addableCustomModels(catalog modelProviderCatalog) []map[string]any {
	items := make([]map[string]any, 0, len(catalog.Models))
	for _, item := range catalog.Models {
		items = append(items, map[string]any{
			"model":      item.Model,
			"model_type": item.ModelType,
		})
	}
	return items
}

func workspaceLoadBalancingPayload(config state.WorkspaceLoadBalancingConfig) map[string]any {
	items := make([]map[string]any, 0, len(config.Configs))
	for _, entry := range config.Configs {
		items = append(items, map[string]any{
			"id":            nullIfEmpty(entry.ID),
			"name":          entry.Name,
			"enabled":       entry.Enabled,
			"credentials":   cloneJSONObject(entry.Credentials),
			"credential_id": nullIfEmpty(entry.CredentialID),
		})
	}
	return map[string]any{
		"enabled": config.Enabled,
		"configs": items,
	}
}

func normalizeWorkspaceLoadBalancing(config *state.WorkspaceLoadBalancingConfig) {
	if config == nil {
		return
	}
	if config.Configs == nil {
		config.Configs = []state.WorkspaceLoadBalancingEntry{}
	}
	for i := range config.Configs {
		if strings.TrimSpace(config.Configs[i].ID) == "" {
			config.Configs[i].ID = generateRuntimeCredentialID("lb")
		}
		if config.Configs[i].Credentials == nil {
			config.Configs[i].Credentials = map[string]any{}
		}
	}
}

func reflectCredential(credentials map[string]any) string {
	data, err := json.Marshal(credentials)
	if err != nil {
		return ""
	}
	return string(data)
}

func generateRuntimeCredentialID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, generateImportID())
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func providerIconDataURI(text, background, foreground string) string {
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64"><rect width="64" height="64" rx="16" fill="%s"/><text x="32" y="40" text-anchor="middle" font-family="Arial, sans-serif" font-size="28" font-weight="700" fill="%s">%s</text></svg>`,
		background,
		foreground,
		text,
	)
	return "data:image/svg+xml;utf8," + url.QueryEscape(svg)
}
