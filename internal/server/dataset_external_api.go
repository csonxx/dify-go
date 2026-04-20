package server

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

const externalKnowledgeAPISecretPlaceholder = "[__HIDDEN__]"

func (s *server) handleDatasetExternalCreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	var payload struct {
		Name                   string                              `json:"name"`
		Description            string                              `json:"description"`
		ExternalKnowledgeID    string                              `json:"external_knowledge_id"`
		ExternalKnowledgeAPIID string                              `json:"external_knowledge_api_id"`
		ExternalRetrievalModel state.DatasetExternalRetrievalModel `json:"external_retrieval_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	dataset, err := s.store.CreateExternalDataset(workspace.ID, currentUser(r), state.CreateExternalDatasetInput{
		Name:                   payload.Name,
		Description:            payload.Description,
		ExternalKnowledgeID:    payload.ExternalKnowledgeID,
		ExternalKnowledgeAPIID: payload.ExternalKnowledgeAPIID,
		ExternalRetrievalModel: payload.ExternalRetrievalModel,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.datasetResponse(dataset))
}

func (s *server) handleDatasetExternalKnowledgeAPICreate(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	input, err := decodeExternalKnowledgeAPIInput(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	api, err := s.store.CreateExternalKnowledgeAPI(workspace.ID, currentUser(r), input, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.externalKnowledgeAPIResponse(api))
}

func (s *server) handleDatasetExternalKnowledgeAPIDetail(w http.ResponseWriter, r *http.Request) {
	api, ok := s.currentUserExternalKnowledgeAPI(r)
	if !ok {
		writeError(w, http.StatusNotFound, "external_knowledge_api_not_found", "External knowledge API not found.")
		return
	}
	writeJSON(w, http.StatusOK, s.externalKnowledgeAPIResponse(api))
}

func (s *server) handleDatasetExternalKnowledgeAPIUpdate(w http.ResponseWriter, r *http.Request) {
	api, ok := s.currentUserExternalKnowledgeAPI(r)
	if !ok {
		writeError(w, http.StatusNotFound, "external_knowledge_api_not_found", "External knowledge API not found.")
		return
	}

	input, err := decodeExternalKnowledgeAPIInput(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	updated, err := s.store.UpdateExternalKnowledgeAPI(api.WorkspaceID, api.ID, input, currentUser(r), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.externalKnowledgeAPIResponse(updated))
}

func (s *server) handleDatasetExternalKnowledgeAPIDelete(w http.ResponseWriter, r *http.Request) {
	api, ok := s.currentUserExternalKnowledgeAPI(r)
	if !ok {
		writeError(w, http.StatusNotFound, "external_knowledge_api_not_found", "External knowledge API not found.")
		return
	}

	if err := s.store.DeleteExternalKnowledgeAPI(api.WorkspaceID, api.ID, currentUser(r), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleDatasetExternalKnowledgeAPIUseCheck(w http.ResponseWriter, r *http.Request) {
	api, ok := s.currentUserExternalKnowledgeAPI(r)
	if !ok {
		writeError(w, http.StatusNotFound, "external_knowledge_api_not_found", "External knowledge API not found.")
		return
	}

	count := s.store.ExternalKnowledgeAPIUseCount(api.WorkspaceID, api.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"is_using": count > 0,
		"count":    count,
	})
}

func (s *server) handleDatasetDocumentDownload(w http.ResponseWriter, r *http.Request) {
	_, document, ok := s.currentUserDatasetDocument(r)
	if !ok {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}

	mimeType := datasetDocumentDownloadMIMEType(document)
	content := datasetDocumentDownloadContent(document)
	writeJSON(w, http.StatusOK, map[string]any{
		"url": "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString([]byte(content)),
	})
}

func (s *server) handleDatasetDocumentDownloadZip(w http.ResponseWriter, r *http.Request) {
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
	if len(payload.DocumentIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Document IDs are required.")
		return
	}

	targets := make(map[string]struct{}, len(payload.DocumentIDs))
	for _, documentID := range payload.DocumentIDs {
		if trimmed := strings.TrimSpace(documentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}
	if len(targets) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Document IDs are required.")
		return
	}

	selected := make([]state.DatasetDocument, 0, len(targets))
	for _, document := range dataset.Documents {
		if _, exists := targets[document.ID]; exists {
			selected = append(selected, document)
		}
	}
	if len(selected) == 0 {
		writeError(w, http.StatusNotFound, "document_not_found", "Document not found.")
		return
	}

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	usedNames := make(map[string]int)
	for _, document := range selected {
		entryName := datasetDocumentDownloadFileName(document, usedNames)
		writer, err := archive.Create(entryName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		if _, err := writer.Write([]byte(datasetDocumentDownloadContent(document))); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}
	if err := archive.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="documents.zip"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buffer.Bytes())
}

func (s *server) currentUserExternalKnowledgeAPI(r *http.Request) (state.ExternalKnowledgeAPI, bool) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		return state.ExternalKnowledgeAPI{}, false
	}
	apiID := strings.TrimSpace(chi.URLParam(r, "apiID"))
	if apiID == "" {
		return state.ExternalKnowledgeAPI{}, false
	}
	return s.store.GetExternalKnowledgeAPI(workspace.ID, apiID)
}

func (s *server) externalKnowledgeAPIResponse(api state.ExternalKnowledgeAPI) map[string]any {
	bindings := s.store.ExternalKnowledgeAPIDatasetBindings(api.WorkspaceID, api.ID)
	datasetBindings := make([]map[string]any, 0, len(bindings))
	for _, binding := range bindings {
		datasetBindings = append(datasetBindings, map[string]any{
			"id":   binding.ID,
			"name": binding.Name,
		})
	}

	return map[string]any{
		"id":          api.ID,
		"tenant_id":   api.WorkspaceID,
		"name":        api.Name,
		"description": api.Description,
		"settings": map[string]any{
			"endpoint": api.Endpoint,
			"api_key":  externalKnowledgeAPISecretPlaceholder,
		},
		"dataset_bindings": datasetBindings,
		"created_by":       api.CreatedBy,
		"created_at":       api.CreatedAt,
	}
}

func decodeExternalKnowledgeAPIInput(r *http.Request) (state.ExternalKnowledgeAPIInput, error) {
	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Settings    struct {
			Endpoint string `json:"endpoint"`
			APIKey   string `json:"api_key"`
		} `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return state.ExternalKnowledgeAPIInput{}, err
	}
	return state.ExternalKnowledgeAPIInput{
		Name:        payload.Name,
		Description: payload.Description,
		Endpoint:    payload.Settings.Endpoint,
		APIKey:      payload.Settings.APIKey,
	}, nil
}

