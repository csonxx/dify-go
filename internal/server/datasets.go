package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) mountDatasetRoutes(r chi.Router) {
	r.Get("/datasets/retrieval-setting", s.handleDatasetRetrievalSetting)
	r.Get("/datasets/process-rule", s.handleDatasetProcessRule)
	r.Post("/datasets/indexing-estimate", s.handleDatasetIndexingEstimate)
	r.Get("/datasets/api-base-info", s.handleDatasetAPIBaseInfo)
	r.Get("/datasets/api-keys", s.handleDatasetAPIKeyList)
	r.Post("/datasets/api-keys", s.handleDatasetAPIKeyCreate)
	r.Delete("/datasets/api-keys/{keyID}", s.handleDatasetAPIKeyDelete)
	r.Post("/datasets/external", s.handleDatasetExternalCreate)
	r.Route("/datasets/external-knowledge-api", func(r chi.Router) {
		r.Get("/", s.handleDatasetExternalKnowledgeAPIList)
		r.Post("/", s.handleDatasetExternalKnowledgeAPICreate)
		r.Route("/{apiID}", func(r chi.Router) {
			r.Get("/", s.handleDatasetExternalKnowledgeAPIDetail)
			r.Patch("/", s.handleDatasetExternalKnowledgeAPIUpdate)
			r.Delete("/", s.handleDatasetExternalKnowledgeAPIDelete)
			r.Get("/use-check", s.handleDatasetExternalKnowledgeAPIUseCheck)
		})
	})
	r.Get("/datasets/metadata/built-in", s.handleDatasetBuiltInMetadataFields)
	r.Get("/datasets/batch_import_status/{jobID}", s.handleDatasetBatchImportStatus)
	r.Get("/datasets", s.handleDatasetList)
	r.Post("/datasets", s.handleDatasetCreate)
	r.Post("/datasets/init", s.handleDatasetInit)
	r.Route("/datasets/{datasetID}", func(r chi.Router) {
		r.Get("/", s.handleDatasetDetail)
		r.Patch("/", s.handleDatasetUpdate)
		r.Delete("/", s.handleDatasetDelete)
		r.Get("/use-check", s.handleDatasetUseCheck)
		r.Get("/related-apps", s.handleDatasetRelatedApps)
		r.Post("/api-keys/enable", s.handleDatasetAPIEnable)
		r.Post("/api-keys/disable", s.handleDatasetAPIDisable)
		r.Get("/metadata", s.handleDatasetMetadataList)
		r.Post("/metadata", s.handleDatasetMetadataCreate)
		r.Patch("/metadata/{metadataID}", s.handleDatasetMetadataRename)
		r.Delete("/metadata/{metadataID}", s.handleDatasetMetadataDelete)
		r.Post("/metadata/built-in/enable", s.handleDatasetBuiltInMetadataEnable)
		r.Post("/metadata/built-in/disable", s.handleDatasetBuiltInMetadataDisable)
		r.Get("/documents", s.handleDatasetDocumentList)
		r.Post("/documents", s.handleDatasetDocumentCreate)
		r.Delete("/documents", s.handleDatasetDocumentDelete)
		r.Post("/documents/download-zip", s.handleDatasetDocumentDownloadZip)
		r.Post("/documents/metadata", s.handleDatasetDocumentMetadataBatchUpdate)
		r.Patch("/documents/status/{action}/batch", s.handleDatasetDocumentBatchAction)
		r.Post("/documents/generate-summary", s.handleDatasetDocumentGenerateSummary)
		r.Get("/documents/{documentID}", s.handleDatasetDocumentDetail)
		r.Get("/documents/{documentID}/download", s.handleDatasetDocumentDownload)
		r.Get("/documents/{documentID}/pipeline-execution-log", s.handleDatasetDocumentPipelineExecutionLog)
		r.Get("/documents/{documentID}/notion/sync", s.handleDatasetDocumentNotionSync)
		r.Get("/documents/{documentID}/website-sync", s.handleDatasetDocumentWebsiteSync)
		r.Put("/documents/{documentID}/metadata", s.handleDatasetDocumentMetadataUpdate)
		r.Get("/documents/{documentID}/indexing-status", s.handleDatasetDocumentIndexingStatus)
		r.Post("/documents/{documentID}/rename", s.handleDatasetDocumentRename)
		r.Patch("/documents/{documentID}/processing/pause", s.handleDatasetDocumentPause)
		r.Patch("/documents/{documentID}/processing/resume", s.handleDatasetDocumentResume)
		r.Get("/documents/{documentID}/segments", s.handleDatasetSegmentList)
		r.Delete("/documents/{documentID}/segments", s.handleDatasetSegmentDelete)
		r.Post("/documents/{documentID}/segments/batch_import", s.handleDatasetSegmentBatchImport)
		r.Post("/documents/{documentID}/segment", s.handleDatasetSegmentCreate)
		r.Patch("/documents/{documentID}/segment/enable", s.handleDatasetSegmentEnable)
		r.Patch("/documents/{documentID}/segment/disable", s.handleDatasetSegmentDisable)
		r.Patch("/documents/{documentID}/segments/{segmentID}", s.handleDatasetSegmentUpdate)
		r.Get("/documents/{documentID}/segments/{segmentID}/child_chunks", s.handleDatasetChildChunkList)
		r.Post("/documents/{documentID}/segments/{segmentID}/child_chunks", s.handleDatasetChildChunkCreate)
		r.Patch("/documents/{documentID}/segments/{segmentID}/child_chunks/{childChunkID}", s.handleDatasetChildChunkUpdate)
		r.Delete("/documents/{documentID}/segments/{segmentID}/child_chunks/{childChunkID}", s.handleDatasetChildChunkDelete)
		r.Get("/batch/{batchID}/indexing-status", s.handleDatasetBatchIndexingStatus)
		r.Get("/auto-disable-logs", s.handleDatasetAutoDisableLogs)
		r.Get("/queries", s.handleDatasetQueries)
		r.Get("/error-docs", s.handleDatasetErrorDocs)
		r.Post("/hit-testing", s.handleDatasetHitTesting)
		r.Post("/external-hit-testing", s.handleDatasetExternalHitTesting)
		r.Post("/retry", s.handleDatasetRetry)
		r.NotFound(s.compatFallback)
	})
}

