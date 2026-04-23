package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleRAGPipelinePublishedRun(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return
	}
	if app.WorkflowPublished == nil {
		writeError(w, http.StatusNotFound, "published_workflow_not_exist", "Published workflow does not exist.")
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

	datasourceInfoList := objectListFromAny(payload["datasource_info_list"])
	if len(datasourceInfoList) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Datasource info list is required.")
		return
	}

	inputs := mapFromAny(payload["inputs"])
	startNodeID := firstImportValue(strings.TrimSpace(stringFromAny(payload["start_node_id"])), ragPipelineDatasourceNodeID(datasourceType))
	documentInput := ragPipelinePublishedDocumentInput(dataset, *app.WorkflowPublished, datasourceType, datasourceInfoList, inputs, startNodeID)
	now := time.Now()
	runContext := ragPipelinePublishedRunContext{
		Dataset:            dataset,
		DatasourceType:     datasourceType,
		DatasourceInfoList: cloneMapList(datasourceInfoList),
		Inputs:             cloneJSONObject(inputs),
		StartNodeID:        startNodeID,
		IsPreview:          ragPipelineBoolValue(payload["is_preview"]),
	}

	if ragPipelineBoolValue(payload["is_preview"]) {
		previewOutputs := ragPipelinePreviewOutputs(documentInput.DocForm, datasourceType, datasourceInfoList, documentInput.ProcessRule)
		run := s.buildPublishedPipelineRun(app, currentUser(r), payload, runContext, ragPipelineStoredRunOutputs(runContext, previewOutputs), now)
		_, _ = s.store.SaveWorkflowRun(app.ID, app.WorkspaceID, currentUser(r), run, now)

		writeJSON(w, http.StatusOK, map[string]any{
			"task_iod":        run.TaskID,
			"workflow_run_id": run.ID,
			"data": map[string]any{
				"id":           run.ID,
				"status":       run.Status,
				"created_at":   run.CreatedAt,
				"elapsed_time": run.ElapsedTime,
				"error":        run.Error,
				"finished_at":  run.FinishedAt,
				"outputs":      cloneJSONObject(previewOutputs),
				"total_steps":  run.TotalSteps,
				"total_tokens": run.TotalTokens,
				"workflow_id":  workflowIDForResponse(app),
			},
		})
		return
	}

	originalDocumentID := strings.TrimSpace(stringFromAny(payload["original_document_id"]))
	runContext.OriginalDocumentID = originalDocumentID
	if originalDocumentID != "" && len(datasourceInfoList) != 1 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Published pipeline reprocess expects a single datasource item.")
		return
	}

	var (
		batchID        string
		documents      []state.DatasetDocument
		updatedDataset state.Dataset
	)
	if originalDocumentID != "" {
		var updatedDocument state.DatasetDocument
		batchID, updatedDocument, updatedDataset, err = s.store.UpdateDatasetDocumentFromInput(
			dataset.ID,
			dataset.WorkspaceID,
			originalDocumentID,
			currentUser(r),
			documentInput,
			now,
		)
		if err == nil {
			documents = []state.DatasetDocument{updatedDocument}
		}
	} else {
		batchID, documents, updatedDataset, err = s.store.CreateDatasetDocuments(
			dataset.ID,
			dataset.WorkspaceID,
			currentUser(r),
			documentInput,
			now,
		)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	runContext.Dataset = updatedDataset
	runContext.BatchID = batchID
	runContext.Documents = ragPipelineCloneDocuments(documents)
	run := s.buildPublishedPipelineRun(app, currentUser(r), payload, runContext, ragPipelineStoredRunOutputs(runContext, nil), now)
	_, _ = s.store.SaveWorkflowRun(app.ID, app.WorkspaceID, currentUser(r), run, now)

	writeJSON(w, http.StatusOK, map[string]any{
		"batch": batchID,
		"dataset": map[string]any{
			"id":              updatedDataset.ID,
			"name":            updatedDataset.Name,
			"description":     updatedDataset.Description,
			"chunk_structure": firstNonEmpty(updatedDataset.DocForm, "text_model"),
		},
		"documents": ragPipelineInitialDocumentListPayload(documents),
	})
}

