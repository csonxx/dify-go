package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

const (
	uploadBatchCountLimit                = 10
	uploadImageFileSizeLimitMB           = 10
	uploadImageFileBatchLimit            = 10
	uploadSingleChunkAttachmentLimit     = 10
	uploadAttachmentImageFileSizeLimitMB = 2
	uploadDocumentFileSizeLimitMB        = 15
	uploadAudioFileSizeLimitMB           = 50
	uploadVideoFileSizeLimitMB           = 100
	uploadWorkflowFileUploadLimit        = 10
	uploadFileUploadLimit                = 5
	uploadedFilePreviewWordLimit         = 3000
)

var (
	documentUploadExtensions = map[string]struct{}{
		"txt": {}, "markdown": {}, "md": {}, "mdx": {}, "pdf": {}, "html": {}, "htm": {},
		"xlsx": {}, "xls": {}, "docx": {}, "csv": {}, "vtt": {}, "properties": {}, "json": {},
		"yaml": {}, "yml": {}, "xml": {},
	}
	imageUploadExtensions = map[string]struct{}{
		"jpg": {}, "jpeg": {}, "png": {}, "gif": {}, "webp": {}, "bmp": {}, "svg": {},
	}
	audioUploadExtensions = map[string]struct{}{
		"mp3": {}, "wav": {}, "m4a": {}, "aac": {}, "ogg": {},
	}
	videoUploadExtensions = map[string]struct{}{
		"mp4": {}, "mov": {}, "avi": {}, "webm": {}, "mkv": {},
	}
	textPreviewExtensions = map[string]struct{}{
		"txt": {}, "markdown": {}, "md": {}, "mdx": {}, "html": {}, "htm": {}, "csv": {},
		"vtt": {}, "properties": {}, "json": {}, "yaml": {}, "yml": {}, "xml": {},
	}
)

type uploadAPIError struct {
	status  int
	code    string
	message string
}

func (e *uploadAPIError) Error() string {
	return e.message
}

func (s *server) fileRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/{fileID}/file-preview", s.handleUploadedFileBinary)
	r.Get("/{fileID}/image-preview", s.handleUploadedFileBinary)
	r.NotFound(s.compatFallback)
	return r
}

func (s *server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	s.handleMultipartFileUpload(w, r)
}

func (s *server) handlePublicFileUpload(w http.ResponseWriter, r *http.Request) {
	s.handleMultipartFileUpload(w, r)
}

func (s *server) handleMultipartFileUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid multipart payload.")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	allFiles := 0
	for _, files := range r.MultipartForm.File {
		allFiles += len(files)
	}
	if allFiles == 0 {
		writeError(w, http.StatusBadRequest, "no_file_uploaded", "File is required.")
		return
	}
	if allFiles > 1 {
		writeError(w, http.StatusBadRequest, "too_many_files", "Only one file can be uploaded at a time.")
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "no_file_uploaded", "File is required.")
		return
	}

	header := files[0]
	filename := sanitizeUploadFilename(header.Filename)
	if filename == "" {
		writeError(w, http.StatusBadRequest, "filename_not_exists", "Filename is required.")
		return
	}

	file, err := header.Open()
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Unable to read uploaded file.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Unable to read uploaded file.")
		return
	}

	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if contentType == "" && len(content) > 0 {
		contentType = http.DetectContentType(content)
	}
	workspace, createdBy, ok := s.uploadRequestWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	recorded, err := s.persistUploadedFile(workspace.ID, createdBy, filename, contentType, content)
	if err != nil {
		writeUploadAPIError(w, err, http.StatusInternalServerError, "upload_failed", "Unable to store uploaded file.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          recorded.ID,
		"name":        recorded.Name,
		"size":        recorded.Size,
		"extension":   recorded.Extension,
		"mime_type":   recorded.MimeType,
		"created_by":  recorded.CreatedBy,
		"created_at":  recorded.CreatedAt,
		"preview_url": recorded.PreviewURL,
		"source_url":  recorded.SourceURL,
	})
}

func (s *server) handleRemoteFileUpload(w http.ResponseWriter, r *http.Request) {
	s.handleRemoteFileUploadRequest(w, r)
}

func (s *server) handlePublicRemoteFileUpload(w http.ResponseWriter, r *http.Request) {
	s.handleRemoteFileUploadRequest(w, r)
}