func (s *server) handleDatasetRetrievalSetting(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"retrieval_method": []string{
			"semantic_search",
			"full_text_search",
			"hybrid_search",
			"keyword_search",
		},
	})
}

func (s *server) handleDatasetProcessRule(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	if documentID := strings.TrimSpace(r.URL.Query().Get("document_id")); documentID != "" {
		_, document, found := s.store.FindDatasetDocument(workspace.ID, documentID)
		if !found {
			writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
			return
		}
		writeJSON(w, http.StatusOK, datasetProcessRulePayload(document.DatasetProcessRule))
		return
	}

	writeJSON(w, http.StatusOK, datasetProcessRulePayload(state.DatasetProcessRule{}))
}

func (s *server) handleDatasetIndexingEstimate(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	dataSource := mapFromAny(payload["info_list"])
	dataSourceType := stringFromAny(dataSource["data_source_type"])
	totalNodes := estimateDatasetNodeCount(dataSourceType, dataSource)
	if totalNodes <= 0 {
		totalNodes = 1
	}

	preview := make([]map[string]any, 0, totalNodes)
	for i := 0; i < totalNodes; i++ {
		preview = append(preview, map[string]any{
			"content":      fmt.Sprintf("Preview chunk %d generated by dify-go dataset compatibility layer.", i+1),
			"child_chunks": []string{},
			"summary":      "Compatible preview summary",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_nodes":    totalNodes,
		"tokens":         totalNodes * 128,
		"total_price":    0,
		"currency":       "USD",
		"total_segments": totalNodes,
		"preview":        preview,
	})
}

func (s *server) handleDatasetAPIBaseInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"api_base_url": "http://localhost:5001/v1/datasets",
	})
}

func (s *server) handleDatasetAPIKeyList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	keys, err := s.store.ListDatasetAPIKeys(workspace.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	data := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		item := map[string]any{
			"id":         key.ID,
			"type":       key.Type,
			"token":      key.Token,
			"created_at": key.CreatedAt,
		}
		if key.LastUsedAt != nil {
			item["last_used_at"] = *key.LastUsedAt
		} else {
			item["last_used_at"] = nil
		}
		data = append(data, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleDatasetAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	key, err := s.store.CreateDatasetAPIKey(workspace.ID, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         key.ID,
		"type":       key.Type,
		"token":      key.Token,
		"created_at": key.CreatedAt,
	})
}

func (s *server) handleDatasetAPIKeyDelete(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	keyID := strings.TrimSpace(chi.URLParam(r, "keyID"))
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Key ID is required.")
		return
	}

	if err := s.store.DeleteDatasetAPIKey(workspace.ID, keyID); err != nil {
		writeError(w, http.StatusNotFound, "api_key_not_found", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleDatasetExternalKnowledgeAPIList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	page := s.store.ListExternalKnowledgeAPIs(
		workspace.ID,
		intQuery(r, "page", 1),
		intQuery(r, "limit", 20),
	)
	data := make([]map[string]any, 0, len(page.Data))
	for _, item := range page.Data {
		data = append(data, s.externalKnowledgeAPIResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": page.HasMore,
		"limit":    page.Limit,
		"page":     page.Page,
		"total":    page.Total,
	})
}

func (s *server) handleDatasetList(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	_ = s.store.RefreshWorkspaceDatasetProcessingProgress(workspace.ID, time.Now())

	page := s.store.ListDatasets(workspace.ID, state.DatasetListFilters{
		Page:    intQuery(r, "page", 1),
		Limit:   intQuery(r, "limit", 20),
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		IDs:     r.URL.Query()["ids"],
	})
	data := make([]map[string]any, 0, len(page.Data))
	for _, dataset := range page.Data {
		data = append(data, s.datasetResponse(dataset))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": page.HasMore,
		"limit":    page.Limit,
		"page":     page.Page,
		"total":    page.Total,
	})
}

func (s *server) handleDatasetCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	user := currentUser(r)

	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	dataset, err := s.store.CreateDataset(workspace.ID, user, state.CreateDatasetInput{
		Name:        payload.Name,
		Description: payload.Description,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.datasetResponse(dataset))
}

func (s *server) handleDatasetInit(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	user := currentUser(r)

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	now := time.Now()
	dataset, err := s.store.CreateDataset(workspace.ID, user, state.CreateDatasetInput{
		Name: datasetNameFromPayload(payload),
	}, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input := datasetDocumentInputFromPayload(payload)
	batchID, documents, updatedDataset, err := s.store.CreateDatasetDocuments(dataset.ID, workspace.ID, user, input, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"dataset":   s.datasetResponse(updatedDataset),
		"batch":     batchID,
		"documents": s.datasetDocumentListPayload(updatedDataset, documents),
	})
}

func (s *server) handleDatasetDetail(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.datasetResponse(dataset))
}

func (s *server) handleDatasetUpdate(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	user := currentUser(r)

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.PatchDataset(dataset.ID, dataset.WorkspaceID, payload, user, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.datasetResponse(updated))
}

func (s *server) handleDatasetDelete(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	removed, err := s.store.DeleteDataset(dataset.ID, dataset.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.datasetResponse(removed))
}

func (s *server) handleDatasetUseCheck(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	count := s.store.DatasetUseCount(dataset.WorkspaceID, dataset.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"is_using": count > 0,
		"count":    count,
	})
}

