package state

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"
)

const (
	datasetPermissionAllTeamMembers = "all_team_members"
	datasetRuntimeModeGeneral       = "general"
	datasetProviderLocal            = "local"
	datasetProviderExternal         = "external"
	documentIndexingStatusCompleted = "completed"
	documentIndexingStatusPaused    = "paused"
	documentIndexingStatusError     = "error"
	documentDisplayStatusAvailable  = "available"
	documentDisplayStatusPaused     = "paused"
	documentDisplayStatusError      = "error"
	documentDisplayStatusEnabled    = "enabled"
	documentDisplayStatusDisabled   = "disabled"
	documentDisplayStatusArchived   = "archived"
	metadataTypeString              = "string"
	metadataTypeNumber              = "number"
	metadataTypeTime                = "time"
	childChunkTypeAutomatic         = "automatic"
	childChunkTypeCustomized        = "customized"
	segmentStatusWaiting            = "waiting"
	segmentStatusCompleted          = "completed"
	segmentStatusError              = "error"
	segmentStatusIndexing           = "indexing"
	batchImportStatusWaiting        = "waiting"
	batchImportStatusProcessing     = "processing"
	batchImportStatusCompleted      = "completed"
	batchImportStatusFailed         = "error"
)

type Dataset struct {
	ID                     string                        `json:"id"`
	WorkspaceID            string                        `json:"workspace_id"`
	Name                   string                        `json:"name"`
	Description            string                        `json:"description"`
	Permission             string                        `json:"permission"`
	DataSourceType         string                        `json:"data_source_type"`
	IndexingTechnique      string                        `json:"indexing_technique"`
	AuthorName             string                        `json:"author_name"`
	CreatedBy              string                        `json:"created_by"`
	UpdatedBy              string                        `json:"updated_by"`
	CreatedAt              int64                         `json:"created_at"`
	UpdatedAt              int64                         `json:"updated_at"`
	DocForm                string                        `json:"doc_form"`
	Provider               string                        `json:"provider"`
	EmbeddingModel         string                        `json:"embedding_model"`
	EmbeddingModelProvider string                        `json:"embedding_model_provider"`
	EmbeddingAvailable     bool                          `json:"embedding_available"`
	IconInfo               DatasetIconInfo               `json:"icon_info"`
	RetrievalModel         DatasetRetrievalModel         `json:"retrieval_model"`
	ExternalKnowledgeInfo  DatasetExternalKnowledgeInfo  `json:"external_knowledge_info"`
	ExternalRetrievalModel DatasetExternalRetrievalModel `json:"external_retrieval_model"`
	BuiltInFieldEnabled    bool                          `json:"built_in_field_enabled"`
	PartialMemberList      []string                      `json:"partial_member_list"`
	RuntimeMode            string                        `json:"runtime_mode"`
	EnableAPI              bool                          `json:"enable_api"`
	IsPublished            bool                          `json:"is_published"`
	IsMultimodal           bool                          `json:"is_multimodal"`
	SummaryIndexSetting    DatasetSummaryIndexSetting    `json:"summary_index_setting"`
	PipelineID             string                        `json:"pipeline_id"`
	MetadataFields         []DatasetMetadataField        `json:"metadata_fields"`
	BatchImportJobs        []DatasetBatchImportJob       `json:"batch_import_jobs"`
	Documents              []DatasetDocument             `json:"documents"`
	Queries                []DatasetQueryRecord          `json:"queries"`
}

type DatasetIconInfo struct {
	Icon           string `json:"icon"`
	IconBackground string `json:"icon_background"`
	IconType       string `json:"icon_type"`
	IconURL        string `json:"icon_url"`
}

type DatasetRetrievalModel struct {
	SearchMethod          string               `json:"search_method"`
	RerankingEnable       bool                 `json:"reranking_enable"`
	RerankingModel        DatasetProviderModel `json:"reranking_model"`
	TopK                  int                  `json:"top_k"`
	ScoreThresholdEnabled bool                 `json:"score_threshold_enabled"`
	ScoreThreshold        float64              `json:"score_threshold"`
	RerankingMode         string               `json:"reranking_mode,omitempty"`
	Weights               map[string]any       `json:"weights,omitempty"`
}

type DatasetProviderModel struct {
	ProviderName string `json:"reranking_provider_name"`
	ModelName    string `json:"reranking_model_name"`
}

type DatasetExternalKnowledgeInfo struct {
	ExternalKnowledgeID          string `json:"external_knowledge_id"`
	ExternalKnowledgeAPIID       string `json:"external_knowledge_api_id"`
	ExternalKnowledgeAPIName     string `json:"external_knowledge_api_name"`
	ExternalKnowledgeAPIEndpoint string `json:"external_knowledge_api_endpoint"`
}

type DatasetExternalRetrievalModel struct {
	TopK                  int     `json:"top_k"`
	ScoreThreshold        float64 `json:"score_threshold"`
	ScoreThresholdEnabled bool    `json:"score_threshold_enabled"`
}

type DatasetSummaryIndexSetting struct {
	Enable            bool   `json:"enable"`
	ModelName         string `json:"model_name"`
	ModelProviderName string `json:"model_provider_name"`
	SummaryPrompt     string `json:"summary_prompt"`
}

type DatasetProcessRule struct {
	Mode                string                     `json:"mode"`
	Rules               DatasetProcessRuleSettings `json:"rules"`
	SummaryIndexSetting DatasetSummaryIndexSetting `json:"summary_index_setting"`
}

type DatasetProcessRuleSettings struct {
	PreProcessingRules   []DatasetPreProcessingRule `json:"pre_processing_rules"`
	Segmentation         DatasetSegmentation        `json:"segmentation"`
	ParentMode           string                     `json:"parent_mode"`
	SubchunkSegmentation DatasetSegmentation        `json:"subchunk_segmentation"`
}

type DatasetPreProcessingRule struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type DatasetSegmentation struct {
	Separator    string `json:"separator"`
	MaxTokens    int    `json:"max_tokens"`
	ChunkOverlap int    `json:"chunk_overlap"`
}

