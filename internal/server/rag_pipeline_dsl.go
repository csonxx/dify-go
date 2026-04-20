package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/langgenius/dify-go/internal/state"
)

const currentRAGPipelineDSLVersion = "0.1.0"
const defaultRAGPipelineDatasetName = "Untitled"
const defaultRAGPipelineIcon = "📙"
const defaultRAGPipelineIconBackground = "#FFF4ED"

type ragPipelineDSLDocument struct {
	Version      string           `yaml:"version"`
	Kind         string           `yaml:"kind"`
	RAGPipeline  map[string]any   `yaml:"rag_pipeline"`
	Workflow     map[string]any   `yaml:"workflow"`
	Dependencies []map[string]any `yaml:"dependencies,omitempty"`
}

type ragPipelineImportResult struct {
	ImportID           string
	Status             string
	PipelineID         string
	DatasetID          string
	CurrentDSLVersion  string
	ImportedDSLVersion string
	Error              string
}

func (s *server) handleRAGPipelineExport(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	dataset, ok := s.store.FindRAGPipelineDataset(workspace.ID, app.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	document, err := s.exportRAGPipelineDSL(app, dataset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	payload, err := yaml.Marshal(document)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to export rag pipeline DSL.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": string(payload)})
}

func (s *server) handleRAGPipelineImport(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Mode        string `json:"mode"`
		YAMLContent string `json:"yaml_content"`
		YAMLURL     string `json:"yaml_url"`
		PipelineID  string `json:"pipeline_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	result, statusCode := s.importRAGPipelineDSL(
		workspace.ID,
		currentUser(r),
		payload.Mode,
		payload.YAMLContent,
		payload.YAMLURL,
		payload.PipelineID,
		time.Now(),
	)
	if result.Status == "failed" {
		writeError(w, statusCode, "invalid_request", firstNonEmpty(strings.TrimSpace(result.Error), "Failed to import rag pipeline DSL."))
		return
	}

	writeJSON(w, statusCode, map[string]any{
		"id":                   result.ImportID,
		"status":               result.Status,
		"pipeline_id":          result.PipelineID,
		"dataset_id":           result.DatasetID,
		"current_dsl_version":  result.CurrentDSLVersion,
		"imported_dsl_version": result.ImportedDSLVersion,
		"error":                nullIfEmpty(result.Error),
	})
}

func (s *server) handleRAGPipelineImportConfirm(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusBadRequest, map[string]any{
		"id":                   chi.URLParam(r, "importID"),
		"status":               "failed",
		"pipeline_id":          "",
		"dataset_id":           "",
		"current_dsl_version":  currentRAGPipelineDSLVersion,
		"imported_dsl_version": "",
		"error":                "This import does not require confirmation.",
	})
}

func (s *server) handleRAGPipelineDatasetCreateFromDSL(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		YAMLContent string `json:"yaml_content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	result, statusCode := s.importRAGPipelineDSL(
		workspace.ID,
		currentUser(r),
		"yaml-content",
		payload.YAMLContent,
		"",
		"",
		time.Now(),
	)
	if result.Status == "failed" {
		writeError(w, statusCode, "invalid_request", firstNonEmpty(strings.TrimSpace(result.Error), "Failed to create rag pipeline dataset."))
		return
	}

	dataset, ok := s.store.GetDataset(result.DatasetID, workspace.ID)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "Imported dataset was not persisted.")
		return
	}

	response := s.datasetResponse(dataset)
	response["dataset_id"] = dataset.ID
	writeJSON(w, http.StatusCreated, response)
}