func (s *server) handleDatasetRelatedApps(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	related := s.store.DatasetRelatedApps(dataset.WorkspaceID, dataset.ID)
	data := make([]map[string]any, 0, len(related))
	for _, item := range related {
		data = append(data, map[string]any{
			"id":              item.ID,
			"name":            item.Name,
			"mode":            item.Mode,
			"icon_type":       nullIfEmpty(item.IconType),
			"icon":            item.Icon,
			"icon_background": nullIfEmpty(item.IconBackground),
			"icon_url":        nil,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  data,
		"total": len(data),
	})
}

func (s *server) handleDatasetAPIEnable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetAPIStatus(w, r, true)
}

func (s *server) handleDatasetAPIDisable(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetAPIStatus(w, r, false)
}

func (s *server) handleDatasetAPIStatus(w http.ResponseWriter, r *http.Request, enabled bool) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	if _, err := s.store.SetDatasetAPIEnabled(dataset.ID, dataset.WorkspaceID, enabled, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentList(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	page, ok := s.store.ListDatasetDocuments(dataset.ID, dataset.WorkspaceID, state.DocumentListFilters{
		Page:    intQuery(r, "page", 1),
		Limit:   intQuery(r, "limit", 20),
		Keyword: strings.TrimSpace(r.URL.Query().Get("keyword")),
		Sort:    strings.TrimSpace(r.URL.Query().Get("sort")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
	})
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     s.datasetDocumentListPayload(dataset, page.Data),
		"has_more": page.HasMore,
		"limit":    page.Limit,
		"page":     page.Page,
		"total":    page.Total,
	})
}

func (s *server) handleDatasetDocumentCreate(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	batchID, documents, updatedDataset, err := s.store.CreateDatasetDocuments(
		dataset.ID,
		dataset.WorkspaceID,
		currentUser(r),
		datasetDocumentInputFromPayload(payload),
		time.Now(),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"dataset":   s.datasetResponse(updatedDataset),
		"batch":     batchID,
		"documents": s.datasetDocumentListPayload(updatedDataset, documents),
	})
}

func (s *server) handleDatasetDocumentDelete(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	if err := s.store.DeleteDatasetDocuments(dataset.ID, dataset.WorkspaceID, r.URL.Query()["document_id"], currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentBatchAction(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	action := strings.TrimSpace(chi.URLParam(r, "action"))
	if err := s.store.ApplyDatasetDocumentAction(dataset.ID, dataset.WorkspaceID, r.URL.Query()["document_id"], action, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentGenerateSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentDetail(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.datasetDocumentDetailResponse(dataset, document, strings.TrimSpace(r.URL.Query().Get("metadata"))))
}

func (s *server) handleDatasetDocumentIndexingStatus(w http.ResponseWriter, r *http.Request) {
	_, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.datasetDocumentIndexingStatusResponse(document))
}

func (s *server) handleDatasetDocumentRename(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}

	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if err := s.store.RenameDatasetDocument(dataset.ID, dataset.WorkspaceID, document.ID, payload.Name, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetDocumentPause(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetDocumentProcessing(w, r, true)
}

func (s *server) handleDatasetDocumentResume(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetDocumentProcessing(w, r, false)
}

func (s *server) handleDatasetDocumentProcessing(w http.ResponseWriter, r *http.Request, paused bool) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if err := s.store.SetDatasetDocumentProcessing(dataset.ID, dataset.WorkspaceID, document.ID, paused, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetBatchIndexingStatus(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	documents, ok := s.store.ListDatasetBatchDocuments(dataset.ID, dataset.WorkspaceID, chi.URLParam(r, "batchID"))
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	data := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		data = append(data, s.datasetDocumentIndexingStatusResponse(document))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handleDatasetAutoDisableLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"document_ids": []any{}})
}

func (s *server) handleDatasetQueries(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	page := intQuery(r, "page", 1)
	limit := intQuery(r, "limit", 20)
	items, total, ok := s.store.ListDatasetQueries(dataset.ID, dataset.WorkspaceID, page, limit)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, datasetQueryRecordPayload(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": page*limit < total,
		"limit":    limit,
		"page":     page,
		"total":    total,
	})
}

func (s *server) handleDatasetErrorDocs(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	documents, ok := s.store.ListDatasetErrorDocuments(dataset.ID, dataset.WorkspaceID)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}
	data := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		data = append(data, s.datasetDocumentIndexingStatusResponse(document))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  data,
		"total": len(data),
	})
}

func (s *server) handleDatasetHitTesting(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	var payload struct {
		Query         string   `json:"query"`
		AttachmentIDs []string `json:"attachment_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	query, err := validateDatasetHitTestingQuery(payload.Query, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	queryItems := s.datasetHitTestingQueryItems(dataset.WorkspaceID, query, payload.AttachmentIDs)
	if len(queryItems) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Query or attachment_ids is required.")
		return
	}

	record := state.DatasetQueryRecord{
		Source:  "hit_testing",
		Queries: queryItems,
	}
	_ = s.store.AddDatasetQueryRecord(dataset.ID, dataset.WorkspaceID, record, currentUser(r), time.Now())

	results := datasetHitResults(dataset, datasetHitTestingSearchText(queryItems))
	writeJSON(w, http.StatusOK, map[string]any{
		"query": map[string]any{
			"content":       query,
			"tsne_position": map[string]any{"x": 0.5, "y": 0.5},
		},
		"records": results,
	})
}

func (s *server) handleDatasetExternalHitTesting(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	var payload struct {
		Query                  string `json:"query"`
		ExternalRetrievalModel *struct {
			TopK                  *int     `json:"top_k"`
			ScoreThreshold        *float64 `json:"score_threshold"`
			ScoreThresholdEnabled *bool    `json:"score_threshold_enabled"`
		} `json:"external_retrieval_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	query, err := validateDatasetHitTestingQuery(payload.Query, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if strings.TrimSpace(dataset.Provider) != "external" {
		writeJSON(w, http.StatusOK, map[string]any{
			"query": map[string]any{
				"content": query,
			},
			"records": []map[string]any{},
		})
		return
	}

	model := resolveDatasetExternalHitTestingModel(dataset.ExternalRetrievalModel, payload.ExternalRetrievalModel)
	records, err := s.fetchDatasetExternalHitTestingRecords(r, dataset, query, model)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	record := state.DatasetQueryRecord{
		Source: "hit_testing",
		Queries: []state.DatasetQueryItem{
			{Content: query, ContentType: "text_query"},
		},
	}
	_ = s.store.AddDatasetQueryRecord(dataset.ID, dataset.WorkspaceID, record, currentUser(r), time.Now())

	writeJSON(w, http.StatusOK, map[string]any{
		"query": map[string]any{
			"content": query,
		},
		"records": records,
	})
}

