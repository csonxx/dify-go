package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

type pluginSpec struct {
	UniqueIdentifier    string
	PluginID            string
	Name                string
	Author              string
	Category            string
	Version             string
	Source              string
	Label               string
	Description         string
	Icon                string
	IconDark            string
	MinimumDifyVersion  string
	Tags                []string
	Verified            bool
	Status              string
	DeprecatedReason    string
	AlternativePluginID string
	Meta                state.WorkspacePluginMeta
}

func (s *server) mountWorkspacePluginRoutes(r chi.Router) {
	r.Get("/workspaces/current/plugin/debugging-key", s.handlePluginDebuggingKey)
	r.Get("/workspaces/current/plugin/list", s.handlePluginList)
	r.Post("/workspaces/current/plugin/list/latest-versions", s.handlePluginLatestVersions)
	r.Post("/workspaces/current/plugin/list/installations/ids", s.handlePluginInstalledByIDs)
	r.Get("/workspaces/current/plugin/icon", s.handlePluginIcon)
	r.Get("/workspaces/current/plugin/asset", s.handlePluginAsset)
	r.Post("/workspaces/current/plugin/upload/pkg", s.handlePluginUploadPackage)
	r.Post("/workspaces/current/plugin/upload/github", s.handlePluginUploadGitHub)
	r.Post("/workspaces/current/plugin/upload/bundle", s.handlePluginUploadBundle)
	r.Post("/workspaces/current/plugin/install/pkg", s.handlePluginInstallPackage)
	r.Post("/workspaces/current/plugin/install/github", s.handlePluginInstallGitHub)
	r.Post("/workspaces/current/plugin/install/marketplace", s.handlePluginInstallMarketplace)
	r.Get("/workspaces/current/plugin/marketplace/pkg", s.handlePluginMarketplaceManifest)
	r.Get("/workspaces/current/plugin/fetch-manifest", s.handlePluginFetchManifest)
	r.Get("/workspaces/current/plugin/tasks", s.handlePluginTaskList)
	r.Get("/workspaces/current/plugin/tasks/{taskID}", s.handlePluginTaskDetail)
	r.Post("/workspaces/current/plugin/tasks/{taskID}/delete", s.handlePluginTaskDelete)
	r.Post("/workspaces/current/plugin/tasks/delete_all", s.handlePluginTaskDeleteAll)
	r.Post("/workspaces/current/plugin/tasks/{taskID}/delete/{identifier}", s.handlePluginTaskDeleteItem)
	r.Post("/workspaces/current/plugin/upgrade/marketplace", s.handlePluginUpgradeMarketplace)
	r.Post("/workspaces/current/plugin/upgrade/github", s.handlePluginUpgradeGitHub)
	r.Post("/workspaces/current/plugin/uninstall", s.handlePluginUninstall)
	r.Post("/workspaces/current/plugin/permission/change", s.handlePluginPermissionChange)
	r.Get("/workspaces/current/plugin/permission/fetch", s.handlePluginPermissionFetch)
	r.Get("/workspaces/current/plugin/parameters/dynamic-options", s.handlePluginDynamicOptions)
	r.Post("/workspaces/current/plugin/parameters/dynamic-options-with-credentials", s.handlePluginDynamicOptionsWithCredentials)
	r.Post("/workspaces/current/plugin/preferences/change", s.handlePluginPreferencesChange)
	r.Get("/workspaces/current/plugin/preferences/fetch", s.handlePluginPreferencesFetch)
	r.Post("/workspaces/current/plugin/preferences/autoupgrade/exclude", s.handlePluginPreferencesExclude)
	r.Get("/workspaces/current/plugin/readme", s.handlePluginReadme)

	r.Get("/rag/pipelines/imports/{pipelineID}/check-dependencies", s.handlePipelineImportCheckDependencies)
}

func (s *server) handlePluginDebuggingKey(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost"
	}
	if index := strings.Index(host, ":"); index > 0 {
		host = host[:index]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":  "dbg_" + workspace.ID,
		"host": host,
		"port": 5003,
	})
}

func (s *server) handlePluginList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	page, pageSize := pagingFromRequest(r)
	items := s.store.ListWorkspaceInstalledPlugins(workspace.ID)
	start, end := pageBounds(page, pageSize, len(items))

	payload := make([]map[string]any, 0, end-start)
	for _, item := range items[start:end] {
		payload = append(payload, s.pluginDetailPayload(workspace.ID, item))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plugins": payload,
		"total":   len(items),
	})
}