type ragPipelinePublishedRunContext struct {
	Dataset            state.Dataset
	DatasourceType     string
	DatasourceInfoList []map[string]any
	Inputs             map[string]any
	StartNodeID        string
	IsPreview          bool
	OriginalDocumentID string
	BatchID            string
	Documents          []state.DatasetDocument
}

func (s *server) buildPublishedPipelineRun(app state.App, user state.User, payload map[string]any, ctx ragPipelinePublishedRunContext, outputs map[string]any, now time.Time) state.WorkflowRun {
	runApp := app
	workflow := state.WorkflowState{}
	if app.WorkflowPublished != nil {
		workflow = state.WorkflowState(*app.WorkflowPublished)
		runApp.WorkflowDraft = &workflow
	}
	run := s.buildWorkflowRunForState(runApp, workflow, user, payload, workflowRunOptions{
		Mode: "published",
	}, now)
	run.Inputs = ragPipelineWorkflowRunInputs(ctx)
	run.Outputs = cloneJSONObject(outputs)
	run.NodeExecutions = s.ragPipelineWorkflowNodeExecutions(runApp, user, ctx, run.Outputs, now)
	run.TotalSteps = len(run.NodeExecutions)
	run.TotalTokens = len(run.NodeExecutions) * 40
	run.ElapsedTime = workflowElapsed(len(run.NodeExecutions))
	return run
}

func ragPipelineStoredRunOutputs(ctx ragPipelinePublishedRunContext, previewOutputs map[string]any) map[string]any {
	outputs := map[string]any{
		"mode":            ragPipelineRunMode(ctx),
		"dataset_id":      ctx.Dataset.ID,
		"pipeline_id":     ctx.Dataset.PipelineID,
		"datasource_type": ctx.DatasourceType,
		"start_node_id":   ctx.StartNodeID,
		"datasource": map[string]any{
			"type":  ctx.DatasourceType,
			"count": len(ctx.DatasourceInfoList),
			"items": cloneMapList(ctx.DatasourceInfoList),
		},
		"processing_inputs": cloneJSONObject(ctx.Inputs),
	}
	if ctx.OriginalDocumentID != "" {
		outputs["original_document_id"] = ctx.OriginalDocumentID
	}
	if ctx.IsPreview {
		for key, value := range cloneJSONObject(previewOutputs) {
			outputs[key] = value
		}
		outputs["preview_result"] = cloneJSONObject(previewOutputs)
		return outputs
	}

	outputs["batch"] = ctx.BatchID
	outputs["document_ids"] = ragPipelineDocumentIDs(ctx.Documents)
	outputs["document_count"] = len(ctx.Documents)
	outputs["documents"] = ragPipelineWorkflowRunDocuments(ctx.Documents)
	return outputs
}

func ragPipelineRunMode(ctx ragPipelinePublishedRunContext) string {
	switch {
	case ctx.IsPreview:
		return "preview"
	case strings.TrimSpace(ctx.OriginalDocumentID) != "":
		return "reprocess"
	default:
		return "create"
	}
}

func ragPipelineWorkflowRunInputs(ctx ragPipelinePublishedRunContext) map[string]any {
	inputs := map[string]any{
		"pipeline_id":          ctx.Dataset.PipelineID,
		"dataset_id":           ctx.Dataset.ID,
		"start_node_id":        ctx.StartNodeID,
		"datasource_type":      ctx.DatasourceType,
		"datasource_info_list": cloneMapList(ctx.DatasourceInfoList),
		"processing_inputs":    cloneJSONObject(ctx.Inputs),
		"is_preview":           ctx.IsPreview,
	}
	if ctx.OriginalDocumentID != "" {
		inputs["original_document_id"] = ctx.OriginalDocumentID
	}
	return inputs
}