func (s *server) handleRemoteFileUploadRequest(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	rawURL := strings.TrimSpace(payload.URL)
	if rawURL == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Valid remote file URL is required.")
		return
	}

	workspace, createdBy, ok := s.uploadRequestWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	filename, contentType, content, err := s.fetchRemoteUploadContent(r, rawURL)
	if err != nil {
		writeUploadAPIError(w, err, http.StatusBadGateway, "remote_file_fetch_failed", "Unable to fetch remote file.")
		return
	}

	recorded, err := s.persistUploadedFile(workspace.ID, createdBy, filename, contentType, content)
	if err != nil {
		writeUploadAPIError(w, err, http.StatusInternalServerError, "upload_failed", "Unable to store remote file.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":        recorded.ID,
		"name":      recorded.Name,
		"size":      recorded.Size,
		"mime_type": recorded.MimeType,
		"url":       recorded.SourceURL,
	})
}

func (s *server) handleFilePreview(w http.ResponseWriter, r *http.Request) {
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	uploadedFile, ok := s.store.GetUploadedFile(workspace.ID, chi.URLParam(r, "fileID"))
	if !ok {
		writeError(w, http.StatusNotFound, "file_not_found", "File not found.")
		return
	}

	content, err := os.ReadFile(s.uploadedFilePath(uploadedFile.StorageKey))
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "file_not_found", "File not found.")
			return
		}
		writeError(w, http.StatusInternalServerError, "file_preview_failed", "Unable to read uploaded file.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"content": uploadedFilePreviewText(uploadedFile, content),
	})
}

func (s *server) handleUploadedFileBinary(w http.ResponseWriter, r *http.Request) {
	uploadedFile, ok := s.store.FindUploadedFile(chi.URLParam(r, "fileID"))
	if !ok {
		http.NotFound(w, r)
		return
	}

	content, err := os.ReadFile(s.uploadedFilePath(uploadedFile.StorageKey))
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		writeError(w, http.StatusInternalServerError, "file_preview_failed", "Unable to read uploaded file.")
		return
	}

	contentType := firstNonEmpty(strings.TrimSpace(uploadedFile.MimeType), http.DetectContentType(content))
	disposition := "inline"
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("as_attachment")), "true") {
		disposition = "attachment"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, uploadedFile.Name))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *server) uploadedFilePath(storageKey string) string {
	return filepath.Join(s.cfg.UploadDir, filepath.FromSlash(storageKey))
}

func allowedUploadExtensions() map[string]struct{} {
	out := map[string]struct{}{}
	for ext := range documentUploadExtensions {
		out[ext] = struct{}{}
	}
	for ext := range imageUploadExtensions {
		out[ext] = struct{}{}
	}
	for ext := range audioUploadExtensions {
		out[ext] = struct{}{}
	}
	for ext := range videoUploadExtensions {
		out[ext] = struct{}{}
	}
	return out
}

func uploadFileSizeLimitBytes(extension string) int64 {
	normalized := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(extension), "."))
	switch {
	case hasExtension(imageUploadExtensions, normalized):
		return int64(uploadImageFileSizeLimitMB) * 1024 * 1024
	case hasExtension(audioUploadExtensions, normalized):
		return int64(uploadAudioFileSizeLimitMB) * 1024 * 1024
	case hasExtension(videoUploadExtensions, normalized):
		return int64(uploadVideoFileSizeLimitMB) * 1024 * 1024
	default:
		return int64(uploadDocumentFileSizeLimitMB) * 1024 * 1024
	}
}

func hasExtension(set map[string]struct{}, extension string) bool {
	_, ok := set[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(extension), "."))]
	return ok
}

func detectUploadExtension(filename, contentType string) string {
	contentType = canonicalUploadContentType(contentType)
	trimmed := strings.TrimSpace(filename)
	if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(trimmed), ".")); ext != "" {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "text/markdown", "text/x-markdown":
		return "md"
	case "text/csv":
		return "csv"
	case "application/pdf":
		return "pdf"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "application/json":
		return "json"
	default:
		return "txt"
	}
}

func canonicalUploadContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		return strings.ToLower(strings.TrimSpace(mediaType))
	}
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	return strings.ToLower(strings.TrimSpace(contentType))
}