func (s *server) handlePluginLatestVersions(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginIDs []string `json:"plugin_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	installed := s.store.ListWorkspaceInstalledPluginsByPluginIDs(workspace.ID, payload.PluginIDs)
	lookup := make(map[string]state.WorkspacePluginInstallation, len(installed))
	for _, item := range installed {
		lookup[item.PluginID] = item
	}

	versions := make(map[string]any, len(payload.PluginIDs))
	for _, pluginID := range payload.PluginIDs {
		pluginID = strings.TrimSpace(pluginID)
		if pluginID == "" {
			continue
		}
		if item, ok := lookup[pluginID]; ok {
			versions[pluginID] = map[string]any{
				"unique_identifier":     item.LatestUniqueIdentifier,
				"version":               item.LatestVersion,
				"status":                item.Status,
				"deprecated_reason":     item.DeprecatedReason,
				"alternative_plugin_id": item.AlternativePluginID,
			}
			continue
		}
		versions[pluginID] = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

func (s *server) handlePluginInstalledByIDs(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginIDs []string `json:"plugin_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	items := s.store.ListWorkspaceInstalledPluginsByPluginIDs(workspace.ID, payload.PluginIDs)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, s.pluginDetailPayload(workspace.ID, item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": result})
}

func (s *server) handlePluginIcon(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimSpace(r.URL.Query().Get("filename"))
	if filename == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "filename is required.")
		return
	}

	label := pluginDisplayNameFromFilename(filename)
	dark := strings.Contains(strings.ToLower(filename), "-dark.")
	background := "#E5E7EB"
	foreground := "#111827"
	if dark {
		background = "#111827"
		foreground = "#F9FAFB"
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(pluginGraphicSVG(label, filename, background, foreground))
}

func (s *server) handlePluginAsset(w http.ResponseWriter, r *http.Request) {
	fileName := strings.TrimSpace(r.URL.Query().Get("file_name"))
	if fileName == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "file_name is required.")
		return
	}
	uniqueIdentifier := strings.TrimSpace(r.URL.Query().Get("plugin_unique_identifier"))
	spec := s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "marketplace", state.WorkspacePluginMeta{})

	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(pluginAssetSVG(spec.Label, fileName))
}

func (s *server) handlePluginUploadPackage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserWorkspace(r); !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	fileName, err := multipartFileName(r, "pkg")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	spec := s.localPackageSpec(fileName)
	writeJSON(w, http.StatusOK, map[string]any{
		"unique_identifier": spec.UniqueIdentifier,
		"manifest":          s.pluginManifestPayload(spec),
	})
}

func (s *server) handlePluginUploadGitHub(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserWorkspace(r); !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Repo    string `json:"repo"`
		Version string `json:"version"`
		Package string `json:"package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	spec := s.githubPluginSpec(payload.Repo, payload.Version, payload.Package, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"unique_identifier": spec.UniqueIdentifier,
		"manifest":          s.pluginManifestPayload(spec),
	})
}

func (s *server) handlePluginUploadBundle(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserWorkspace(r); !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	fileName, err := multipartFileName(r, "bundle")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	spec := s.localPackageSpec(fileName)
	writeJSON(w, http.StatusOK, []map[string]any{
		{
			"type": "package",
			"value": map[string]any{
				"unique_identifier": spec.UniqueIdentifier,
				"manifest":          s.pluginManifestPayload(spec),
			},
		},
	})
}

func (s *server) handlePluginInstallPackage(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginUniqueIdentifiers []string `json:"plugin_unique_identifiers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	uniqueIdentifier := firstString(payload.PluginUniqueIdentifiers)
	if uniqueIdentifier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifiers is required.")
		return
	}

	spec := s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "package", state.WorkspacePluginMeta{})
	plugin, task, err := s.store.UpsertWorkspacePlugin(workspace.ID, state.UpsertWorkspacePluginInput{
		PluginUniqueIdentifier: uniqueIdentifier,
		PluginID:               spec.PluginID,
		Name:                   spec.Name,
		Author:                 spec.Author,
		Category:               spec.Category,
		Version:                spec.Version,
		LatestVersion:          spec.Version,
		LatestUniqueIdentifier: spec.UniqueIdentifier,
		Source:                 spec.Source,
		Verified:               spec.Verified,
		Label:                  spec.Label,
		Description:            spec.Description,
		Icon:                   spec.Icon,
		IconDark:               spec.IconDark,
		MinimumDifyVersion:     spec.MinimumDifyVersion,
		Tags:                   spec.Tags,
		Status:                 state.WorkspacePluginStatusActive,
		Meta:                   spec.Meta,
		TaskStatus:             state.WorkspacePluginTaskStatusSuccess,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plugin_unique_identifier": plugin.PluginUniqueIdentifier,
		"all_installed":            false,
		"task_id":                  task.ID,
	})
}