func (s *server) handleDatasetRetry(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset_not_found", "Dataset not found.")
		return
	}

	var payload struct {
		DocumentIDs []string `json:"document_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	if err := s.store.RetryDatasetDocuments(dataset.ID, dataset.WorkspaceID, payload.DocumentIDs, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) currentUserDataset(r *http.Request) (state.Dataset, bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		return state.Dataset{}, false
	}
	datasetID := strings.TrimSpace(chi.URLParam(r, "datasetID"))
	if datasetID == "" {
		return state.Dataset{}, false
	}
	if dataset, _, err := s.store.RefreshDatasetProcessingProgress(datasetID, workspace.ID, time.Now()); err == nil {
		return dataset, true
	}
	return s.store.GetDataset(datasetID, workspace.ID)
}

func (s *server) currentUserDatasetDocument(r *http.Request) (state.Dataset, state.DatasetDocument, bool) {
	dataset, ok := s.currentUserDataset(r)
	if !ok {
		return state.Dataset{}, state.DatasetDocument{}, false
	}
	documentID := strings.TrimSpace(chi.URLParam(r, "documentID"))
	if documentID == "" {
		return state.Dataset{}, state.DatasetDocument{}, false
	}
	document, ok := s.store.GetDatasetDocument(dataset.ID, dataset.WorkspaceID, documentID)
	if !ok {
		return state.Dataset{}, state.DatasetDocument{}, false
	}
	return dataset, document, true
}

func (s *server) datasetResponse(dataset state.Dataset) map[string]any {
	totalDocuments := len(dataset.Documents)
	totalAvailableDocuments := 0
	wordCount := 0
	for _, document := range dataset.Documents {
		wordCount += document.WordCount
		if document.Enabled && !document.Archived {
			totalAvailableDocuments++
		}
	}

	isPublished := dataset.IsPublished
	if dataset.PipelineID != "" {
		if app, ok := s.store.GetApp(dataset.PipelineID, dataset.WorkspaceID); ok {
			isPublished = app.WorkflowPublished != nil
		}
	}

	return map[string]any{
		"id":                        dataset.ID,
		"name":                      dataset.Name,
		"indexing_status":           aggregateDatasetIndexingStatus(dataset),
		"icon_info":                 datasetIconInfoPayload(dataset.IconInfo),
		"description":               dataset.Description,
		"permission":                dataset.Permission,
		"data_source_type":          nullIfEmpty(dataset.DataSourceType),
		"indexing_technique":        nullIfEmpty(dataset.IndexingTechnique),
		"author_name":               dataset.AuthorName,
		"created_by":                dataset.CreatedBy,
		"created_at":                dataset.CreatedAt,
		"updated_by":                dataset.UpdatedBy,
		"updated_at":                dataset.UpdatedAt,
		"app_count":                 s.store.DatasetUseCount(dataset.WorkspaceID, dataset.ID),
		"doc_form":                  nullIfEmpty(dataset.DocForm),
		"document_count":            totalDocuments,
		"total_document_count":      totalDocuments,
		"total_available_documents": totalAvailableDocuments,
		"word_count":                wordCount,
		"provider":                  firstNonEmpty(dataset.Provider, "local"),
		"embedding_model":           dataset.EmbeddingModel,
		"embedding_model_provider":  dataset.EmbeddingModelProvider,
		"embedding_available":       dataset.EmbeddingAvailable,
		"retrieval_model_dict":      datasetRetrievalModelPayload(dataset.RetrievalModel),
		"retrieval_model":           datasetRetrievalModelPayload(dataset.RetrievalModel),
		"tags":                      []any{},
		"partial_member_list":       dataset.PartialMemberList,
		"external_knowledge_info": map[string]any{
			"external_knowledge_id":           nullIfEmpty(dataset.ExternalKnowledgeInfo.ExternalKnowledgeID),
			"external_knowledge_api_id":       nullIfEmpty(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID),
			"external_knowledge_api_name":     nullIfEmpty(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName),
			"external_knowledge_api_endpoint": nullIfEmpty(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint),
		},
		"external_retrieval_model": map[string]any{
			"top_k":                   dataset.ExternalRetrievalModel.TopK,
			"score_threshold":         dataset.ExternalRetrievalModel.ScoreThreshold,
			"score_threshold_enabled": dataset.ExternalRetrievalModel.ScoreThresholdEnabled,
		},
		"built_in_field_enabled": dataset.BuiltInFieldEnabled,
		"doc_metadata":           datasetMetadataDefinitionPayload(dataset.MetadataFields),
		"keyword_number":         len(dataset.MetadataFields),
		"pipeline_id":            nullIfEmpty(dataset.PipelineID),
		"is_published":           isPublished,
		"runtime_mode":           firstNonEmpty(dataset.RuntimeMode, "general"),
		"enable_api":             dataset.EnableAPI,
		"is_multimodal":          dataset.IsMultimodal,
		"summary_index_setting": map[string]any{
			"enable":              dataset.SummaryIndexSetting.Enable,
			"model_name":          nullIfEmpty(dataset.SummaryIndexSetting.ModelName),
			"model_provider_name": nullIfEmpty(dataset.SummaryIndexSetting.ModelProviderName),
			"summary_prompt":      nullIfEmpty(dataset.SummaryIndexSetting.SummaryPrompt),
		},
	}
}

func (s *server) datasetDocumentListPayload(dataset state.Dataset, documents []state.DatasetDocument) []map[string]any {
	data := make([]map[string]any, 0, len(documents))
	for _, document := range documents {
		data = append(data, map[string]any{
			"id":                      document.ID,
			"batch":                   document.Batch,
			"position":                document.Position,
			"dataset_id":              document.DatasetID,
			"data_source_type":        document.DataSourceType,
			"data_source_info":        cloneJSONObject(document.DataSourceInfo),
			"dataset_process_rule_id": document.DatasetProcessRuleID,
			"name":                    document.Name,
			"created_from":            document.CreatedFrom,
			"created_by":              document.CreatedBy,
			"created_at":              document.CreatedAt,
			"indexing_status":         document.IndexingStatus,
			"display_status":          document.DisplayStatus,
			"completed_segments":      document.CompletedSegments,
			"total_segments":          document.TotalSegments,
			"doc_form":                document.DocForm,
			"doc_language":            document.DocLanguage,
			"summary_index_status":    nullIfEmpty(document.SummaryIndexStatus),
			"enabled":                 document.Enabled,
			"word_count":              document.WordCount,
			"error":                   nullIfEmpty(document.Error),
			"archived":                document.Archived,
			"updated_at":              document.UpdatedAt,
			"hit_count":               document.HitCount,
			"data_source_detail_dict": cloneJSONObject(document.DataSourceDetailDict),
			"doc_metadata":            datasetDocumentMetadataListPayload(dataset, document, false),
		})
	}
	return data
}

func (s *server) datasetDocumentDetailResponse(dataset state.Dataset, document state.DatasetDocument, metadataMode string) map[string]any {
	response := s.datasetDocumentListPayload(dataset, []state.DatasetDocument{document})[0]
	response["created_api_request_id"] = document.CreatedAPIRequestID
	response["processing_started_at"] = document.ProcessingStartedAt
	response["parsing_completed_at"] = document.ParsingCompletedAt
	response["cleaning_completed_at"] = document.CleaningCompletedAt
	response["splitting_completed_at"] = document.SplittingCompletedAt
	response["tokens"] = document.Tokens
	response["indexing_latency"] = document.IndexingLatency
	response["completed_at"] = document.CompletedAt
	response["paused_by"] = nullIfEmpty(document.PausedBy)
	response["paused_at"] = nullableInt64(document.PausedAt)
	response["stopped_at"] = nullableInt64(document.StoppedAt)
	response["disabled_at"] = nullableInt64(document.DisabledAt)
	response["disabled_by"] = nullIfEmpty(document.DisabledBy)
	response["archived_reason"] = nullIfEmpty(document.ArchivedReason)
	response["archived_by"] = nullIfEmpty(document.ArchivedBy)
	response["archived_at"] = nullableInt64(document.ArchivedAt)
	response["doc_type"] = nullIfEmpty(document.DocType)
	response["segment_count"] = document.SegmentCount
	response["average_segment_length"] = datasetAverageSegmentLength(document)
	response["dataset_process_rule"] = datasetProcessRulePayload(document.DatasetProcessRule)
	response["document_process_rule"] = datasetProcessRulePayload(document.DocumentProcessRule)
	switch metadataMode {
	case "only":
		response["doc_metadata"] = datasetDocumentMetadataListPayload(dataset, document, true)
	default:
		response["doc_metadata"] = cloneJSONObject(stringMapToAny(document.DocMetadata))
	}
	return response
}

func (s *server) datasetDocumentIndexingStatusResponse(document state.DatasetDocument) map[string]any {
	return map[string]any{
		"id":                     document.ID,
		"indexing_status":        document.IndexingStatus,
		"processing_started_at":  document.ProcessingStartedAt,
		"parsing_completed_at":   document.ParsingCompletedAt,
		"cleaning_completed_at":  document.CleaningCompletedAt,
		"splitting_completed_at": document.SplittingCompletedAt,
		"completed_at":           nullableInt64(document.CompletedAt),
		"paused_at":              nullableInt64(document.PausedAt),
		"error":                  nullIfEmpty(document.Error),
		"stopped_at":             nullableInt64(document.StoppedAt),
		"completed_segments":     document.CompletedSegments,
		"total_segments":         document.TotalSegments,
	}
}

func datasetIconInfoPayload(icon state.DatasetIconInfo) map[string]any {
	return map[string]any{
		"icon":            icon.Icon,
		"icon_background": nullIfEmpty(icon.IconBackground),
		"icon_type":       icon.IconType,
		"icon_url":        nullIfEmpty(icon.IconURL),
	}
}

func datasetRetrievalModelPayload(model state.DatasetRetrievalModel) map[string]any {
	return map[string]any{
		"search_method":    firstNonEmpty(model.SearchMethod, "semantic_search"),
		"reranking_enable": model.RerankingEnable,
		"reranking_model": map[string]any{
			"reranking_provider_name": nullIfEmpty(model.RerankingModel.ProviderName),
			"reranking_model_name":    nullIfEmpty(model.RerankingModel.ModelName),
		},
		"top_k":                   model.TopK,
		"score_threshold_enabled": model.ScoreThresholdEnabled,
		"score_threshold":         model.ScoreThreshold,
		"reranking_mode":          nullIfEmpty(model.RerankingMode),
		"weights":                 cloneJSONObject(model.Weights),
	}
}

func datasetProcessRulePayload(rule state.DatasetProcessRule) map[string]any {
	if rule.Mode == "" {
		rule = state.DatasetProcessRule{}
	}
	if rule.Mode == "" {
		payload := datasetProcessRulePayload(state.DatasetProcessRule{
			Mode: "custom",
		})
		payload["rules"] = map[string]any{
			"pre_processing_rules": []map[string]any{
				{"id": "remove_extra_spaces", "enabled": false},
				{"id": "remove_urls_emails", "enabled": false},
			},
			"segmentation": map[string]any{
				"separator":     "\n\n",
				"max_tokens":    1000,
				"chunk_overlap": 50,
			},
			"parent_mode": "paragraph",
			"subchunk_segmentation": map[string]any{
				"separator":     "\n",
				"max_tokens":    300,
				"chunk_overlap": 30,
			},
		}
		payload["limits"] = state.DatasetProcessRuleLimits()
		return payload
	}

	preRules := make([]map[string]any, 0, len(rule.Rules.PreProcessingRules))
	for _, item := range rule.Rules.PreProcessingRules {
		preRules = append(preRules, map[string]any{
			"id":      item.ID,
			"enabled": item.Enabled,
		})
	}

	return map[string]any{
		"mode": rule.Mode,
		"rules": map[string]any{
			"pre_processing_rules": preRules,
			"segmentation": map[string]any{
				"separator":     rule.Rules.Segmentation.Separator,
				"max_tokens":    rule.Rules.Segmentation.MaxTokens,
				"chunk_overlap": rule.Rules.Segmentation.ChunkOverlap,
			},
			"parent_mode": rule.Rules.ParentMode,
			"subchunk_segmentation": map[string]any{
				"separator":     rule.Rules.SubchunkSegmentation.Separator,
				"max_tokens":    rule.Rules.SubchunkSegmentation.MaxTokens,
				"chunk_overlap": rule.Rules.SubchunkSegmentation.ChunkOverlap,
			},
		},
		"limits": state.DatasetProcessRuleLimits(),
		"summary_index_setting": map[string]any{
			"enable":              rule.SummaryIndexSetting.Enable,
			"model_name":          nullIfEmpty(rule.SummaryIndexSetting.ModelName),
			"model_provider_name": nullIfEmpty(rule.SummaryIndexSetting.ModelProviderName),
			"summary_prompt":      nullIfEmpty(rule.SummaryIndexSetting.SummaryPrompt),
		},
	}
}

func datasetDocumentInputFromPayload(payload map[string]any) state.CreateDatasetDocumentInput {
	var retrievalModel state.DatasetRetrievalModel
	decodeInto(payload["retrieval_model"], &retrievalModel)

	var processRule state.DatasetProcessRule
	decodeInto(payload["process_rule"], &processRule)

	var summaryIndexSetting state.DatasetSummaryIndexSetting
	decodeInto(payload["summary_index_setting"], &summaryIndexSetting)

	dataSource := mapFromAny(payload["data_source"])
	return state.CreateDatasetDocumentInput{
		DataSourceType:         firstNonEmpty(stringFromAny(dataSource["type"]), stringFromAny(payload["data_source_type"])),
		DataSource:             dataSource,
		DocForm:                stringFromAny(payload["doc_form"]),
		DocLanguage:            stringFromAny(payload["doc_language"]),
		IndexingTechnique:      stringFromAny(payload["indexing_technique"]),
		RetrievalModel:         retrievalModel,
		EmbeddingModel:         stringFromAny(payload["embedding_model"]),
		EmbeddingModelProvider: stringFromAny(payload["embedding_model_provider"]),
		ProcessRule:            processRule,
		SummaryIndexSetting:    summaryIndexSetting,
		CreatedFrom:            "web",
	}
}

func decodeInto(value any, dest any) {
	if value == nil || dest == nil {
		return
	}
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, dest)
}

func datasetNameFromPayload(payload map[string]any) string {
	if name := strings.TrimSpace(stringFromAny(payload["name"])); name != "" {
		return name
	}
	dataSource := mapFromAny(payload["data_source"])
	infoList := mapFromAny(dataSource["info_list"])
	if notionItems := anySlice(infoList["notion_info_list"]); len(notionItems) > 0 {
		first := mapFromAny(notionItems[0])
		pages := anySlice(first["pages"])
		if len(pages) > 0 {
			page := mapFromAny(pages[0])
			if pageName := strings.TrimSpace(stringFromAny(page["page_name"])); pageName != "" {
				return pageName
			}
		}
	}
	if websiteInfo := mapFromAny(infoList["website_info_list"]); len(websiteInfo) > 0 {
		urls := anySlice(websiteInfo["urls"])
		if len(urls) > 0 {
			if rawURL, ok := urls[0].(string); ok && strings.TrimSpace(rawURL) != "" {
				name := strings.TrimSpace(rawURL)
				name = strings.TrimPrefix(name, "https://")
				name = strings.TrimPrefix(name, "http://")
				if slash := strings.IndexRune(name, '/'); slash >= 0 {
					name = name[:slash]
				}
				return name
			}
		}
	}
	if fileInfo := mapFromAny(infoList["file_info_list"]); len(fileInfo) > 0 {
		fileIDs := anySlice(fileInfo["file_ids"])
		if len(fileIDs) > 0 {
			if fileID, ok := fileIDs[0].(string); ok && strings.TrimSpace(fileID) != "" {
				return strings.TrimSpace(fileID)
			}
		}
	}
	return "Knowledge Base"
}

func estimateDatasetNodeCount(dataSourceType string, infoList map[string]any) int {
	switch strings.TrimSpace(dataSourceType) {
	case "notion_import":
		count := 0
		for _, item := range anySlice(infoList["notion_info_list"]) {
			count += len(anySlice(mapFromAny(item)["pages"]))
		}
		return count
	case "website_crawl":
		return len(anySlice(mapFromAny(infoList["website_info_list"])["urls"]))
	default:
		return len(anySlice(mapFromAny(infoList["file_info_list"])["file_ids"]))
	}
}

func aggregateDatasetIndexingStatus(dataset state.Dataset) string {
	if len(dataset.Documents) == 0 {
		return "completed"
	}
	hasPaused := false
	hasError := false
	hasRunning := false
	for _, document := range dataset.Documents {
		switch document.IndexingStatus {
		case "error":
			hasError = true
		case "paused":
			hasPaused = true
		case "completed":
		default:
			hasRunning = true
		}
	}
	switch {
	case hasError:
		return "error"
	case hasPaused:
		return "paused"
	case hasRunning:
		return "indexing"
	default:
		return "completed"
	}
}

func datasetQueryRecordPayload(record state.DatasetQueryRecord) map[string]any {
	queries := make([]map[string]any, 0, len(record.Queries))
	for _, item := range record.Queries {
		query := map[string]any{
			"content":      item.Content,
			"content_type": item.ContentType,
			"file_info":    nil,
		}
		if item.FileInfo != nil {
			query["file_info"] = map[string]any{
				"id":         item.FileInfo.ID,
				"name":       item.FileInfo.Name,
				"size":       item.FileInfo.Size,
				"extension":  item.FileInfo.Extension,
				"mime_type":  item.FileInfo.MimeType,
				"source_url": item.FileInfo.SourceURL,
			}
		}
		queries = append(queries, query)
	}
	return map[string]any{
		"id":              record.ID,
		"source":          record.Source,
		"source_app_id":   nullIfEmpty(record.SourceAppID),
		"created_by_role": firstNonEmpty(record.CreatedByRole, "account"),
		"created_by":      record.CreatedBy,
		"created_at":      record.CreatedAt,
		"queries":         queries,
	}
}

func (s *server) datasetHitTestingQueryItems(workspaceID, textQuery string, attachmentIDs []string) []state.DatasetQueryItem {
	items := make([]state.DatasetQueryItem, 0, 1+len(attachmentIDs))
	if textQuery = strings.TrimSpace(textQuery); textQuery != "" {
		items = append(items, state.DatasetQueryItem{
			Content:     textQuery,
			ContentType: "text_query",
		})
	}

	seen := map[string]struct{}{}
	for _, attachmentID := range attachmentIDs {
		attachmentID = strings.TrimSpace(attachmentID)
		if attachmentID == "" {
			continue
		}
		if _, ok := seen[attachmentID]; ok {
			continue
		}
		seen[attachmentID] = struct{}{}

		uploadedFile, ok := s.store.GetUploadedFile(workspaceID, attachmentID)
		if !ok {
			uploadedFile, ok = s.store.FindUploadedFile(attachmentID)
		}
		if !ok {
			continue
		}

		attachment := &state.DatasetAttachment{
			ID:        uploadedFile.ID,
			Name:      uploadedFile.Name,
			Size:      uploadedFile.Size,
			Extension: uploadedFile.Extension,
			MimeType:  uploadedFile.MimeType,
			SourceURL: uploadedFile.SourceURL,
		}
		items = append(items, state.DatasetQueryItem{
			Content:     attachment.SourceURL,
			ContentType: "image_query",
			FileInfo:    attachment,
		})
	}
	return items
}

func datasetHitTestingSearchText(items []state.DatasetQueryItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.ContentType == "text_query" && strings.TrimSpace(item.Content) != "" {
			return strings.TrimSpace(item.Content)
		}
		if item.FileInfo != nil && strings.TrimSpace(item.FileInfo.Name) != "" {
			parts = append(parts, strings.TrimSpace(item.FileInfo.Name))
		}
	}
	return strings.Join(parts, " ")
}

func validateDatasetHitTestingQuery(raw string, allowEmpty bool) (string, error) {
	query := strings.TrimSpace(raw)
	if query == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("Query is required.")
	}
	if utf8.RuneCountInString(query) > 250 {
		return "", fmt.Errorf("Query cannot exceed 250 characters.")
	}
	return query, nil
}

func resolveDatasetExternalHitTestingModel(base state.DatasetExternalRetrievalModel, override *struct {
	TopK                  *int     `json:"top_k"`
	ScoreThreshold        *float64 `json:"score_threshold"`
	ScoreThresholdEnabled *bool    `json:"score_threshold_enabled"`
}) state.DatasetExternalRetrievalModel {
	model := base
	if model.TopK <= 0 {
		model.TopK = 4
	}
	if model.ScoreThreshold == 0 {
		model.ScoreThreshold = 0.5
	}
	if override == nil {
		return model
	}
	if override.TopK != nil && *override.TopK > 0 {
		model.TopK = *override.TopK
	}
	if override.ScoreThreshold != nil {
		model.ScoreThreshold = *override.ScoreThreshold
	}
	if override.ScoreThresholdEnabled != nil {
		model.ScoreThresholdEnabled = *override.ScoreThresholdEnabled
	}
	return model
}

func (s *server) fetchDatasetExternalHitTestingRecords(r *http.Request, dataset state.Dataset, query string, model state.DatasetExternalRetrievalModel) ([]map[string]any, error) {
	apiID := strings.TrimSpace(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID)
	if apiID == "" {
		return nil, fmt.Errorf("External knowledge API is not configured.")
	}

	api, ok := s.store.GetExternalKnowledgeAPI(dataset.WorkspaceID, apiID)
	if !ok {
		return nil, fmt.Errorf("External knowledge API not found.")
	}

	endpoint := strings.TrimRight(strings.TrimSpace(api.Endpoint), "/")
	if endpoint == "" {
		return nil, fmt.Errorf("External knowledge API endpoint is required.")
	}

	knowledgeID := strings.TrimSpace(dataset.ExternalKnowledgeInfo.ExternalKnowledgeID)
	if knowledgeID == "" {
		return nil, fmt.Errorf("External knowledge ID is required.")
	}

	scoreThreshold := 0.0
	if model.ScoreThresholdEnabled {
		scoreThreshold = model.ScoreThreshold
	}

	requestPayload := map[string]any{
		"retrieval_setting": map[string]any{
			"top_k":           model.TopK,
			"score_threshold": scoreThreshold,
		},
		"query":              escapeDatasetHitTestingSearchQuery(query),
		"knowledge_id":       knowledgeID,
		"metadata_condition": nil,
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("Failed to build external retrieval request.")
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint+"/retrieval", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Failed to create external retrieval request.")
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey := strings.TrimSpace(api.APIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve from external knowledge API: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read external knowledge API response.")
	}
	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(responseBody))
		if message == "" {
			message = fmt.Sprintf("External knowledge API returned status %d.", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", message)
	}

	var response struct {
		Records []struct {
			Content  string         `json:"content"`
			Title    string         `json:"title"`
			Score    *float64       `json:"score"`
			Metadata map[string]any `json:"metadata"`
		} `json:"records"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("Invalid external knowledge API response.")
	}

	records := make([]map[string]any, 0, len(response.Records))
	for _, item := range response.Records {
		if model.ScoreThresholdEnabled && item.Score != nil && *item.Score < model.ScoreThreshold {
			continue
		}

		record := map[string]any{
			"content":  item.Content,
			"title":    externalHitTestingRecordTitle(dataset, item.Title, item.Metadata),
			"score":    nil,
			"metadata": nil,
		}
		if item.Score != nil {
			record["score"] = *item.Score
		}
		if len(item.Metadata) > 0 {
			record["metadata"] = item.Metadata
		}
		records = append(records, record)
		if len(records) >= model.TopK {
			break
		}
	}

	return records, nil
}

