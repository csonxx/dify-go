package state

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

type DatasetSegmentFilters struct {
	Page    int
	Limit   int
	Keyword string
	Enabled string
}

type DatasetSegmentPage struct {
	Page       int
	Limit      int
	Total      int
	TotalPages int
	HasMore    bool
	Data       []DatasetSegment
}

type DatasetChildChunkFilters struct {
	Page    int
	Limit   int
	Keyword string
}

type DatasetChildChunkPage struct {
	Page       int
	Limit      int
	Total      int
	TotalPages int
	Data       []DatasetChildChunk
}

type DatasetDocumentMetadataItem struct {
	ID    string
	Name  string
	Type  string
	Value any
}

type DatasetDocumentMetadataOperation struct {
	DocumentID    string
	MetadataList  []DatasetDocumentMetadataItem
	PartialUpdate bool
}

type DatasetSegmentInput struct {
	Content               string
	Answer                *string
	Summary               *string
	Keywords              *[]string
	AttachmentIDs         *[]string
	RegenerateChildChunks bool
}

func DefaultBuiltInDatasetMetadataFields() []DatasetMetadataField {
	return []DatasetMetadataField{
		{ID: "built-in-source", Name: "source", Type: metadataTypeString},
		{ID: "built-in-created-at", Name: "created_at", Type: metadataTypeTime},
		{ID: "built-in-updated-at", Name: "updated_at", Type: metadataTypeTime},
		{ID: "built-in-segment-count", Name: "segment_count", Type: metadataTypeNumber},
		{ID: "built-in-hit-count", Name: "hit_count", Type: metadataTypeNumber},
	}
}

func (s *Store) ListDatasetMetadataFields(datasetID, workspaceID string) ([]DatasetMetadataField, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return nil, false
	}
	return cloneDatasetMetadataFieldList(s.state.Datasets[datasetIndex].MetadataFields), true
}

func (s *Store) CreateDatasetMetadataField(datasetID, workspaceID string, input DatasetMetadataField, user User, now time.Time) (DatasetMetadataField, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetMetadataField{}, fmt.Errorf("dataset not found")
	}

	field := DatasetMetadataField{
		ID:   generateID("meta"),
		Name: strings.TrimSpace(input.Name),
		Type: strings.TrimSpace(input.Type),
	}
	normalizeDatasetMetadataField(&field)
	if field.Name == "" {
		return DatasetMetadataField{}, fmt.Errorf("metadata name is required")
	}
	if datasetMetadataNameExists(s.state.Datasets[datasetIndex].MetadataFields, field.Name, "") {
		return DatasetMetadataField{}, fmt.Errorf("metadata name already exists")
	}

	s.state.Datasets[datasetIndex].MetadataFields = append(s.state.Datasets[datasetIndex].MetadataFields, field)
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetMetadataField{}, err
	}
	return field, nil
}

func (s *Store) RenameDatasetMetadataField(datasetID, workspaceID, metadataID, name string, user User, now time.Time) (DatasetMetadataField, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetMetadataField{}, fmt.Errorf("dataset not found")
	}
	fieldIndex := findDatasetMetadataFieldIndexLocked(&s.state.Datasets[datasetIndex], metadataID)
	if fieldIndex < 0 {
		return DatasetMetadataField{}, fmt.Errorf("metadata field not found")
	}

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return DatasetMetadataField{}, fmt.Errorf("metadata name is required")
	}
	if datasetMetadataNameExists(s.state.Datasets[datasetIndex].MetadataFields, trimmedName, metadataID) {
		return DatasetMetadataField{}, fmt.Errorf("metadata name already exists")
	}

	s.state.Datasets[datasetIndex].MetadataFields[fieldIndex].Name = trimmedName
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetMetadataField{}, err
	}
	return s.state.Datasets[datasetIndex].MetadataFields[fieldIndex], nil
}