type DatasetDocument struct {
	ID                   string                      `json:"id"`
	Batch                string                      `json:"batch"`
	Position             int                         `json:"position"`
	DatasetID            string                      `json:"dataset_id"`
	DataSourceType       string                      `json:"data_source_type"`
	DataSourceInfo       map[string]any              `json:"data_source_info"`
	DatasetProcessRuleID string                      `json:"dataset_process_rule_id"`
	Name                 string                      `json:"name"`
	CreatedFrom          string                      `json:"created_from"`
	CreatedBy            string                      `json:"created_by"`
	CreatedAt            int64                       `json:"created_at"`
	IndexingStatus       string                      `json:"indexing_status"`
	DisplayStatus        string                      `json:"display_status"`
	CompletedSegments    int                         `json:"completed_segments"`
	TotalSegments        int                         `json:"total_segments"`
	DocForm              string                      `json:"doc_form"`
	DocLanguage          string                      `json:"doc_language"`
	SummaryIndexStatus   string                      `json:"summary_index_status"`
	Enabled              bool                        `json:"enabled"`
	WordCount            int                         `json:"word_count"`
	Error                string                      `json:"error"`
	Archived             bool                        `json:"archived"`
	UpdatedAt            int64                       `json:"updated_at"`
	HitCount             int                         `json:"hit_count"`
	DataSourceDetailDict map[string]any              `json:"data_source_detail_dict"`
	DocMetadata          map[string]string           `json:"doc_metadata"`
	MetadataValues       map[string]string           `json:"metadata_values"`
	DatasetProcessRule   DatasetProcessRule          `json:"dataset_process_rule"`
	DocumentProcessRule  DatasetProcessRule          `json:"document_process_rule"`
	CreatedAPIRequestID  string                      `json:"created_api_request_id"`
	ProcessingStartedAt  int64                       `json:"processing_started_at"`
	ParsingCompletedAt   int64                       `json:"parsing_completed_at"`
	CleaningCompletedAt  int64                       `json:"cleaning_completed_at"`
	SplittingCompletedAt int64                       `json:"splitting_completed_at"`
	Tokens               int                         `json:"tokens"`
	IndexingLatency      int64                       `json:"indexing_latency"`
	CompletedAt          int64                       `json:"completed_at"`
	PausedBy             string                      `json:"paused_by"`
	PausedAt             int64                       `json:"paused_at"`
	StoppedAt            int64                       `json:"stopped_at"`
	DisabledAt           int64                       `json:"disabled_at"`
	DisabledBy           string                      `json:"disabled_by"`
	ArchivedReason       string                      `json:"archived_reason"`
	ArchivedBy           string                      `json:"archived_by"`
	ArchivedAt           int64                       `json:"archived_at"`
	DocType              string                      `json:"doc_type"`
	SegmentCount         int                         `json:"segment_count"`
	Content              string                      `json:"content"`
	SignContent          string                      `json:"sign_content"`
	Keywords             []string                    `json:"keywords"`
	Summary              string                      `json:"summary"`
	Attachments          []DatasetAttachment         `json:"attachments"`
	ChildChunks          []DatasetChildChunk         `json:"child_chunks"`
	Segments             []DatasetSegment            `json:"segments"`
	PipelineExecutionLog DatasetPipelineExecutionLog `json:"pipeline_execution_log"`
}

type DatasetAttachment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	SourceURL string `json:"source_url"`
}