func escapeDatasetHitTestingSearchQuery(query string) string {
	return strings.ReplaceAll(query, `"`, `\"`)
}

func externalHitTestingRecordTitle(dataset state.Dataset, title string, metadata map[string]any) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	if source := strings.TrimSpace(stringFromAny(metadata["x-amz-bedrock-kb-source-uri"])); source != "" {
		return source
	}
	if source := strings.TrimSpace(stringFromAny(metadata["source"])); source != "" {
		return source
	}
	return firstNonEmpty(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName, dataset.Name, "External Knowledge")
}

func datasetHitResults(dataset state.Dataset, query string) []map[string]any {
	type scored struct {
		document state.DatasetDocument
		score    float64
	}
	scoredDocuments := make([]scored, 0, len(dataset.Documents))
	for _, document := range dataset.Documents {
		score := datasetSearchScore(query, document.Name+" "+document.Content)
		if score <= 0 {
			continue
		}
		scoredDocuments = append(scoredDocuments, scored{document: document, score: score})
	}
	if len(scoredDocuments) == 0 && len(dataset.Documents) > 0 {
		scoredDocuments = append(scoredDocuments, scored{document: dataset.Documents[0], score: 0.3})
	}

	results := make([]map[string]any, 0, len(scoredDocuments))
	for i, item := range scoredDocuments {
		document := item.document
		results = append(results, map[string]any{
			"segment": map[string]any{
				"id":              "seg_" + document.ID,
				"document":        map[string]any{"id": document.ID, "data_source_type": document.DataSourceType, "name": document.Name, "doc_type": firstNonEmpty(document.DocType, "others")},
				"content":         document.Content,
				"sign_content":    document.SignContent,
				"position":        i + 1,
				"word_count":      document.WordCount,
				"tokens":          document.Tokens,
				"keywords":        document.Keywords,
				"hit_count":       document.HitCount,
				"index_node_hash": document.ID,
				"answer":          "",
			},
			"content": map[string]any{
				"id":              "seg_" + document.ID,
				"document":        map[string]any{"id": document.ID, "data_source_type": document.DataSourceType, "name": document.Name, "doc_type": firstNonEmpty(document.DocType, "others")},
				"content":         document.Content,
				"sign_content":    document.SignContent,
				"position":        i + 1,
				"word_count":      document.WordCount,
				"tokens":          document.Tokens,
				"keywords":        document.Keywords,
				"hit_count":       document.HitCount,
				"index_node_hash": document.ID,
				"answer":          "",
			},
			"score": item.score,
			"tsne_position": map[string]any{
				"x": float64(i+1) / float64(len(scoredDocuments)+1),
				"y": 1 - (float64(i+1) / float64(len(scoredDocuments)+1)),
			},
			"child_chunks": datasetChildChunkPayload(document.ChildChunks),
			"files":        datasetAttachmentsPayload(document.Attachments),
			"summary":      nullIfEmpty(document.Summary),
		})
		if len(results) >= 5 {
			break
		}
	}
	return results
}