func (s *Store) DeleteDatasetMetadataField(datasetID, workspaceID, metadataID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	fieldIndex := findDatasetMetadataFieldIndexLocked(&s.state.Datasets[datasetIndex], metadataID)
	if fieldIndex < 0 {
		return fmt.Errorf("metadata field not found")
	}

	s.state.Datasets[datasetIndex].MetadataFields = append(
		s.state.Datasets[datasetIndex].MetadataFields[:fieldIndex],
		s.state.Datasets[datasetIndex].MetadataFields[fieldIndex+1:]...,
	)
	for i := range s.state.Datasets[datasetIndex].Documents {
		delete(s.state.Datasets[datasetIndex].Documents[i].MetadataValues, metadataID)
	}
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) SetDatasetBuiltInFieldEnabled(datasetID, workspaceID string, enabled bool, user User, now time.Time) (Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return Dataset{}, fmt.Errorf("dataset not found")
	}

	s.state.Datasets[datasetIndex].BuiltInFieldEnabled = enabled
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return Dataset{}, err
	}
	return cloneDataset(s.state.Datasets[datasetIndex]), nil
}

func (s *Store) BatchUpdateDatasetDocumentMetadata(datasetID, workspaceID string, operations []DatasetDocumentMetadataOperation, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	if len(operations) == 0 {
		return fmt.Errorf("metadata operations are required")
	}

	for _, op := range operations {
		documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], strings.TrimSpace(op.DocumentID))
		if documentIndex < 0 {
			return fmt.Errorf("document not found")
		}
		document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
		nextValues := map[string]string{}
		if op.PartialUpdate {
			nextValues = cloneStringMap(document.MetadataValues)
		}
		for _, item := range op.MetadataList {
			field := ensureDatasetMetadataFieldLocked(&s.state.Datasets[datasetIndex], item)
			nextValues[field.ID] = metadataStringValue(item.Value)
		}
		document.MetadataValues = nextValues
		document.UpdatedAt = now.UTC().Unix()
	}

	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) UpdateDatasetDocumentMetadataProfile(datasetID, workspaceID, documentID, docType string, metadata map[string]any, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return fmt.Errorf("document not found")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	document.DocType = strings.TrimSpace(docType)
	document.DocMetadata = map[string]string{}
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		document.DocMetadata[trimmedKey] = metadataStringValue(value)
	}
	document.UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) ListDatasetSegments(datasetID, workspaceID, documentID string, filters DatasetSegmentFilters) (DatasetSegmentPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetSegmentPage{}, false
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetSegmentPage{}, false
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items := make([]DatasetSegment, 0, len(s.state.Datasets[datasetIndex].Documents[documentIndex].Segments))
	for _, segment := range s.state.Datasets[datasetIndex].Documents[documentIndex].Segments {
		if filters.Keyword != "" && !strings.Contains(strings.ToLower(segment.Content+" "+segment.Answer+" "+segment.Summary), strings.ToLower(strings.TrimSpace(filters.Keyword))) {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(filters.Enabled)) {
		case "true":
			if !segment.Enabled {
				continue
			}
		case "false":
			if segment.Enabled {
				continue
			}
		}
		items = append(items, cloneDatasetSegment(segment))
	}

	slices.SortFunc(items, func(a, b DatasetSegment) int {
		if a.Position == b.Position {
			return bcmp(a.ID, b.ID)
		}
		if a.Position < b.Position {
			return -1
		}
		return 1
	})

	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return DatasetSegmentPage{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
		HasMore:    end < total,
		Data:       cloneDatasetSegmentList(items[start:end]),
	}, true
}

func (s *Store) GetDatasetSegment(datasetID, workspaceID, documentID, segmentID string) (DatasetSegment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetSegment{}, false
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetSegment{}, false
	}
	segmentIndex := findDatasetSegmentIndexLocked(&s.state.Datasets[datasetIndex].Documents[documentIndex], segmentID)
	if segmentIndex < 0 {
		return DatasetSegment{}, false
	}
	return cloneDatasetSegment(s.state.Datasets[datasetIndex].Documents[documentIndex].Segments[segmentIndex]), true
}

