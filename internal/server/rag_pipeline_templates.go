package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/langgenius/dify-go/internal/state"
)

type builtinRAGPipelineTemplate struct {
	ID             string
	Name           string
	Description    string
	ChunkStructure string
	IconInfo       state.DatasetIconInfo
	CreatedBy      string
	Workflow       map[string]any
}

func (s *server) handleRAGPipelineTemplateList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	templateType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if templateType == "" {
		templateType = "built-in"
	}

	switch templateType {
	case "built-in":
		items := builtinRAGPipelineTemplates(strings.TrimSpace(r.URL.Query().Get("language")))
		data := make([]map[string]any, 0, len(items))
		for _, item := range items {
			data = append(data, map[string]any{
				"id":              item.ID,
				"name":            item.Name,
				"icon":            datasetIconInfoPayload(item.IconInfo),
				"description":     item.Description,
				"position":        len(data) + 1,
				"chunk_structure": item.ChunkStructure,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"pipeline_templates": data})
	case "customized":
		items := s.store.ListPipelineTemplates(workspace.ID)
		data := make([]map[string]any, 0, len(items))
		for _, item := range items {
			data = append(data, s.pipelineTemplateListItem(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"pipeline_templates": data})
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "Unsupported pipeline template type.")
	}
}

func (s *server) handleRAGPipelineTemplateDetail(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "template_id is required.")
		return
	}

	templateType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if templateType == "" {
		templateType = "built-in"
	}

	switch templateType {
	case "built-in":
		template, ok := findBuiltinRAGPipelineTemplate(templateID, strings.TrimSpace(r.URL.Query().Get("language")))
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "Pipeline template not found.")
			return
		}
		writeJSON(w, http.StatusOK, s.builtinPipelineTemplateDetailPayload(template))
	case "customized":
		template, ok := s.store.GetPipelineTemplate(templateID, workspace.ID)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "Pipeline template not found.")
			return
		}
		writeJSON(w, http.StatusOK, s.pipelineTemplateDetailPayload(template))
	default:
		writeError(w, http.StatusBadRequest, "invalid_request", "Unsupported pipeline template type.")
	}
}

func (s *server) handleCustomizedRAGPipelineTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "template_id is required.")
		return
	}

	var payload struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		IconInfo    map[string]any `json:"icon_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Template name is required.")
		return
	}

	template, ok := s.store.GetPipelineTemplate(templateID, workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Pipeline template not found.")
		return
	}

	iconInfo := datasetIconInfoFromPayload(payload.IconInfo, template.IconInfo, payload.Name)
	exportData, chunkStructure, err := rewriteRAGPipelineTemplateExportData(template.ExportData, payload.Name, payload.Description, iconInfo)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updated, err := s.store.UpdatePipelineTemplate(template.ID, template.WorkspaceID, currentUser(r), state.UpdatePipelineTemplateInput{
		Name:           payload.Name,
		Description:    payload.Description,
		ChunkStructure: chunkStructure,
		IconInfo:       &iconInfo,
		ExportData:     &exportData,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pipeline_id":     updated.ID,
		"name":            updated.Name,
		"icon":            datasetIconInfoPayload(updated.IconInfo),
		"description":     updated.Description,
		"position":        updated.Position,
		"chunk_structure": updated.ChunkStructure,
	})
}

func (s *server) handleCustomizedRAGPipelineTemplateDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "template_id is required.")
		return
	}

	if err := s.store.DeletePipelineTemplate(templateID, workspace.ID); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Pipeline template not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"code": 200})
}

func (s *server) handleCustomizedRAGPipelineTemplateExport(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "template_id is required.")
		return
	}

	template, ok := s.store.GetPipelineTemplate(templateID, workspace.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "Pipeline template not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": template.ExportData})
}

func (s *server) handleRAGPipelinePublishCustomizedTemplate(w http.ResponseWriter, r *http.Request) {
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

	var payload struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		IconInfo    map[string]any `json:"icon_info"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if strings.TrimSpace(payload.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Template name is required.")
		return
	}

	document, err := s.exportRAGPipelineDSL(app, dataset)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	iconInfo := datasetIconInfoFromPayload(payload.IconInfo, dataset.IconInfo, payload.Name)
	document.RAGPipeline["name"] = strings.TrimSpace(payload.Name)
	document.RAGPipeline["description"] = strings.TrimSpace(payload.Description)
	document.RAGPipeline["icon"] = iconInfo.Icon
	document.RAGPipeline["icon_type"] = iconInfo.IconType
	document.RAGPipeline["icon_background"] = iconInfo.IconBackground
	if strings.TrimSpace(iconInfo.IconURL) != "" {
		document.RAGPipeline["icon_url"] = iconInfo.IconURL
	} else {
		document.RAGPipeline["icon_url"] = nil
	}

	exported, err := yaml.Marshal(document)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to publish pipeline template.")
		return
	}

	_, err = s.store.CreatePipelineTemplate(workspace.ID, currentUser(r), state.CreatePipelineTemplateInput{
		Name:           payload.Name,
		Description:    payload.Description,
		ChunkStructure: firstNonEmpty(pipelineTemplateChunkStructure(document.Workflow, ""), dataset.DocForm),
		IconInfo:       iconInfo,
		ExportData:     string(exported),
		Language:       currentUser(r).InterfaceLanguage,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) pipelineTemplateListItem(item state.PipelineTemplate) map[string]any {
	return map[string]any{
		"id":              item.ID,
		"name":            item.Name,
		"icon":            datasetIconInfoPayload(item.IconInfo),
		"description":     item.Description,
		"position":        item.Position,
		"chunk_structure": item.ChunkStructure,
	}
}

func (s *server) pipelineTemplateDetailPayload(item state.PipelineTemplate) map[string]any {
	document, err := decodeRAGPipelineDSL([]byte(item.ExportData))
	if err != nil {
		document = ragPipelineDSLDocument{
			Version: currentRAGPipelineDSLVersion,
			Kind:    "rag_pipeline",
			RAGPipeline: map[string]any{
				"name":            item.Name,
				"description":     item.Description,
				"icon":            item.IconInfo.Icon,
				"icon_type":       item.IconInfo.IconType,
				"icon_background": item.IconInfo.IconBackground,
				"icon_url":        nullIfEmpty(item.IconInfo.IconURL),
			},
			Workflow: map[string]any{
				"graph": normalizeWorkflowGraph(map[string]any{}),
			},
		}
	}

	return map[string]any{
		"id":              item.ID,
		"name":            item.Name,
		"icon_info":       datasetIconInfoPayload(item.IconInfo),
		"description":     item.Description,
		"chunk_structure": pipelineTemplateChunkStructure(document.Workflow, item.ChunkStructure),
		"export_data":     item.ExportData,
		"graph":           normalizeWorkflowGraph(mapFromAny(document.Workflow["graph"])),
		"created_by":      s.pipelineTemplateCreatedByName(item.CreatedBy),
	}
}

func (s *server) builtinPipelineTemplateDetailPayload(item builtinRAGPipelineTemplate) map[string]any {
	exportData := builtinRAGPipelineTemplateExportData(item)
	return map[string]any{
		"id":              item.ID,
		"name":            item.Name,
		"icon_info":       datasetIconInfoPayload(item.IconInfo),
		"description":     item.Description,
		"chunk_structure": item.ChunkStructure,
		"export_data":     exportData,
		"graph":           normalizeWorkflowGraph(mapFromAny(item.Workflow["graph"])),
		"created_by":      item.CreatedBy,
	}
}

func (s *server) pipelineTemplateCreatedByName(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	if user, ok := s.store.GetUser(userID); ok {
		return user.Name
	}
	return ""
}

func builtinRAGPipelineTemplates(language string) []builtinRAGPipelineTemplate {
	return []builtinRAGPipelineTemplate{
		newBuiltinRAGPipelineTemplate(
			"builtin-general",
			localizedBuiltinTemplateText(language, "General RAG Pipeline", "通用知识库流程", "汎用ナレッジパイプライン"),
			localizedBuiltinTemplateText(language, "Start from a general-purpose retrieval and chunking workflow.", "从通用检索与切分流程开始搭建。", "汎用的な検索とチャンク分割フローから始めます。"),
			"text_model",
			state.DatasetIconInfo{IconType: "emoji", Icon: "📙", IconBackground: "#FFF4ED"},
		),
		newBuiltinRAGPipelineTemplate(
			"builtin-parent-child",
			localizedBuiltinTemplateText(language, "Parent-Child Pipeline", "父子分块流程", "親子チャンクパイプライン"),
			localizedBuiltinTemplateText(language, "Use hierarchical chunks for long-form source material.", "适合长文档的父子分块检索流程。", "長文ソース向けの階層チャンク検索フローです。"),
			"hierarchical_model",
			state.DatasetIconInfo{IconType: "emoji", Icon: "🧩", IconBackground: "#DBEAFE"},
		),
		newBuiltinRAGPipelineTemplate(
			"builtin-qa",
			localizedBuiltinTemplateText(language, "Q&A Pipeline", "问答知识库流程", "Q&A パイプライン"),
			localizedBuiltinTemplateText(language, "Optimize for FAQ-style chunks and question answering.", "适合 FAQ 与问答式切分的流程。", "FAQ や質問応答向けのフローです。"),
			"qa_model",
			state.DatasetIconInfo{IconType: "emoji", Icon: "❓", IconBackground: "#DCFCE7"},
		),
	}
}

func findBuiltinRAGPipelineTemplate(id, language string) (builtinRAGPipelineTemplate, bool) {
	for _, item := range builtinRAGPipelineTemplates(language) {
		if item.ID == id {
			return item, true
		}
	}
	return builtinRAGPipelineTemplate{}, false
}

func newBuiltinRAGPipelineTemplate(id, name, description, chunkStructure string, iconInfo state.DatasetIconInfo) builtinRAGPipelineTemplate {
	return builtinRAGPipelineTemplate{
		ID:             id,
		Name:           name,
		Description:    description,
		ChunkStructure: chunkStructure,
		IconInfo:       iconInfo,
		CreatedBy:      "dify-go",
		Workflow: map[string]any{
			"graph": map[string]any{
				"nodes": []map[string]any{
					{
						"id": "knowledge-node",
						"data": map[string]any{
							"title":                    "Knowledge Index",
							"type":                     "knowledge-index",
							"chunk_structure":          chunkStructure,
							"indexing_technique":       "high_quality",
							"retrieval_model":          map[string]any{"search_method": "semantic_search", "top_k": 4, "score_threshold_enabled": false, "score_threshold": 0.5},
							"keyword_number":           10,
							"summary_index_setting":    map[string]any{"enable": false, "model_name": "", "model_provider_name": "", "summary_prompt": ""},
							"embedding_model":          "",
							"embedding_model_provider": "",
						},
					},
				},
				"edges":    []any{},
				"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
			},
			"features":               map[string]any{},
			"environment_variables":  []any{},
			"conversation_variables": []any{},
			"rag_pipeline_variables": []any{},
		},
	}
}

func builtinRAGPipelineTemplateExportData(item builtinRAGPipelineTemplate) string {
	document := ragPipelineDSLDocument{
		Version: currentRAGPipelineDSLVersion,
		Kind:    "rag_pipeline",
		RAGPipeline: map[string]any{
			"name":            item.Name,
			"description":     item.Description,
			"icon":            item.IconInfo.Icon,
			"icon_type":       item.IconInfo.IconType,
			"icon_background": item.IconInfo.IconBackground,
			"icon_url":        nullIfEmpty(item.IconInfo.IconURL),
		},
		Workflow: item.Workflow,
	}
	payload, err := yaml.Marshal(document)
	if err != nil {
		return ""
	}
	return string(payload)
}

func localizedBuiltinTemplateText(language, english, chinese, japanese string) string {
	switch strings.TrimSpace(language) {
	case "zh-Hans", "zh-CN":
		return chinese
	case "ja-JP":
		return japanese
	default:
		return english
	}
}

func pipelineTemplateChunkStructure(workflow map[string]any, fallback string) string {
	nodeData := ragPipelineKnowledgeIndexNodeData(mapFromAny(workflow["graph"]))
	if chunkStructure := strings.TrimSpace(stringFromAny(nodeData["chunk_structure"])); chunkStructure != "" {
		return chunkStructure
	}
	return firstNonEmpty(strings.TrimSpace(fallback), "text_model")
}

func datasetIconInfoFromPayload(payload map[string]any, fallback state.DatasetIconInfo, name string) state.DatasetIconInfo {
	if len(payload) == 0 {
		return fallback
	}

	return state.DatasetIconInfo{
		Icon:           firstImportValue(strings.TrimSpace(stringFromAny(payload["icon"])), fallback.Icon),
		IconBackground: firstImportValue(strings.TrimSpace(stringFromAny(payload["icon_background"])), fallback.IconBackground),
		IconType:       firstImportValue(strings.TrimSpace(stringFromAny(payload["icon_type"])), fallback.IconType),
		IconURL:        firstImportValue(strings.TrimSpace(stringFromAny(payload["icon_url"])), fallback.IconURL),
	}
}

func rewriteRAGPipelineTemplateExportData(exportData, name, description string, iconInfo state.DatasetIconInfo) (string, string, error) {
	document, err := decodeRAGPipelineDSL([]byte(exportData))
	if err != nil {
		return "", "", err
	}

	document.RAGPipeline["name"] = strings.TrimSpace(name)
	document.RAGPipeline["description"] = strings.TrimSpace(description)
	document.RAGPipeline["icon"] = iconInfo.Icon
	document.RAGPipeline["icon_type"] = iconInfo.IconType
	document.RAGPipeline["icon_background"] = iconInfo.IconBackground
	if strings.TrimSpace(iconInfo.IconURL) != "" {
		document.RAGPipeline["icon_url"] = iconInfo.IconURL
	} else {
		document.RAGPipeline["icon_url"] = nil
	}

	payload, err := yaml.Marshal(document)
	if err != nil {
		return "", "", err
	}

	return string(payload), pipelineTemplateChunkStructure(document.Workflow, ""), nil
}
