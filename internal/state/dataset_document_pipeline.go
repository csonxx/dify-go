package state

import (
	"fmt"
	"strings"
	"time"
)

const (
	pipelineDatasourceLocalFile      = "local_file"
	pipelineDatasourceOnlineDocument = "online_document"
	pipelineDatasourceWebsiteCrawl   = "website_crawl"
	pipelineDatasourceOnlineDrive    = "online_drive"
)

type DatasetPipelineExecutionLog struct {
	DatasourceType   string         `json:"datasource_type"`
	DatasourceInfo   map[string]any `json:"datasource_info"`
	InputData        map[string]any `json:"input_data"`
	DatasourceNodeID string         `json:"datasource_node_id"`
	LastSyncedAt     int64          `json:"last_synced_at"`
}

func normalizeDatasetPipelineExecutionLog(log *DatasetPipelineExecutionLog, document DatasetDocument) {
	if log.DatasourceType == "" {
		log.DatasourceType = pipelineDatasourceTypeForDocument(document.DataSourceType)
	}
	if log.DatasourceInfo == nil || len(log.DatasourceInfo) == 0 {
		log.DatasourceInfo = datasetPipelineDatasourceInfo(document)
	} else {
		log.DatasourceInfo = cloneMap(log.DatasourceInfo)
	}
	if log.InputData == nil || len(log.InputData) == 0 {
		log.InputData = datasetPipelineInputData(document)
	} else {
		log.InputData = cloneMap(log.InputData)
	}
	if log.DatasourceNodeID == "" {
		log.DatasourceNodeID = datasetPipelineDatasourceNodeID(log.DatasourceType)
	}
}

func (s *Store) GetDatasetDocumentPipelineExecutionLog(datasetID, workspaceID, documentID string) (DatasetPipelineExecutionLog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetPipelineExecutionLog{}, false
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetPipelineExecutionLog{}, false
	}

	log := cloneDatasetPipelineExecutionLog(s.state.Datasets[datasetIndex].Documents[documentIndex].PipelineExecutionLog)
	normalizeDatasetPipelineExecutionLog(&log, s.state.Datasets[datasetIndex].Documents[documentIndex])
	return log, true
}

func (s *Store) SyncDatasetDocumentSource(datasetID, workspaceID, documentID string, user User, now time.Time) (DatasetDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	datasetIndex := s.findDatasetIndexLocked(datasetID, workspaceID)
	if datasetIndex < 0 {
		return DatasetDocument{}, fmt.Errorf("dataset not found")
	}
	documentIndex := findDatasetDocumentIndexLocked(&s.state.Datasets[datasetIndex], documentID)
	if documentIndex < 0 {
		return DatasetDocument{}, fmt.Errorf("document not found")
	}

	document := &s.state.Datasets[datasetIndex].Documents[documentIndex]
	switch strings.TrimSpace(document.DataSourceType) {
	case "notion_import", "website_crawl":
	default:
		return DatasetDocument{}, fmt.Errorf("document source does not support sync")
	}

	timestamp := now.UTC().Unix()
	document.IndexingStatus = documentIndexingStatusCompleted
	document.DisplayStatus = documentDisplayStatusAvailable
	document.Error = ""
	document.ProcessingStartedAt = timestamp
	document.ParsingCompletedAt = timestamp
	document.CleaningCompletedAt = timestamp
	document.SplittingCompletedAt = timestamp
	document.CompletedAt = timestamp
	document.UpdatedAt = timestamp
	document.PausedAt = 0
	document.PausedBy = ""
	document.StoppedAt = 0
	document.PipelineExecutionLog = cloneDatasetPipelineExecutionLog(document.PipelineExecutionLog)
	document.PipelineExecutionLog.LastSyncedAt = timestamp
	normalizeDatasetPipelineExecutionLog(&document.PipelineExecutionLog, *document)

	s.state.Datasets[datasetIndex].UpdatedAt = timestamp
	s.state.Datasets[datasetIndex].UpdatedBy = user.ID
	if err := s.saveLocked(); err != nil {
		return DatasetDocument{}, err
	}
	return cloneDatasetDocument(*document), nil
}

func pipelineDatasourceTypeForDocument(dataSourceType string) string {
	switch strings.TrimSpace(dataSourceType) {
	case "notion_import":
		return pipelineDatasourceOnlineDocument
	case "website_crawl":
		return pipelineDatasourceWebsiteCrawl
	case pipelineDatasourceOnlineDrive:
		return pipelineDatasourceOnlineDrive
	case pipelineDatasourceLocalFile, "upload_file":
		fallthrough
	default:
		return pipelineDatasourceLocalFile
	}
}