type DatasetChildChunk struct {
	ID        string  `json:"id"`
	SegmentID string  `json:"segment_id"`
	Content   string  `json:"content"`
	Position  int     `json:"position"`
	Score     float64 `json:"score"`
	WordCount int     `json:"word_count"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
	Type      string  `json:"type"`
}

type DatasetMetadataField struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type DatasetBatchImportJob struct {
	ID         string `json:"id"`
	DatasetID  string `json:"dataset_id"`
	DocumentID string `json:"document_id"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

type DatasetSegment struct {
	ID            string              `json:"id"`
	Position      int                 `json:"position"`
	DocumentID    string              `json:"document_id"`
	Content       string              `json:"content"`
	SignContent   string              `json:"sign_content"`
	WordCount     int                 `json:"word_count"`
	Tokens        int                 `json:"tokens"`
	Keywords      []string            `json:"keywords"`
	IndexNodeID   string              `json:"index_node_id"`
	IndexNodeHash string              `json:"index_node_hash"`
	HitCount      int                 `json:"hit_count"`
	Enabled       bool                `json:"enabled"`
	DisabledAt    int64               `json:"disabled_at"`
	DisabledBy    string              `json:"disabled_by"`
	Status        string              `json:"status"`
	CreatedBy     string              `json:"created_by"`
	CreatedAt     int64               `json:"created_at"`
	IndexingAt    int64               `json:"indexing_at"`
	CompletedAt   int64               `json:"completed_at"`
	Error         string              `json:"error"`
	StoppedAt     int64               `json:"stopped_at"`
	Answer        string              `json:"answer"`
	Summary       string              `json:"summary"`
	ChildChunks   []DatasetChildChunk `json:"child_chunks"`
	UpdatedAt     int64               `json:"updated_at"`
	Attachments   []DatasetAttachment `json:"attachments"`
}

type DatasetQueryRecord struct {
	ID            string             `json:"id"`
	Source        string             `json:"source"`
	SourceAppID   string             `json:"source_app_id"`
	CreatedByRole string             `json:"created_by_role"`
	CreatedBy     string             `json:"created_by"`
	CreatedAt     int64              `json:"created_at"`
	Queries       []DatasetQueryItem `json:"queries"`
}

type DatasetQueryItem struct {
	Content     string             `json:"content"`
	ContentType string             `json:"content_type"`
	FileInfo    *DatasetAttachment `json:"file_info,omitempty"`
}

type DatasetListFilters struct {
	Page    int
	Limit   int
	Keyword string
	IDs     []string
}

type DatasetPage struct {
	Page    int
	Limit   int
	Total   int
	HasMore bool
	Data    []Dataset
}

type DocumentListFilters struct {
	Page    int
	Limit   int
	Keyword string
	Sort    string
	Status  string
}

type DatasetDocumentPage struct {
	Page    int
	Limit   int
	Total   int
	HasMore bool
	Data    []DatasetDocument
}

type CreateDatasetInput struct {
	Name        string
	Description string
}

type CreateDatasetDocumentInput struct {
	DataSourceType         string
	DataSource             map[string]any
	DocForm                string
	DocLanguage            string
	IndexingTechnique      string
	RetrievalModel         DatasetRetrievalModel
	EmbeddingModel         string
	EmbeddingModelProvider string
	ProcessRule            DatasetProcessRule
	SummaryIndexSetting    DatasetSummaryIndexSetting
	CreatedFrom            string
}

type RelatedAppSummary struct {
	ID             string
	Name           string
	Mode           string
	IconType       string
	Icon           string
	IconBackground string
}

func normalizeDataset(dataset *Dataset) {
	dataset.Permission = firstNonEmpty(dataset.Permission, datasetPermissionAllTeamMembers)
	dataset.Provider = firstNonEmpty(dataset.Provider, datasetProviderLocal)
	dataset.RuntimeMode = firstNonEmpty(dataset.RuntimeMode, datasetRuntimeModeGeneral)
	dataset.IconInfo = normalizeDatasetIconInfo(dataset.IconInfo, dataset.Name)
	dataset.RetrievalModel = normalizeDatasetRetrievalModel(dataset.RetrievalModel)
	dataset.ExternalRetrievalModel = normalizeDatasetExternalRetrievalModel(dataset.ExternalRetrievalModel)
	if dataset.PartialMemberList == nil {
		dataset.PartialMemberList = []string{}
	}
	if dataset.MetadataFields == nil {
		dataset.MetadataFields = []DatasetMetadataField{}
	}
	for i := range dataset.MetadataFields {
		normalizeDatasetMetadataField(&dataset.MetadataFields[i])
	}
	if dataset.BatchImportJobs == nil {
		dataset.BatchImportJobs = []DatasetBatchImportJob{}
	}
	if dataset.Documents == nil {
		dataset.Documents = []DatasetDocument{}
	}
	if dataset.Queries == nil {
		dataset.Queries = []DatasetQueryRecord{}
	}
	for i := range dataset.Documents {
		normalizeDatasetDocument(&dataset.Documents[i])
	}
}

func normalizeDatasetDocument(document *DatasetDocument) {
	if document.IndexingStatus == "" {
		document.IndexingStatus = documentIndexingStatusCompleted
	}
	if document.DisplayStatus == "" {
		document.DisplayStatus = documentDisplayStatusAvailable
	}
	document.DatasetProcessRule = normalizeDatasetProcessRule(document.DatasetProcessRule)
	document.DocumentProcessRule = normalizeDatasetProcessRule(document.DocumentProcessRule)
	if document.DataSourceInfo == nil {
		document.DataSourceInfo = map[string]any{}
	}
	if document.DataSourceDetailDict == nil {
		document.DataSourceDetailDict = map[string]any{}
	}
	if document.DocMetadata == nil {
		document.DocMetadata = map[string]string{}
	}
	if document.MetadataValues == nil {
		document.MetadataValues = map[string]string{}
	}
	if document.Keywords == nil {
		document.Keywords = []string{}
	}
	if document.Attachments == nil {
		document.Attachments = []DatasetAttachment{}
	}
	if document.ChildChunks == nil {
		document.ChildChunks = []DatasetChildChunk{}
	}
	for i := range document.ChildChunks {
		normalizeDatasetChildChunk(&document.ChildChunks[i], document.ID)
	}
	if document.Segments == nil {
		document.Segments = []DatasetSegment{}
	}
	if len(document.Segments) == 0 && (document.Content != "" || len(document.ChildChunks) > 0 || len(document.Attachments) > 0) {
		document.Segments = []DatasetSegment{datasetSegmentFromDocument(*document)}
	}
	for i := range document.Segments {
		normalizeDatasetSegment(&document.Segments[i], document)
	}
	normalizeDatasetPipelineExecutionLog(&document.PipelineExecutionLog, *document)
	syncDatasetDocumentFromSegments(document)
}

func normalizeDatasetMetadataField(field *DatasetMetadataField) {
	field.Name = strings.TrimSpace(field.Name)
	switch strings.TrimSpace(field.Type) {
	case metadataTypeNumber, metadataTypeTime:
		field.Type = strings.TrimSpace(field.Type)
	default:
		field.Type = metadataTypeString
	}
}

func normalizeDatasetChildChunk(chunk *DatasetChildChunk, segmentID string) {
	chunk.SegmentID = firstNonEmpty(chunk.SegmentID, segmentID)
	chunk.Content = strings.TrimSpace(chunk.Content)
	if chunk.Position <= 0 {
		chunk.Position = 1
	}
	if chunk.WordCount <= 0 {
		chunk.WordCount = max(estimateWordCount(chunk.Content), 1)
	}
	if chunk.Type == "" {
		chunk.Type = childChunkTypeAutomatic
	}
}

func normalizeDatasetSegment(segment *DatasetSegment, document *DatasetDocument) {
	segment.DocumentID = firstNonEmpty(segment.DocumentID, document.ID)
	if segment.Position <= 0 {
		segment.Position = 1
	}
	if segment.SignContent == "" {
		segment.SignContent = segment.Content
	}
	if segment.WordCount <= 0 {
		segment.WordCount = max(estimateWordCount(segment.Content), 1)
	}
	if segment.Tokens <= 0 {
		segment.Tokens = estimateTokenCount(segment.Content)
	}
	if segment.Keywords == nil {
		segment.Keywords = []string{}
	}
	if segment.Attachments == nil {
		segment.Attachments = []DatasetAttachment{}
	}
	if segment.ChildChunks == nil {
		segment.ChildChunks = []DatasetChildChunk{}
	}
	for i := range segment.ChildChunks {
		normalizeDatasetChildChunk(&segment.ChildChunks[i], segment.ID)
	}
	if segment.Status == "" {
		segment.Status = segmentStatusCompleted
	}
	if segment.CreatedBy == "" {
		segment.CreatedBy = document.CreatedBy
	}
	if segment.CreatedAt == 0 {
		segment.CreatedAt = firstNonZeroInt64(document.CreatedAt, document.UpdatedAt)
	}
	if segment.IndexingAt == 0 {
		segment.IndexingAt = firstNonZeroInt64(document.ProcessingStartedAt, segment.CreatedAt)
	}
	if segment.CompletedAt == 0 && segment.Status == segmentStatusCompleted {
		segment.CompletedAt = firstNonZeroInt64(document.CompletedAt, document.UpdatedAt, segment.CreatedAt)
	}
	if segment.UpdatedAt == 0 {
		segment.UpdatedAt = firstNonZeroInt64(document.UpdatedAt, segment.CreatedAt)
	}
	if segment.IndexNodeID == "" {
		segment.IndexNodeID = generateID("node")
	}
	if segment.IndexNodeHash == "" {
		segment.IndexNodeHash = firstNonEmpty(segment.ID, document.ID)
	}
}

func normalizeDatasetProcessRule(rule DatasetProcessRule) DatasetProcessRule {
	if rule.Mode == "" {
		rule = defaultDatasetProcessRule()
	}
	if rule.Rules.PreProcessingRules == nil {
		rule.Rules.PreProcessingRules = defaultDatasetProcessRule().Rules.PreProcessingRules
	}
	if rule.Rules.Segmentation.Separator == "" {
		rule.Rules.Segmentation = defaultDatasetProcessRule().Rules.Segmentation
	}
	if rule.Rules.SubchunkSegmentation.Separator == "" {
		rule.Rules.SubchunkSegmentation = defaultDatasetProcessRule().Rules.SubchunkSegmentation
	}
	rule.Rules.ParentMode = firstNonEmpty(rule.Rules.ParentMode, "paragraph")
	return rule
}

func normalizeDatasetIconInfo(icon DatasetIconInfo, name string) DatasetIconInfo {
	icon.Icon = firstNonEmpty(icon.Icon, "📚")
	icon.IconType = firstNonEmpty(icon.IconType, "emoji")
	icon.IconBackground = firstNonEmpty(icon.IconBackground, "#FFF4ED")
	if icon.IconURL == "" && icon.IconType == "image" {
		icon.IconType = "emoji"
	}
	return icon
}

func normalizeDatasetRetrievalModel(model DatasetRetrievalModel) DatasetRetrievalModel {
	model.SearchMethod = firstNonEmpty(model.SearchMethod, "semantic_search")
	if model.TopK <= 0 {
		model.TopK = 4
	}
	if model.ScoreThreshold == 0 {
		model.ScoreThreshold = 0.5
	}
	return model
}

func normalizeDatasetExternalRetrievalModel(model DatasetExternalRetrievalModel) DatasetExternalRetrievalModel {
	if model.TopK <= 0 {
		model.TopK = 4
	}
	if model.ScoreThreshold == 0 {
		model.ScoreThreshold = 0.5
	}
	return model
}

func defaultDatasetProcessRule() DatasetProcessRule {
	return DatasetProcessRule{
		Mode: "custom",
		Rules: DatasetProcessRuleSettings{
			PreProcessingRules: []DatasetPreProcessingRule{
				{ID: "remove_extra_spaces", Enabled: false},
				{ID: "remove_urls_emails", Enabled: false},
			},
			Segmentation: DatasetSegmentation{
				Separator:    "\n\n",
				MaxTokens:    1000,
				ChunkOverlap: 50,
			},
			ParentMode: "paragraph",
			SubchunkSegmentation: DatasetSegmentation{
				Separator:    "\n",
				MaxTokens:    300,
				ChunkOverlap: 30,
			},
		},
	}
}

func DatasetProcessRuleLimits() map[string]any {
	return map[string]any{
		"indexing_max_segmentation_tokens_length": 4000,
	}
}

func (s *Store) ListDatasets(workspaceID string, filters DatasetListFilters) DatasetPage {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

	idSet := make(map[string]struct{}, len(filters.IDs))
	for _, id := range filters.IDs {
		if trimmed := strings.TrimSpace(id); trimmed != "" {
			idSet[trimmed] = struct{}{}
		}
	}

	items := make([]Dataset, 0)
	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[dataset.ID]; !ok {
				continue
			}
		}
		if filters.Keyword != "" && !strings.Contains(strings.ToLower(dataset.Name), strings.ToLower(strings.TrimSpace(filters.Keyword))) {
			continue
		}
		items = append(items, cloneDataset(dataset))
	}

	slices.SortFunc(items, func(a, b Dataset) int {
		if a.UpdatedAt == b.UpdatedAt {
			return bcmp(a.ID, b.ID)
		}
		if a.UpdatedAt > b.UpdatedAt {
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

	return DatasetPage{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: end < total,
		Data:    cloneDatasetList(items[start:end]),
	}
}

func (s *Store) GetDataset(datasetID, workspaceID string) (Dataset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.findDatasetIndexLocked(datasetID, workspaceID)
	if index < 0 {
		return Dataset{}, false
	}
	return cloneDataset(s.state.Datasets[index]), true
}

func (s *Store) CreateDataset(workspaceID string, user User, input CreateDatasetInput, now time.Time) (Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Dataset{}, fmt.Errorf("dataset name is required")
	}

	timestamp := now.UTC().Unix()
	dataset := Dataset{
		ID:                     generateID("dataset"),
		WorkspaceID:            workspaceID,
		Name:                   name,
		Description:            strings.TrimSpace(input.Description),
		Permission:             datasetPermissionAllTeamMembers,
		AuthorName:             ownerDisplayName(user),
		CreatedBy:              user.ID,
		UpdatedBy:              user.ID,
		CreatedAt:              timestamp,
		UpdatedAt:              timestamp,
		Provider:               datasetProviderLocal,
		EmbeddingAvailable:     true,
		IconInfo:               normalizeDatasetIconInfo(DatasetIconInfo{}, name),
		RetrievalModel:         normalizeDatasetRetrievalModel(DatasetRetrievalModel{}),
		ExternalRetrievalModel: normalizeDatasetExternalRetrievalModel(DatasetExternalRetrievalModel{}),
		BuiltInFieldEnabled:    false,
		PartialMemberList:      []string{},
		RuntimeMode:            datasetRuntimeModeGeneral,
		MetadataFields:         []DatasetMetadataField{},
		BatchImportJobs:        []DatasetBatchImportJob{},
		Documents:              []DatasetDocument{},
		Queries:                []DatasetQueryRecord{},
	}

	s.state.Datasets = append(s.state.Datasets, dataset)
	if err := s.saveLocked(); err != nil {
		return Dataset{}, err
	}
	return cloneDataset(dataset), nil
}

func (s *Store) PatchDataset(datasetID, workspaceID string, patch map[string]any, user User, now time.Time) (Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findDatasetIndexLocked(datasetID, workspaceID)
	if index < 0 {
		return Dataset{}, fmt.Errorf("dataset not found")
	}

	dataset := s.state.Datasets[index]
	if value, ok := patch["name"]; ok {
		if name, ok := value.(string); ok && strings.TrimSpace(name) != "" {
			dataset.Name = strings.TrimSpace(name)
		}
	}
	if value, ok := patch["description"]; ok {
		if description, ok := value.(string); ok {
			dataset.Description = strings.TrimSpace(description)
		}
	}
	if value, ok := patch["permission"]; ok {
		if permission, ok := value.(string); ok && strings.TrimSpace(permission) != "" {
			dataset.Permission = strings.TrimSpace(permission)
		}
	}
	if value, ok := patch["partial_member_list"]; ok {
		members := make([]string, 0)
		for _, item := range anySlice(value) {
			switch typed := item.(type) {
			case string:
				if trimmed := strings.TrimSpace(typed); trimmed != "" {
					members = append(members, trimmed)
				}
			case map[string]any:
				if userID := strings.TrimSpace(stringValue(typed["user_id"], "")); userID != "" {
					members = append(members, userID)
				}
			}
		}
		dataset.PartialMemberList = members
	}
	if value, ok := patch["indexing_technique"]; ok {
		if technique, ok := value.(string); ok {
			dataset.IndexingTechnique = strings.TrimSpace(technique)
		}
	}
	if value, ok := patch["embedding_model"]; ok {
		if embeddingModel, ok := value.(string); ok {
			dataset.EmbeddingModel = strings.TrimSpace(embeddingModel)
		}
	}
	if value, ok := patch["embedding_model_provider"]; ok {
		if embeddingProvider, ok := value.(string); ok {
			dataset.EmbeddingModelProvider = strings.TrimSpace(embeddingProvider)
		}
	}
	if value, ok := patch["doc_form"]; ok {
		if docForm, ok := value.(string); ok {
			dataset.DocForm = strings.TrimSpace(docForm)
		}
	}
	if value, ok := patch["runtime_mode"]; ok {
		if runtimeMode, ok := value.(string); ok && strings.TrimSpace(runtimeMode) != "" {
			dataset.RuntimeMode = strings.TrimSpace(runtimeMode)
		}
	}
	if value, ok := patch["provider"]; ok {
		if provider, ok := value.(string); ok && strings.TrimSpace(provider) != "" {
			dataset.Provider = strings.TrimSpace(provider)
		}
	}
	if value, ok := patch["is_published"]; ok {
		if published, ok := value.(bool); ok {
			dataset.IsPublished = published
		}
	}
	if value, ok := patch["enable_api"]; ok {
		if enabled, ok := value.(bool); ok {
			dataset.EnableAPI = enabled
		}
	}
	if value, ok := patch["is_multimodal"]; ok {
		if multimodal, ok := value.(bool); ok {
			dataset.IsMultimodal = multimodal
		}
	}
	if value, ok := patch["icon_info"]; ok {
		iconMap := mapStringAny(value)
		dataset.IconInfo = normalizeDatasetIconInfo(DatasetIconInfo{
			Icon:           stringValue(iconMap["icon"], ""),
			IconBackground: stringValue(iconMap["icon_background"], ""),
			IconType:       stringValue(iconMap["icon_type"], ""),
			IconURL:        stringValue(iconMap["icon_url"], ""),
		}, dataset.Name)
	}
	if value, ok := patch["retrieval_model"]; ok {
		dataset.RetrievalModel = parseDatasetRetrievalModel(value)
	}
	if value, ok := patch["retrieval_model_dict"]; ok {
		dataset.RetrievalModel = parseDatasetRetrievalModel(value)
	}
	if value, ok := patch["summary_index_setting"]; ok {
		dataset.SummaryIndexSetting = parseDatasetSummaryIndexSetting(value)
	}
	if value, ok := patch["external_knowledge_id"]; ok {
		if externalKnowledgeID, ok := value.(string); ok {
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeID = strings.TrimSpace(externalKnowledgeID)
		}
	}
	if value, ok := patch["external_knowledge_api_id"]; ok {
		if externalKnowledgeAPIID, ok := value.(string); ok {
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID = strings.TrimSpace(externalKnowledgeAPIID)
		}
	}
	if value, ok := patch["external_retrieval_model"]; ok {
		dataset.ExternalRetrievalModel = parseDatasetExternalRetrievalModel(value)
	}

	dataset.IconInfo = normalizeDatasetIconInfo(dataset.IconInfo, dataset.Name)
	dataset.RetrievalModel = normalizeDatasetRetrievalModel(dataset.RetrievalModel)
	s.hydrateDatasetExternalKnowledgeInfoLocked(&dataset)
	dataset.ExternalRetrievalModel = normalizeDatasetExternalRetrievalModel(dataset.ExternalRetrievalModel)
	dataset.UpdatedAt = now.UTC().Unix()
	dataset.UpdatedBy = user.ID

	s.state.Datasets[index] = dataset
	if err := s.saveLocked(); err != nil {
		return Dataset{}, err
	}
	return cloneDataset(dataset), nil
}

func (s *Store) DeleteDataset(datasetID, workspaceID string) (Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findDatasetIndexLocked(datasetID, workspaceID)
	if index < 0 {
		return Dataset{}, fmt.Errorf("dataset not found")
	}
	removed := cloneDataset(s.state.Datasets[index])
	s.state.Datasets = append(s.state.Datasets[:index], s.state.Datasets[index+1:]...)
	if err := s.saveLocked(); err != nil {
		return Dataset{}, err
	}
	return removed, nil
}

func (s *Store) SetDatasetAPIEnabled(datasetID, workspaceID string, enabled bool, user User, now time.Time) (Dataset, error) {
	return s.PatchDataset(datasetID, workspaceID, map[string]any{"enable_api": enabled}, user, now)
}

func (s *Store) ListDatasetDocuments(datasetID, workspaceID string, filters DocumentListFilters) (DatasetDocumentPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.findDatasetIndexLocked(datasetID, workspaceID)
	if index < 0 {
		return DatasetDocumentPage{}, false
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

	documents := make([]DatasetDocument, 0, len(s.state.Datasets[index].Documents))
	for _, document := range s.state.Datasets[index].Documents {
		if filters.Keyword != "" && !strings.Contains(strings.ToLower(document.Name), strings.ToLower(strings.TrimSpace(filters.Keyword))) {
			continue
		}
		if filters.Status != "" && filters.Status != "all" && !datasetDocumentMatchesStatus(document, filters.Status) {
			continue
		}
		documents = append(documents, cloneDatasetDocument(document))
	}

	slices.SortFunc(documents, func(a, b DatasetDocument) int {
		switch filters.Sort {
		case "hit_count":
			if a.HitCount == b.HitCount {
				return bcmp(a.ID, b.ID)
			}
			if a.HitCount < b.HitCount {
				return -1
			}
			return 1
		case "-hit_count":
			if a.HitCount == b.HitCount {
				return bcmp(a.ID, b.ID)
			}
			if a.HitCount > b.HitCount {
				return -1
			}
			return 1
		case "created_at":
			if a.CreatedAt == b.CreatedAt {
				return bcmp(a.ID, b.ID)
			}
			if a.CreatedAt < b.CreatedAt {
				return -1
			}
			return 1
		default:
			if a.CreatedAt == b.CreatedAt {
				return bcmp(a.ID, b.ID)
			}
			if a.CreatedAt > b.CreatedAt {
				return -1
			}
			return 1
		}
	})

	total := len(documents)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return DatasetDocumentPage{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: end < total,
		Data:    cloneDatasetDocumentList(documents[start:end]),
	}, true
}

func (s *Store) GetDatasetDocument(datasetID, workspaceID, documentID string) (DatasetDocument, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetDocument{}, false
	}

	for _, document := range s.state.Datasets[datasetIndex].Documents {
		if document.ID == documentID {
			return cloneDatasetDocument(document), true
		}
	}
	return DatasetDocument{}, false
}

func (s *Store) FindDatasetDocument(workspaceID, documentID string) (Dataset, DatasetDocument, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		for _, document := range dataset.Documents {
			if document.ID == documentID {
				return cloneDataset(dataset), cloneDatasetDocument(document), true
			}
		}
	}
	return Dataset{}, DatasetDocument{}, false
}

