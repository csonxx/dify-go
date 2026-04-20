package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleWorkflowDraftGet(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, s.workflowResponse(*app.WorkflowDraft))
}

func (s *server) handleWorkflowDraftSync(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}

	var payload struct {
		Graph                 map[string]any   `json:"graph"`
		Features              map[string]any   `json:"features"`
		EnvironmentVariables  []map[string]any `json:"environment_variables"`
		ConversationVariables []map[string]any `json:"conversation_variables"`
		RagPipelineVariables  []map[string]any `json:"rag_pipeline_variables"`
		Hash                  string           `json:"hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	workflow, err := s.store.SyncWorkflowDraft(app.ID, app.WorkspaceID, currentUser(r), payload.Graph, payload.Features, payload.EnvironmentVariables, payload.ConversationVariables, payload.RagPipelineVariables, payload.Hash, time.Now())
	if err != nil {
		if err.Error() == "draft_workflow_not_sync" {
			writeError(w, http.StatusConflict, "draft_workflow_not_sync", "Draft workflow is out of sync.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":     "success",
		"updated_at": workflow.UpdatedAt,
		"hash":       workflow.Hash,
	})
}

func (s *server) handleWorkflowDraftEnvironmentVariables(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	items, _, exists := s.store.WorkflowEnvironmentVariables(app.ID, app.WorkspaceID)
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *server) handleWorkflowDraftConversationVariables(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	items := []map[string]any{}
	if app.WorkflowDraft != nil {
		items = append(items, cloneMapList(app.WorkflowDraft.ConversationVariables)...)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": buildInspectVars(items, "conversation")})
}

func (s *server) handleWorkflowDraftEnvironmentVariablesUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		EnvironmentVariables []map[string]any `json:"environment_variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateWorkflowEnvironmentVariables(app.ID, app.WorkspaceID, currentUser(r), payload.EnvironmentVariables, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftSystemVariables(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": []map[string]any{
			inspectVar("sys_query", "sys", "query", "Current user query", []string{"sys", "query"}, "string", "", false),
			inspectVar("sys_files", "sys", "files", "Uploaded files", []string{"sys", "files"}, "array[file]", []any{}, false),
			inspectVar("sys_conversation_id", "sys", "conversation_id", "Conversation identifier", []string{"sys", "conversation_id"}, "string", "", false),
		},
	})
}

func (s *server) handleWorkflowDraftVariables(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	items := []map[string]any{}
	if app.WorkflowDraft != nil {
		items = append(items, buildInspectVars(app.WorkflowDraft.EnvironmentVariables, "env")...)
		items = append(items, buildInspectVars(app.WorkflowDraft.ConversationVariables, "conversation")...)
	}
	items = append(items,
		inspectVar("sys_query", "sys", "query", "Current user query", []string{"sys", "query"}, "string", "", false),
		inspectVar("sys_files", "sys", "files", "Uploaded files", []string{"sys", "files"}, "array[file]", []any{}, false),
		inspectVar("sys_conversation_id", "sys", "conversation_id", "Conversation identifier", []string{"sys", "conversation_id"}, "string", "", false),
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"total": len(items),
		"items": items,
	})
}

func (s *server) handleWorkflowDraftNodeVariables(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
}

func (s *server) handleWorkflowDraftConversationVariablesUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		ConversationVariables []map[string]any `json:"conversation_variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateWorkflowConversationVariables(app.ID, app.WorkspaceID, currentUser(r), payload.ConversationVariables, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftFeaturesUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		Features map[string]any `json:"features"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if _, err := s.store.UpdateWorkflowFeatures(app.ID, app.WorkspaceID, currentUser(r), payload.Features, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDefaultBlockConfigs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, defaultWorkflowBlockConfigs(""))
}

func (s *server) handleWorkflowDefaultBlockConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	blockType := strings.TrimSpace(chi.URLParam(r, "blockType"))
	configType := parseCodeLanguageQuery(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"type":   blockType,
		"config": workflowBlockConfig(blockType, configType),
	})
}

func (s *server) handleWorkflowPublishedGet(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}

	if app.WorkflowPublished != nil {
		writeJSON(w, http.StatusOK, s.workflowResponse(*app.WorkflowPublished))
		return
	}
	if app.WorkflowDraft != nil {
		fallback := *app.WorkflowDraft
		fallback.CreatedAt = 0
		fallback.UpdatedAt = 0
		fallback.ToolPublished = false
		writeJSON(w, http.StatusOK, s.workflowResponse(fallback))
		return
	}

	writeError(w, http.StatusNotFound, "published_workflow_not_exist", "Published workflow does not exist.")
}

