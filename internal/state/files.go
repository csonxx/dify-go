package state

import (
	"fmt"
	"strings"
)

type UploadedFile struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	Extension   string `json:"extension"`
	MimeType    string `json:"mime_type"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   int64  `json:"created_at"`
	StorageKey  string `json:"storage_key"`
	SourceURL   string `json:"source_url"`
	PreviewURL  string `json:"preview_url"`
}

func normalizeUploadedFile(file *UploadedFile) {
	file.ID = strings.TrimSpace(file.ID)
	file.WorkspaceID = strings.TrimSpace(file.WorkspaceID)
	file.Name = strings.TrimSpace(file.Name)
	file.Extension = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(file.Extension), "."))
	file.MimeType = strings.TrimSpace(file.MimeType)
	file.CreatedBy = strings.TrimSpace(file.CreatedBy)
	file.StorageKey = strings.TrimSpace(file.StorageKey)
	file.SourceURL = strings.TrimSpace(file.SourceURL)
	file.PreviewURL = strings.TrimSpace(file.PreviewURL)
	if file.Extension == "" && file.Name != "" {
		file.Extension = datasetFileExtension(file.Name)
	}
	if file.MimeType == "" {
		file.MimeType = datasetMimeType(file.Extension)
	}
	if file.SourceURL == "" && file.ID != "" {
		file.SourceURL = "/files/" + file.ID + "/file-preview"
	}
	if file.PreviewURL == "" {
		file.PreviewURL = file.SourceURL
	}
}

func cloneUploadedFile(src UploadedFile) UploadedFile {
	return src
}

func (s *Store) RecordUploadedFile(file UploadedFile) (UploadedFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizeUploadedFile(&file)
	if file.ID == "" {
		return UploadedFile{}, fmt.Errorf("file id is required")
	}
	if file.WorkspaceID == "" {
		return UploadedFile{}, fmt.Errorf("workspace id is required")
	}
	if file.Name == "" {
		return UploadedFile{}, fmt.Errorf("file name is required")
	}
	if file.StorageKey == "" {
		return UploadedFile{}, fmt.Errorf("storage key is required")
	}

	for i := range s.state.UploadedFiles {
		if s.state.UploadedFiles[i].ID != file.ID {
			continue
		}
		s.state.UploadedFiles[i] = file
		if err := s.saveLocked(); err != nil {
			return UploadedFile{}, err
		}
		return cloneUploadedFile(file), nil
	}

	s.state.UploadedFiles = append([]UploadedFile{file}, s.state.UploadedFiles...)
	if err := s.saveLocked(); err != nil {
		return UploadedFile{}, err
	}
	return cloneUploadedFile(file), nil
}

func (s *Store) GetUploadedFile(workspaceID, fileID string) (UploadedFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, ok := s.findUploadedFileLocked(workspaceID, fileID)
	if !ok {
		return UploadedFile{}, false
	}
	return cloneUploadedFile(file), true
}

func (s *Store) FindUploadedFile(fileID string) (UploadedFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	file, ok := s.findUploadedFileAnyWorkspaceLocked(fileID)
	if !ok {
		return UploadedFile{}, false
	}
	return cloneUploadedFile(file), true
}

func (s *Store) findUploadedFileLocked(workspaceID, fileID string) (UploadedFile, bool) {
	trimmedWorkspaceID := strings.TrimSpace(workspaceID)
	trimmedFileID := strings.TrimSpace(fileID)
	for _, file := range s.state.UploadedFiles {
		if file.ID != trimmedFileID {
			continue
		}
		if trimmedWorkspaceID != "" && file.WorkspaceID != trimmedWorkspaceID {
			continue
		}
		return file, true
	}
	return UploadedFile{}, false
}

func (s *Store) findUploadedFileAnyWorkspaceLocked(fileID string) (UploadedFile, bool) {
	trimmedFileID := strings.TrimSpace(fileID)
	for _, file := range s.state.UploadedFiles {
		if file.ID == trimmedFileID {
			return file, true
		}
	}
	return UploadedFile{}, false
}

func datasetAttachmentFromUploadedFile(file UploadedFile) DatasetAttachment {
	return DatasetAttachment{
		ID:        file.ID,
		Name:      file.Name,
		Size:      file.Size,
		Extension: file.Extension,
		MimeType:  file.MimeType,
		SourceURL: file.SourceURL,
	}
}
