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

type workflowRunOptions struct {
	ChatMode        bool
	SelectedNodeIDs []string
	Mode            string
}

type workflowNodeMeta struct {
	ID              string
	NodeType        string
	Title           string
	PredecessorNode string
}

func (s *server) handleWorkflowVersionList(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}

	items, hasMore, page, exists := s.store.ListWorkflowVersions(
		app.ID,
		app.WorkspaceID,
		intQuery(r, "page", 1),
		intQuery(r, "limit", 10),
		strings.TrimSpace(r.URL.Query().Get("user_id")),
		boolQuery(r, "named_only"),
	)
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	responseItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, s.workflowResponse(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":    responseItems,
		"has_more": hasMore,
		"page":     page,
	})
}

func (s *server) handleWorkflowVersionUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
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

	version, err := s.store.UpdateWorkflowVersionMetadata(app.ID, app.WorkspaceID, chi.URLParam(r, "versionID"), currentUser(r), payload.MarkedName, payload.MarkedComment, time.Now())
	if err != nil {
		if err.Error() == "workflow_version_not_found" {
			writeError(w, http.StatusNotFound, "workflow_version_not_found", "Workflow version does not exist.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.workflowResponse(version))
}

func (s *server) handleWorkflowVersionDelete(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	err := s.store.DeleteWorkflowVersion(app.ID, app.WorkspaceID, chi.URLParam(r, "versionID"), currentUser(r), time.Now())
	if err != nil {
		switch err.Error() {
		case "workflow_version_not_found":
			writeError(w, http.StatusNotFound, "workflow_version_not_found", "Workflow version does not exist.")
		case "latest_workflow_version_cannot_delete":
			writeError(w, http.StatusBadRequest, "latest_workflow_version_cannot_delete", "Latest workflow version cannot be deleted.")
		default:
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowVersionRestore(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	version, err := s.store.RestoreWorkflowVersion(app.ID, app.WorkspaceID, chi.URLParam(r, "versionID"), currentUser(r), time.Now())
	if err != nil {
		if err.Error() == "workflow_version_not_found" {
			writeError(w, http.StatusNotFound, "workflow_version_not_found", "Workflow version does not exist.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := s.syncLinkedRAGPipelineDatasetFromWorkflow(app, version, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":     "success",
		"updated_at": version.UpdatedAt,
		"hash":       version.Hash,
	})
}

func (s *server) handleWorkflowRunHistory(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	runs, exists := s.store.ListWorkflowRuns(app.ID, app.WorkspaceID)
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	data := make([]map[string]any, 0, len(runs))
	for _, run := range runs {
		data = append(data, s.workflowRunHistoryResponse(run))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleWorkflowRunDetail(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	run, found, exists := s.store.GetWorkflowRun(app.ID, app.WorkspaceID, chi.URLParam(r, "runID"))
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "workflow_run_not_found", "Workflow run does not exist.")
		return
	}

	writeJSON(w, http.StatusOK, s.workflowRunDetailResponse(run))
}

func (s *server) handleWorkflowRunNodeExecutions(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	run, found, exists := s.store.GetWorkflowRun(app.ID, app.WorkspaceID, chi.URLParam(r, "runID"))
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "workflow_run_not_found", "Workflow run does not exist.")
		return
	}

	data := make([]map[string]any, 0, len(run.NodeExecutions))
	for _, execution := range run.NodeExecutions {
		data = append(data, s.nodeTracingResponse(execution))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleWorkflowRunStop(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	if _, err := s.store.StopWorkflowRun(app.ID, app.WorkspaceID, chi.URLParam(r, "taskID"), currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftRun(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	s.handleWorkflowRunSSEWithPayload(w, r, workflowRunOptions{
		ChatMode: strings.Contains(r.URL.Path, "/advanced-chat/"),
		Mode:     "draft",
	}, payload)
}

func (s *server) handleWorkflowDraftTriggerRun(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	selectedNodeIDs := workflowNodeIDList(payload["node_ids"])
	if len(selectedNodeIDs) == 0 {
		if nodeID := stringFromAny(payload["node_id"]); nodeID != "" {
			selectedNodeIDs = []string{nodeID}
		}
	}
	s.handleWorkflowRunSSEWithPayload(w, r, workflowRunOptions{
		SelectedNodeIDs: selectedNodeIDs,
		Mode:            "trigger",
	}, payload)
}

func (s *server) handleWorkflowDraftTriggerRunAll(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	s.handleWorkflowRunSSEWithPayload(w, r, workflowRunOptions{
		SelectedNodeIDs: workflowNodeIDList(payload["node_ids"]),
		Mode:            "trigger-all",
	}, payload)
}

func (s *server) handleWorkflowRunSSE(w http.ResponseWriter, r *http.Request, options workflowRunOptions) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	s.handleWorkflowRunSSEWithPayload(w, r, options, payload)
}

func (s *server) handleWorkflowRunSSEWithPayload(w http.ResponseWriter, r *http.Request, options workflowRunOptions, payload map[string]any) {
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

	now := time.Now()
	run := s.buildWorkflowRun(app, currentUser(r), payload, options, now)
	if _, err := s.store.SaveWorkflowRun(app.ID, app.WorkspaceID, currentUser(r), run, now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist workflow run.")
		return
	}

	events := s.workflowRunEvents(app, run, options.ChatMode)
	if err := s.streamWorkflowEvents(w, r, events); err != nil {
		return
	}
}

func (s *server) handleWorkflowDraftNodeRun(w http.ResponseWriter, r *http.Request) {
	s.handleWorkflowSingleNodeRun(w, r, false)
}

func (s *server) handleWorkflowDraftNodeTriggerRun(w http.ResponseWriter, r *http.Request) {
	s.handleWorkflowSingleNodeRun(w, r, true)
}

func (s *server) handleWorkflowSingleNodeRun(w http.ResponseWriter, r *http.Request, isTrigger bool) {
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

	execution := s.buildSingleNodeExecution(app, currentUser(r), chi.URLParam(r, "nodeID"), payload, isTrigger, time.Now())
	if _, err := s.store.SaveWorkflowNodeRun(app.ID, app.WorkspaceID, currentUser(r), execution, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist node run.")
		return
	}

	writeJSON(w, http.StatusOK, s.nodeTracingResponse(execution))
}

func (s *server) handleWorkflowDraftIterationNodeRun(w http.ResponseWriter, r *http.Request) {
	s.handleWorkflowStructuredNodeStream(w, r, "iteration")
}

func (s *server) handleWorkflowDraftLoopNodeRun(w http.ResponseWriter, r *http.Request) {
	s.handleWorkflowStructuredNodeStream(w, r, "loop")
}

func (s *server) handleWorkflowStructuredNodeStream(w http.ResponseWriter, r *http.Request, mode string) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
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

	now := time.Now()
	execution := s.buildSingleNodeExecution(app, currentUser(r), chi.URLParam(r, "nodeID"), payload, false, now)
	if _, err := s.store.SaveWorkflowNodeRun(app.ID, app.WorkspaceID, currentUser(r), execution, now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist node run.")
		return
	}

	run := state.WorkflowRun{
		ID:             runtimeID("run"),
		TaskID:         runtimeID("task"),
		Version:        "draft",
		Graph:          cloneJSONObject(app.WorkflowDraft.Graph),
		Inputs:         workflowRunInputs(payload),
		Status:         "succeeded",
		Outputs:        cloneJSONObject(execution.Outputs),
		ElapsedTime:    execution.ElapsedTime,
		TotalTokens:    execution.TotalTokens,
		TotalPrice:     execution.TotalPrice,
		Currency:       execution.Currency,
		TotalSteps:     1,
		CreatedAt:      now.UTC().Unix(),
		FinishedAt:     now.UTC().Unix(),
		CreatedBy:      currentUser(r).ID,
		NodeExecutions: []state.WorkflowNodeExecution{execution},
	}

	events := []map[string]any{
		s.workflowStartedEvent(app, run, strings.Contains(r.URL.Path, "/advanced-chat/")),
		s.workflowRuntimeNodeEvent(run, execution, mode+"_started", execution.Status),
		s.workflowRuntimeNodeEvent(run, execution, "node_started", "running"),
		s.workflowRuntimeNodeEvent(run, execution, "node_finished", execution.Status),
		s.workflowRuntimeNodeEvent(run, execution, mode+"_completed", execution.Status),
		s.workflowFinishedEvent(app, run),
	}
	if err := s.streamWorkflowEvents(w, r, events); err != nil {
		return
	}
}

func (s *server) handleWorkflowDraftNodeLastRun(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	execution, found, exists := s.store.GetWorkflowNodeRun(app.ID, app.WorkspaceID, chi.URLParam(r, "nodeID"))
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "node_run_not_found", "Node last run does not exist.")
		return
	}

	writeJSON(w, http.StatusOK, s.nodeTracingResponse(execution))
}

func (s *server) handleWorkflowDraftVariablesDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftNodeVariablesDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftVariableDelete(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserApp(r); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleWorkflowDraftVariableReset(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	value := any(nil)
	varID := chi.URLParam(r, "varID")
	if app.WorkflowDraft != nil {
		for _, item := range app.WorkflowDraft.EnvironmentVariables {
			if stringFromAny(item["id"]) == varID || stringFromAny(item["name"]) == varID {
				value = item["value"]
				break
			}
		}
		if value == nil {
			for _, item := range app.WorkflowDraft.ConversationVariables {
				if stringFromAny(item["id"]) == varID || stringFromAny(item["name"]) == varID {
					value = item["value"]
					break
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *server) handleConversationVariablesCurrentValues(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	items := make([]map[string]any, 0)
	if app.WorkflowDraft != nil {
		for _, item := range app.WorkflowDraft.ConversationVariables {
			items = append(items, map[string]any{
				"id":         stringValueAny(item["id"], stringFromAny(item["name"])),
				"name":       stringFromAny(item["name"]),
				"value":      item["value"],
				"value_type": stringValueAny(item["value_type"], "string"),
				"updated_at": app.WorkflowDraft.UpdatedAt,
				"created_at": app.WorkflowDraft.CreatedAt,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     items,
		"has_more": false,
		"limit":    len(items),
		"total":    len(items),
		"page":     1,
	})
}

func (s *server) buildWorkflowRun(app state.App, user state.User, payload map[string]any, options workflowRunOptions, now time.Time) state.WorkflowRun {
	nodes := workflowGraphNodes(app.WorkflowDraft.Graph, options.SelectedNodeIDs)
	inputs := workflowRunInputs(payload)
	outputs := workflowRunOutputs(inputs, nodes, options.Mode)
	nodeExecutions := make([]state.WorkflowNodeExecution, 0, len(nodes))
	for index, node := range nodes {
		output := workflowNodeOutputs(node, inputs, outputs, index)
		nodeExecutions = append(nodeExecutions, state.WorkflowNodeExecution{
			ID:                runtimeID("node"),
			Index:             index,
			PredecessorNodeID: node.PredecessorNode,
			NodeID:            node.ID,
			NodeType:          node.NodeType,
			Title:             node.Title,
			Inputs:            workflowNodeInputs(node, inputs, payload),
			ProcessData: map[string]any{
				"summary": fmt.Sprintf("%s completed", node.Title),
				"mode":    options.Mode,
			},
			Outputs:     output,
			Status:      "succeeded",
			ElapsedTime: workflowNodeElapsed(index),
			TotalTokens: 24 + index*8,
			TotalPrice:  0,
			Currency:    "USD",
			CreatedAt:   now.UTC().Unix(),
			FinishedAt:  now.UTC().Unix(),
			CreatedBy:   user.ID,
		})
	}

	run := state.WorkflowRun{
		ID:             runtimeID("run"),
		TaskID:         runtimeID("task"),
		Version:        workflowRunVersion(app),
		Graph:          cloneJSONObject(app.WorkflowDraft.Graph),
		Inputs:         inputs,
		Status:         "succeeded",
		Outputs:        outputs,
		ElapsedTime:    workflowElapsed(len(nodeExecutions)),
		TotalTokens:    len(nodeExecutions) * 32,
		TotalPrice:     0,
		Currency:       "USD",
		TotalSteps:     len(nodeExecutions),
		CreatedAt:      now.UTC().Unix(),
		FinishedAt:     now.UTC().Unix(),
		CreatedBy:      user.ID,
		NodeExecutions: nodeExecutions,
	}
	if options.ChatMode {
		run.ConversationID = runtimeID("conv")
		run.MessageID = runtimeID("msg")
	}
	return run
}

func (s *server) buildSingleNodeExecution(app state.App, user state.User, nodeID string, payload map[string]any, isTrigger bool, now time.Time) state.WorkflowNodeExecution {
	node := workflowNodeDefinition(app.WorkflowDraft.Graph, nodeID)
	inputs := workflowRunInputs(payload)
	mode := "single"
	if isTrigger {
		mode = "trigger"
	}
	outputs := workflowNodeOutputs(node, inputs, map[string]any{
		"text":      fmt.Sprintf("%s executed successfully.", node.Title),
		"node_id":   node.ID,
		"node_type": node.NodeType,
	}, 0)

	return state.WorkflowNodeExecution{
		ID:                runtimeID("node"),
		Index:             0,
		PredecessorNodeID: node.PredecessorNode,
		NodeID:            node.ID,
		NodeType:          node.NodeType,
		Title:             node.Title,
		Inputs:            workflowNodeInputs(node, inputs, payload),
		ProcessData: map[string]any{
			"summary": fmt.Sprintf("%s %s run", node.Title, mode),
			"mode":    mode,
		},
		Outputs:     outputs,
		Status:      "succeeded",
		ElapsedTime: 0.12,
		TotalTokens: 18,
		TotalPrice:  0,
		Currency:    "USD",
		CreatedAt:   now.UTC().Unix(),
		FinishedAt:  now.UTC().Unix(),
		CreatedBy:   user.ID,
	}
}

func (s *server) workflowRunEvents(app state.App, run state.WorkflowRun, includeConversation bool) []map[string]any {
	events := []map[string]any{s.workflowStartedEvent(app, run, includeConversation)}
	for _, execution := range run.NodeExecutions {
		events = append(events, s.workflowRuntimeNodeEvent(run, execution, "node_started", "running"))
		events = append(events, s.workflowRuntimeNodeEvent(run, execution, "node_finished", execution.Status))
	}
	events = append(events, s.workflowFinishedEvent(app, run))
	return events
}

func (s *server) workflowStartedEvent(app state.App, run state.WorkflowRun, includeConversation bool) map[string]any {
	payload := map[string]any{
		"task_id":         run.TaskID,
		"workflow_run_id": run.ID,
		"event":           "workflow_started",
		"data": map[string]any{
			"id":          run.ID,
			"workflow_id": workflowIDForResponse(app),
			"created_at":  run.CreatedAt,
		},
	}
	if includeConversation {
		payload["conversation_id"] = run.ConversationID
		payload["message_id"] = run.MessageID
	}
	return payload
}

func (s *server) workflowFinishedEvent(app state.App, run state.WorkflowRun) map[string]any {
	return map[string]any{
		"task_id":         run.TaskID,
		"workflow_run_id": run.ID,
		"event":           "workflow_finished",
		"data": map[string]any{
			"id":           run.ID,
			"workflow_id":  workflowIDForResponse(app),
			"status":       run.Status,
			"outputs":      cloneJSONObject(run.Outputs),
			"error":        run.Error,
			"elapsed_time": run.ElapsedTime,
			"total_tokens": run.TotalTokens,
			"total_steps":  run.TotalSteps,
			"created_at":   run.CreatedAt,
			"created_by":   s.workflowActor(run.CreatedBy),
			"finished_at":  run.FinishedAt,
			"files":        []any{},
		},
	}
}

func (s *server) workflowRuntimeNodeEvent(run state.WorkflowRun, execution state.WorkflowNodeExecution, eventName, status string) map[string]any {
	nodePayload := s.nodeTracingResponse(execution)
	nodePayload["status"] = status
	if status == "running" {
		delete(nodePayload, "outputs")
		nodePayload["finished_at"] = 0
	}

	return map[string]any{
		"task_id":         run.TaskID,
		"workflow_run_id": run.ID,
		"event":           eventName,
		"data":            nodePayload,
	}
}

func (s *server) streamWorkflowEvents(w http.ResponseWriter, r *http.Request, events []map[string]any) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "Streaming is not supported.")
		return fmt.Errorf("streaming unsupported")
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for _, event := range events {
		if err := writeWorkflowEvent(w, event); err != nil {
			return err
		}
		flusher.Flush()
		if err := r.Context().Err(); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) workflowRunHistoryResponse(run state.WorkflowRun) map[string]any {
	return map[string]any{
		"id":                 run.ID,
		"version":            run.Version,
		"conversation_id":    nullIfEmpty(run.ConversationID),
		"message_id":         nullIfEmpty(run.MessageID),
		"graph":              cloneJSONObject(run.Graph),
		"inputs":             cloneJSONObject(run.Inputs),
		"status":             run.Status,
		"outputs":            cloneJSONObject(run.Outputs),
		"error":              nullIfEmpty(run.Error),
		"elapsed_time":       run.ElapsedTime,
		"total_tokens":       run.TotalTokens,
		"total_steps":        run.TotalSteps,
		"created_at":         run.CreatedAt,
		"finished_at":        run.FinishedAt,
		"created_by_account": s.workflowActor(run.CreatedBy),
	}
}

func (s *server) workflowRunDetailResponse(run state.WorkflowRun) map[string]any {
	return map[string]any{
		"id":                run.ID,
		"version":           run.Version,
		"graph":             cloneJSONObject(run.Graph),
		"inputs":            stringifyWorkflowValue(run.Inputs),
		"inputs_truncated":  false,
		"status":            run.Status,
		"outputs":           stringifyWorkflowValue(run.Outputs),
		"outputs_truncated": false,
		"outputs_full_content": map[string]any{
			"download_url": "",
		},
		"error":              nullIfEmpty(run.Error),
		"elapsed_time":       run.ElapsedTime,
		"total_tokens":       run.TotalTokens,
		"total_steps":        run.TotalSteps,
		"created_by_role":    "account",
		"created_by_account": s.workflowActor(run.CreatedBy),
		"created_at":         run.CreatedAt,
		"finished_at":        run.FinishedAt,
		"exceptions_count":   0,
	}
}

func (s *server) nodeTracingResponse(execution state.WorkflowNodeExecution) map[string]any {
	return map[string]any{
		"id":                     execution.ID,
		"index":                  execution.Index,
		"predecessor_node_id":    execution.PredecessorNodeID,
		"node_id":                execution.NodeID,
		"node_type":              execution.NodeType,
		"title":                  execution.Title,
		"inputs":                 cloneJSONObject(execution.Inputs),
		"inputs_truncated":       false,
		"process_data":           cloneJSONObject(execution.ProcessData),
		"process_data_truncated": false,
		"outputs":                cloneJSONObject(execution.Outputs),
		"outputs_truncated":      false,
		"outputs_full_content": map[string]any{
			"download_url": "",
		},
		"status":       execution.Status,
		"error":        nullIfEmpty(execution.Error),
		"elapsed_time": execution.ElapsedTime,
		"execution_metadata": map[string]any{
			"total_tokens": execution.TotalTokens,
			"total_price":  execution.TotalPrice,
			"currency":     firstImportValue(execution.Currency, "USD"),
		},
		"metadata": map[string]any{
			"iterator_length": 0,
			"iterator_index":  0,
			"loop_length":     0,
			"loop_index":      0,
		},
		"created_at":  execution.CreatedAt,
		"created_by":  s.workflowActor(execution.CreatedBy),
		"finished_at": execution.FinishedAt,
	}
}

func decodeJSONObjectBody(r *http.Request) (map[string]any, error) {
	if r.Body == nil {
		return map[string]any{}, nil
	}

	payload := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if err == io.EOF {
			return map[string]any{}, nil
		}
		return nil, err
	}
	return payload, nil
}

func writeWorkflowEvent(w http.ResponseWriter, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

func workflowRunInputs(payload map[string]any) map[string]any {
	inputs := mapFromAny(payload["inputs"])
	if query := stringFromAny(payload["query"]); query != "" {
		inputs["query"] = query
	}
	if files, ok := payload["files"].([]any); ok && len(files) > 0 {
		inputs["files"] = files
	}
	if nodeID := stringFromAny(payload["node_id"]); nodeID != "" {
		inputs["node_id"] = nodeID
	}
	if nodeIDs := workflowNodeIDList(payload["node_ids"]); len(nodeIDs) > 0 {
		values := make([]any, 0, len(nodeIDs))
		for _, nodeID := range nodeIDs {
			values = append(values, nodeID)
		}
		inputs["node_ids"] = values
	}
	if conversationID := stringFromAny(payload["conversation_id"]); conversationID != "" {
		inputs["conversation_id"] = conversationID
	}
	return inputs
}

func workflowRunOutputs(inputs map[string]any, nodes []workflowNodeMeta, mode string) map[string]any {
	text := "Workflow run completed successfully."
	switch mode {
	case "trigger":
		text = "Trigger debug completed successfully."
	case "trigger-all":
		text = "All trigger nodes completed successfully."
	}

	outputs := map[string]any{
		"text":       text,
		"node_count": len(nodes),
	}
	if len(inputs) > 0 {
		outputs["inputs"] = cloneJSONObject(inputs)
	}
	if query := stringFromAny(inputs["query"]); query != "" {
		outputs["query"] = query
	}
	return outputs
}

func workflowNodeOutputs(node workflowNodeMeta, inputs, finalOutputs map[string]any, index int) map[string]any {
	switch node.NodeType {
	case "start":
		return map[string]any{
			"received": cloneJSONObject(inputs),
		}
	case "answer", "end":
		return cloneJSONObject(finalOutputs)
	default:
		return map[string]any{
			"text":    fmt.Sprintf("%s completed", node.Title),
			"node_id": node.ID,
			"step":    index + 1,
		}
	}
}

func workflowNodeInputs(node workflowNodeMeta, inputs, payload map[string]any) map[string]any {
	if len(inputs) > 0 {
		return cloneJSONObject(inputs)
	}
	if len(payload) > 0 {
		return cloneJSONObject(payload)
	}
	return map[string]any{
		"node_id": node.ID,
	}
}

func workflowGraphNodes(graph map[string]any, selectedNodeIDs []string) []workflowNodeMeta {
	nodes := workflowGraphNodesAll(graph)
	if len(selectedNodeIDs) == 0 {
		return nodes
	}

	nodeMap := make(map[string]workflowNodeMeta, len(nodes))
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}

	filtered := make([]workflowNodeMeta, 0, len(selectedNodeIDs))
	for _, nodeID := range selectedNodeIDs {
		if node, ok := nodeMap[nodeID]; ok {
			filtered = append(filtered, node)
			continue
		}
		filtered = append(filtered, workflowNodeMeta{
			ID:       nodeID,
			NodeType: "custom",
			Title:    fallbackNodeTitle("custom"),
		})
	}
	return filtered
}

func workflowGraphNodesAll(graph map[string]any) []workflowNodeMeta {
	predecessors := workflowPredecessors(graph)
	rawNodes, ok := graph["nodes"].([]any)
	if !ok || len(rawNodes) == 0 {
		return []workflowNodeMeta{{
			ID:       "start",
			NodeType: "start",
			Title:    "Start",
		}}
	}

	nodes := make([]workflowNodeMeta, 0, len(rawNodes))
	for _, raw := range rawNodes {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		data := mapFromAny(item["data"])
		nodeID := firstImportValue(stringFromAny(item["id"]), runtimeID("node"))
		nodeType := firstImportValue(stringFromAny(data["type"]), stringFromAny(item["type"]))
		if nodeType == "" {
			nodeType = "custom"
		}
		title := firstImportValue(stringFromAny(data["title"]), fallbackNodeTitle(nodeType))
		nodes = append(nodes, workflowNodeMeta{
			ID:              nodeID,
			NodeType:        nodeType,
			Title:           title,
			PredecessorNode: predecessors[nodeID],
		})
	}
	if len(nodes) == 0 {
		return []workflowNodeMeta{{
			ID:       "start",
			NodeType: "start",
			Title:    "Start",
		}}
	}
	return nodes
}

func workflowNodeDefinition(graph map[string]any, nodeID string) workflowNodeMeta {
	for _, node := range workflowGraphNodesAll(graph) {
		if node.ID == nodeID {
			return node
		}
	}
	if strings.TrimSpace(nodeID) == "" {
		nodeID = runtimeID("node")
	}
	return workflowNodeMeta{
		ID:       nodeID,
		NodeType: "custom",
		Title:    fallbackNodeTitle("custom"),
	}
}

func workflowPredecessors(graph map[string]any) map[string]string {
	rawEdges, ok := graph["edges"].([]any)
	if !ok || len(rawEdges) == 0 {
		return map[string]string{}
	}

	predecessors := make(map[string]string, len(rawEdges))
	for _, raw := range rawEdges {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		target := stringFromAny(item["target"])
		source := stringFromAny(item["source"])
		if target != "" && source != "" {
			predecessors[target] = source
		}
	}
	return predecessors
}

func workflowNodeIDList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return []string{}
	}
	nodeIDs := make([]string, 0, len(items))
	for _, item := range items {
		if nodeID := stringFromAny(item); nodeID != "" {
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	return nodeIDs
}

func fallbackNodeTitle(nodeType string) string {
	switch nodeType {
	case "start":
		return "Start"
	case "end":
		return "End"
	case "answer":
		return "Answer"
	case "llm":
		return "LLM"
	default:
		if nodeType == "" {
			return "Node"
		}
		return strings.Title(strings.ReplaceAll(nodeType, "-", " "))
	}
}

func workflowRunVersion(app state.App) string {
	if app.WorkflowDraft != nil && strings.TrimSpace(app.WorkflowDraft.Version) != "" {
		return app.WorkflowDraft.Version
	}
	return "draft"
}

func workflowIDForResponse(app state.App) string {
	if app.Workflow != nil && strings.TrimSpace(app.Workflow.ID) != "" {
		return app.Workflow.ID
	}
	if app.WorkflowPublished != nil && strings.TrimSpace(app.WorkflowPublished.ID) != "" {
		return app.WorkflowPublished.ID
	}
	if app.WorkflowDraft != nil && strings.TrimSpace(app.WorkflowDraft.ID) != "" {
		return app.WorkflowDraft.ID
	}
	return ""
}

func workflowElapsed(steps int) float64 {
	if steps <= 0 {
		return 0.12
	}
	return float64(steps) * 0.12
}

func workflowNodeElapsed(index int) float64 {
	return 0.08 + (float64(index) * 0.04)
}

func stringifyWorkflowValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func runtimeID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
}