func (s *server) handleWorkflowPublish(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}

	var payload struct {
		MarkedName    string `json:"marked_name"`
		MarkedComment string `json:"marked_comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	published, err := s.store.PublishWorkflow(app.ID, app.WorkspaceID, currentUser(r), payload.MarkedName, payload.MarkedComment, time.Now())
	if err != nil {
		if err.Error() == "draft_workflow_not_exist" {
			writeError(w, http.StatusNotFound, "draft_workflow_not_exist", "Draft workflow does not exist.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":     "success",
		"created_at": published.CreatedAt,
	})
}

func (s *server) workflowResponse(workflow state.WorkflowState) map[string]any {
	return map[string]any{
		"id":                     workflow.ID,
		"graph":                  workflow.Graph,
		"features":               workflow.Features,
		"created_at":             workflow.CreatedAt,
		"created_by":             s.workflowActor(workflow.CreatedBy),
		"hash":                   workflow.Hash,
		"updated_at":             workflow.UpdatedAt,
		"updated_by":             s.workflowActor(workflow.UpdatedBy),
		"tool_published":         workflow.ToolPublished,
		"environment_variables":  workflow.EnvironmentVariables,
		"conversation_variables": workflow.ConversationVariables,
		"rag_pipeline_variables": workflow.RagPipelineVariables,
		"version":                workflow.Version,
		"marked_name":            workflow.MarkedName,
		"marked_comment":         workflow.MarkedComment,
	}
}

func (s *server) workflowActor(userID string) map[string]any {
	user, ok := s.store.GetUser(userID)
	if !ok {
		return map[string]any{
			"id":    userID,
			"name":  "",
			"email": "",
		}
	}
	return map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	}
}

func isWorkflowApp(app state.App) bool {
	return app.Workflow != nil || app.Mode == "workflow" || app.Mode == "advanced-chat"
}

func defaultWorkflowBlockConfigs(codeLanguage string) []map[string]any {
	types := []string{
		"start",
		"end",
		"answer",
		"llm",
		"knowledge-retrieval",
		"question-classifier",
		"if-else",
		"code",
		"http-request",
		"variable-assigner",
		"variable-aggregator",
		"tool",
		"parameter-extractor",
		"iteration",
		"agent",
		"loop",
		"human-input",
		"trigger-schedule",
		"trigger-webhook",
		"trigger-plugin",
	}

	items := make([]map[string]any, 0, len(types))
	for _, blockType := range types {
		items = append(items, map[string]any{
			"type":   blockType,
			"config": workflowBlockConfig(blockType, codeLanguage),
		})
	}
	return items
}

func workflowBlockConfig(blockType, codeLanguage string) map[string]any {
	switch blockType {
	case "start":
		return map[string]any{"variables": []any{}}
	case "llm":
		return map[string]any{
			"model": map[string]any{
				"provider": "langgenius/openai/openai",
				"name":     "gpt-4o-mini",
				"mode":     "chat",
				"completion_params": map[string]any{
					"temperature": 0.7,
				},
			},
			"prompt_template": []map[string]any{{"role": "system", "text": ""}},
			"context":         map[string]any{"enabled": false, "variable_selector": []any{}},
			"vision":          map[string]any{"enabled": false},
		}
	case "answer":
		return map[string]any{"answer": ""}
	case "code":
		language := strings.TrimSpace(codeLanguage)
		if language == "" {
			language = "python3"
		}
		template := "def main():\n    return {}\n"
		if language == "javascript" {
			template = "function main() {\n  return {}\n}\n"
		}
		return map[string]any{
			"code":          template,
			"code_language": language,
			"variables":     []any{},
			"outputs":       map[string]any{},
		}
	default:
		return map[string]any{}
	}
}

func parseCodeLanguageQuery(r *http.Request) string {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(query), &payload); err != nil {
		return ""
	}
	if value, ok := payload["code_language"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func buildInspectVars(items []map[string]any, kind string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		name, _ := item["name"].(string)
		description, _ := item["description"].(string)
		valueType, _ := item["value_type"].(string)
		value := item["value"]
		if valueType == "" {
			valueType = "string"
		}
		prefix := "env"
		if kind == "conversation" {
			prefix = "conversation"
		}
		out = append(out, inspectVar(stringValueAny(item["id"], name), prefix, name, description, []string{prefix, name}, valueType, value, true))
	}
	return out
}

func inspectVar(id, kind, name, description string, selector []string, valueType string, value any, edited bool) map[string]any {
	return map[string]any{
		"id":           id,
		"type":         kind,
		"name":         name,
		"description":  description,
		"selector":     selector,
		"value_type":   valueType,
		"value":        value,
		"edited":       edited,
		"visible":      true,
		"is_truncated": false,
		"full_content": map[string]any{
			"size_bytes":   0,
			"download_url": "",
		},
	}
}

func cloneMapList(items []map[string]any) []map[string]any {
	if items == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, len(items))
	for i, item := range items {
		out[i] = cloneJSONObject(item)
	}
	return out
}

func stringValueAny(value any, fallback string) string {
	if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
		return strings.TrimSpace(str)
	}
	return fallback
}