func (s *Store) AddDatasetSegment(datasetID, workspaceID, documentID string, input DatasetSegmentInput, user User, now time.Time) (DatasetSegment, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetSegment{}, "", fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetSegment{}, "", fmt.Errorf("document not found")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	timestamp := now.UTC().Unix()
	segment := DatasetSegment{
		ID:            generateID("seg"),
		Position:      len(document.Segments) + 1,
		DocumentID:    document.ID,
		Content:       strings.TrimSpace(input.Content),
		SignContent:   strings.TrimSpace(input.Content),
		WordCount:     max(estimateWordCount(input.Content), 1),
		Tokens:        estimateTokenCount(input.Content),
		Keywords:      cloneStringPointerSlice(input.Keywords),
		IndexNodeID:   generateID("node"),
		IndexNodeHash: generateID("nodehash"),
		HitCount:      0,
		Enabled:       true,
		Status:        segmentStatusCompleted,
		CreatedBy:     user.ID,
		CreatedAt:     timestamp,
		IndexingAt:    timestamp,
		CompletedAt:   timestamp,
		UpdatedAt:     timestamp,
		Attachments:   attachmentsFromIDs(input.AttachmentIDs),
	}
	if segment.Content == "" {
		return DatasetSegment{}, "", fmt.Errorf("segment content is required")
	}
	if input.Answer != nil {
		segment.Answer = strings.TrimSpace(*input.Answer)
	}
	if input.Summary != nil {
		segment.Summary = strings.TrimSpace(*input.Summary)
	}
	if len(segment.Keywords) == 0 {
		segment.Keywords = datasetDocumentKeywords(document.Name, segment.Content)
	}
	if document.DocForm == "hierarchical_model" {
		segment.ChildChunks = segmentChildChunksFromContent(segment.Content, segment.ID, now, childChunkTypeAutomatic)
	}

	document.Segments = append(document.Segments, segment)
	sortDatasetSegmentsLocked(document)
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetSegment{}, "", err
	}

	segmentIndex := findDatasetSegmentIndexLocked(document, segment.ID)
	return cloneDatasetSegment(document.Segments[segmentIndex]), document.DocForm, nil
}

func (s *Store) UpdateDatasetSegment(datasetID, workspaceID, documentID, segmentID string, input DatasetSegmentInput, user User, now time.Time) (DatasetSegment, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetSegment{}, "", fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetSegment{}, "", fmt.Errorf("document not found")
	}
	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	segmentIndex := findDatasetSegmentIndexLocked(document, segmentID)
	if segmentIndex < 0 {
		return DatasetSegment{}, "", fmt.Errorf("segment not found")
	}

	segment := &document.Segments[segmentIndex]
	timestamp := now.UTC().Unix()
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return DatasetSegment{}, "", fmt.Errorf("segment content is required")
	}
	segment.Content = content
	segment.SignContent = content
	segment.WordCount = max(estimateWordCount(content), 1)
	segment.Tokens = estimateTokenCount(content)
	if input.Answer != nil {
		segment.Answer = strings.TrimSpace(*input.Answer)
	}
	if input.Summary != nil {
		segment.Summary = strings.TrimSpace(*input.Summary)
	}
	if input.Keywords != nil {
		segment.Keywords = cloneStringSlice(*input.Keywords)
	}
	if input.AttachmentIDs != nil {
		segment.Attachments = attachmentsFromIDs(input.AttachmentIDs)
	}
	if input.RegenerateChildChunks {
		segment.ChildChunks = segmentChildChunksFromContent(segment.Content, segment.ID, now, childChunkTypeAutomatic)
	}
	segment.UpdatedAt = timestamp
	segment.CompletedAt = timestamp
	segment.Status = segmentStatusCompleted
	sortDatasetSegmentsLocked(document)
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetSegment{}, "", err
	}
	return cloneDatasetSegment(document.Segments[segmentIndex]), document.DocForm, nil
}