func sanitizeUploadFilename(filename string) string {
	name := filepath.Base(strings.TrimSpace(filename))
	replacer := strings.NewReplacer("/", "_", "\\", "_", "\x00", "")
	name = replacer.Replace(name)
	if len(name) > 200 {
		extension := filepath.Ext(name)
		base := strings.TrimSuffix(name, extension)
		if len(extension) > 32 {
			extension = extension[:32]
		}
		maxBaseLength := 200 - len(extension)
		if maxBaseLength < 1 {
			maxBaseLength = 1
		}
		if len(base) > maxBaseLength {
			base = base[:maxBaseLength]
		}
		name = base + extension
	}
	return strings.TrimSpace(name)
}

func uploadedFilePreviewText(file state.UploadedFile, content []byte) string {
	if len(content) == 0 {
		return ""
	}

	extension := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(file.Extension), "."))
	mimeType := strings.ToLower(strings.TrimSpace(file.MimeType))
	if (strings.HasPrefix(mimeType, "text/") || hasExtension(textPreviewExtensions, extension) || mimeType == "application/json" || mimeType == "application/xml") && utf8.Valid(content) {
		return truncatePreview(string(content), uploadedFilePreviewWordLimit)
	}
	if strings.HasPrefix(mimeType, "image/") {
		return fmt.Sprintf("Binary image %s is available at %s.", firstNonEmpty(file.Name, file.ID), firstNonEmpty(file.SourceURL, "/files/"+file.ID+"/file-preview"))
	}
	return fmt.Sprintf("Preview for %s is not implemented in dify-go yet.", firstNonEmpty(file.Name, file.ID))
}

func truncatePreview(text string, limit int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}

func generateUploadFileID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return "file_" + hex.EncodeToString(buf)
}

func maxUploadTransferBytes() int64 {
	return int64(uploadVideoFileSizeLimitMB) * 1024 * 1024
}

func writeUploadAPIError(w http.ResponseWriter, err error, fallbackStatus int, fallbackCode, fallbackMessage string) {
	var apiErr *uploadAPIError
	if errors.As(err, &apiErr) {
		writeError(w, apiErr.status, apiErr.code, apiErr.message)
		return
	}
	writeError(w, fallbackStatus, fallbackCode, fallbackMessage)
}

func (s *server) uploadRequestWorkspace(r *http.Request) (state.Workspace, string, bool) {
	user := currentUser(r)
	if user.ID != "" {
		workspace, ok := s.store.UserWorkspace(user.ID)
		return workspace, user.ID, ok
	}

	if token := accessTokenFromRequest(r); token != "" {
		if session, ok := s.sessions.Get(token); ok {
			if user, ok := s.store.GetUser(session.UserID); ok {
				workspace, ok := s.store.UserWorkspace(user.ID)
				return workspace, user.ID, ok
			}
		}
	}

	workspace, ok := s.store.PrimaryWorkspace()
	return workspace, "", ok
}

func accessTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(readCookie(r, accessTokenCookie)); token != "" {
		return token
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}

func (s *server) persistUploadedFile(workspaceID, createdBy, filename, contentType string, content []byte) (state.UploadedFile, error) {
	filename = sanitizeUploadFilename(filename)
	if filename == "" {
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusBadRequest,
			code:    "filename_not_exists",
			message: "Filename is required.",
		}
	}

	contentType = canonicalUploadContentType(contentType)
	if (contentType == "" || contentType == "application/octet-stream") && len(content) > 0 {
		contentType = canonicalUploadContentType(http.DetectContentType(content))
	}

	extension := detectUploadExtension(filename, contentType)
	if _, ok := allowedUploadExtensions()[extension]; !ok {
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusUnsupportedMediaType,
			code:    "unsupported_file_type",
			message: "Unsupported file type.",
		}
	}
	if int64(len(content)) > uploadFileSizeLimitBytes(extension) {
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusRequestEntityTooLarge,
			code:    "file_too_large",
			message: "Uploaded file exceeds the supported size limit.",
		}
	}
	if extension != "" && filepath.Ext(filename) == "" {
		filename += "." + extension
	}

	fileID := generateUploadFileID()
	storageKey := path.Join(workspaceID, fileID+"."+extension)
	fullPath := s.uploadedFilePath(storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusInternalServerError,
			code:    "upload_failed",
			message: "Unable to prepare upload storage.",
		}
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusInternalServerError,
			code:    "upload_failed",
			message: "Unable to store uploaded file.",
		}
	}

	uploadedFile := state.UploadedFile{
		ID:          fileID,
		WorkspaceID: workspaceID,
		Name:        filename,
		Size:        int64(len(content)),
		Extension:   extension,
		MimeType:    contentType,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UTC().Unix(),
		StorageKey:  storageKey,
		SourceURL:   "/files/" + fileID + "/file-preview",
		PreviewURL:  "/files/" + fileID + "/file-preview",
	}
	if uploadedFile.MimeType == "" {
		uploadedFile.MimeType = canonicalUploadContentType(http.DetectContentType(content))
	}

	recorded, err := s.store.RecordUploadedFile(uploadedFile)
	if err != nil {
		_ = os.Remove(fullPath)
		return state.UploadedFile{}, &uploadAPIError{
			status:  http.StatusInternalServerError,
			code:    "upload_failed",
			message: "Unable to persist uploaded file metadata.",
		}
	}
	return recorded, nil
}