func (s *server) ragPipelineWorkflowNodeExecutions(app state.App, user state.User, ctx ragPipelinePublishedRunContext, runOutputs map[string]any, now time.Time) []state.WorkflowNodeExecution {
	nodes := workflowGraphNodes(app.WorkflowDraft.Graph, nil)
	executions := make([]state.WorkflowNodeExecution, 0, len(nodes))
	for index, node := range nodes {
		executions = append(executions, state.WorkflowNodeExecution{
			ID:                runtimeID("node"),
			Index:             index,
			PredecessorNodeID: node.PredecessorNode,
			NodeID:            node.ID,
			NodeType:          node.NodeType,
			Title:             node.Title,
			Inputs:            ragPipelineNodeExecutionInputs(node, ctx),
			ProcessData:       ragPipelineNodeProcessData(node, ctx),
			Outputs:           ragPipelineNodeExecutionOutputs(node, ctx, runOutputs, index == len(nodes)-1),
			Status:            "succeeded",
			ElapsedTime:       workflowNodeElapsed(index),
			TotalTokens:       32 + index*10,
			TotalPrice:        0,
			Currency:          "USD",
			CreatedAt:         now.UTC().Unix(),
			FinishedAt:        now.UTC().Unix(),
			CreatedBy:         user.ID,
		})
	}
	return executions
}

func ragPipelineNodeExecutionInputs(node workflowNodeMeta, ctx ragPipelinePublishedRunContext) map[string]any {
	switch node.NodeType {
	case "knowledge-index":
		return map[string]any{
			"dataset_id":           ctx.Dataset.ID,
			"pipeline_id":          ctx.Dataset.PipelineID,
			"start_node_id":        ctx.StartNodeID,
			"datasource_type":      ctx.DatasourceType,
			"datasource_info_list": cloneMapList(ctx.DatasourceInfoList),
			"processing_inputs":    cloneJSONObject(ctx.Inputs),
		}
	case "start":
		return ragPipelineWorkflowRunInputs(ctx)
	default:
		if len(ctx.Inputs) > 0 {
			return cloneJSONObject(ctx.Inputs)
		}
		return map[string]any{
			"datasource_type": ctx.DatasourceType,
			"start_node_id":   ctx.StartNodeID,
		}
	}
}

func ragPipelineNodeProcessData(node workflowNodeMeta, ctx ragPipelinePublishedRunContext) map[string]any {
	mode := ragPipelineRunMode(ctx)
	switch node.NodeType {
	case "knowledge-index":
		summary := "Knowledge indexing completed."
		switch mode {
		case "preview":
			summary = fmt.Sprintf("Generated preview from %d datasource item(s).", len(ctx.DatasourceInfoList))
		case "reprocess":
			summary = fmt.Sprintf("Reprocessed %d document(s) into batch %s.", len(ctx.Documents), ctx.BatchID)
		default:
			summary = fmt.Sprintf("Created %d document(s) in batch %s.", len(ctx.Documents), ctx.BatchID)
		}
		return map[string]any{
			"summary":          summary,
			"mode":             mode,
			"phase":            "knowledge-index",
			"datasource_count": len(ctx.DatasourceInfoList),
			"document_count":   len(ctx.Documents),
		}
	default:
		return map[string]any{
			"summary": fmt.Sprintf("%s completed for %s mode.", node.Title, mode),
			"mode":    mode,
			"phase":   "pipeline",
		}
	}
}

