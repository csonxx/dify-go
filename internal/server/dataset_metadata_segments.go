package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleDatasetBuiltInMetadataFields(w http.ResponseWriter, r *http.Request) {
	fields := state.DefaultBuiltInDatasetMetadataFields()
	data := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		data = append(data, map[string]any{
			"name": field.Name,
			"type": field.Type,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"fields": data})
}

func (s *server) handleDatasetMetadataList(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"doc_metadata":           datasetMetadataListPayload(dataset),
		"built_in_field_enabled": dataset.BuiltInFieldEnabled,
	})
}

func (s *server) handleDatasetMetadataCreate(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	var payload struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	field, err := s.store.CreateDatasetMetadataField(dataset.ID, dataset.WorkspaceID, state.DatasetMetadataField{
		Name: payload.Name,
		Type: payload.Type,
	}, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":    field.ID,
		"name":  field.Name,
		"type":  field.Type,
		"count": 0,
	})
}

func (s *server) handleDatasetMetadataRename(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	field, err := s.store.RenameDatasetMetadataField(dataset.ID, dataset.WorkspaceID, chi.URLParam(r, "metadataID"), payload.Name, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    field.ID,
		"name":  field.Name,
		"type":  field.Type,
		"count": datasetMetadataCount(dataset, field.ID),
	})
}

func (s *server) handleDatasetMetadataDelete(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	if err := s.store.DeleteDatasetMetadataField(dataset.ID, dataset.WorkspaceID, chi.URLParam(r, "metadataID"), currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetBuiltInMetadataEnable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetBuiltInMetadataStatus(w, r, true)
}

func (s *server) handleDatasetBuiltInMetadataDisable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetBuiltInMetadataStatus(w, r, false)
}

func (s *server) handleDatasetBuiltInMetadataStatus(w http.ResponseWriter, r *http.Request, enabled bool) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	if _, err := s.store.SetDatasetBuiltInFieldEnabled(dataset.ID, dataset.WorkspaceID, enabled, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentMetadataBatchUpdate(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	var payload struct {
		OperationData []struct {
			DocumentID    string `json:"document_id"`
			PartialUpdate bool   `json:"partial_update"`
			MetadataList  []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Type  string `json:"type"`
				Value any    `json:"value"`
			} `json:"metadata_list"`
		} `json:"operation_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	operations := make([]state.DatasetDocumentMetadataOperation, 0, len(payload.OperationData))
	for _, item := range payload.OperationData {
		op := state.DatasetDocumentMetadataOperation{
			DocumentID:    item.DocumentID,
			PartialUpdate: item.PartialUpdate,
			MetadataList:  make([]state.DatasetDocumentMetadataItem, 0, len(item.MetadataList)),
		}
		for _, meta := range item.MetadataList {
			op.MetadataList = append(op.MetadataList, state.DatasetDocumentMetadataItem{
				ID:    meta.ID,
				Name:  meta.Name,
				Type:  meta.Type,
				Value: meta.Value,
			})
		}
		operations = append(operations, op)
	}
	if err := s.store.BatchUpdateDatasetDocumentMetadata(dataset.ID, dataset.WorkspaceID, operations, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentMetadataUpdate(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	var payload struct {
		DocType     string         `json:"doc_type"`
		DocMetadata map[string]any `json:"doc_metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if err := s.store.UpdateDatasetDocumentMetadataProfile(dataset.ID, dataset.WorkspaceID, document.ID, payload.DocType, payload.DocMetadata, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetSegmentList(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	page, ok := s.store.ListDatasetSegments(dataset.ID, dataset.WorkspaceID, document.ID, state.DatasetSegmentFilters{
		Page:    intQuery(r, "page", 1),
		Limit:   intQuery(r, "limit", 20),
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Enabled: strings.TrimSpace(r.URL.Query().Get("enabled")),
	})
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	data := make([]map[string]any, 0, len(page.Data))
	for _, segment := range page.Data {
		data = append(data, datasetSegmentPayload(segment))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":        data,
		"has_more":    page.HasMore,
		"limit":       page.Limit,
		"total":       page.Total,
		"total_pages": page.TotalPages,
		"page":        page.Page,
	})
}

func (s *server) handleDatasetSegmentCreate(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	input, err := datasetSegmentInputFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	segment, docForm, err := s.store.AddDatasetSegment(dataset.ID, dataset.WorkspaceID, document.ID, input, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"data":     datasetSegmentPayload(segment),
		"doc_form": nullIfEmpty(docForm),
	})
}