func (s *server) fetchRemoteUploadContent(r *http.Request, rawURL string) (string, string, []byte, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsedURL == nil || parsedURL.Host == "" {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "Valid remote file URL is required.",
		}
	}
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
	default:
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "Only HTTP and HTTPS remote files are supported.",
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var headHeader http.Header
	headReq, err := http.NewRequestWithContext(r.Context(), http.MethodHead, parsedURL.String(), nil)
	if err == nil {
		if headResp, err := client.Do(headReq); err == nil {
			headHeader = headResp.Header.Clone()
			if headResp.ContentLength > maxUploadTransferBytes() {
				headResp.Body.Close()
				return "", "", nil, &uploadAPIError{
					status:  http.StatusRequestEntityTooLarge,
					code:    "file_too_large",
					message: "Uploaded file exceeds the supported size limit.",
				}
			}
			headResp.Body.Close()
		}
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "Valid remote file URL is required.",
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadGateway,
			code:    "remote_file_fetch_failed",
			message: "Unable to fetch remote file.",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadGateway,
			code:    "remote_file_fetch_failed",
			message: fmt.Sprintf("Remote file fetch returned HTTP %d.", resp.StatusCode),
		}
	}
	if resp.ContentLength > maxUploadTransferBytes() {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusRequestEntityTooLarge,
			code:    "file_too_large",
			message: "Uploaded file exceeds the supported size limit.",
		}
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxUploadTransferBytes()+1))
	if err != nil {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusBadGateway,
			code:    "remote_file_fetch_failed",
			message: "Unable to read remote file.",
		}
	}
	if int64(len(content)) > maxUploadTransferBytes() {
		return "", "", nil, &uploadAPIError{
			status:  http.StatusRequestEntityTooLarge,
			code:    "file_too_large",
			message: "Uploaded file exceeds the supported size limit.",
		}
	}

	contentType := canonicalUploadContentType(firstNonEmpty(resp.Header.Get("Content-Type"), headHeader.Get("Content-Type")))
	if (contentType == "" || contentType == "application/octet-stream") && len(content) > 0 {
		contentType = canonicalUploadContentType(http.DetectContentType(content))
	}

	filename := remoteUploadFilename(parsedURL, resp.Header, headHeader, contentType)
	return filename, contentType, content, nil
}

func remoteUploadFilename(parsedURL *url.URL, responseHeader, fallbackHeader http.Header, contentType string) string {
	for _, header := range []http.Header{responseHeader, fallbackHeader} {
		if filename := filenameFromContentDisposition(header.Get("Content-Disposition")); filename != "" {
			if filepath.Ext(filename) == "" {
				if extension := detectUploadExtension(filename, contentType); extension != "" {
					filename += "." + extension
				}
			}
			return sanitizeUploadFilename(filename)
		}
	}

	filename := ""
	if parsedURL != nil {
		base := path.Base(strings.TrimSpace(parsedURL.Path))
		if base != "." && base != "/" {
			filename = sanitizeUploadFilename(base)
		}
	}
	if filename == "" {
		filename = "remote-file"
	}
	if filepath.Ext(filename) == "" {
		if extension := detectUploadExtension(filename, contentType); extension != "" {
			filename += "." + extension
		}
	}
	return sanitizeUploadFilename(filename)
}

func filenameFromContentDisposition(contentDisposition string) string {
	if strings.TrimSpace(contentDisposition) == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return ""
	}
	return sanitizeUploadFilename(params["filename"])
}