func ragPipelineNodeExecutionOutputs(node workflowNodeMeta, ctx ragPipelinePublishedRunContext, runOutputs map[string]any, isLast bool) map[string]any {
	switch node.NodeType {
	case "knowledge-index":
		outputs := map[string]any{
			"mode":             ragPipelineRunMode(ctx),
			"dataset_id":       ctx.Dataset.ID,
			"pipeline_id":      ctx.Dataset.PipelineID,
			"datasource_type":  ctx.DatasourceType,
			"start_node_id":    ctx.StartNodeID,
			"datasource_count": len(ctx.DatasourceInfoList),
		}
		if ctx.IsPreview {
			if preview := mapFromAny(runOutputs["preview_result"]); len(preview) > 0 {
				outputs["preview_result"] = preview
			} else {
				outputs["preview_result"] = cloneJSONObject(runOutputs)
			}
			return outputs
		}
		outputs["batch"] = ctx.BatchID
		outputs["documents"] = ragPipelineWorkflowRunDocuments(ctx.Documents)
		outputs["document_ids"] = ragPipelineDocumentIDs(ctx.Documents)
		outputs["document_count"] = len(ctx.Documents)
		return outputs
	default:
		if isLast {
			return cloneJSONObject(runOutputs)
		}
		return map[string]any{
			"mode":             ragPipelineRunMode(ctx),
			"datasource_type":  ctx.DatasourceType,
			"datasource_count": len(ctx.DatasourceInfoList),
		}
	}
}

func ragPipelineWorkflowRunDocuments(documents []state.DatasetDocument) []map[string]any {
	items := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		items = append(items, map[string]any{
			"id":               document.ID,
			"name":             document.Name,
			"data_source_type": document.DataSourceType,
			"indexing_status":  document.IndexingStatus,
			"display_status":   document.DisplayStatus,
		})
	}
	return items
}

func ragPipelineCloneDocuments(documents []state.DatasetDocument) []state.DatasetDocument {
	if documents == nil {
		return []state.DatasetDocument{}
	}
	return append([]state.DatasetDocument(nil), documents...)
}

func ragPipelinePublishedDocumentInput(dataset state.Dataset, workflow state.WorkflowState, datasourceType string, datasourceInfoList []map[string]any, inputs map[string]any, startNodeID string) state.CreateDatasetDocumentInput {
	workflowPatch := ragPipelineDatasetPatchFromWorkflow(map[string]any{
		"graph": workflow.Graph,
	})

	retrievalModel := dataset.RetrievalModel
	if patchValue := mapFromAny(workflowPatch["retrieval_model"]); len(patchValue) > 0 {
		decodeInto(patchValue, &retrievalModel)
	}

	summaryIndexSetting := dataset.SummaryIndexSetting
	if patchValue := mapFromAny(workflowPatch["summary_index_setting"]); len(patchValue) > 0 {
		decodeInto(patchValue, &summaryIndexSetting)
	}
	if enabled, ok := ragPipelineBoolValueWithOK(inputs["summary_index_enabled"]); ok {
		summaryIndexSetting.Enable = enabled
	}

	processRule := ragPipelineProcessRuleFromInputs(inputs, summaryIndexSetting)
	normalizedInputs := ragPipelineExecutionInputs(inputs, processRule, summaryIndexSetting, firstNonEmpty(stringFromAny(inputs["doc_language"]), "English"), firstNonEmpty(stringFromAny(inputs["doc_form"]), stringFromAny(workflowPatch["doc_form"]), dataset.DocForm, "text_model"))

	return state.CreateDatasetDocumentInput{
		DataSourceType:         datasourceType,
		DataSource:             ragPipelineDataSourcePayload(datasourceInfoList),
		DocForm:                firstNonEmpty(stringFromAny(normalizedInputs["doc_form"]), stringFromAny(workflowPatch["doc_form"]), dataset.DocForm, "text_model"),
		DocLanguage:            firstNonEmpty(stringFromAny(normalizedInputs["doc_language"]), "English"),
		IndexingTechnique:      firstNonEmpty(stringFromAny(workflowPatch["indexing_technique"]), dataset.IndexingTechnique),
		RetrievalModel:         retrievalModel,
		EmbeddingModel:         firstNonEmpty(stringFromAny(workflowPatch["embedding_model"]), dataset.EmbeddingModel),
		EmbeddingModelProvider: firstNonEmpty(stringFromAny(workflowPatch["embedding_model_provider"]), dataset.EmbeddingModelProvider),
		ProcessRule:            processRule,
		SummaryIndexSetting:    summaryIndexSetting,
		CreatedFrom:            "rag-pipeline",
		PipelineExecutionLogs:  ragPipelineExecutionLogs(datasourceType, datasourceInfoList, normalizedInputs, startNodeID),
	}
}