func (s *server) handlePluginInstallGitHub(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginUniqueIdentifier string `json:"plugin_unique_identifier"`
		Repo                   string `json:"repo"`
		Version                string `json:"version"`
		Package                string `json:"package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	spec := s.githubPluginSpec(payload.Repo, payload.Version, payload.Package, payload.PluginUniqueIdentifier)
	plugin, task, err := s.store.UpsertWorkspacePlugin(workspace.ID, state.UpsertWorkspacePluginInput{
		PluginUniqueIdentifier: spec.UniqueIdentifier,
		PluginID:               spec.PluginID,
		Name:                   spec.Name,
		Author:                 spec.Author,
		Category:               spec.Category,
		Version:                spec.Version,
		LatestVersion:          spec.Version,
		LatestUniqueIdentifier: spec.UniqueIdentifier,
		Source:                 spec.Source,
		Verified:               spec.Verified,
		Label:                  spec.Label,
		Description:            spec.Description,
		Icon:                   spec.Icon,
		IconDark:               spec.IconDark,
		MinimumDifyVersion:     spec.MinimumDifyVersion,
		Tags:                   spec.Tags,
		Status:                 state.WorkspacePluginStatusActive,
		Meta:                   spec.Meta,
		TaskStatus:             state.WorkspacePluginTaskStatusSuccess,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plugin_unique_identifier": plugin.PluginUniqueIdentifier,
		"all_installed":            false,
		"task_id":                  task.ID,
	})
}

func (s *server) handlePluginInstallMarketplace(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginUniqueIdentifiers []string `json:"plugin_unique_identifiers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	uniqueIdentifier := firstString(payload.PluginUniqueIdentifiers)
	if uniqueIdentifier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifiers is required.")
		return
	}

	spec := s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "marketplace", state.WorkspacePluginMeta{})
	plugin, task, err := s.store.UpsertWorkspacePlugin(workspace.ID, state.UpsertWorkspacePluginInput{
		PluginUniqueIdentifier: uniqueIdentifier,
		PluginID:               spec.PluginID,
		Name:                   spec.Name,
		Author:                 spec.Author,
		Category:               spec.Category,
		Version:                spec.Version,
		LatestVersion:          spec.Version,
		LatestUniqueIdentifier: spec.UniqueIdentifier,
		Source:                 spec.Source,
		Verified:               spec.Verified,
		Label:                  spec.Label,
		Description:            spec.Description,
		Icon:                   spec.Icon,
		IconDark:               spec.IconDark,
		MinimumDifyVersion:     spec.MinimumDifyVersion,
		Tags:                   spec.Tags,
		Status:                 state.WorkspacePluginStatusActive,
		Meta:                   spec.Meta,
		TaskStatus:             state.WorkspacePluginTaskStatusSuccess,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"plugin_unique_identifier": plugin.PluginUniqueIdentifier,
		"all_installed":            false,
		"task_id":                  task.ID,
	})
}

func (s *server) handlePluginMarketplaceManifest(w http.ResponseWriter, r *http.Request) {
	uniqueIdentifier := strings.TrimSpace(r.URL.Query().Get("plugin_unique_identifier"))
	if uniqueIdentifier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifier is required.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"manifest": s.pluginManifestPayload(s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "marketplace", state.WorkspacePluginMeta{})),
	})
}

func (s *server) handlePluginFetchManifest(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	uniqueIdentifier := strings.TrimSpace(r.URL.Query().Get("plugin_unique_identifier"))
	if uniqueIdentifier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifier is required.")
		return
	}

	if item, ok := s.store.GetWorkspaceInstalledPluginByUniqueIdentifier(workspace.ID, uniqueIdentifier); ok {
		writeJSON(w, http.StatusOK, map[string]any{"manifest": s.pluginManifestPayload(pluginSpecFromInstallation(item))})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"manifest": s.pluginManifestPayload(s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "marketplace", state.WorkspacePluginMeta{})),
	})
}