func (s *Store) CreateDatasetDocuments(datasetID, workspaceID string, user User, input CreateDatasetDocumentInput, now time.Time) (string, []DatasetDocument, Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return "", nil, Dataset{}, fmt.Errorf("dataset not found")
	}

	dataset := s.state.Datasets[datasetIndex]
	timestamp := now.UTC().Unix()
	batchID := generateID("batch")
	processRule := normalizeDatasetProcessRule(input.ProcessRule)
	if processRule.Mode == "" {
		processRule = defaultDatasetProcessRule()
	}
	retrievalModel := normalizeDatasetRetrievalModel(input.RetrievalModel)

	specs := datasetDocumentSpecs(input.DataSourceType, input.DataSource, user, timestamp, batchID)
	if len(specs) == 0 {
		specs = []datasetDocumentSpec{{
			Name:     firstNonEmpty(dataset.Name, "Knowledge Document"),
			Content:  "Imported document for " + firstNonEmpty(dataset.Name, "knowledge base"),
			Keywords: []string{"knowledge", "document"},
		}}
	}

	created := make([]DatasetDocument, 0, len(specs))
	for i, spec := range specs {
		content := firstNonEmpty(spec.Content, spec.Name)
		document := DatasetDocument{
			ID:                   generateID("doc"),
			Batch:                batchID,
			Position:             i + 1,
			DatasetID:            dataset.ID,
			DataSourceType:       firstNonEmpty(input.DataSourceType, dataset.DataSourceType, "upload_file"),
			DataSourceInfo:       cloneMap(spec.DataSourceInfo),
			DatasetProcessRuleID: generateID("rule"),
			Name:                 firstNonEmpty(spec.Name, fmt.Sprintf("Document %d", i+1)),
			CreatedFrom:          firstNonEmpty(input.CreatedFrom, "web"),
			CreatedBy:            user.ID,
			CreatedAt:            timestamp,
			IndexingStatus:       documentIndexingStatusCompleted,
			DisplayStatus:        documentDisplayStatusAvailable,
			CompletedSegments:    1,
			TotalSegments:        1,
			DocForm:              firstNonEmpty(input.DocForm, dataset.DocForm, "text_model"),
			DocLanguage:          firstNonEmpty(input.DocLanguage, "English"),
			SummaryIndexStatus:   "completed",
			Enabled:              true,
			WordCount:            max(estimateWordCount(content), 1),
			Error:                "",
			Archived:             false,
			UpdatedAt:            timestamp,
			HitCount:             0,
			DataSourceDetailDict: cloneMap(spec.DataSourceDetailDict),
			DocMetadata:          map[string]string{},
			MetadataValues:       map[string]string{},
			DatasetProcessRule:   processRule,
			DocumentProcessRule:  processRule,
			CreatedAPIRequestID:  generateID("req"),
			ProcessingStartedAt:  timestamp,
			ParsingCompletedAt:   timestamp,
			CleaningCompletedAt:  timestamp,
			SplittingCompletedAt: timestamp,
			Tokens:               estimateTokenCount(content),
			IndexingLatency:      1,
			CompletedAt:          timestamp,
			DocType:              "others",
			SegmentCount:         1,
			Content:              content,
			SignContent:          content,
			Keywords:             cloneStringSlice(spec.Keywords),
			Summary:              firstNonEmpty(spec.Summary, "Go migration compatible dataset document"),
			Attachments:          cloneDatasetAttachmentList(spec.Attachments),
			ChildChunks:          cloneDatasetChildChunkList(spec.ChildChunks),
			Segments:             []DatasetSegment{},
		}
		normalizeDatasetDocument(&document)
		dataset.Documents = append(dataset.Documents, document)
		created = append(created, cloneDatasetDocument(document))
	}

	if input.DataSourceType != "" {
		dataset.DataSourceType = input.DataSourceType
	}
	if input.IndexingTechnique != "" {
		dataset.IndexingTechnique = input.IndexingTechnique
	}
	if input.DocForm != "" {
		dataset.DocForm = input.DocForm
	}
	if input.EmbeddingModel != "" {
		dataset.EmbeddingModel = input.EmbeddingModel
	}
	if input.EmbeddingModelProvider != "" {
		dataset.EmbeddingModelProvider = input.EmbeddingModelProvider
	}
	dataset.RetrievalModel = retrievalModel
	dataset.SummaryIndexSetting = input.SummaryIndexSetting
	dataset.AuthorName = ownerDisplayName(user)
	dataset.UpdatedAt = timestamp
	dataset.UpdatedBy = user.ID
	s.state.Datasets[datasetIndex] = dataset
	if err := s.saveLocked(); err != nil {
		return "", nil, Dataset{}, err
	}
	return batchID, created, cloneDataset(dataset), nil
}