func (s *Store) SetDatasetSegmentEnabled(datasetID, workspaceID, documentID string, segmentIDs []string, enabled bool, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return fmt.Errorf("document not found")
	}
	if len(segmentIDs) == 0 {
		return fmt.Errorf("segment ids are required")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	targets := make(map[string]struct{}, len(segmentIDs))
	for _, segmentID := range segmentIDs {
		if trimmed := strings.TrimSpace(segmentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("segment ids are required")
	}

	timestamp := now.UTC().Unix()
	updated := false
	for i := range document.Segments {
		segment := &document.Segments[i]
		if _, ok := targets[segment.ID]; !ok {
			continue
		}
		segment.Enabled = enabled
		if enabled {
			segment.DisabledAt = 0
			segment.DisabledBy = ""
		} else {
			segment.DisabledAt = timestamp
			segment.DisabledBy = user.ID
		}
		segment.UpdatedAt = timestamp
		updated = true
	}
	if !updated {
		return fmt.Errorf("segment not found")
	}

	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) DeleteDatasetSegments(datasetID, workspaceID, documentID string, segmentIDs []string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return fmt.Errorf("document not found")
	}
	if len(segmentIDs) == 0 {
		return fmt.Errorf("segment ids are required")
	}

	targets := make(map[string]struct{}, len(segmentIDs))
	for _, segmentID := range segmentIDs {
		if trimmed := strings.TrimSpace(segmentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("segment ids are required")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	filtered := document.Segments[:0]
	removed := false
	for _, segment := range document.Segments {
		if _, ok := targets[segment.ID]; ok {
			removed = true
			continue
		}
		filtered = append(filtered, segment)
	}
	if !removed {
		return fmt.Errorf("segment not found")
	}
	document.Segments = filtered
	sortDatasetSegmentsLocked(document)
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) ListDatasetChildChunks(datasetID, workspaceID, documentID, segmentID string, filters DatasetChildChunkFilters) (DatasetChildChunkPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetChildChunkPage{}, false
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetChildChunkPage{}, false
	}
	segmentIndex := findDatasetSegmentIndexLocked(&s.state.Datasets[datasetIndex].Documents[documentIndex], segmentID)
	if segmentIndex < 0 {
		return DatasetChildChunkPage{}, false
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items := make([]DatasetChildChunk, 0, len(s.state.Datasets[datasetIndex].Documents[documentIndex].Segments[segmentIndex].ChildChunks))
	for _, chunk := range s.state.Datasets[datasetIndex].Documents[documentIndex].Segments[segmentIndex].ChildChunks {
		if filters.Keyword != "" && !strings.Contains(strings.ToLower(chunk.Content), strings.ToLower(strings.TrimSpace(filters.Keyword))) {
			continue
		}
		items = append(items, cloneDatasetChildChunk(chunk))
	}

	slices.SortFunc(items, func(a, b DatasetChildChunk) int {
		if a.Position == b.Position {
			return bcmp(a.ID, b.ID)
		}
		if a.Position < b.Position {
			return -1
		}
		return 1
	})

	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return DatasetChildChunkPage{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
		Data:       cloneDatasetChildChunkList(items[start:end]),
	}, true
}

func (s *Store) AddDatasetChildChunk(datasetID, workspaceID, documentID, segmentID, content string, user User, now time.Time) (DatasetChildChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("document not found")
	}
	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	segmentIndex := findDatasetSegmentIndexLocked(document, segmentID)
	if segmentIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("segment not found")
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return DatasetChildChunk{}, fmt.Errorf("child chunk content is required")
	}

	segment := &document.Segments[segmentIndex]
	timestamp := now.UTC().Unix()
	chunk := DatasetChildChunk{
		ID:        generateID("child"),
		SegmentID: segment.ID,
		Content:   trimmedContent,
		Position:  len(segment.ChildChunks) + 1,
		WordCount: max(estimateWordCount(trimmedContent), 1),
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
		Type:      childChunkTypeCustomized,
	}
	segment.ChildChunks = append(segment.ChildChunks, chunk)
	segment.UpdatedAt = timestamp
	sortDatasetChildChunksLocked(&segment.ChildChunks)
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetChildChunk{}, err
	}
	chunkIndex := findDatasetChildChunkIndexLocked(segment.ChildChunks, chunk.ID)
	return cloneDatasetChildChunk(segment.ChildChunks[chunkIndex]), nil
}