func (s *server) importRAGPipelineDSL(workspaceID string, user state.User, mode, yamlContent, yamlURL, pipelineID string, now time.Time) (ragPipelineImportResult, int) {
	result := ragPipelineImportResult{
		ImportID:           generateImportID(),
		Status:             "failed",
		CurrentDSLVersion:  currentRAGPipelineDSLVersion,
		ImportedDSLVersion: currentRAGPipelineDSLVersion,
	}

	content, err := resolveImportContent(mode, yamlContent, yamlURL)
	if err != nil {
		result.Error = err.Error()
		return result, http.StatusBadRequest
	}

	document, err := decodeRAGPipelineDSL(content)
	if err != nil {
		result.Error = err.Error()
		return result, http.StatusBadRequest
	}
	result.ImportedDSLVersion = document.Version

	var app state.App
	var dataset state.Dataset
	if trimmedPipelineID := strings.TrimSpace(pipelineID); trimmedPipelineID != "" {
		app, dataset, err = s.resolveExistingRAGPipeline(workspaceID, trimmedPipelineID)
		if err != nil {
			result.Error = err.Error()
			return result, http.StatusNotFound
		}
	} else {
		dataset, app, err = s.store.CreateRAGPipelineDataset(workspaceID, user, state.CreateRAGPipelineDatasetInput{
			Name:        firstImportValue(ragPipelineDSLName(document.RAGPipeline), ""),
			Description: ragPipelineDSLDescription(document.RAGPipeline),
		}, now)
		if err != nil {
			result.Error = err.Error()
			return result, http.StatusBadRequest
		}
	}

	app, err = s.applyRAGPipelineMetadata(app, document, now)
	if err != nil {
		result.Error = err.Error()
		return result, http.StatusBadRequest
	}

	dataset, err = s.applyRAGPipelineDatasetMetadata(dataset, document, user, now)
	if err != nil {
		result.Error = err.Error()
		return result, http.StatusBadRequest
	}

	workflowPayload := ragPipelineWorkflowPayload(document.Workflow, app)
	_, err = s.store.SyncWorkflowDraft(
		app.ID,
		workspaceID,
		user,
		workflowPayload.Graph,
		workflowPayload.Features,
		workflowPayload.EnvironmentVariables,
		workflowPayload.ConversationVariables,
		workflowPayload.RAGPipelineVariables,
		"",
		now,
	)
	if err != nil {
		result.Error = err.Error()
		return result, http.StatusBadRequest
	}

	result.Status = "completed"
	result.PipelineID = app.ID
	result.DatasetID = dataset.ID
	return result, http.StatusOK
}

func (s *server) resolveExistingRAGPipeline(workspaceID, pipelineID string) (state.App, state.Dataset, error) {
	app, ok := s.findPipelineDependencyApp(workspaceID, pipelineID)
	if !ok {
		return state.App{}, state.Dataset{}, fmt.Errorf("App not found.")
	}

	dataset, ok := s.store.FindRAGPipelineDataset(workspaceID, app.ID)
	if !ok {
		return state.App{}, state.Dataset{}, fmt.Errorf("Dataset not found.")
	}

	return app, dataset, nil
}

func (s *server) applyRAGPipelineMetadata(app state.App, document ragPipelineDSLDocument, now time.Time) (state.App, error) {
	meta := document.RAGPipeline
	update := state.UpdateAppInput{
		Name:                firstImportValue(ragPipelineDSLName(meta), app.Name),
		Description:         firstImportValue(ragPipelineDSLDescription(meta), app.Description),
		IconType:            firstImportValue(strings.TrimSpace(stringFromAny(meta["icon_type"])), app.IconType),
		Icon:                firstImportValue(strings.TrimSpace(stringFromAny(meta["icon"])), app.Icon),
		IconBackground:      firstImportValue(strings.TrimSpace(stringFromAny(meta["icon_background"])), app.IconBackground),
		UseIconAsAnswerIcon: boolPtr(app.UseIconAsAnswerIcon),
		MaxActiveRequests:   app.MaxActiveRequests,
	}
	return s.store.UpdateApp(app.ID, app.WorkspaceID, update, now)
}

func (s *server) applyRAGPipelineDatasetMetadata(dataset state.Dataset, document ragPipelineDSLDocument, user state.User, now time.Time) (state.Dataset, error) {
	meta := document.RAGPipeline
	patch := map[string]any{
		"name":        firstImportValue(ragPipelineDSLName(meta), dataset.Name),
		"description": firstImportValue(ragPipelineDSLDescription(meta), dataset.Description),
		"icon_info": map[string]any{
			"icon":            firstImportValue(strings.TrimSpace(stringFromAny(meta["icon"])), dataset.IconInfo.Icon),
			"icon_type":       firstImportValue(strings.TrimSpace(stringFromAny(meta["icon_type"])), dataset.IconInfo.IconType),
			"icon_background": firstImportValue(strings.TrimSpace(stringFromAny(meta["icon_background"])), dataset.IconInfo.IconBackground),
			"icon_url":        firstImportValue(strings.TrimSpace(stringFromAny(meta["icon_url"])), dataset.IconInfo.IconURL),
		},
	}

	for key, value := range ragPipelineDatasetPatchFromWorkflow(document.Workflow) {
		patch[key] = value
	}

	return s.store.PatchDataset(dataset.ID, dataset.WorkspaceID, patch, user, now)
}

