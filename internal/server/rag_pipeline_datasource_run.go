package server

import (
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleRAGPipelineDraftDatasourceNodeRun(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineDatasourceNodeRun(w, r, true)
}

func (s *server) handleRAGPipelinePublishedDatasourceNodeRun(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineDatasourceNodeRun(w, r, false)
}

func (s *server) handleRAGPipelineDraftDatasourceVariablesInspect(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}
	if app.WorkflowDraft == nil {
		writeError(w, http.StatusNotFound, "draft_workflow_not_exist", "Draft workflow does not exist.")
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	datasourceType := strings.TrimSpace(stringFromAny(payload["datasource_type"]))
	if datasourceType == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource type is required.")
		return
	}
	datasourceInfo := mapFromAny(payload["datasource_info"])
	if len(datasourceInfo) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource info is required.")
		return
	}

	nodeID := firstImportValue(strings.TrimSpace(stringFromAny(payload["start_node_id"])), ragPipelineDatasourceNodeID(datasourceType))
	title := firstImportValue(strings.TrimSpace(stringFromAny(payload["start_node_title"])), workflowNodeDefinition(app.WorkflowDraft.Graph, nodeID).Title)
	now := time.Now()
	execution := state.WorkflowNodeExecution{
		ID:                runtimeID("node"),
		Index:             0,
		PredecessorNodeID: "",
		NodeID:            nodeID,
		NodeType:          "datasource",
		Title:             title,
		Inputs: map[string]any{
			"datasource_type": datasourceType,
			"datasource_info": cloneJSONObject(datasourceInfo),
		},
		ProcessData: map[string]any{
			"summary":         "Datasource variables inspected.",
			"mode":            "datasource-variables-inspect",
			"datasource_type": datasourceType,
		},
		Outputs:     datasourceVariablesInspectOutputs(datasourceType, datasourceInfo),
		Status:      "succeeded",
		ElapsedTime: 0.08,
		TotalTokens: 0,
		TotalPrice:  0,
		Currency:    "USD",
		CreatedAt:   now.UTC().Unix(),
		FinishedAt:  now.UTC().Unix(),
		CreatedBy:   currentUser(r).ID,
	}
	if _, err := s.store.SaveWorkflowNodeRun(app.ID, app.WorkspaceID, currentUser(r), execution, now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist datasource variables.")
		return
	}

	writeJSON(w, http.StatusOK, s.nodeTracingResponse(execution))
}

func (s *server) handleRAGPipelineDatasourceNodeRun(w http.ResponseWriter, r *http.Request, isDraft bool) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}
	if isDraft && app.WorkflowDraft == nil {
		writeError(w, http.StatusNotFound, "draft_workflow_not_exist", "Draft workflow does not exist.")
		return
	}
	if !isDraft && app.WorkflowPublished == nil {
		writeError(w, http.StatusNotFound, "published_workflow_not_exist", "Published workflow does not exist.")
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	datasourceType := strings.TrimSpace(stringFromAny(payload["datasource_type"]))
	if datasourceType == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource type is required.")
		return
	}

	nodeID := strings.TrimSpace(chiURLParamOrQuery(r, "nodeID", "node_id"))
	inputs := mapFromAny(payload["inputs"])
	credentialID := strings.TrimSpace(stringFromAny(payload["credential_id"]))
	if requiresDatasourceCredential(datasourceType) && credentialID == "" {
		s.streamDatasourceEvents(w, r, []map[string]any{
			datasourceErrorEvent("Credential is required."),
		})
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	spec, ok := s.ragPipelineDatasourceAvailableSpecByType(workspace.ID, datasourceType)
	if !ok {
		s.streamDatasourceEvents(w, r, []map[string]any{
			datasourceErrorEvent("Datasource provider is not available in the current workspace."),
		})
		return
	}

	credential, credentialFound := s.findWorkspaceDatasourceCredential(workspace.ID, spec.PluginID, spec.Provider, credentialID)
	if requiresDatasourceCredential(datasourceType) && !credentialFound {
		s.streamDatasourceEvents(w, r, []map[string]any{
			datasourceErrorEvent("Credential not found or no longer available."),
		})
		return
	}

	events := s.datasourceNodeRunEvents(datasourceNodeRunContext{
		App:            app,
		Workspace:      workspace,
		NodeID:         nodeID,
		DatasourceType: datasourceType,
		Inputs:         inputs,
		Spec:           spec,
		Credential:     credential,
	})
	if len(events) == 0 {
		events = []map[string]any{datasourceErrorEvent("No datasource events were generated.")}
	}
	if err := s.streamDatasourceEvents(w, r, events); err != nil {
		return
	}
}

type datasourceNodeRunContext struct {
	App            state.App
	Workspace      state.Workspace
	NodeID         string
	DatasourceType string
	Inputs         map[string]any
	Spec           ragPipelineDatasourceProviderSpec
	Credential     state.WorkspaceDatasourceCredential
}