func (s *Store) UpdateDatasetChildChunk(datasetID, workspaceID, documentID, segmentID, childChunkID, content string, user User, now time.Time) (DatasetChildChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("document not found")
	}
	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	segmentIndex := findDatasetSegmentIndexLocked(document, segmentID)
	if segmentIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("segment not found")
	}
	childIndex := findDatasetChildChunkIndexLocked(document.Segments[segmentIndex].ChildChunks, childChunkID)
	if childIndex < 0 {
		return DatasetChildChunk{}, fmt.Errorf("child chunk not found")
	}

	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		return DatasetChildChunk{}, fmt.Errorf("child chunk content is required")
	}

	timestamp := now.UTC().Unix()
	chunk := &document.Segments[segmentIndex].ChildChunks[childIndex]
	chunk.Content = trimmedContent
	chunk.WordCount = max(estimateWordCount(trimmedContent), 1)
	chunk.UpdatedAt = timestamp
	chunk.Type = childChunkTypeCustomized
	document.Segments[segmentIndex].UpdatedAt = timestamp
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetChildChunk{}, err
	}
	return cloneDatasetChildChunk(document.Segments[segmentIndex].ChildChunks[childIndex]), nil
}

func (s *Store) DeleteDatasetChildChunk(datasetID, workspaceID, documentID, segmentID, childChunkID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return fmt.Errorf("document not found")
	}
	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	segmentIndex := findDatasetSegmentIndexLocked(document, segmentID)
	if segmentIndex < 0 {
		return fmt.Errorf("segment not found")
	}

	segment := &document.Segments[segmentIndex]
	childIndex := findDatasetChildChunkIndexLocked(segment.ChildChunks, childChunkID)
	if childIndex < 0 {
		return fmt.Errorf("child chunk not found")
	}

	segment.ChildChunks = append(segment.ChildChunks[:childIndex], segment.ChildChunks[childIndex+1:]...)
	sortDatasetChildChunksLocked(&segment.ChildChunks)
	segment.UpdatedAt = now.UTC().Unix()
	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) CreateDatasetBatchImportJob(datasetID, workspaceID, documentID, uploadFileID string, user User, now time.Time) (DatasetBatchImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetBatchImportJob{}, fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetBatchImportJob{}, fmt.Errorf("document not found")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	timestamp := now.UTC().Unix()
	job := DatasetBatchImportJob{
		ID:         generateID("job"),
		DatasetID:  datasetID,
		DocumentID: documentID,
		Status:     batchImportStatusCompleted,
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp,
	}

	importLabel := firstNonEmpty(strings.TrimSpace(uploadFileID), "batch-import")
	if document.DocForm == "hierarchical_model" && document.DocumentProcessRule.Rules.ParentMode == "full-doc" {
		if len(document.Segments) == 0 {
			document.Segments = []DatasetSegment{datasetSegmentFromDocument(*document)}
		}
		segment := &document.Segments[0]
		for i := 0; i < 2; i++ {
			segment.ChildChunks = append(segment.ChildChunks, DatasetChildChunk{
				ID:        generateID("child"),
				SegmentID: segment.ID,
				Content:   fmt.Sprintf("Imported child chunk %d from %s.", i+1, importLabel),
				Position:  len(segment.ChildChunks) + 1,
				WordCount: 6,
				CreatedAt: timestamp,
				UpdatedAt: timestamp,
				Type:      childChunkTypeCustomized,
			})
		}
		sortDatasetChildChunksLocked(&segment.ChildChunks)
		segment.UpdatedAt = timestamp
	} else {
		for i := 0; i < 2; i++ {
			document.Segments = append(document.Segments, DatasetSegment{
				ID:            generateID("seg"),
				Position:      len(document.Segments) + 1,
				DocumentID:    document.ID,
				Content:       fmt.Sprintf("Imported segment %d from %s.", i+1, importLabel),
				SignContent:   fmt.Sprintf("Imported segment %d from %s.", i+1, importLabel),
				WordCount:     6,
				Tokens:        12,
				Keywords:      []string{"imported", "batch"},
				IndexNodeID:   generateID("node"),
				IndexNodeHash: generateID("nodehash"),
				Enabled:       true,
				Status:        segmentStatusCompleted,
				CreatedBy:     user.ID,
				CreatedAt:     timestamp,
				IndexingAt:    timestamp,
				CompletedAt:   timestamp,
				UpdatedAt:     timestamp,
				Attachments:   []DatasetAttachment{},
				ChildChunks:   []DatasetChildChunk{},
			})
		}
		sortDatasetSegmentsLocked(document)
	}

	syncDatasetDocumentFromSegments(document)
	document.UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].BatchImportJobs = append([]DatasetBatchImportJob{job}, s.state.Datasets[datasetIndex].BatchImportJobs...)
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetBatchImportJob{}, err
	}
	return job, nil
}