func (s *server) handlePluginTaskList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	page, pageSize := pagingFromRequest(r)
	items := s.store.ListWorkspacePluginTasks(workspace.ID)
	start, end := pageBounds(page, pageSize, len(items))

	payload := make([]map[string]any, 0, end-start)
	for _, item := range items[start:end] {
		payload = append(payload, pluginTaskPayload(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": payload})
}

func (s *server) handlePluginTaskDetail(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	task, ok := s.store.GetWorkspacePluginTask(workspace.ID, chi.URLParam(r, "taskID"))
	if !ok {
		writeError(w, http.StatusNotFound, "plugin_task_not_found", "Plugin task not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"task": pluginTaskPayload(task)})
}

func (s *server) handlePluginTaskDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if err := s.store.DeleteWorkspacePluginTask(workspace.ID, chi.URLParam(r, "taskID")); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginTaskDeleteAll(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if err := s.store.DeleteAllWorkspacePluginTasks(workspace.ID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginTaskDeleteItem(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if err := s.store.DeleteWorkspacePluginTaskItem(workspace.ID, chi.URLParam(r, "taskID"), chi.URLParam(r, "identifier")); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginUpgradeMarketplace(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		OriginalPluginUniqueIdentifier string `json:"original_plugin_unique_identifier"`
		NewPluginUniqueIdentifier      string `json:"new_plugin_unique_identifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.OriginalPluginUniqueIdentifier) == "" || strings.TrimSpace(payload.NewPluginUniqueIdentifier) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Both original and new plugin unique identifiers are required.")
		return
	}

	source := "marketplace"
	if existing, ok := s.store.GetWorkspaceInstalledPluginByUniqueIdentifier(workspace.ID, payload.OriginalPluginUniqueIdentifier); ok && existing.Source != "" {
		source = existing.Source
	}
	spec := s.pluginSpecFromUniqueIdentifier(payload.NewPluginUniqueIdentifier, source, state.WorkspacePluginMeta{})
	_, task, err := s.store.UpsertWorkspacePlugin(workspace.ID, state.UpsertWorkspacePluginInput{
		OriginalPluginUniqueIdentifier: payload.OriginalPluginUniqueIdentifier,
		PluginUniqueIdentifier:         spec.UniqueIdentifier,
		PluginID:                       spec.PluginID,
		Name:                           spec.Name,
		Author:                         spec.Author,
		Category:                       spec.Category,
		Version:                        spec.Version,
		LatestVersion:                  spec.Version,
		LatestUniqueIdentifier:         spec.UniqueIdentifier,
		Source:                         spec.Source,
		Verified:                       spec.Verified,
		Label:                          spec.Label,
		Description:                    spec.Description,
		Icon:                           spec.Icon,
		IconDark:                       spec.IconDark,
		MinimumDifyVersion:             spec.MinimumDifyVersion,
		Tags:                           spec.Tags,
		Status:                         state.WorkspacePluginStatusActive,
		Meta:                           spec.Meta,
		TaskStatus:                     state.WorkspacePluginTaskStatusSuccess,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"all_installed": false,
		"task_id":       task.ID,
	})
}

func (s *server) handlePluginUpgradeGitHub(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		OriginalPluginUniqueIdentifier string `json:"original_plugin_unique_identifier"`
		NewPluginUniqueIdentifier      string `json:"new_plugin_unique_identifier"`
		Repo                           string `json:"repo"`
		Version                        string `json:"version"`
		Package                        string `json:"package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.OriginalPluginUniqueIdentifier) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "original_plugin_unique_identifier is required.")
		return
	}

	spec := s.githubPluginSpec(payload.Repo, payload.Version, payload.Package, payload.NewPluginUniqueIdentifier)
	_, task, err := s.store.UpsertWorkspacePlugin(workspace.ID, state.UpsertWorkspacePluginInput{
		OriginalPluginUniqueIdentifier: payload.OriginalPluginUniqueIdentifier,
		PluginUniqueIdentifier:         spec.UniqueIdentifier,
		PluginID:                       spec.PluginID,
		Name:                           spec.Name,
		Author:                         spec.Author,
		Category:                       spec.Category,
		Version:                        spec.Version,
		LatestVersion:                  spec.Version,
		LatestUniqueIdentifier:         spec.UniqueIdentifier,
		Source:                         spec.Source,
		Verified:                       spec.Verified,
		Label:                          spec.Label,
		Description:                    spec.Description,
		Icon:                           spec.Icon,
		IconDark:                       spec.IconDark,
		MinimumDifyVersion:             spec.MinimumDifyVersion,
		Tags:                           spec.Tags,
		Status:                         state.WorkspacePluginStatusActive,
		Meta:                           spec.Meta,
		TaskStatus:                     state.WorkspacePluginTaskStatusSuccess,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"all_installed": false,
		"task_id":       task.ID,
	})
}

func (s *server) handlePluginUninstall(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginInstallationID string `json:"plugin_installation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.DeleteWorkspacePluginInstallation(workspace.ID, payload.PluginInstallationID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginPermissionChange(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !isAdminOrOwner(user) {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners or admins can change plugin permissions.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	current, _ := s.store.GetWorkspacePluginPreferences(workspace.ID)
	var payload struct {
		InstallPermission string `json:"install_permission"`
		DebugPermission   string `json:"debug_permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	current.Permission.InstallPermission = firstNonEmpty(payload.InstallPermission, current.Permission.InstallPermission)
	current.Permission.DebugPermission = firstNonEmpty(payload.DebugPermission, current.Permission.DebugPermission)

	if err := s.store.UpdateWorkspacePluginPreferences(workspace.ID, current); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginPermissionFetch(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	preferences, _ := s.store.GetWorkspacePluginPreferences(workspace.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"install_permission": preferences.Permission.InstallPermission,
		"debug_permission":   preferences.Permission.DebugPermission,
	})
}

func (s *server) handlePluginDynamicOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"options": []any{}})
}

func (s *server) handlePluginDynamicOptionsWithCredentials(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"options": []any{}})
}

func (s *server) handlePluginPreferencesChange(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if !isAdminOrOwner(user) {
		writeError(w, http.StatusForbidden, "forbidden", "Only workspace owners or admins can change plugin preferences.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Permission struct {
			InstallPermission string `json:"install_permission"`
			DebugPermission   string `json:"debug_permission"`
		} `json:"permission"`
		AutoUpgrade struct {
			StrategySetting  string   `json:"strategy_setting"`
			UpgradeTimeOfDay int      `json:"upgrade_time_of_day"`
			UpgradeMode      string   `json:"upgrade_mode"`
			ExcludePlugins   []string `json:"exclude_plugins"`
			IncludePlugins   []string `json:"include_plugins"`
		} `json:"auto_upgrade"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	preferences := state.WorkspacePluginPreferences{
		Permission: state.WorkspacePluginPermissionSettings{
			InstallPermission: payload.Permission.InstallPermission,
			DebugPermission:   payload.Permission.DebugPermission,
		},
		AutoUpgrade: state.WorkspacePluginAutoUpgradeSettings{
			StrategySetting:  payload.AutoUpgrade.StrategySetting,
			UpgradeTimeOfDay: payload.AutoUpgrade.UpgradeTimeOfDay,
			UpgradeMode:      payload.AutoUpgrade.UpgradeMode,
			ExcludePlugins:   payload.AutoUpgrade.ExcludePlugins,
			IncludePlugins:   payload.AutoUpgrade.IncludePlugins,
		},
	}
	if err := s.store.UpdateWorkspacePluginPreferences(workspace.ID, preferences); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginPreferencesFetch(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	preferences, _ := s.store.GetWorkspacePluginPreferences(workspace.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"permission": map[string]any{
			"install_permission": preferences.Permission.InstallPermission,
			"debug_permission":   preferences.Permission.DebugPermission,
		},
		"auto_upgrade": map[string]any{
			"strategy_setting":    preferences.AutoUpgrade.StrategySetting,
			"upgrade_time_of_day": preferences.AutoUpgrade.UpgradeTimeOfDay,
			"upgrade_mode":        preferences.AutoUpgrade.UpgradeMode,
			"exclude_plugins":     preferences.AutoUpgrade.ExcludePlugins,
			"include_plugins":     preferences.AutoUpgrade.IncludePlugins,
		},
	})
}

func (s *server) handlePluginPreferencesExclude(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		PluginID string `json:"plugin_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if err := s.store.ExcludeWorkspacePluginAutoUpgrade(workspace.ID, payload.PluginID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *server) handlePluginReadme(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	uniqueIdentifier := strings.TrimSpace(r.URL.Query().Get("plugin_unique_identifier"))
	if uniqueIdentifier == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "plugin_unique_identifier is required.")
		return
	}

	spec := s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "marketplace", state.WorkspacePluginMeta{})
	if item, ok := s.store.GetWorkspaceInstalledPluginByUniqueIdentifier(workspace.ID, uniqueIdentifier); ok {
		spec = pluginSpecFromInstallation(item)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"readme": pluginReadmeMarkdown(spec),
	})
}