func (s *server) exportRAGPipelineDSL(app state.App, dataset state.Dataset) (ragPipelineDSLDocument, error) {
	if app.WorkflowDraft == nil {
		return ragPipelineDSLDocument{}, fmt.Errorf("Missing draft workflow configuration, please check.")
	}

	iconInfo := dataset.IconInfo
	document := ragPipelineDSLDocument{
		Version: currentRAGPipelineDSLVersion,
		Kind:    "rag_pipeline",
		RAGPipeline: map[string]any{
			"name":            firstNonEmpty(dataset.Name, app.Name, defaultRAGPipelineDatasetName),
			"icon":            firstNonEmpty(iconInfo.Icon, app.Icon, defaultRAGPipelineIcon),
			"icon_type":       firstNonEmpty(iconInfo.IconType, app.IconType, "emoji"),
			"icon_background": firstNonEmpty(iconInfo.IconBackground, app.IconBackground, defaultRAGPipelineIconBackground),
			"icon_url":        nullIfEmpty(iconInfo.IconURL),
			"description":     firstNonEmpty(dataset.Description, app.Description),
		},
		Workflow: map[string]any{
			"graph":                  cloneJSONObject(app.WorkflowDraft.Graph),
			"features":               cloneJSONObject(app.WorkflowDraft.Features),
			"environment_variables":  cloneMapList(app.WorkflowDraft.EnvironmentVariables),
			"conversation_variables": cloneMapList(app.WorkflowDraft.ConversationVariables),
			"rag_pipeline_variables": cloneMapList(app.WorkflowDraft.RagPipelineVariables),
		},
	}

	if dependencies := s.ragPipelineDependencies(app); len(dependencies) > 0 {
		document.Dependencies = dependencies
	}

	return document, nil
}

func (s *server) ragPipelineDependencies(app state.App) []map[string]any {
	candidates := s.appDependencyCandidates(app)
	seen := make(map[string]struct{}, len(candidates))
	result := make([]map[string]any, 0, len(candidates))

	for _, candidate := range candidates {
		candidate = normalizePluginDependencyCandidate(candidate)
		key := firstNonEmpty(candidate.UniqueIdentifier, candidate.PluginID, candidate.Meta.Repo)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		payload := s.pluginDependencyPayloadFromCandidate(candidate, state.WorkspacePluginInstallation{}, false)
		if payload != nil {
			result = append(result, payload)
		}
	}

	return dedupePluginDependencyPayloads(result)
}

type ragPipelineWorkflowImportPayload struct {
	Graph                 map[string]any
	Features              map[string]any
	EnvironmentVariables  []map[string]any
	ConversationVariables []map[string]any
	RAGPipelineVariables  []map[string]any
}

func ragPipelineWorkflowPayload(workflow map[string]any, app state.App) ragPipelineWorkflowImportPayload {
	current := workflowStateForImport(app)

	graph := mapFromAny(workflow["graph"])
	if len(graph) == 0 {
		graph = cloneJSONObject(current.Graph)
	}

	features := mapFromAny(workflow["features"])
	if len(features) == 0 && workflow["features"] == nil {
		features = cloneJSONObject(current.Features)
	}

	environmentVariables := objectListFromAny(workflow["environment_variables"])
	if len(environmentVariables) == 0 && workflow["environment_variables"] == nil {
		environmentVariables = cloneMapList(current.EnvironmentVariables)
	}

	conversationVariables := objectListFromAny(workflow["conversation_variables"])
	if len(conversationVariables) == 0 && workflow["conversation_variables"] == nil {
		conversationVariables = cloneMapList(current.ConversationVariables)
	}

	ragPipelineVariables := objectListFromAny(workflow["rag_pipeline_variables"])
	if len(ragPipelineVariables) == 0 && workflow["rag_pipeline_variables"] == nil {
		ragPipelineVariables = cloneMapList(current.RagPipelineVariables)
	}

	return ragPipelineWorkflowImportPayload{
		Graph:                 normalizeWorkflowGraph(graph),
		Features:              features,
		EnvironmentVariables:  environmentVariables,
		ConversationVariables: conversationVariables,
		RAGPipelineVariables:  ragPipelineVariables,
	}
}

func workflowStateForImport(app state.App) state.WorkflowState {
	switch {
	case app.WorkflowDraft != nil:
		return *app.WorkflowDraft
	case app.WorkflowPublished != nil:
		return *app.WorkflowPublished
	default:
		now := time.Now().UTC().Unix()
		return state.WorkflowState{
			Graph:                 map[string]any{"nodes": []any{}, "edges": []any{}, "viewport": map[string]any{"x": 0, "y": 0, "zoom": 1}},
			Features:              map[string]any{},
			EnvironmentVariables:  []map[string]any{},
			ConversationVariables: []map[string]any{},
			RagPipelineVariables:  []map[string]any{},
			CreatedAt:             now,
			UpdatedAt:             now,
		}
	}
}