func (s *Store) GetDatasetBatchImportJob(workspaceID, jobID string) (DatasetBatchImportJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		for _, job := range dataset.BatchImportJobs {
			if job.ID == jobID {
				return job, true
			}
		}
	}
	return DatasetBatchImportJob{}, false
}

func datasetSegmentFromDocument(document DatasetDocument) DatasetSegment {
	segmentID := "seg_" + document.ID
	childChunks := cloneDatasetChildChunkList(document.ChildChunks)
	for i := range childChunks {
		childChunks[i].SegmentID = segmentID
		normalizeDatasetChildChunk(&childChunks[i], segmentID)
	}
	segment := DatasetSegment{
		ID:            segmentID,
		Position:      1,
		DocumentID:    document.ID,
		Content:       document.Content,
		SignContent:   firstNonEmpty(document.SignContent, document.Content),
		WordCount:     max(max(document.WordCount, estimateWordCount(document.Content)), 1),
		Tokens:        max(document.Tokens, estimateTokenCount(document.Content)),
		Keywords:      cloneStringSlice(document.Keywords),
		IndexNodeID:   "node_" + document.ID,
		IndexNodeHash: document.ID,
		HitCount:      document.HitCount,
		Enabled:       true,
		Status:        segmentStatusCompleted,
		CreatedBy:     document.CreatedBy,
		CreatedAt:     firstNonZeroInt64(document.CreatedAt, document.UpdatedAt),
		IndexingAt:    firstNonZeroInt64(document.ProcessingStartedAt, document.CreatedAt, document.UpdatedAt),
		CompletedAt:   firstNonZeroInt64(document.CompletedAt, document.UpdatedAt, document.CreatedAt),
		Answer:        "",
		Summary:       document.Summary,
		ChildChunks:   childChunks,
		UpdatedAt:     firstNonZeroInt64(document.UpdatedAt, document.CreatedAt),
		Attachments:   cloneDatasetAttachmentList(document.Attachments),
	}
	if document.DocForm == "qa_model" {
		segment.Answer = document.Summary
	}
	normalizeDatasetSegment(&segment, &document)
	return segment
}

func syncDatasetDocumentFromSegments(document *DatasetDocument) {
	sortDatasetSegmentsLocked(document)
	if len(document.Segments) == 0 {
		document.Content = ""
		document.SignContent = ""
		document.Keywords = []string{}
		document.Summary = ""
		document.Attachments = []DatasetAttachment{}
		document.ChildChunks = []DatasetChildChunk{}
		document.WordCount = 0
		document.Tokens = 0
		document.HitCount = 0
		document.SegmentCount = 0
		document.TotalSegments = 0
		document.CompletedSegments = 0
		return
	}

	contentParts := make([]string, 0, len(document.Segments))
	signParts := make([]string, 0, len(document.Segments))
	keywordSeen := map[string]struct{}{}
	keywords := make([]string, 0, 8)
	attachmentSeen := map[string]struct{}{}
	attachments := make([]DatasetAttachment, 0)
	childChunks := make([]DatasetChildChunk, 0)
	wordCount := 0
	tokens := 0
	hitCount := 0
	completedCount := 0
	summary := ""

	for _, segment := range document.Segments {
		contentParts = append(contentParts, strings.TrimSpace(segment.Content))
		signParts = append(signParts, firstNonEmpty(segment.SignContent, segment.Content))
		wordCount += segment.WordCount
		tokens += segment.Tokens
		hitCount += segment.HitCount
		if segment.Status == segmentStatusCompleted {
			completedCount++
		}
		if summary == "" {
			summary = strings.TrimSpace(segment.Summary)
		}
		for _, keyword := range segment.Keywords {
			trimmedKeyword := strings.TrimSpace(keyword)
			if trimmedKeyword == "" {
				continue
			}
			if _, ok := keywordSeen[trimmedKeyword]; ok {
				continue
			}
			keywordSeen[trimmedKeyword] = struct{}{}
			keywords = append(keywords, trimmedKeyword)
		}
		for _, attachment := range segment.Attachments {
			key := firstNonEmpty(attachment.ID, attachment.Name)
			if _, ok := attachmentSeen[key]; ok {
				continue
			}
			attachmentSeen[key] = struct{}{}
			attachments = append(attachments, attachment)
		}
		childChunks = append(childChunks, cloneDatasetChildChunkList(segment.ChildChunks)...)
	}

	sortDatasetChildChunksLocked(&childChunks)
	document.Content = strings.TrimSpace(strings.Join(contentParts, "\n\n"))
	document.SignContent = strings.TrimSpace(strings.Join(signParts, "\n\n"))
	document.Keywords = keywords
	document.Summary = summary
	document.Attachments = attachments
	document.ChildChunks = childChunks
	document.WordCount = wordCount
	document.Tokens = tokens
	document.HitCount = hitCount
	document.SegmentCount = len(document.Segments)
	document.TotalSegments = len(document.Segments)
	document.CompletedSegments = completedCount
}

