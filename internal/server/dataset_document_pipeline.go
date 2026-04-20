package server

import (
	"net/http"
	"time"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handleDatasetDocumentPipelineExecutionLog(w http.ResponseWriter, r *http.Request) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}

	log, ok := s.store.GetDatasetDocumentPipelineExecutionLog(dataset.ID, dataset.WorkspaceID, document.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}

	writeJSON(w, http.StatusOK, datasetDocumentPipelineExecutionLogResponse(log))
}

func (s *server) handleDatasetDocumentNotionSync(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetDocumentSync(w, r, "notion_import")
}

func (s *server) handleDatasetDocumentWebsiteSync(w http.ResponseWriter, r *http.Request) {
	s.handleDatasetDocumentSync(w, r, "website_crawl")
}

func (s *server) handleDatasetDocumentSync(w http.ResponseWriter, r *http.Request, expectedType string) {
	dataset, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}
	if document.DataSourceType != expectedType {
		writeError(w, http.StatusBadRequest, "invalid_request", "Document source does not support this sync operation.")
		return
	}

	if _, err := s.store.SyncDatasetDocumentSource(dataset.ID, dataset.WorkspaceID, document.ID, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func datasetDocumentPipelineExecutionLogResponse(log state.DatasetPipelineExecutionLog) map[string]any {
	return map[string]any{
		"datasource_type":    log.DatasourceType,
		"datasource_info":    cloneJSONObject(log.DatasourceInfo),
		"input_data":         cloneJSONObject(log.InputData),
		"datasource_node_id": log.DatasourceNodeID,
	}
}