func (s *Store) RenameDatasetDocument(datasetID, workspaceID, documentID, name string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	for i := range s.state.Datasets[datasetIndex].Documents {
		if s.state.Datasets[datasetIndex].Documents[i].ID != documentID {
			continue
		}
		s.state.Datasets[datasetIndex].Documents[i].Name = firstNonEmpty(name, s.state.Datasets[datasetIndex].Documents[i].Name)
		s.state.Datasets[datasetIndex].Documents[i].UpdatedAt = now.UTC().Unix()
		s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
		s.state.Datasets[datasetIndex].UpdatedBy = user.ID
		return s.saveLocked()
	}
	return fmt.Errorf("document not found")
}

func (s *Store) SetDatasetDocumentProcessing(datasetID, workspaceID, documentID string, paused bool, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	for i := range s.state.Datasets[datasetIndex].Documents {
		document := &s.state.Datasets[datasetIndex].Documents[i]
		if document.ID != documentID {
			continue
		}
		timestamp := now.UTC().Unix()
		if paused {
			document.IndexingStatus = documentIndexingStatusPaused
			document.DisplayStatus = documentDisplayStatusPaused
			document.PausedBy = user.ID
			document.PausedAt = timestamp
		} else {
			document.IndexingStatus = documentIndexingStatusCompleted
			document.DisplayStatus = documentDisplayStatusAvailable
			document.PausedBy = ""
			document.PausedAt = 0
			document.CompletedAt = timestamp
		}
		document.UpdatedAt = timestamp
		s.state.Datasets[datasetIndex].UpdatedAt = timestamp
		s.state.Datasets[datasetIndex].UpdatedBy = user.ID
		return s.saveLocked()
	}
	return fmt.Errorf("document not found")
}

