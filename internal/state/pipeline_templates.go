package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const defaultPipelineTemplateChunkStructure = "text_model"

type PipelineTemplate struct {
	ID             string          `json:"id"`
	WorkspaceID    string          `json:"workspace_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	ChunkStructure string          `json:"chunk_structure"`
	IconInfo       DatasetIconInfo `json:"icon_info"`
	ExportData     string          `json:"export_data"`
	Position       int             `json:"position"`
	Language       string          `json:"language"`
	CreatedBy      string          `json:"created_by"`
	UpdatedBy      string          `json:"updated_by"`
	CreatedAt      int64           `json:"created_at"`
	UpdatedAt      int64           `json:"updated_at"`
}

type CreatePipelineTemplateInput struct {
	Name           string
	Description    string
	ChunkStructure string
	IconInfo       DatasetIconInfo
	ExportData     string
	Language       string
}

type UpdatePipelineTemplateInput struct {
	Name           string
	Description    string
	ChunkStructure string
	IconInfo       *DatasetIconInfo
	ExportData     *string
}

func (s *Store) ListPipelineTemplates(workspaceID string) []PipelineTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]PipelineTemplate, 0, len(s.state.PipelineTemplates))
	for _, item := range s.state.PipelineTemplates {
		if item.WorkspaceID != workspaceID {
			continue
		}
		items = append(items, clonePipelineTemplate(item))
	}

	slices.SortFunc(items, func(a, b PipelineTemplate) int {
		if a.Position == b.Position {
			if a.UpdatedAt == b.UpdatedAt {
				return strings.Compare(a.ID, b.ID)
			}
			if a.UpdatedAt > b.UpdatedAt {
				return -1
			}
			return 1
		}
		if a.Position < b.Position {
			return -1
		}
		return 1
	})

	return items
}

func (s *Store) GetPipelineTemplate(id, workspaceID string) (PipelineTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.findPipelineTemplateIndexLocked(id, workspaceID)
	if index < 0 {
		return PipelineTemplate{}, false
	}
	return clonePipelineTemplate(s.state.PipelineTemplates[index]), true
}

func (s *Store) CreatePipelineTemplate(workspaceID string, user User, input CreatePipelineTemplateInput, now time.Time) (PipelineTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return PipelineTemplate{}, fmt.Errorf("template name is required")
	}
	exportData := strings.TrimSpace(input.ExportData)
	if exportData == "" {
		return PipelineTemplate{}, fmt.Errorf("template export data is required")
	}

	timestamp := now.UTC().Unix()
	template := PipelineTemplate{
		ID:             generateID("pipeline_tpl"),
		WorkspaceID:    workspaceID,
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		ChunkStructure: strings.TrimSpace(input.ChunkStructure),
		IconInfo:       input.IconInfo,
		ExportData:     exportData,
		Position:       s.nextPipelineTemplatePositionLocked(workspaceID),
		Language:       firstNonEmpty(strings.TrimSpace(input.Language), "en-US"),
		CreatedBy:      user.ID,
		UpdatedBy:      user.ID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}
	normalizePipelineTemplate(&template)

	s.state.PipelineTemplates = append(s.state.PipelineTemplates, template)
	if err := s.saveLocked(); err != nil {
		return PipelineTemplate{}, err
	}
	return clonePipelineTemplate(template), nil
}

func (s *Store) UpdatePipelineTemplate(id, workspaceID string, user User, input UpdatePipelineTemplateInput, now time.Time) (PipelineTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findPipelineTemplateIndexLocked(id, workspaceID)
	if index < 0 {
		return PipelineTemplate{}, fmt.Errorf("template not found")
	}

	template := s.state.PipelineTemplates[index]
	if name := strings.TrimSpace(input.Name); name != "" {
		template.Name = name
	}
	template.Description = strings.TrimSpace(input.Description)
	if chunkStructure := strings.TrimSpace(input.ChunkStructure); chunkStructure != "" {
		template.ChunkStructure = chunkStructure
	}
	if input.IconInfo != nil {
		template.IconInfo = *input.IconInfo
	}
	if input.ExportData != nil {
		template.ExportData = strings.TrimSpace(*input.ExportData)
	}
	template.UpdatedAt = now.UTC().Unix()
	template.UpdatedBy = user.ID
	normalizePipelineTemplate(&template)

	s.state.PipelineTemplates[index] = template
	if err := s.saveLocked(); err != nil {
		return PipelineTemplate{}, err
	}
	return clonePipelineTemplate(template), nil
}

func (s *Store) DeletePipelineTemplate(id, workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findPipelineTemplateIndexLocked(id, workspaceID)
	if index < 0 {
		return fmt.Errorf("template not found")
	}

	s.state.PipelineTemplates = append(s.state.PipelineTemplates[:index], s.state.PipelineTemplates[index+1:]...)
	return s.saveLocked()
}

func (s *Store) findPipelineTemplateIndexLocked(id, workspaceID string) int {
	for i, item := range s.state.PipelineTemplates {
		if item.ID == id && item.WorkspaceID == workspaceID {
			return i
		}
	}
	return -1
}

func (s *Store) nextPipelineTemplatePositionLocked(workspaceID string) int {
	next := 1
	for _, item := range s.state.PipelineTemplates {
		if item.WorkspaceID != workspaceID {
			continue
		}
		if item.Position >= next {
			next = item.Position + 1
		}
	}
	return next
}

func normalizePipelineTemplate(item *PipelineTemplate) {
	if item == nil {
		return
	}
	item.Name = strings.TrimSpace(item.Name)
	item.Description = strings.TrimSpace(item.Description)
	item.ChunkStructure = firstNonEmpty(strings.TrimSpace(item.ChunkStructure), defaultPipelineTemplateChunkStructure)
	item.Language = firstNonEmpty(strings.TrimSpace(item.Language), "en-US")
	if item.Position <= 0 {
		item.Position = 1
	}
	item.IconInfo = normalizeDatasetIconInfo(item.IconInfo, item.Name)
	item.ExportData = strings.TrimSpace(item.ExportData)
}

func clonePipelineTemplate(item PipelineTemplate) PipelineTemplate {
	cloned := item
	cloned.IconInfo = DatasetIconInfo{
		Icon:           item.IconInfo.Icon,
		IconBackground: item.IconInfo.IconBackground,
		IconType:       item.IconInfo.IconType,
		IconURL:        item.IconInfo.IconURL,
	}
	return cloned
}