func ragPipelinePreviewOutputs(docForm, datasourceType string, datasourceInfoList []map[string]any, processRule state.DatasetProcessRule) map[string]any {
	return buildPreviewEstimateOutputs(docForm, datasourceType, datasourceInfoList, processRule)
}

func ragPipelineInitialDocumentListPayload(documents []state.DatasetDocument) []map[string]any {
	data := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		data = append(data, map[string]any{
			"id":               document.ID,
			"data_source_info": cloneJSONObject(document.DataSourceInfo),
			"data_source_type": document.DataSourceType,
			"enable":           document.Enabled,
			"error":            document.Error,
			"indexing_status":  document.IndexingStatus,
			"name":             document.Name,
			"position":         document.Position,
		})
	}
	return data
}

func ragPipelineDocumentIDs(documents []state.DatasetDocument) []any {
	ids := make([]any, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.ID)
	}
	return ids
}

func ragPipelineDataSourcePayload(datasourceInfoList []map[string]any) map[string]any {
	return map[string]any{
		"info_list": map[string]any{
			"datasource_info_list": cloneMapList(datasourceInfoList),
		},
	}
}

func ragPipelineExecutionLogs(datasourceType string, datasourceInfoList []map[string]any, inputs map[string]any, startNodeID string) []state.DatasetPipelineExecutionLog {
	logs := make([]state.DatasetPipelineExecutionLog, 0, len(datasourceInfoList))
	for _, info := range datasourceInfoList {
		logs = append(logs, state.DatasetPipelineExecutionLog{
			DatasourceType:   datasourceType,
			DatasourceInfo:   cloneJSONObject(info),
			InputData:        cloneJSONObject(inputs),
			DatasourceNodeID: startNodeID,
		})
	}
	return logs
}

func ragPipelineProcessRuleFromInputs(inputs map[string]any, summaryIndexSetting state.DatasetSummaryIndexSetting) state.DatasetProcessRule {
	rule := state.DatasetProcessRule{
		Mode: "custom",
		Rules: state.DatasetProcessRuleSettings{
			PreProcessingRules: []state.DatasetPreProcessingRule{
				{ID: "remove_extra_spaces", Enabled: false},
				{ID: "remove_urls_emails", Enabled: false},
			},
			Segmentation: state.DatasetSegmentation{
				Separator:    "\n\n",
				MaxTokens:    1000,
				ChunkOverlap: 50,
			},
			ParentMode: "paragraph",
			SubchunkSegmentation: state.DatasetSegmentation{
				Separator:    "\n",
				MaxTokens:    300,
				ChunkOverlap: 30,
			},
		},
		SummaryIndexSetting: summaryIndexSetting,
	}

	if separator := stringFromAny(inputs["separator"]); separator != "" {
		rule.Rules.Segmentation.Separator = separator
	}
	if chunkSize, ok := ragPipelineIntValue(inputs["chunk_size"]); ok && chunkSize > 0 {
		rule.Rules.Segmentation.MaxTokens = chunkSize
	}
	if chunkOverlap, ok := ragPipelineIntValue(inputs["chunk_overlap"]); ok && chunkOverlap >= 0 {
		rule.Rules.Segmentation.ChunkOverlap = chunkOverlap
	}
	if separator := stringFromAny(inputs["subchunk_separator"]); separator != "" {
		rule.Rules.SubchunkSegmentation.Separator = separator
	}
	if chunkSize, ok := ragPipelineIntValue(inputs["subchunk_chunk_size"]); ok && chunkSize > 0 {
		rule.Rules.SubchunkSegmentation.MaxTokens = chunkSize
	}
	if chunkOverlap, ok := ragPipelineIntValue(inputs["subchunk_chunk_overlap"]); ok && chunkOverlap >= 0 {
		rule.Rules.SubchunkSegmentation.ChunkOverlap = chunkOverlap
	}
	if parentMode := stringFromAny(inputs["parent_mode"]); parentMode != "" {
		rule.Rules.ParentMode = parentMode
	}
	if enabled, ok := ragPipelineBoolValueWithOK(inputs["remove_extra_spaces"]); ok {
		rule.Rules.PreProcessingRules[0].Enabled = enabled
	}
	if enabled, ok := ragPipelineBoolValueWithOK(inputs["remove_urls_emails"]); ok {
		rule.Rules.PreProcessingRules[1].Enabled = enabled
	}
	return rule
}