func datasetAttachmentsPayload(attachments []state.DatasetAttachment) []map[string]any {
	data := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		data = append(data, map[string]any{
			"id":         item.ID,
			"name":       item.Name,
			"size":       item.Size,
			"extension":  item.Extension,
			"mime_type":  item.MimeType,
			"source_url": item.SourceURL,
		})
	}
	return data
}

func datasetChildChunkPayload(chunks []state.DatasetChildChunk) any {
	if len(chunks) == 0 {
		return nil
	}
	data := make([]map[string]any, 0, len(chunks))
	for _, item := range chunks {
		data = append(data, map[string]any{
			"id":       item.ID,
			"content":  item.Content,
			"position": item.Position,
			"score":    item.Score,
		})
	}
	return data
}

func datasetSearchScore(query, content string) float64 {
	queryTokens := tokenizeDatasetText(query)
	if len(queryTokens) == 0 {
		return 0
	}
	contentSet := map[string]struct{}{}
	for _, token := range tokenizeDatasetText(content) {
		contentSet[token] = struct{}{}
	}
	matches := 0
	for _, token := range queryTokens {
		if _, ok := contentSet[token]; ok {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return float64(matches) / float64(len(queryTokens))
}

func tokenizeDatasetText(input string) []string {
	input = strings.ToLower(input)
	return strings.FieldsFunc(input, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

func stringMapToAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func anySlice(value any) []any {
	if value == nil {
		return []any{}
	}
	if items, ok := value.([]any); ok {
		out := make([]any, len(items))
		copy(out, items)
		return out
	}
	return []any{}
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