func segmentChildChunksFromContent(content, segmentID string, now time.Time, chunkType string) []DatasetChildChunk {
	lines := splitSegmentContent(content)
	chunks := make([]DatasetChildChunk, 0, len(lines))
	timestamp := now.UTC().Unix()
	for i, line := range lines {
		chunks = append(chunks, DatasetChildChunk{
			ID:        generateID("child"),
			SegmentID: segmentID,
			Content:   line,
			Position:  i + 1,
			WordCount: max(estimateWordCount(line), 1),
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
			Type:      chunkType,
		})
	}
	return chunks
}

func splitSegmentContent(content string) []string {
	lines := strings.FieldsFunc(strings.TrimSpace(content), func(r rune) bool {
		return r == '\n'
	})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 && strings.TrimSpace(content) != "" {
		out = []string{strings.TrimSpace(content)}
	}
	if len(out) == 0 {
		out = []string{"Empty child chunk"}
	}
	return out
}

func attachmentsFromIDs(ids *[]string) []DatasetAttachment {
	if ids == nil {
		return []DatasetAttachment{}
	}
	items := make([]DatasetAttachment, 0, len(*ids))
	for _, id := range *ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		items = append(items, DatasetAttachment{
			ID:        trimmed,
			Name:      trimmed,
			Extension: datasetFileExtension(trimmed),
			MimeType:  datasetMimeType(datasetFileExtension(trimmed)),
			SourceURL: "/files/" + trimmed,
		})
	}
	return items
}

func cloneDatasetMetadataFieldList(src []DatasetMetadataField) []DatasetMetadataField {
	if src == nil {
		return []DatasetMetadataField{}
	}
	out := make([]DatasetMetadataField, len(src))
	copy(out, src)
	return out
}

func cloneDatasetSegment(src DatasetSegment) DatasetSegment {
	segment := DatasetSegment{
		ID:            src.ID,
		Position:      src.Position,
		DocumentID:    src.DocumentID,
		Content:       src.Content,
		SignContent:   src.SignContent,
		WordCount:     src.WordCount,
		Tokens:        src.Tokens,
		Keywords:      cloneStringSlice(src.Keywords),
		IndexNodeID:   src.IndexNodeID,
		IndexNodeHash: src.IndexNodeHash,
		HitCount:      src.HitCount,
		Enabled:       src.Enabled,
		DisabledAt:    src.DisabledAt,
		DisabledBy:    src.DisabledBy,
		Status:        src.Status,
		CreatedBy:     src.CreatedBy,
		CreatedAt:     src.CreatedAt,
		IndexingAt:    src.IndexingAt,
		CompletedAt:   src.CompletedAt,
		Error:         src.Error,
		StoppedAt:     src.StoppedAt,
		Answer:        src.Answer,
		Summary:       src.Summary,
		ChildChunks:   cloneDatasetChildChunkList(src.ChildChunks),
		UpdatedAt:     src.UpdatedAt,
		Attachments:   cloneDatasetAttachmentList(src.Attachments),
	}
	return segment
}