func (s *server) handleDatasetSegmentUpdate(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	input, err := datasetSegmentInputFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	segment, docForm, err := s.store.UpdateDatasetSegment(dataset.ID, dataset.WorkspaceID, document.ID, chi.URLParam(r, "segmentID"), input, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     datasetSegmentPayload(segment),
		"doc_form": nullIfEmpty(docForm),
	})
}

func (s *server) handleDatasetSegmentEnable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetSegmentStatus(w, r, true)
}

func (s *server) handleDatasetSegmentDisable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetSegmentStatus(w, r, false)
}

func (s *server) handleDatasetSegmentStatus(w http.ResponseWriter, r *http.Request, enabled bool) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err := s.store.SetDatasetSegmentEnabled(dataset.ID, dataset.WorkspaceID, document.ID, r.URL.Query()["segment_id"], enabled, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetSegmentDelete(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err := s.store.DeleteDatasetSegments(dataset.ID, dataset.WorkspaceID, document.ID, r.URL.Query()["segment_id"], currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetChildChunkList(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	page, ok := s.store.ListDatasetChildChunks(dataset.ID, dataset.WorkspaceID, document.ID, chi.URLParam(r, "segmentID"), state.DatasetChildChunkFilters{
		Page:    intQuery(r, "page", 1),
		Limit:   intQuery(r, "limit", 20),
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
	})
	if !ok {
		writeError(w, http.StatusNotFound, "segment_not_found", "Segment not found.")
		return
	}
	data := make([]map[string]any, 0, len(page.Data))
	for _, chunk := range page.Data {
		data = append(data, datasetChildChunkDetailPayload(chunk))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":        data,
		"total":       page.Total,
		"total_pages": page.TotalPages,
		"page":        page.Page,
		"limit":       page.Limit,
	})
}

func (s *server) handleDatasetChildChunkCreate(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	chunk, err := s.store.AddDatasetChildChunk(dataset.ID, dataset.WorkspaceID, document.ID, chi.URLParam(r, "segmentID"), payload.Content, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": datasetChildChunkDetailPayload(chunk)})
}

func (s *server) handleDatasetChildChunkUpdate(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	chunk, err := s.store.UpdateDatasetChildChunk(dataset.ID, dataset.WorkspaceID, document.ID, chi.URLParam(r, "segmentID"), chi.URLParam(r, "childChunkID"), payload.Content, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": datasetChildChunkDetailPayload(chunk)})
}

func (s *server) handleDatasetChildChunkDelete(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err := s.store.DeleteDatasetChildChunk(dataset.ID, dataset.WorkspaceID, document.ID, chi.URLParam(r, "segmentID"), chi.URLParam(r, "childChunkID"), currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetSegmentBatchImport(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	var payload struct {
		UploadFileID string `json:"upload_file_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	job, err := s.store.CreateDatasetBatchImportJob(dataset.ID, dataset.WorkspaceID, document.ID, payload.UploadFileID, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"job_id":     job.ID,
		"job_status": job.Status,
	})
}

func (s *server) handleDatasetBatchImportStatus(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	job, ok := s.store.GetDatasetBatchImportJob(workspace.ID, chi.URLParam(r, "jobID"))
	if !ok {
		writeError(w, http.StatusNotFound, "job_not_found", "Batch import job not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"job_id":     job.ID,
		"job_status": job.Status,
	})
}

func datasetMetadataListPayload(dataset state.Dataset) []map[string]any {
	data := make([]map[string]any, 0, len(dataset.MetadataFields))
	for _, field := range dataset.MetadataFields {
		data = append(data, map[string]any{
			"id":    field.ID,
			"name":  field.Name,
			"type":  field.Type,
			"count": datasetMetadataCount(dataset, field.ID),
		})
	}
	return data
}

func datasetMetadataDefinitionPayload(fields []state.DatasetMetadataField) []map[string]any {
	data := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		data = append(data, map[string]any{
			"id":    field.ID,
			"name":  field.Name,
			"type":  field.Type,
			"value": nil,
		})
	}
	return data
}

func datasetMetadataCount(dataset state.Dataset, metadataID string) int {
	count := 0
	for _, document := range dataset.Documents {
		if _, ok := document.MetadataValues[metadataID]; ok {
			count++
		}
	}
	return count
}

func datasetDocumentMetadataListPayload(dataset state.Dataset, document state.DatasetDocument, includeBuiltIn bool) []map[string]any {
	data := make([]map[string]any, 0, len(document.MetadataValues)+5)
	for _, field := range dataset.MetadataFields {
		value, ok := document.MetadataValues[field.ID]
		if !ok {
			continue
		}
		data = append(data, map[string]any{
			"id":    field.ID,
			"name":  field.Name,
			"type":  field.Type,
			"value": datasetMetadataValueByType(field.Type, value),
		})
	}
	if includeBuiltIn && dataset.BuiltInFieldEnabled {
		for _, field := range state.DefaultBuiltInDatasetMetadataFields() {
			data = append(data, map[string]any{
				"id":    "built-in",
				"name":  field.Name,
				"type":  field.Type,
				"value": datasetBuiltInMetadataValue(field.Name, document),
			})
		}
	}
	return data
}

func datasetBuiltInMetadataValue(name string, document state.DatasetDocument) any {
	switch name {
	case "source":
		return document.DataSourceType
	case "created_at":
		return nullableInt64(document.CreatedAt)
	case "updated_at":
		return nullableInt64(firstNonZero(document.CompletedAt, document.UpdatedAt))
	case "segment_count":
		return document.SegmentCount
	case "hit_count":
		return document.HitCount
	default:
		return nil
	}
}

func datasetMetadataValueByType(fieldType, raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	switch fieldType {
	case "number", "time":
		if value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil {
			return value
		}
		if value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			return value
		}
	}
	return raw
}

func datasetSegmentPayload(segment state.DatasetSegment) map[string]any {
	return map[string]any{
		"id":              segment.ID,
		"position":        segment.Position,
		"document_id":     segment.DocumentID,
		"content":         segment.Content,
		"sign_content":    segment.SignContent,
		"word_count":      segment.WordCount,
		"tokens":          segment.Tokens,
		"keywords":        cloneStringSliceAny(segment.Keywords),
		"index_node_id":   segment.IndexNodeID,
		"index_node_hash": segment.IndexNodeHash,
		"hit_count":       segment.HitCount,
		"enabled":         segment.Enabled,
		"disabled_at":     nullableInt64(segment.DisabledAt),
		"disabled_by":     nullIfEmpty(segment.DisabledBy),
		"status":          segment.Status,
		"created_by":      segment.CreatedBy,
		"created_at":      segment.CreatedAt,
		"indexing_at":     segment.IndexingAt,
		"completed_at":    nullableInt64(segment.CompletedAt),
		"error":           nullIfEmpty(segment.Error),
		"stopped_at":      nullableInt64(segment.StoppedAt),
		"answer":          nullIfEmpty(segment.Answer),
		"summary":         nullIfEmpty(segment.Summary),
		"child_chunks":    datasetChildChunkDetailListPayload(segment.ChildChunks),
		"updated_at":      segment.UpdatedAt,
		"attachments":     datasetAttachmentsPayload(segment.Attachments),
	}
}

func datasetChildChunkDetailListPayload(chunks []state.DatasetChildChunk) []map[string]any {
	data := make([]map[string]any, 0, len(chunks))
	for _, chunk := range chunks {
		data = append(data, datasetChildChunkDetailPayload(chunk))
	}
	return data
}

func datasetChildChunkDetailPayload(chunk state.DatasetChildChunk) map[string]any {
	return map[string]any{
		"id":         chunk.ID,
		"position":   chunk.Position,
		"segment_id": chunk.SegmentID,
		"content":    chunk.Content,
		"word_count": chunk.WordCount,
		"created_at": chunk.CreatedAt,
		"updated_at": chunk.UpdatedAt,
		"type":       nullIfEmpty(chunk.Type),
	}
}

func datasetSegmentInputFromRequest(r *http.Request) (state.DatasetSegmentInput, error) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return state.DatasetSegmentInput{}, err
	}
	input := state.DatasetSegmentInput{
		Content: strings.TrimSpace(stringFromAny(payload["content"])),
	}
	if value, ok := payload["answer"]; ok {
		answer := strings.TrimSpace(stringFromAny(value))
		input.Answer = &answer
	}
	if value, ok := payload["summary"]; ok {
		summary := strings.TrimSpace(stringFromAny(value))
		input.Summary = &summary
	}
	if value, ok := payload["keywords"]; ok {
		keywords := stringSliceFromJSONAny(value)
		input.Keywords = &keywords
	}
	if value, ok := payload["attachment_ids"]; ok {
		attachmentIDs := stringSliceFromJSONAny(value)
		input.AttachmentIDs = &attachmentIDs
	}
	if value, ok := payload["regenerate_child_chunks"].(bool); ok {
		input.RegenerateChildChunks = value
	}
	return input, nil
}

func stringSliceFromJSONAny(value any) []string {
	items := anySlice(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return out
}

func cloneStringSliceAny(values []string) []any {
	data := make([]any, 0, len(values))
	for _, value := range values {
		data = append(data, value)
	}
	return data
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func datasetAverageSegmentLength(document state.DatasetDocument) int {
	if document.SegmentCount <= 0 {
		return 0
	}
	return document.WordCount / document.SegmentCount
}