func (s *server) handleAppImportCheckDependencies(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"leaked_dependencies": []any{}})
}

func (s *server) handlePipelineImportCheckDependencies(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(chi.URLParam(r, "pipelineID")) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "pipelineID is required.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"leaked_dependencies": []any{}})
}

func (s *server) pluginDetailPayload(workspaceID string, plugin state.WorkspacePluginInstallation) map[string]any {
	settings, _ := s.store.GetWorkspaceToolSettings(workspaceID)
	endpointsSetups := 0
	endpointsActive := 0
	for _, endpoint := range settings.Endpoints {
		if endpoint.PluginID != plugin.PluginID {
			continue
		}
		endpointsSetups++
		if endpoint.Enabled {
			endpointsActive++
		}
	}

	metaPayload := any(nil)
	if strings.TrimSpace(plugin.Meta.Repo) != "" || strings.TrimSpace(plugin.Meta.Version) != "" || strings.TrimSpace(plugin.Meta.Package) != "" {
		metaPayload = map[string]any{
			"repo":    nullIfEmpty(plugin.Meta.Repo),
			"version": nullIfEmpty(firstNonEmpty(plugin.Meta.Version, plugin.Version)),
			"package": nullIfEmpty(plugin.Meta.Package),
		}
	}

	return map[string]any{
		"id":                       plugin.ID,
		"created_at":               unixToRFC3339(plugin.CreatedAt),
		"updated_at":               unixToRFC3339(plugin.UpdatedAt),
		"name":                     plugin.Name,
		"plugin_id":                plugin.PluginID,
		"plugin_unique_identifier": plugin.PluginUniqueIdentifier,
		"declaration":              s.pluginManifestPayload(pluginSpecFromInstallation(plugin)),
		"installation_id":          plugin.ID,
		"tenant_id":                workspaceID,
		"endpoints_setups":         endpointsSetups,
		"endpoints_active":         endpointsActive,
		"version":                  plugin.Version,
		"latest_version":           plugin.LatestVersion,
		"latest_unique_identifier": plugin.LatestUniqueIdentifier,
		"source":                   plugin.Source,
		"meta":                     metaPayload,
		"status":                   plugin.Status,
		"deprecated_reason":        plugin.DeprecatedReason,
		"alternative_plugin_id":    plugin.AlternativePluginID,
	}
}