func ragPipelineExecutionInputs(inputs map[string]any, rule state.DatasetProcessRule, summaryIndexSetting state.DatasetSummaryIndexSetting, docLanguage, docForm string) map[string]any {
	normalized := cloneJSONObject(inputs)
	normalized["separator"] = firstImportValue(stringFromAny(normalized["separator"]), rule.Rules.Segmentation.Separator)
	normalized["chunk_size"] = ragPipelineIntValueOrDefault(normalized["chunk_size"], rule.Rules.Segmentation.MaxTokens)
	normalized["chunk_overlap"] = ragPipelineIntValueOrDefault(normalized["chunk_overlap"], rule.Rules.Segmentation.ChunkOverlap)
	normalized["subchunk_separator"] = firstImportValue(stringFromAny(normalized["subchunk_separator"]), rule.Rules.SubchunkSegmentation.Separator)
	normalized["subchunk_chunk_size"] = ragPipelineIntValueOrDefault(normalized["subchunk_chunk_size"], rule.Rules.SubchunkSegmentation.MaxTokens)
	normalized["subchunk_chunk_overlap"] = ragPipelineIntValueOrDefault(normalized["subchunk_chunk_overlap"], rule.Rules.SubchunkSegmentation.ChunkOverlap)
	normalized["parent_mode"] = firstImportValue(stringFromAny(normalized["parent_mode"]), rule.Rules.ParentMode)
	normalized["doc_language"] = firstImportValue(stringFromAny(normalized["doc_language"]), docLanguage)
	normalized["doc_form"] = firstImportValue(stringFromAny(normalized["doc_form"]), docForm)
	normalized["summary_index_enabled"] = ragPipelineBoolValueOrDefault(normalized["summary_index_enabled"], summaryIndexSetting.Enable)
	normalized["remove_extra_spaces"] = ragPipelineBoolValueOrDefault(normalized["remove_extra_spaces"], rule.Rules.PreProcessingRules[0].Enabled)
	normalized["remove_urls_emails"] = ragPipelineBoolValueOrDefault(normalized["remove_urls_emails"], rule.Rules.PreProcessingRules[1].Enabled)
	return normalized
}

func ragPipelineDatasourceNodeID(datasourceType string) string {
	switch strings.TrimSpace(datasourceType) {
	case "online_document":
		return "datasource-online-document"
	case "website_crawl":
		return "datasource-website-crawl"
	case "online_drive":
		return "datasource-online-drive"
	default:
		return "datasource-local-file"
	}
}

func ragPipelineBoolValue(value any) bool {
	boolean, _ := ragPipelineBoolValueWithOK(value)
	return boolean
}

func ragPipelineBoolValueWithOK(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true, true
		case "false", "0", "no":
			return false, true
		}
	}
	return false, false
}

func ragPipelineBoolValueOrDefault(value any, fallback bool) bool {
	if boolean, ok := ragPipelineBoolValueWithOK(value); ok {
		return boolean
	}
	return fallback
}

func ragPipelineIntValue(value any) (int, bool) {
	if number, ok := numberValue(value); ok {
		return int(number), true
	}
	if str, ok := value.(string); ok {
		parsed, err := strconv.Atoi(strings.TrimSpace(str))
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func ragPipelineIntValueOrDefault(value any, fallback int) int {
	if parsed, ok := ragPipelineIntValue(value); ok {
		return parsed
	}
	return fallback
}