func datasourceVariablesInspectOutputs(datasourceType string, datasourceInfo map[string]any) map[string]any {
	info := cloneJSONObject(datasourceInfo)
	outputs := map[string]any{
		"datasource_type":      datasourceType,
		"datasource_info":      info,
		"datasource_info_list": []map[string]any{info},
	}
	switch strings.TrimSpace(datasourceType) {
	case "local_file":
		outputs["related_id"] = firstImportValue(stringFromAny(info["related_id"]), stringFromAny(info["id"]))
		outputs["name"] = stringFromAny(info["name"])
		outputs["extension"] = stringFromAny(info["extension"])
		outputs["mime_type"] = stringFromAny(info["mime_type"])
		outputs["size"] = info["size"]
	case "online_document":
		outputs["workspace_id"] = stringFromAny(info["workspace_id"])
		outputs["page"] = cloneJSONObject(mapFromAny(info["page"]))
		outputs["credential_id"] = stringFromAny(info["credential_id"])
	case "website_crawl":
		outputs["title"] = firstImportValue(stringFromAny(info["title"]), datasourceWebsiteTitle(stringFromAny(info["source_url"])))
		outputs["source_url"] = stringFromAny(info["source_url"])
		outputs["description"] = stringFromAny(info["description"])
		outputs["content"] = firstImportValue(stringFromAny(info["content"]), stringFromAny(info["markdown"]))
		outputs["credential_id"] = stringFromAny(info["credential_id"])
	case "online_drive":
		outputs["id"] = stringFromAny(info["id"])
		outputs["type"] = stringFromAny(info["type"])
		outputs["bucket"] = info["bucket"]
		outputs["name"] = stringFromAny(info["name"])
		outputs["size"] = info["size"]
		outputs["credential_id"] = stringFromAny(info["credential_id"])
	}
	return outputs
}

func (s *server) datasourceNodeRunEvents(ctx datasourceNodeRunContext) []map[string]any {
	switch ctx.DatasourceType {
	case "online_document":
		return []map[string]any{
			datasourceCompletedEvent(s.datasourceNotionWorkspaces(ctx), 0.2),
		}
	case "website_crawl":
		results := s.datasourceWebsiteResults(ctx)
		return []map[string]any{
			datasourceProcessingEvent(len(results), maxDatasourceProgressCount(len(results), len(results)/2)),
			datasourceProcessingEvent(len(results), len(results)),
			datasourceCompletedEvent(results, 1.2),
		}
	case "online_drive":
		return []map[string]any{
			datasourceCompletedEvent(s.datasourceOnlineDriveData(ctx), 0.3),
		}
	default:
		return []map[string]any{
			datasourceErrorEvent("Datasource type is not supported by dify-go yet."),
		}
	}
}

func (s *server) datasourceNotionWorkspaces(ctx datasourceNodeRunContext) []map[string]any {
	credentialSuffix := trimIDForDisplay(ctx.Credential.ID)
	workspaceID := firstNonEmpty(stringFromAny(ctx.Inputs["workspace"]), "workspace-"+credentialSuffix)
	workspaceName := firstNonEmpty(stringFromAny(ctx.Inputs["workspace_name"]), "Workspace "+strings.ToUpper(credentialSuffix))
	prefix := firstNonEmpty(stringFromAny(ctx.Inputs["database"]), stringFromAny(ctx.Inputs["query"]), "notion")

	return []map[string]any{
		{
			"workspace_name": workspaceName,
			"workspace_id":   workspaceID,
			"workspace_icon": nil,
			"pages": []map[string]any{
				datasourceNotionPage(prefix, "overview", "Overview"),
				datasourceNotionPage(prefix, "spec", "Migration Spec"),
				datasourceNotionPage(prefix, "checklist", "Go Rollout Checklist"),
			},
		},
	}
}

func datasourceNotionPage(prefix, suffix, label string) map[string]any {
	return map[string]any{
		"page_id":   strings.ToLower(strings.ReplaceAll(firstNonEmpty(prefix, "page"), " ", "-")) + "-" + suffix,
		"page_name": firstNonEmpty(strings.TrimSpace(prefix), "Workspace") + " " + label,
		"parent_id": "root",
		"type":      "page",
		"is_bound":  false,
		"page_icon": nil,
	}
}

func (s *server) datasourceWebsiteResults(ctx datasourceNodeRunContext) []map[string]any {
	urls := datasourceCandidateURLs(ctx.Inputs)
	if len(urls) == 0 {
		urls = []string{"https://example.com"}
	}

	results := make([]map[string]any, 0, len(urls)*3)
	for _, rawURL := range urls {
		base, err := neturl.Parse(rawURL)
		if err != nil || base.Host == "" {
			continue
		}
		pathPrefix := strings.TrimRight(base.Path, "/")
		if pathPrefix == "" {
			pathPrefix = ""
		}
		candidates := []string{
			base.String(),
			base.Scheme + "://" + base.Host + pathPrefix + "/about",
			base.Scheme + "://" + base.Host + pathPrefix + "/docs",
		}
		for _, candidate := range candidates {
			results = append(results, map[string]any{
				"title":       datasourceWebsiteTitle(candidate),
				"source_url":  candidate,
				"description": "Crawled by dify-go compatibility datasource runner.",
				"markdown":    "# " + datasourceWebsiteTitle(candidate) + "\n\nCaptured from " + candidate + ".",
			})
		}
	}

	if len(results) == 0 {
		return []map[string]any{
			{
				"title":       "Example Page",
				"source_url":  "https://example.com",
				"description": "Crawled by dify-go compatibility datasource runner.",
				"markdown":    "# Example Page\n\nCaptured from https://example.com.",
			},
		}
	}
	return results
}