func (s *server) pluginManifestPayload(spec pluginSpec) map[string]any {
	manifest := map[string]any{
		"plugin_id":                spec.PluginID,
		"plugin_unique_identifier": spec.UniqueIdentifier,
		"version":                  spec.Version,
		"author":                   spec.Author,
		"icon":                     spec.Icon,
		"icon_dark":                spec.IconDark,
		"name":                     spec.Name,
		"category":                 spec.Category,
		"label":                    localizedText(spec.Label),
		"description":              localizedText(spec.Description),
		"created_at":               time.Now().UTC().Format(time.RFC3339),
		"resource":                 map[string]any{},
		"plugins":                  map[string]any{},
		"verified":                 spec.Verified,
		"endpoint":                 map[string]any{"settings": []any{}, "endpoints": []any{}},
		"model":                    map[string]any{},
		"tags":                     spec.Tags,
		"agent_strategy":           map[string]any{},
		"meta": map[string]any{
			"version":              spec.Version,
			"minimum_dify_version": spec.MinimumDifyVersion,
		},
		"trigger": map[string]any{
			"events": []any{},
			"identity": map[string]any{
				"author": spec.Author,
				"name":   spec.Name,
				"label":  localizedText(spec.Label),
			},
			"subscription_constructor": map[string]any{
				"credentials_schema": []any{},
				"oauth_schema": map[string]any{
					"client_schema":      []any{},
					"credentials_schema": []any{},
				},
				"parameters": []any{},
			},
			"subscription_schema": []any{},
		},
	}

	switch spec.Category {
	case "tool", "model", "datasource":
		manifest["tool"] = map[string]any{
			"identity": map[string]any{
				"author":      spec.Author,
				"name":        spec.Name,
				"description": localizedText(spec.Description),
				"icon":        spec.Icon,
				"label":       localizedText(spec.Label),
				"tags":        spec.Tags,
			},
			"credentials_schema": []any{},
		}
	case "extension":
		manifest["endpoint"] = map[string]any{
			"settings": []map[string]any{
				{
					"name":        "secret",
					"label":       localizedText("Secret"),
					"placeholder": localizedText("Optional shared secret"),
					"type":        "text-input",
					"required":    false,
				},
			},
			"endpoints": []map[string]any{
				{"path": "", "method": "POST"},
			},
		}
	case "trigger":
		manifest["trigger"] = map[string]any{
			"events": []map[string]any{
				{
					"name": "event",
					"identity": map[string]any{
						"author":   spec.Author,
						"name":     spec.Name,
						"label":    localizedText(spec.Label),
						"provider": spec.PluginID,
					},
					"description":   localizedText(spec.Description),
					"parameters":    []any{},
					"output_schema": map[string]any{},
				},
			},
			"identity": map[string]any{
				"author": spec.Author,
				"name":   spec.Name,
				"label":  localizedText(spec.Label),
			},
			"subscription_constructor": map[string]any{
				"credentials_schema": []any{},
				"oauth_schema": map[string]any{
					"client_schema":      []any{},
					"credentials_schema": []any{},
				},
				"parameters": []any{},
			},
			"subscription_schema": []any{},
		}
	}

	return manifest
}

func pluginTaskPayload(task state.WorkspacePluginTask) map[string]any {
	plugins := make([]map[string]any, 0, len(task.Plugins))
	for _, item := range task.Plugins {
		plugins = append(plugins, map[string]any{
			"plugin_unique_identifier": item.PluginUniqueIdentifier,
			"plugin_id":                item.PluginID,
			"source":                   item.Source,
			"status":                   item.Status,
			"message":                  item.Message,
			"icon":                     item.Icon,
			"labels":                   localizedText(item.Label),
		})
	}

	return map[string]any{
		"id":                task.ID,
		"created_at":        unixToRFC3339(task.CreatedAt),
		"updated_at":        unixToRFC3339(task.UpdatedAt),
		"status":            task.Status,
		"total_plugins":     task.TotalPlugins,
		"completed_plugins": task.CompletedPlugins,
		"plugins":           plugins,
	}
}

func pluginSpecFromInstallation(item state.WorkspacePluginInstallation) pluginSpec {
	return pluginSpec{
		UniqueIdentifier:    item.PluginUniqueIdentifier,
		PluginID:            item.PluginID,
		Name:                item.Name,
		Author:              item.Author,
		Category:            item.Category,
		Version:             item.Version,
		Source:              item.Source,
		Label:               item.Label,
		Description:         item.Description,
		Icon:                item.Icon,
		IconDark:            item.IconDark,
		MinimumDifyVersion:  item.MinimumDifyVersion,
		Tags:                item.Tags,
		Verified:            item.Verified,
		Status:              item.Status,
		DeprecatedReason:    item.DeprecatedReason,
		AlternativePluginID: item.AlternativePluginID,
		Meta:                item.Meta,
	}
}