func (s *Store) ApplyDatasetDocumentAction(datasetID, workspaceID string, documentIDs []string, action string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	if len(documentIDs) == 0 {
		return fmt.Errorf("document ids are required")
	}

	targets := make(map[string]struct{}, len(documentIDs))
	for _, documentID := range documentIDs {
		if trimmed := strings.TrimSpace(documentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("document ids are required")
	}

	timestamp := now.UTC().Unix()
	updated := false
	for i := range s.state.Datasets[datasetIndex].Documents {
		document := &s.state.Datasets[datasetIndex].Documents[i]
		if _, ok := targets[document.ID]; !ok {
			continue
		}
		switch action {
		case "enable":
			document.Enabled = true
			document.DisplayStatus = documentDisplayStatusEnabled
			document.DisabledAt = 0
			document.DisabledBy = ""
		case "disable":
			document.Enabled = false
			document.DisplayStatus = documentDisplayStatusDisabled
			document.DisabledAt = timestamp
			document.DisabledBy = user.ID
		case "archive", "un_archive":
			document.Archived = action == "archive"
			if document.Archived {
				document.DisplayStatus = documentDisplayStatusArchived
				document.ArchivedAt = timestamp
				document.ArchivedBy = user.ID
				document.ArchivedReason = "rule_modified"
			} else {
				document.DisplayStatus = documentDisplayStatusAvailable
				document.ArchivedAt = 0
				document.ArchivedBy = ""
				document.ArchivedReason = ""
			}
		default:
			return fmt.Errorf("unsupported document action")
		}
		document.UpdatedAt = timestamp
		updated = true
	}
	if !updated {
		return fmt.Errorf("document not found")
	}
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) DeleteDatasetDocuments(datasetID, workspaceID string, documentIDs []string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}

	targets := make(map[string]struct{}, len(documentIDs))
	for _, documentID := range documentIDs {
		if trimmed := strings.TrimSpace(documentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("document ids are required")
	}

	filtered := s.state.Datasets[datasetIndex].Documents[:0]
	removed := false
	for _, document := range s.state.Datasets[datasetIndex].Documents {
		if _, ok := targets[document.ID]; ok {
			removed = true
			continue
		}
		filtered = append(filtered, document)
	}
	if !removed {
		return fmt.Errorf("document not found")
	}

	s.state.Datasets[datasetIndex].Documents = filtered
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) RetryDatasetDocuments(datasetID, workspaceID string, documentIDs []string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}

	targets := make(map[string]struct{}, len(documentIDs))
	for _, documentID := range documentIDs {
		if trimmed := strings.TrimSpace(documentID); trimmed != "" {
			targets[trimmed] = struct{}{}
		}
	}

	timestamp := now.UTC().Unix()
	updated := false
	for i := range s.state.Datasets[datasetIndex].Documents {
		document := &s.state.Datasets[datasetIndex].Documents[i]
		if len(targets) > 0 {
			if _, ok := targets[document.ID]; !ok {
				continue
			}
		} else if document.IndexingStatus != documentIndexingStatusError {
			continue
		}

		document.IndexingStatus = documentIndexingStatusCompleted
		document.DisplayStatus = documentDisplayStatusAvailable
		document.Error = ""
		document.CompletedAt = timestamp
		document.UpdatedAt = timestamp
		updated = true
	}
	if !updated {
		return fmt.Errorf("document not found")
	}
	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) ListDatasetBatchDocuments(datasetID, workspaceID, batchID string) ([]DatasetDocument, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return nil, false
	}
	items := make([]DatasetDocument, 0)
	for _, document := range s.state.Datasets[datasetIndex].Documents {
		if document.Batch == batchID {
			items = append(items, cloneDatasetDocument(document))
		}
	}
	return items, true
}

func (s *Store) AddDatasetQueryRecord(datasetID, workspaceID string, record DatasetQueryRecord, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return fmt.Errorf("dataset not found")
	}
	if record.ID == "" {
		record.ID = generateID("query")
	}
	if record.CreatedAt == 0 {
		record.CreatedAt = now.UTC().Unix()
	}
	if record.CreatedBy == "" {
		record.CreatedBy = user.ID
	}
	if record.CreatedByRole == "" {
		record.CreatedByRole = "account"
	}
	if record.Source == "" {
		record.Source = "hit_testing"
	}
	if record.Queries == nil {
		record.Queries = []DatasetQueryItem{}
	}
	s.state.Datasets[datasetIndex].Queries = append([]DatasetQueryRecord{record}, s.state.Datasets[datasetIndex].Queries...)
	s.state.Datasets[datasetIndex].UpdatedAt = now.UTC().Unix()
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	return s.saveLocked()
}