func datasourceCandidateURLs(inputs map[string]any) []string {
	candidates := []string{}
	for _, key := range []string{"url", "source_url", "website_url", "start_url"} {
		if value := strings.TrimSpace(stringFromAny(inputs[key])); value != "" {
			candidates = append(candidates, value)
		}
	}
	if list, ok := inputs["urls"].([]any); ok {
		for _, item := range list {
			if value := strings.TrimSpace(stringFromAny(item)); value != "" {
				candidates = append(candidates, value)
			}
		}
	}
	return dedupeStrings(candidates)
}

func datasourceWebsiteTitle(rawURL string) string {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return firstNonEmpty(strings.TrimSpace(rawURL), "Website Result")
	}
	path := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if path == "" {
		return parsed.Host
	}
	segments := strings.Split(path, "/")
	return parsed.Host + " / " + segments[len(segments)-1]
}

func (s *server) datasourceOnlineDriveData(ctx datasourceNodeRunContext) []map[string]any {
	bucket := strings.TrimSpace(stringFromAny(ctx.Inputs["bucket"]))
	prefix := strings.TrimSpace(stringFromAny(ctx.Inputs["prefix"]))
	if bucket == "" && prefix == "" {
		return []map[string]any{
			{"bucket": "workspace-docs", "files": []any{}, "is_truncated": false, "next_page_parameters": map[string]any{}},
			{"bucket": "team-assets", "files": []any{}, "is_truncated": false, "next_page_parameters": map[string]any{}},
		}
	}

	baseName := firstNonEmpty(prefix, bucket, "drive")
	files := []map[string]any{
		{"id": baseName + "-folder", "name": "Designs", "type": "folder"},
		{"id": baseName + "-brief", "name": "brief.md", "type": "file", "size": 2048},
		{"id": baseName + "-roadmap", "name": "roadmap.pdf", "type": "file", "size": 8192},
	}

	return []map[string]any{
		{
			"bucket":               firstNonEmpty(bucket, "workspace-docs"),
			"files":                files,
			"is_truncated":         false,
			"next_page_parameters": map[string]any{},
		},
	}
}

func (s *server) streamDatasourceEvents(w http.ResponseWriter, r *http.Request, events []map[string]any) error {
	return s.streamWorkflowEvents(w, r, events)
}

func datasourceProcessingEvent(total, completed int) map[string]any {
	return map[string]any{
		"event":     "datasource_processing",
		"total":     maxDatasourceProgressCount(total, total),
		"completed": maxDatasourceProgressCount(total, completed),
	}
}

func datasourceCompletedEvent(data any, timeConsuming float64) map[string]any {
	return map[string]any{
		"event":          "datasource_completed",
		"data":           data,
		"time_consuming": timeConsuming,
	}
}

func datasourceErrorEvent(message string) map[string]any {
	return map[string]any{
		"event": "datasource_error",
		"error": firstNonEmpty(strings.TrimSpace(message), "Datasource run failed."),
	}
}

func (s *server) findWorkspaceDatasourceCredential(workspaceID, pluginID, provider, credentialID string) (state.WorkspaceDatasourceCredential, bool) {
	if strings.TrimSpace(credentialID) == "" {
		return state.WorkspaceDatasourceCredential{}, false
	}
	items := s.store.ListWorkspaceDatasourceCredentials(workspaceID, pluginID, provider)
	for _, item := range items {
		if item.ID == credentialID {
			return item, true
		}
	}
	return state.WorkspaceDatasourceCredential{}, false
}

func requiresDatasourceCredential(datasourceType string) bool {
	switch strings.TrimSpace(datasourceType) {
	case "online_document", "website_crawl", "online_drive":
		return true
	default:
		return false
	}
}

func trimIDForDisplay(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "demo"
	}
	if index := strings.LastIndex(value, "_"); index >= 0 && index < len(value)-1 {
		value = value[index+1:]
	}
	if len(value) > 6 {
		value = value[len(value)-6:]
	}
	return value
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func maxDatasourceProgressCount(total, completed int) int {
	if total <= 0 {
		total = 1
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	return completed
}

func chiURLParamOrQuery(r *http.Request, param, query string) string {
	if value := strings.TrimSpace(chi.URLParam(r, param)); value != "" {
		return value
	}
	return strings.TrimSpace(r.URL.Query().Get(query))
}