func (s *server) pluginSpecFromUniqueIdentifier(uniqueIdentifier, source string, meta state.WorkspacePluginMeta) pluginSpec {
	uniqueIdentifier = strings.TrimSpace(uniqueIdentifier)
	pluginID := pluginIDFromUniqueIdentifier(uniqueIdentifier)
	if pluginID == "" {
		pluginID = firstNonEmpty(strings.TrimSpace(meta.Repo), uniqueIdentifier)
	}

	version := firstNonEmpty(strings.TrimSpace(meta.Version), pluginVersionFromUniqueIdentifier(uniqueIdentifier), "1.0.0")
	author := pluginAuthor(pluginID)
	name := pluginName(pluginID)
	if source == "github" && strings.TrimSpace(meta.Repo) != "" {
		author = pluginAuthor(meta.Repo)
		name = pluginName(meta.Repo)
		pluginID = firstNonEmpty(pluginID, strings.TrimSpace(meta.Repo))
	}
	label := pluginLabel(pluginID, name)
	category := inferPluginCategory(pluginID, source)

	return pluginSpec{
		UniqueIdentifier:    firstNonEmpty(uniqueIdentifier, pluginID+"@"+version),
		PluginID:            pluginID,
		Name:                name,
		Author:              author,
		Category:            category,
		Version:             version,
		Source:              firstNonEmpty(source, "marketplace"),
		Label:               label,
		Description:         fmt.Sprintf("%s plugin managed by dify-go during the Go migration.", label),
		Icon:                pluginIconFilename(pluginID, false),
		IconDark:            pluginIconFilename(pluginID, true),
		MinimumDifyVersion:  "0.0.0",
		Tags:                pluginTags(category, source),
		Verified:            source == "marketplace" || strings.HasPrefix(pluginID, "langgenius/"),
		Status:              state.WorkspacePluginStatusActive,
		DeprecatedReason:    "",
		AlternativePluginID: "",
		Meta: state.WorkspacePluginMeta{
			Repo:    strings.TrimSpace(meta.Repo),
			Version: version,
			Package: strings.TrimSpace(meta.Package),
		},
	}
}

func (s *server) githubPluginSpec(repo, version, pkg, uniqueIdentifier string) pluginSpec {
	repo = normalizeRepository(repo)
	meta := state.WorkspacePluginMeta{
		Repo:    repo,
		Version: firstNonEmpty(strings.TrimSpace(version), "1.0.0"),
		Package: strings.TrimSpace(pkg),
	}
	if strings.TrimSpace(uniqueIdentifier) == "" {
		uniqueIdentifier = fmt.Sprintf("%s@%s", repo, meta.Version)
	}
	return s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "github", meta)
}

func (s *server) localPackageSpec(fileName string) pluginSpec {
	fileName = strings.TrimSpace(fileName)
	base := fileName
	if index := strings.LastIndex(base, "."); index > 0 {
		base = base[:index]
	}
	base = normalizeIdentifier(base)
	if base == "" {
		base = "package-plugin"
	}
	version := "1.0.0"
	uniqueIdentifier := fmt.Sprintf("local/%s@%s", base, version)
	return s.pluginSpecFromUniqueIdentifier(uniqueIdentifier, "package", state.WorkspacePluginMeta{
		Package: fileName,
		Version: version,
	})
}

func pluginReadmeMarkdown(spec pluginSpec) string {
	lines := []string{
		"# " + spec.Label,
		"",
		"This plugin entry is currently served by `dify-go` as part of the incremental Go backend migration.",
		"",
		"- Source: `" + firstNonEmpty(spec.Source, "marketplace") + "`",
		"- Plugin ID: `" + firstNonEmpty(spec.PluginID, spec.UniqueIdentifier) + "`",
		"- Unique Identifier: `" + spec.UniqueIdentifier + "`",
		"- Version: `" + firstNonEmpty(spec.Meta.Version, spec.Version) + "`",
	}
	if strings.TrimSpace(spec.Meta.Repo) != "" {
		lines = append(lines, "- Repository: `"+spec.Meta.Repo+"`")
	}
	if strings.TrimSpace(spec.Meta.Package) != "" {
		lines = append(lines, "- Package: `"+spec.Meta.Package+"`")
	}
	lines = append(lines,
		"",
		"![Overview](./_assets/overview.svg)",
		"",
		"## Notes",
		"",
		"- The frontend is kept unchanged and continues to call Dify-compatible APIs.",
		"- This compatibility implementation keeps installation metadata, task polling, README rendering, and basic plugin management on the Go side.",
	)
	return strings.Join(lines, "\n")
}

func multipartFileName(r *http.Request, field string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return "", fmt.Errorf("%s is required", field)
	}
	defer file.Close()
	if _, err := io.Copy(io.Discard, file); err != nil {
		return "", fmt.Errorf("read %s: %w", field, err)
	}
	return strings.TrimSpace(header.Filename), nil
}