func (s *Store) ListDatasetQueries(datasetID, workspaceID string, page, limit int) ([]DatasetQueryRecord, int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return nil, 0, false
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	total := len(s.state.Datasets[datasetIndex].Queries)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	items := cloneDatasetQueryRecordList(s.state.Datasets[datasetIndex].Queries[start:end])
	return items, total, true
}

func (s *Store) ListDatasetErrorDocuments(datasetID, workspaceID string) ([]DatasetDocument, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return nil, false
	}
	items := make([]DatasetDocument, 0)
	for _, document := range s.state.Datasets[datasetIndex].Documents {
		if document.IndexingStatus == documentIndexingStatusError {
			items = append(items, cloneDatasetDocument(document))
		}
	}
	return items, true
}

func (s *Store) DatasetUseCount(workspaceID, datasetID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, app := range s.state.Apps {
		if app.WorkspaceID != workspaceID {
			continue
		}
		if appUsesDataset(app, datasetID) {
			count++
		}
	}
	return count
}

func (s *Store) DatasetRelatedApps(workspaceID, datasetID string) []RelatedAppSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]RelatedAppSummary, 0)
	for _, app := range s.state.Apps {
		if app.WorkspaceID != workspaceID {
			continue
		}
		if !appUsesDataset(app, datasetID) {
			continue
		}
		items = append(items, RelatedAppSummary{
			ID:             app.ID,
			Name:           app.Name,
			Mode:           app.Mode,
			IconType:       app.IconType,
			Icon:           app.Icon,
			IconBackground: app.IconBackground,
		})
	}
	return items
}

func (s *Store) findDatasetIndexLocked(datasetID, workspaceID string) int {
	for i, dataset := range s.state.Datasets {
		if dataset.ID == datasetID && dataset.WorkspaceID == workspaceID {
			return i
		}
	}
	return -1
}

func datasetDocumentMatchesStatus(document DatasetDocument, status string) bool {
	status = strings.TrimSpace(status)
	if status == "" || status == "all" {
		return true
	}
	if status == document.DisplayStatus || status == document.IndexingStatus {
		return true
	}
	switch status {
	case "available":
		return document.Enabled && !document.Archived && document.IndexingStatus == documentIndexingStatusCompleted
	case "enabled":
		return document.Enabled
	case "disabled":
		return !document.Enabled
	case "archived":
		return document.Archived
	case "paused":
		return document.IndexingStatus == documentIndexingStatusPaused
	case "error":
		return document.IndexingStatus == documentIndexingStatusError
	case "indexing", "queuing":
		return document.IndexingStatus != documentIndexingStatusCompleted && document.IndexingStatus != documentIndexingStatusError && document.IndexingStatus != documentIndexingStatusPaused
	default:
		return false
	}
}

func appUsesDataset(app App, datasetID string) bool {
	if valueContainsString(app.ModelConfig, datasetID) {
		return true
	}
	if app.WorkflowDraft != nil && valueContainsString(app.WorkflowDraft.Graph, datasetID) {
		return true
	}
	if app.WorkflowPublished != nil && valueContainsString(app.WorkflowPublished.Graph, datasetID) {
		return true
	}
	for _, version := range app.WorkflowVersions {
		if valueContainsString(version.Graph, datasetID) {
			return true
		}
	}
	return false
}

func valueContainsString(value any, target string) bool {
	switch typed := value.(type) {
	case string:
		return typed == target
	case []any:
		for _, item := range typed {
			if valueContainsString(item, target) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if valueContainsString(item, target) {
				return true
			}
		}
	}
	return false
}

type datasetDocumentSpec struct {
	Name                 string
	Content              string
	Summary              string
	Keywords             []string
	DataSourceInfo       map[string]any
	DataSourceDetailDict map[string]any
	Attachments          []DatasetAttachment
	ChildChunks          []DatasetChildChunk
}

func datasetDocumentSpecs(dataSourceType string, dataSource map[string]any, user User, timestamp int64, batchID string) []datasetDocumentSpec {
	infoList := mapStringAny(dataSource["info_list"])
	switch strings.TrimSpace(dataSourceType) {
	case "notion_import":
		return notionDocumentSpecs(infoList, user, batchID)
	case "website_crawl":
		return websiteDocumentSpecs(infoList)
	default:
		return fileDocumentSpecs(infoList, user, timestamp, batchID)
	}
}

func fileDocumentSpecs(infoList map[string]any, user User, timestamp int64, batchID string) []datasetDocumentSpec {
	fileInfoList := mapStringAny(infoList["file_info_list"])
	fileIDs := stringSliceFromAny(fileInfoList["file_ids"])
	specs := make([]datasetDocumentSpec, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		name := firstNonEmpty(strings.TrimSpace(fileID), "upload-file")
		extension := datasetFileExtension(name)
		content := "Imported file " + name + " through dify-go dataset compatibility layer."
		specs = append(specs, datasetDocumentSpec{
			Name:     name,
			Content:  content,
			Summary:  "Compatible imported file document",
			Keywords: datasetDocumentKeywords(name, content),
			DataSourceInfo: map[string]any{
				"upload_file": map[string]any{
					"id":         fileID,
					"name":       name,
					"size":       0,
					"mime_type":  datasetMimeType(extension),
					"created_at": timestamp,
					"created_by": user.ID,
					"extension":  extension,
				},
				"job_id": batchID,
				"url":    "",
			},
			DataSourceDetailDict: map[string]any{
				"upload_file": map[string]any{
					"name":      name,
					"extension": extension,
				},
			},
			Attachments: []DatasetAttachment{{
				ID:        generateID("att"),
				Name:      name,
				Extension: extension,
				MimeType:  datasetMimeType(extension),
				SourceURL: "",
			}},
		})
	}
	return specs
}

func notionDocumentSpecs(infoList map[string]any, user User, batchID string) []datasetDocumentSpec {
	items := anySlice(infoList["notion_info_list"])
	specs := make([]datasetDocumentSpec, 0)
	for _, item := range items {
		info := mapStringAny(item)
		credentialID := stringValue(info["credential_id"], "")
		workspaceID := stringValue(info["workspace_id"], "")
		for _, pageItem := range anySlice(info["pages"]) {
			pageMap := mapStringAny(pageItem)
			name := firstNonEmpty(stringValue(pageMap["page_name"], ""), "Notion Page")
			pageID := stringValue(pageMap["page_id"], "")
			pageType := firstNonEmpty(stringValue(pageMap["type"], ""), "page")
			content := "Imported notion page " + name + " through dify-go dataset compatibility layer."
			specs = append(specs, datasetDocumentSpec{
				Name:     name,
				Content:  content,
				Summary:  "Compatible imported notion document",
				Keywords: datasetDocumentKeywords(name, content),
				DataSourceInfo: map[string]any{
					"credential_id": credentialID,
					"workspace_id":  workspaceID,
					"page": map[string]any{
						"page_id":          pageID,
						"page_name":        name,
						"type":             pageType,
						"parent_id":        "",
						"page_icon":        stringValue(pageMap["page_icon"], ""),
						"last_edited_time": "",
					},
					"job_id": batchID,
				},
			})
		}
	}
	return specs
}