func cloneDatasetSegmentList(src []DatasetSegment) []DatasetSegment {
	if src == nil {
		return []DatasetSegment{}
	}
	out := make([]DatasetSegment, len(src))
	for i, item := range src {
		out[i] = cloneDatasetSegment(item)
	}
	return out
}

func cloneDatasetChildChunk(src DatasetChildChunk) DatasetChildChunk {
	return DatasetChildChunk{
		ID:        src.ID,
		SegmentID: src.SegmentID,
		Content:   src.Content,
		Position:  src.Position,
		Score:     src.Score,
		WordCount: src.WordCount,
		CreatedAt: src.CreatedAt,
		UpdatedAt: src.UpdatedAt,
		Type:      src.Type,
	}
}

func findDatasetMetadataFieldIndexLocked(dataset *Dataset, metadataID string) int {
	for i, field := range dataset.MetadataFields {
		if field.ID == metadataID {
			return i
		}
	}
	return -1
}

func findDatasetDocumentIndexLocked(dataset *Dataset, documentID string) int {
	for i, document := range dataset.Documents {
		if document.ID == documentID {
			return i
		}
	}
	return -1
}

func findDatasetSegmentIndexLocked(document *DatasetDocument, segmentID string) int {
	for i, segment := range document.Segments {
		if segment.ID == segmentID {
			return i
		}
	}
	return -1
}

func findDatasetChildChunkIndexLocked(chunks []DatasetChildChunk, childChunkID string) int {
	for i, chunk := range chunks {
		if chunk.ID == childChunkID {
			return i
		}
	}
	return -1
}

func datasetMetadataNameExists(fields []DatasetMetadataField, name, excludeID string) bool {
	for _, field := range fields {
		if excludeID != "" && field.ID == excludeID {
			continue
		}
		if strings.EqualFold(field.Name, name) {
			return true
		}
	}
	return false
}

func ensureDatasetMetadataFieldLocked(dataset *Dataset, item DatasetDocumentMetadataItem) DatasetMetadataField {
	metadataID := strings.TrimSpace(item.ID)
	if metadataID != "" {
		if index := findDatasetMetadataFieldIndexLocked(dataset, metadataID); index >= 0 {
			return dataset.MetadataFields[index]
		}
	}
	name := strings.TrimSpace(item.Name)
	for _, field := range dataset.MetadataFields {
		if strings.EqualFold(field.Name, name) {
			return field
		}
	}
	field := DatasetMetadataField{
		ID:   firstNonEmpty(metadataID, generateID("meta")),
		Name: firstNonEmpty(name, "metadata"),
		Type: strings.TrimSpace(item.Type),
	}
	normalizeDatasetMetadataField(&field)
	dataset.MetadataFields = append(dataset.MetadataFields, field)
	return field
}

func metadataStringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == math.Trunc(typed) {
			return fmt.Sprintf("%.0f", typed)
		}
		return fmt.Sprintf("%v", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func sortDatasetSegmentsLocked(document *DatasetDocument) {
	slices.SortFunc(document.Segments, func(a, b DatasetSegment) int {
		if a.Position == b.Position {
			return bcmp(a.ID, b.ID)
		}
		if a.Position < b.Position {
			return -1
		}
		return 1
	})
	for i := range document.Segments {
		document.Segments[i].Position = i + 1
		sortDatasetChildChunksLocked(&document.Segments[i].ChildChunks)
	}
}

func sortDatasetChildChunksLocked(chunks *[]DatasetChildChunk) {
	slices.SortFunc(*chunks, func(a, b DatasetChildChunk) int {
		if a.Position == b.Position {
			return bcmp(a.ID, b.ID)
		}
		if a.Position < b.Position {
			return -1
		}
		return 1
	})
	for i := range *chunks {
		(*chunks)[i].Position = i + 1
	}
}

func totalPages(total, limit int) int {
	if total == 0 || limit <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func cloneStringPointerSlice(src *[]string) []string {
	if src == nil {
		return []string{}
	}
	return cloneStringSlice(*src)
}
