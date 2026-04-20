package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
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

func (s *server) fileRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/{fileID}/file-preview", s.handleUploadedFileBinary)
	r.Get("/{fileID}/image-preview", s.handleUploadedFileBinary)
	r.NotFound(s.compatFallback)
	return r
}

func (s *server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
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
	extension := detectUploadExtension(filename, contentType)
	if _, ok := allowedUploadExtensions()[extension]; !ok {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_file_type", "Unsupported file type.")
		return
	}

	limitBytes := uploadFileSizeLimitBytes(extension)
	if int64(len(content)) > limitBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Uploaded file exceeds the supported size limit.")
		return
	}
	if extension != "" && filepath.Ext(filename) == "" {
		filename = filename + "." + extension
	}

	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	user := currentUser(r)
	fileID := generateUploadFileID()
	storageKey := path.Join(workspace.ID, fileID+"."+extension)
	fullPath := s.uploadedFilePath(storageKey)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "upload_failed", "Unable to prepare upload storage.")
		return
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "upload_failed", "Unable to store uploaded file.")
		return
	}

	uploadedFile := state.UploadedFile{
		ID:          fileID,
		WorkspaceID: workspace.ID,
		Name:        filename,
		Size:        int64(len(content)),
		Extension:   extension,
		MimeType:    contentType,
		CreatedBy:   user.ID,
		CreatedAt:   time.Now().UTC().Unix(),
		StorageKey:  storageKey,
		SourceURL:   "/files/" + fileID + "/file-preview",
		PreviewURL:  "/files/" + fileID + "/file-preview",
	}
	if uploadedFile.MimeType == "" {
		uploadedFile.MimeType = http.DetectContentType(content)
	}

	recorded, err := s.store.RecordUploadedFile(uploadedFile)
	if err != nil {
		_ = os.Remove(fullPath)
		writeError(w, http.StatusInternalServerError, "upload_failed", "Unable to persist uploaded file metadata.")
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