func decodeRAGPipelineDSL(content []byte) (ragPipelineDSLDocument, error) {
	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return ragPipelineDSLDocument{}, fmt.Errorf("invalid DSL content")
	}

	documentMap, ok := normalizeDocumentValue(raw).(map[string]any)
	if !ok {
		return ragPipelineDSLDocument{}, fmt.Errorf("invalid DSL content")
	}

	document := ragPipelineDSLDocument{
		Version:      firstNonEmpty(strings.TrimSpace(stringFromAny(documentMap["version"])), currentRAGPipelineDSLVersion),
		Kind:         firstNonEmpty(strings.TrimSpace(stringFromAny(documentMap["kind"])), "rag_pipeline"),
		RAGPipeline:  mapFromAny(documentMap["rag_pipeline"]),
		Workflow:     mapFromAny(documentMap["workflow"]),
		Dependencies: objectListFromAny(documentMap["dependencies"]),
	}

	if len(document.Workflow) == 0 {
		document.Workflow = ragPipelineWorkflowFallbackDocument(documentMap)
	}

	if strings.TrimSpace(document.Kind) != "" && !strings.Contains(strings.ToLower(document.Kind), "rag_pipeline") {
		return ragPipelineDSLDocument{}, fmt.Errorf("dsl kind must be rag_pipeline")
	}
	if len(document.Workflow) == 0 {
		return ragPipelineDSLDocument{}, fmt.Errorf("dsl workflow is required")
	}

	return document, nil
}

func ragPipelineWorkflowFallbackDocument(document map[string]any) map[string]any {
	workflow := map[string]any{}
	for _, key := range []string{"graph", "features", "environment_variables", "conversation_variables", "rag_pipeline_variables"} {
		if value, ok := document[key]; ok {
			workflow[key] = normalizeDocumentValue(value)
		}
	}
	return workflow
}

func ragPipelineDatasetPatchFromWorkflow(workflow map[string]any) map[string]any {
	nodeData := ragPipelineKnowledgeIndexNodeData(mapFromAny(workflow["graph"]))
	if len(nodeData) == 0 {
		return map[string]any{}
	}

	patch := map[string]any{}
	if chunkStructure := strings.TrimSpace(stringFromAny(nodeData["chunk_structure"])); chunkStructure != "" {
		patch["doc_form"] = chunkStructure
	}
	if technique := strings.TrimSpace(stringFromAny(nodeData["indexing_technique"])); technique != "" {
		patch["indexing_technique"] = technique
	}
	if retrievalModel := mapFromAny(nodeData["retrieval_model"]); len(retrievalModel) > 0 {
		patch["retrieval_model"] = retrievalModel
	}
	if embeddingModel := strings.TrimSpace(stringFromAny(nodeData["embedding_model"])); embeddingModel != "" {
		patch["embedding_model"] = embeddingModel
	}
	if embeddingProvider := strings.TrimSpace(stringFromAny(nodeData["embedding_model_provider"])); embeddingProvider != "" {
		patch["embedding_model_provider"] = embeddingProvider
	}
	if summaryIndexSetting := mapFromAny(nodeData["summary_index_setting"]); len(summaryIndexSetting) > 0 {
		patch["summary_index_setting"] = summaryIndexSetting
	}
	return patch
}

func ragPipelineKnowledgeIndexNodeData(graph map[string]any) map[string]any {
	for _, item := range anySlice(graph["nodes"]) {
		node := mapFromAny(item)
		data := mapFromAny(node["data"])
		if strings.TrimSpace(stringFromAny(data["type"])) == "knowledge-index" {
			return data
		}
	}
	return map[string]any{}
}

func ragPipelineDSLName(meta map[string]any) string {
	return strings.TrimSpace(stringFromAny(meta["name"]))
}

func ragPipelineDSLDescription(meta map[string]any) string {
	return strings.TrimSpace(stringFromAny(meta["description"]))
}

func objectListFromAny(value any) []map[string]any {
	items := anySlice(normalizeDocumentValue(value))
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := mapFromAny(item)
		if len(entry) == 0 {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func normalizeWorkflowGraph(graph map[string]any) map[string]any {
	if len(graph) == 0 {
		return map[string]any{
			"nodes":    []any{},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		}
	}

	out := cloneJSONObject(graph)
	if out["nodes"] == nil {
		out["nodes"] = []any{}
	}
	if out["edges"] == nil {
		out["edges"] = []any{}
	}
	if viewport := mapFromAny(out["viewport"]); len(viewport) == 0 {
		out["viewport"] = map[string]any{"x": 0, "y": 0, "zoom": 1}
	}
	return out
}