func pluginAuthor(pluginID string) string {
	parts := strings.Split(strings.TrimSpace(pluginID), "/")
	if len(parts) > 1 {
		return parts[0]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return "local"
}

func pluginName(pluginID string) string {
	parts := strings.Split(strings.TrimSpace(pluginID), "/")
	if len(parts) == 0 {
		return "plugin"
	}
	name := parts[len(parts)-1]
	if strings.TrimSpace(name) == "" {
		return "plugin"
	}
	return name
}

func pluginLabel(pluginID, name string) string {
	if label, ok := map[string]string{
		"langgenius/openai":    "OpenAI",
		"langgenius/anthropic": "Anthropic",
		"langgenius/gemini":    "Gemini",
		"langgenius/x":         "xAI",
		"langgenius/deepseek":  "DeepSeek",
		"langgenius/tongyi":    "Tongyi",
	}[pluginID]; ok {
		return label
	}
	return humanizeIdentifier(firstNonEmpty(name, pluginID))
}

func inferPluginCategory(pluginID, source string) string {
	switch {
	case strings.HasPrefix(pluginID, "langgenius/openai"),
		strings.HasPrefix(pluginID, "langgenius/anthropic"),
		strings.HasPrefix(pluginID, "langgenius/gemini"),
		strings.HasPrefix(pluginID, "langgenius/x"),
		strings.HasPrefix(pluginID, "langgenius/deepseek"),
		strings.HasPrefix(pluginID, "langgenius/tongyi"),
		strings.Contains(pluginID, "/model"):
		return "model"
	case strings.Contains(pluginID, "trigger"):
		return "trigger"
	case strings.Contains(pluginID, "datasource"):
		return "datasource"
	case strings.Contains(pluginID, "agent"):
		return "agent-strategy"
	case strings.Contains(pluginID, "endpoint"), strings.Contains(pluginID, "extension"):
		return "extension"
	case source == "package" && strings.Contains(pluginID, "local/extension"):
		return "extension"
	default:
		return "tool"
	}
}

func pluginTags(category, source string) []string {
	items := []string{category}
	if strings.TrimSpace(source) != "" {
		items = append(items, source)
	}
	return items
}

func pluginVersionFromUniqueIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.LastIndex(value, "@"); index > 0 && index+1 < len(value) {
		return strings.TrimSpace(value[index+1:])
	}
	lastSlash := strings.LastIndex(value, "/")
	if index := strings.LastIndex(value, ":"); index > lastSlash && index+1 < len(value) {
		return strings.TrimSpace(value[index+1:])
	}
	return ""
}

func pluginIconFilename(pluginID string, dark bool) string {
	base := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(pluginID), "/", "-"), "_", "-"), "-")
	if base == "" {
		base = "plugin"
	}
	if dark {
		return base + "-dark.svg"
	}
	return base + ".svg"
}

func pluginDisplayNameFromFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if index := strings.LastIndex(filename, "."); index > 0 {
		filename = filename[:index]
	}
	filename = strings.TrimSuffix(filename, "-dark")
	return humanizeIdentifier(strings.ReplaceAll(filename, "-", "_"))
}

func pluginGraphicSVG(label, filename, background, foreground string) []byte {
	initial := "P"
	label = strings.TrimSpace(label)
	if label != "" {
		initial = strings.ToUpper(label[:1])
	}
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="96" height="96" viewBox="0 0 96 96"><rect width="96" height="96" rx="24" fill="%s"/><text x="48" y="56" text-anchor="middle" font-family="Arial, sans-serif" font-size="34" font-weight="700" fill="%s">%s</text><text x="48" y="82" text-anchor="middle" font-family="Arial, sans-serif" font-size="8" fill="%s">%s</text></svg>`,
		background,
		foreground,
		initial,
		foreground,
		xmlEscape(filename),
	)
	return []byte(svg)
}

func pluginAssetSVG(label, fileName string) []byte {
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="960" height="480" viewBox="0 0 960 480"><defs><linearGradient id="g" x1="0" x2="1"><stop offset="0%%" stop-color="#F97316"/><stop offset="100%%" stop-color="#FDBA74"/></linearGradient></defs><rect width="960" height="480" rx="32" fill="#FFF7ED"/><rect x="40" y="40" width="880" height="400" rx="28" fill="url(#g)" opacity="0.12"/><text x="72" y="144" font-family="Arial, sans-serif" font-size="44" font-weight="700" fill="#9A3412">%s</text><text x="72" y="206" font-family="Arial, sans-serif" font-size="24" fill="#C2410C">Compatibility asset rendered by dify-go</text><text x="72" y="256" font-family="Arial, sans-serif" font-size="18" fill="#C2410C">File: %s</text></svg>`,
		xmlEscape(firstNonEmpty(label, "Plugin Asset")),
		xmlEscape(fileName),
	)
	return []byte(svg)
}

func normalizeRepository(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	return strings.Trim(repo, "/")
}

func firstString(items []string) string {
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func isAdminOrOwner(user state.User) bool {
	role := strings.ToLower(strings.TrimSpace(user.Role))
	return role == "owner" || role == "admin"
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}