func websiteDocumentSpecs(infoList map[string]any) []datasetDocumentSpec {
	info := mapStringAny(infoList["website_info_list"])
	urls := stringSliceFromAny(info["urls"])
	provider := stringValue(info["provider"], "")
	jobID := stringValue(info["job_id"], "")
	specs := make([]datasetDocumentSpec, 0, len(urls))
	for _, rawURL := range urls {
		title := datasetDocumentNameFromURL(rawURL)
		content := "Indexed website content from " + rawURL + " through dify-go dataset compatibility layer."
		specs = append(specs, datasetDocumentSpec{
			Name:     title,
			Content:  content,
			Summary:  "Compatible indexed website document",
			Keywords: datasetDocumentKeywords(title, content),
			DataSourceInfo: map[string]any{
				"title":       title,
				"source_url":  rawURL,
				"description": "",
				"content":     content,
				"provider":    provider,
				"job_id":      jobID,
			},
		})
	}
	return specs
}

func datasetDocumentNameFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return firstNonEmpty(strings.TrimSpace(rawURL), "Website Document")
	}
	name := path.Base(strings.TrimSpace(parsed.Path))
	if name == "." || name == "/" || name == "" {
		name = parsed.Host
	}
	return firstNonEmpty(name, "Website Document")
}

func datasetFileExtension(name string) string {
	parts := strings.Split(strings.TrimSpace(name), ".")
	if len(parts) <= 1 {
		return "txt"
	}
	return strings.ToLower(parts[len(parts)-1])
}

func datasetMimeType(extension string) string {
	switch strings.ToLower(strings.TrimPrefix(extension, ".")) {
	case "pdf":
		return "application/pdf"
	case "md":
		return "text/markdown"
	case "html":
		return "text/html"
	default:
		return "text/plain"
	}
}

func datasetDocumentKeywords(name, content string) []string {
	tokens := tokenize(name + " " + content)
	seen := map[string]struct{}{}
	out := make([]string, 0, 5)
	for _, token := range tokens {
		if len(token) < 3 {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func tokenize(input string) []string {
	input = strings.ToLower(input)
	return strings.FieldsFunc(input, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

func estimateWordCount(content string) int {
	return len(strings.Fields(content))
}

func estimateTokenCount(content string) int {
	words := estimateWordCount(content)
	if words == 0 {
		return 0
	}
	return words * 2
}

func mapStringAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	default:
		return map[string]any{}
	}
}

func anySlice(value any) []any {
	if value == nil {
		return []any{}
	}
	if items, ok := value.([]any); ok {
		return cloneSlice(items)
	}
	return []any{}
}

func stringSliceFromAny(value any) []string {
	items := anySlice(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return out
}

func parseDatasetRetrievalModel(value any) DatasetRetrievalModel {
	model := normalizeDatasetRetrievalModel(DatasetRetrievalModel{})
	item := mapStringAny(value)
	if len(item) == 0 {
		return model
	}
	model.SearchMethod = firstNonEmpty(stringValue(item["search_method"], ""), model.SearchMethod)
	if topK, ok := item["top_k"].(float64); ok && int(topK) > 0 {
		model.TopK = int(topK)
	}
	if threshold, ok := item["score_threshold"].(float64); ok {
		model.ScoreThreshold = threshold
	}
	if enabled, ok := item["score_threshold_enabled"].(bool); ok {
		model.ScoreThresholdEnabled = enabled
	}
	if enabled, ok := item["reranking_enable"].(bool); ok {
		model.RerankingEnable = enabled
	}
	model.RerankingMode = stringValue(item["reranking_mode"], "")
	model.Weights = cloneMap(mapStringAny(item["weights"]))
	rerankingModel := mapStringAny(item["reranking_model"])
	model.RerankingModel = DatasetProviderModel{
		ProviderName: stringValue(rerankingModel["reranking_provider_name"], ""),
		ModelName:    stringValue(rerankingModel["reranking_model_name"], ""),
	}
	return normalizeDatasetRetrievalModel(model)
}

func parseDatasetSummaryIndexSetting(value any) DatasetSummaryIndexSetting {
	item := mapStringAny(value)
	if len(item) == 0 {
		return DatasetSummaryIndexSetting{}
	}
	setting := DatasetSummaryIndexSetting{
		ModelName:         stringValue(item["model_name"], ""),
		ModelProviderName: stringValue(item["model_provider_name"], ""),
		SummaryPrompt:     stringValue(item["summary_prompt"], ""),
	}
	if enabled, ok := item["enable"].(bool); ok {
		setting.Enable = enabled
	}
	return setting
}

func parseDatasetExternalRetrievalModel(value any) DatasetExternalRetrievalModel {
	model := normalizeDatasetExternalRetrievalModel(DatasetExternalRetrievalModel{})
	item := mapStringAny(value)
	if len(item) == 0 {
		return model
	}
	if topK, ok := item["top_k"].(float64); ok && int(topK) > 0 {
		model.TopK = int(topK)
	}
	if threshold, ok := item["score_threshold"].(float64); ok {
		model.ScoreThreshold = threshold
	}
	if enabled, ok := item["score_threshold_enabled"].(bool); ok {
		model.ScoreThresholdEnabled = enabled
	}
	return normalizeDatasetExternalRetrievalModel(model)
}

func cloneDataset(src Dataset) Dataset {
	data, err := json.Marshal(src)
	if err != nil {
		return Dataset{}
	}
	var out Dataset
	if err := json.Unmarshal(data, &out); err != nil {
		return Dataset{}
	}
	normalizeDataset(&out)
	return out
}

func cloneDatasetList(src []Dataset) []Dataset {
	out := make([]Dataset, len(src))
	for i, item := range src {
		out[i] = cloneDataset(item)
	}
	return out
}

func cloneDatasetDocument(src DatasetDocument) DatasetDocument {
	data, err := json.Marshal(src)
	if err != nil {
		return DatasetDocument{}
	}
	var out DatasetDocument
	if err := json.Unmarshal(data, &out); err != nil {
		return DatasetDocument{}
	}
	normalizeDatasetDocument(&out)
	return out
}

func cloneDatasetDocumentList(src []DatasetDocument) []DatasetDocument {
	out := make([]DatasetDocument, len(src))
	for i, item := range src {
		out[i] = cloneDatasetDocument(item)
	}
	return out
}

func cloneDatasetAttachmentList(src []DatasetAttachment) []DatasetAttachment {
	if src == nil {
		return []DatasetAttachment{}
	}
	out := make([]DatasetAttachment, len(src))
	copy(out, src)
	return out
}

func cloneDatasetChildChunkList(src []DatasetChildChunk) []DatasetChildChunk {
	if src == nil {
		return []DatasetChildChunk{}
	}
	out := make([]DatasetChildChunk, len(src))
	copy(out, src)
	return out
}

func cloneDatasetQueryRecordList(src []DatasetQueryRecord) []DatasetQueryRecord {
	if src == nil {
		return []DatasetQueryRecord{}
	}
	data, err := json.Marshal(src)
	if err != nil {
		return []DatasetQueryRecord{}
	}
	out := []DatasetQueryRecord{}
	if err := json.Unmarshal(data, &out); err != nil {
		return []DatasetQueryRecord{}
	}
	return out
}

func ownerDisplayName(user User) string {
	return firstNonEmpty(user.Name, user.Email, "Anonymous")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