func datasetDocumentDownloadContent(document state.DatasetDocument) string {
	content := strings.TrimSpace(document.Content)
	if content != "" {
		return content
	}
	if len(document.Segments) > 0 {
		segments := make([]string, 0, len(document.Segments))
		for _, segment := range document.Segments {
			if trimmed := strings.TrimSpace(segment.Content); trimmed != "" {
				segments = append(segments, trimmed)
			}
		}
		if len(segments) > 0 {
			return strings.Join(segments, "\n\n")
		}
	}
	if summary := strings.TrimSpace(document.Summary); summary != "" {
		return summary
	}
	return "Compatible download generated by dify-go for " + firstNonEmpty(document.Name, document.ID, "document") + "."
}

func datasetDocumentDownloadFileName(document state.DatasetDocument, usedNames map[string]int) string {
	name := sanitizeDownloadFileName(firstNonEmpty(strings.TrimSpace(document.Name), document.ID, "document"))
	extension := strings.TrimPrefix(filepath.Ext(name), ".")
	if extension == "" {
		extension = datasetDocumentDownloadExtension(document)
		name += "." + extension
	}

	key := strings.ToLower(name)
	count := usedNames[key]
	usedNames[key] = count + 1
	if count == 0 {
		return name
	}

	suffix := "-" + strconv.Itoa(count+1)
	baseName := strings.TrimSuffix(name, "."+extension)
	return baseName + suffix + "." + extension
}

func datasetDocumentDownloadExtension(document state.DatasetDocument) string {
	if nameExtension := strings.TrimPrefix(filepath.Ext(strings.TrimSpace(document.Name)), "."); nameExtension != "" {
		return strings.ToLower(nameExtension)
	}
	for _, attachment := range document.Attachments {
		if extension := strings.TrimSpace(strings.TrimPrefix(attachment.Extension, ".")); extension != "" {
			return strings.ToLower(extension)
		}
	}
	uploadFile := mapFromAny(document.DataSourceInfo["upload_file"])
	if extension := strings.TrimSpace(strings.TrimPrefix(stringFromAny(uploadFile["extension"]), ".")); extension != "" {
		return strings.ToLower(extension)
	}
	return "txt"
}

func datasetDocumentDownloadMIMEType(document state.DatasetDocument) string {
	for _, attachment := range document.Attachments {
		if mimeType := strings.TrimSpace(attachment.MimeType); mimeType != "" {
			return mimeType
		}
	}

	uploadFile := mapFromAny(document.DataSourceInfo["upload_file"])
	if mimeType := strings.TrimSpace(stringFromAny(uploadFile["mime_type"])); mimeType != "" {
		return mimeType
	}

	switch datasetDocumentDownloadExtension(document) {
	case "pdf":
		return "application/pdf"
	case "md":
		return "text/markdown"
	case "html", "htm":
		return "text/html"
	case "json":
		return "application/json"
	default:
		return "text/plain;charset=utf-8"
	}
}

func sanitizeDownloadFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "document"
	}
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "document"
	}
	return name
}