func datasetPipelineDatasourceInfo(document DatasetDocument) map[string]any {
	switch pipelineDatasourceTypeForDocument(document.DataSourceType) {
	case pipelineDatasourceOnlineDocument:
		return map[string]any{
			"credential_id": stringValue(document.DataSourceInfo["credential_id"], ""),
			"workspace_id":  stringValue(document.DataSourceInfo["workspace_id"], ""),
			"page":          cloneMap(mapStringAny(document.DataSourceInfo["page"])),
		}
	case pipelineDatasourceWebsiteCrawl:
		return map[string]any{
			"content":     firstNonEmpty(stringValue(document.DataSourceInfo["content"], ""), document.Content),
			"description": stringValue(document.DataSourceInfo["description"], ""),
			"source_url":  stringValue(document.DataSourceInfo["source_url"], ""),
			"title":       firstNonEmpty(stringValue(document.DataSourceInfo["title"], ""), document.Name),
		}
	case pipelineDatasourceOnlineDrive:
		return map[string]any{
			"id":   stringValue(document.DataSourceInfo["id"], ""),
			"type": stringValue(document.DataSourceInfo["type"], ""),
			"name": firstNonEmpty(stringValue(document.DataSourceInfo["name"], ""), document.Name),
			"size": document.DataSourceInfo["size"],
		}
	default:
		uploadFile := mapStringAny(document.DataSourceInfo["upload_file"])
		if len(uploadFile) > 0 {
			return map[string]any{
				"related_id": firstNonEmpty(stringValue(uploadFile["id"], ""), document.ID),
				"name":       firstNonEmpty(stringValue(uploadFile["name"], ""), document.Name),
				"extension":  firstNonEmpty(stringValue(uploadFile["extension"], ""), datasetFileExtension(document.Name)),
				"size":       uploadFile["size"],
				"mime_type":  stringValue(uploadFile["mime_type"], ""),
			}
		}
		if uploadFileID := stringValue(document.DataSourceInfo["upload_file_id"], ""); uploadFileID != "" {
			return map[string]any{
				"related_id": uploadFileID,
				"name":       firstNonEmpty(document.Name, "upload-file"),
				"extension":  datasetFileExtension(document.Name),
			}
		}
		return map[string]any{
			"related_id": document.ID,
			"name":       firstNonEmpty(document.Name, "document"),
			"extension":  datasetFileExtension(document.Name),
		}
	}
}

func datasetPipelineInputData(document DatasetDocument) map[string]any {
	rule := normalizeDatasetProcessRule(document.DocumentProcessRule)
	return map[string]any{
		"separator":              rule.Rules.Segmentation.Separator,
		"chunk_size":             rule.Rules.Segmentation.MaxTokens,
		"chunk_overlap":          rule.Rules.Segmentation.ChunkOverlap,
		"subchunk_separator":     rule.Rules.SubchunkSegmentation.Separator,
		"subchunk_chunk_size":    rule.Rules.SubchunkSegmentation.MaxTokens,
		"subchunk_chunk_overlap": rule.Rules.SubchunkSegmentation.ChunkOverlap,
		"parent_mode":            rule.Rules.ParentMode,
		"doc_language":           document.DocLanguage,
		"doc_form":               document.DocForm,
		"summary_index_enabled":  document.SummaryIndexStatus == "completed",
		"remove_extra_spaces":    datasetPreProcessingEnabled(rule, "remove_extra_spaces"),
		"remove_urls_emails":     datasetPreProcessingEnabled(rule, "remove_urls_emails"),
	}
}

func datasetPreProcessingEnabled(rule DatasetProcessRule, id string) bool {
	for _, item := range rule.Rules.PreProcessingRules {
		if item.ID == id {
			return item.Enabled
		}
	}
	return false
}

func datasetPipelineDatasourceNodeID(datasourceType string) string {
	switch datasourceType {
	case pipelineDatasourceOnlineDocument:
		return "datasource-online-document"
	case pipelineDatasourceWebsiteCrawl:
		return "datasource-website-crawl"
	case pipelineDatasourceOnlineDrive:
		return "datasource-online-drive"
	default:
		return "datasource-local-file"
	}
}

func cloneDatasetPipelineExecutionLog(src DatasetPipelineExecutionLog) DatasetPipelineExecutionLog {
	out := DatasetPipelineExecutionLog{
		DatasourceType:   src.DatasourceType,
		DatasourceInfo:   cloneMap(src.DatasourceInfo),
		InputData:        cloneMap(src.InputData),
		DatasourceNodeID: src.DatasourceNodeID,
		LastSyncedAt:     src.LastSyncedAt,
	}
	return out
}
