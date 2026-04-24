package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/langgenius/dify-go/internal/config"
)

type serverTestEnv struct {
	t         *testing.T
	server    *httptest.Server
	client    *http.Client
	stateFile string
}

type uploadResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	MimeType   string `json:"mime_type"`
	URL        string `json:"url"`
	SourceURL  string `json:"source_url"`
	PreviewURL string `json:"preview_url"`
}

type datasetCreateResponse struct {
	ID string `json:"id"`
}

type publicWebAppInfoResponse struct {
	AppID          string         `json:"app_id"`
	CanReplaceLogo bool           `json:"can_replace_logo"`
	CustomConfig   map[string]any `json:"custom_config"`
	EnableSite     bool           `json:"enable_site"`
	EndUserID      any            `json:"end_user_id"`
	Site           map[string]any `json:"site"`
}

type ragPipelineDatasetResponse struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	ID                string `json:"id"`
	PipelineID        string `json:"pipeline_id"`
	RuntimeMode       string `json:"runtime_mode"`
	Permission        string `json:"permission"`
	DocForm           string `json:"doc_form"`
	IndexingTechnique string `json:"indexing_technique"`
	EmbeddingModel    string `json:"embedding_model"`
	EmbeddingProvider string `json:"embedding_model_provider"`
	IsPublished       bool   `json:"is_published"`
}

type workflowDraftResponse struct {
	Graph                 map[string]any   `json:"graph"`
	Features              map[string]any   `json:"features"`
	ID                    string           `json:"id"`
	Hash                  string           `json:"hash"`
	EnvironmentVariables  []map[string]any `json:"environment_variables"`
	ConversationVariables []map[string]any `json:"conversation_variables"`
	RagPipelineVariables  []map[string]any `json:"rag_pipeline_variables"`
}

type workflowSyncResponse struct {
	Result    string `json:"result"`
	UpdatedAt int64  `json:"updated_at"`
	Hash      string `json:"hash"`
}

type ragPipelineParametersResponse struct {
	Variables []map[string]any `json:"variables"`
}

type ragPipelineExportResponse struct {
	Data string `json:"data"`
}

type ragPipelineImportResponse struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	PipelineID         string `json:"pipeline_id"`
	DatasetID          string `json:"dataset_id"`
	CurrentDSLVersion  string `json:"current_dsl_version"`
	ImportedDSLVersion string `json:"imported_dsl_version"`
	Error              string `json:"error"`
}

type publishedPipelineRunPreviewResponse struct {
	TaskIOD       string `json:"task_iod"`
	WorkflowRunID string `json:"workflow_run_id"`
	Data          struct {
		ID          string         `json:"id"`
		Status      string         `json:"status"`
		CreatedAt   int64          `json:"created_at"`
		ElapsedTime float64        `json:"elapsed_time"`
		Error       string         `json:"error"`
		FinishedAt  int64          `json:"finished_at"`
		Outputs     map[string]any `json:"outputs"`
		TotalSteps  int            `json:"total_steps"`
		TotalTokens int            `json:"total_tokens"`
		WorkflowID  string         `json:"workflow_id"`
	} `json:"data"`
}

type publishedPipelineRunResponse struct {
	Batch   string `json:"batch"`
	Dataset struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		ChunkStructure string `json:"chunk_structure"`
	} `json:"dataset"`
	Documents []struct {
		ID             string         `json:"id"`
		DataSourceInfo map[string]any `json:"data_source_info"`
		DataSourceType string         `json:"data_source_type"`
		Enable         bool           `json:"enable"`
		Error          string         `json:"error"`
		IndexingStatus string         `json:"indexing_status"`
		Name           string         `json:"name"`
		Position       int            `json:"position"`
	} `json:"documents"`
}

type indexingEstimatePreviewResponse struct {
	TotalNodes     int    `json:"total_nodes"`
	Tokens         int    `json:"tokens"`
	TotalSegments  int    `json:"total_segments"`
	Currency       string `json:"currency"`
	ChunkStructure string `json:"chunk_structure"`
	ParentMode     string `json:"parent_mode"`
	Preview        []struct {
		Content     string   `json:"content"`
		ChildChunks []string `json:"child_chunks"`
		Summary     string   `json:"summary"`
	} `json:"preview"`
	QAPreview []struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	} `json:"qa_preview"`
}

type pipelineExecutionLogResponse struct {
	DatasourceInfo   map[string]any `json:"datasource_info"`
	DatasourceType   string         `json:"datasource_type"`
	InputData        map[string]any `json:"input_data"`
	DatasourceNodeID string         `json:"datasource_node_id"`
}

type datasourceAuthListResponse struct {
	Result []map[string]any `json:"result"`
}

type datasourceCredentialResponse struct {
	Credential map[string]any `json:"credential"`
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	ID         string         `json:"id"`
	IsDefault  bool           `json:"is_default"`
	AvatarURL  string         `json:"avatar_url"`
}

type datasourceCredentialListResponse struct {
	Result []datasourceCredentialResponse `json:"result"`
}

type datasourceOAuthURLResponse struct {
	AuthorizationURL string `json:"authorization_url"`
	State            string `json:"state"`
	ContextID        string `json:"context_id"`
}

type pipelineTemplateListResponse struct {
	PipelineTemplates []struct {
		ID             string         `json:"id"`
		Name           string         `json:"name"`
		Description    string         `json:"description"`
		Position       int            `json:"position"`
		ChunkStructure string         `json:"chunk_structure"`
		Icon           map[string]any `json:"icon"`
	} `json:"pipeline_templates"`
}

type pipelineTemplateDetailResponse struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	ChunkStructure string         `json:"chunk_structure"`
	IconInfo       map[string]any `json:"icon_info"`
	ExportData     string         `json:"export_data"`
	Graph          map[string]any `json:"graph"`
	CreatedBy      string         `json:"created_by"`
}

type externalKnowledgeAPIResponse struct {
	ID       string `json:"id"`
	Settings struct {
		Endpoint string `json:"endpoint"`
		APIKey   string `json:"api_key"`
	} `json:"settings"`
}

type datasetDocumentCreateResponse struct {
	Batch     string `json:"batch"`
	Documents []struct {
		ID string `json:"id"`
	} `json:"documents"`
}

type indexingStatusResponse struct {
	ID                   string `json:"id"`
	IndexingStatus       string `json:"indexing_status"`
	ProcessingStartedAt  int64  `json:"processing_started_at"`
	ParsingCompletedAt   any    `json:"parsing_completed_at"`
	CleaningCompletedAt  any    `json:"cleaning_completed_at"`
	SplittingCompletedAt any    `json:"splitting_completed_at"`
	CompletedAt          any    `json:"completed_at"`
	PausedAt             any    `json:"paused_at"`
	Error                any    `json:"error"`
	StoppedAt            any    `json:"stopped_at"`
	CompletedSegments    int    `json:"completed_segments"`
	TotalSegments        int    `json:"total_segments"`
}

type indexingStatusBatchResponse struct {
	Data []indexingStatusResponse `json:"data"`
}

type datasetMetadataFieldResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type datasetMetadataListResponse struct {
	DocMetadata []datasetMetadataFieldResponse `json:"doc_metadata"`
}

type datasetDocumentMetadataOnlyResponse struct {
	ID          string                        `json:"id"`
	DocMetadata []datasetDocumentMetadataItem `json:"doc_metadata"`
}

type datasetDocumentMetadataItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type datasetSegmentEnvelope struct {
	Data datasetSegmentResponse `json:"data"`
}

type datasetSegmentResponse struct {
	ID          string                      `json:"id"`
	Content     string                      `json:"content"`
	Attachments []datasetAttachmentResponse `json:"attachments"`
	ChildChunks []datasetChildChunkResponse `json:"child_chunks"`
}

type datasetAttachmentResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	SourceURL string `json:"source_url"`
}

type datasetChildChunkResponse struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type datasetChildChunkListResponse struct {
	Data  []datasetChildChunkResponse `json:"data"`
	Total int                         `json:"total"`
}

type datasetSegmentListResponse struct {
	Data  []datasetSegmentResponse `json:"data"`
	Total int                      `json:"total"`
}

type batchImportResponse struct {
	JobID     string `json:"job_id"`
	JobStatus string `json:"job_status"`
}

type datasetHitTestingResponse struct {
	Query struct {
		Content string `json:"content"`
	} `json:"query"`
}

type externalDatasetHitTestingResponse struct {
	Query struct {
		Content string `json:"content"`
	} `json:"query"`
	Records []struct {
		Content  string         `json:"content"`
		Title    string         `json:"title"`
		Score    float64        `json:"score"`
		Metadata map[string]any `json:"metadata"`
	} `json:"records"`
}

type datasetQueriesResponse struct {
	Data []struct {
		Queries []struct {
			Content     string                     `json:"content"`
			ContentType string                     `json:"content_type"`
			FileInfo    *datasetAttachmentResponse `json:"file_info"`
		} `json:"queries"`
	} `json:"data"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func TestUploadsAndDatasetHitTestingAttachmentQueries(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	remoteContent := []byte("# Remote Guide\n\nhello from remote upload\n")
	remoteSource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="guide.md"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(remoteContent)))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(remoteContent)
	}))
	defer remoteSource.Close()

	consoleLocal := env.uploadFile("/console/api/files/upload", true, "local.md", "text/markdown", []byte("# Local\n\nhello from local\n"))
	if consoleLocal.ID == "" {
		t.Fatal("expected console local upload id")
	}
	if consoleLocal.SourceURL != "/files/"+consoleLocal.ID+"/file-preview" {
		t.Fatalf("unexpected console local source_url: %q", consoleLocal.SourceURL)
	}

	consoleRemote := postJSON[uploadResponse](env, http.MethodPost, "/console/api/remote-files/upload", map[string]any{
		"url": remoteSource.URL + "/guide",
	}, true, http.StatusCreated)
	if consoleRemote.Name != "guide.md" {
		t.Fatalf("unexpected console remote name: %q", consoleRemote.Name)
	}
	if consoleRemote.URL != "/files/"+consoleRemote.ID+"/file-preview" {
		t.Fatalf("unexpected console remote url: %q", consoleRemote.URL)
	}

	publicImage := env.uploadFile("/api/files/upload", false, "pixel.png", "image/png", mustPNG(t))
	if publicImage.ID == "" {
		t.Fatal("expected public image upload id")
	}
	previewBody := env.getBytes(publicImage.SourceURL, false, http.StatusOK)
	if !bytes.Equal(previewBody, mustPNG(t)) {
		t.Fatalf("public upload preview body mismatch: got %d bytes", len(previewBody))
	}

	publicRemote := postJSON[uploadResponse](env, http.MethodPost, "/api/remote-files/upload", map[string]any{
		"url": remoteSource.URL + "/guide",
	}, false, http.StatusCreated)
	if publicRemote.Name != "guide.md" {
		t.Fatalf("unexpected public remote name: %q", publicRemote.Name)
	}
	publicRemoteBody := env.getBytes(publicRemote.URL, false, http.StatusOK)
	if !bytes.Equal(publicRemoteBody, remoteContent) {
		t.Fatalf("public remote preview body mismatch: got %q", string(publicRemoteBody))
	}

	dataset := postJSON[datasetCreateResponse](env, http.MethodPost, "/console/api/datasets", map[string]any{
		"name":        "Uploads Dataset",
		"description": "integration test",
	}, true, http.StatusCreated)

	documents := postJSON[datasetDocumentCreateResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents", map[string]any{
		"data_source": map[string]any{
			"type": "upload_file",
			"info_list": map[string]any{
				"file_info_list": map[string]any{
					"file_ids": []string{consoleRemote.ID},
				},
			},
		},
		"indexing_technique": "high_quality",
	}, true, http.StatusCreated)
	if len(documents.Documents) != 1 || documents.Documents[0].ID == "" {
		t.Fatalf("unexpected document create response: %+v", documents)
	}

	hitResult := postJSON[datasetHitTestingResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/hit-testing", map[string]any{
		"query":          "guide",
		"attachment_ids": []string{publicImage.ID},
		"retrieval_model": map[string]any{
			"search_method": "semantic_search",
		},
	}, true, http.StatusOK)
	if hitResult.Query.Content != "guide" {
		t.Fatalf("unexpected hit testing query: %+v", hitResult.Query)
	}

	queryRecords := getJSON[datasetQueriesResponse](env, "/console/api/datasets/"+dataset.ID+"/queries?page=1&limit=1", true, http.StatusOK)
	if len(queryRecords.Data) != 1 {
		t.Fatalf("expected one query record, got %d", len(queryRecords.Data))
	}
	if len(queryRecords.Data[0].Queries) != 2 {
		t.Fatalf("expected text + image queries, got %+v", queryRecords.Data[0].Queries)
	}
	if queryRecords.Data[0].Queries[0].ContentType != "text_query" || queryRecords.Data[0].Queries[0].Content != "guide" {
		t.Fatalf("unexpected first query item: %+v", queryRecords.Data[0].Queries[0])
	}
	imageQuery := queryRecords.Data[0].Queries[1]
	if imageQuery.ContentType != "image_query" {
		t.Fatalf("unexpected image query content type: %+v", imageQuery)
	}
	if imageQuery.FileInfo == nil || imageQuery.FileInfo.ID != publicImage.ID {
		t.Fatalf("unexpected image query file info: %+v", imageQuery.FileInfo)
	}
	if imageQuery.FileInfo.SourceURL != publicImage.SourceURL {
		t.Fatalf("unexpected image query source url: %q", imageQuery.FileInfo.SourceURL)
	}
}

func TestDatasetMetadataSegmentsChildChunksAndBatchImportFlows(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	documentUpload := env.uploadFile("/console/api/files/upload", true, "knowledge.md", "text/markdown", []byte("# Knowledge\n\nhello dataset\n"))
	dataset := postJSON[datasetCreateResponse](env, http.MethodPost, "/console/api/datasets", map[string]any{
		"name":        "Segments Dataset",
		"description": "integration test",
	}, true, http.StatusCreated)

	documentCreate := postJSON[datasetDocumentCreateResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents", map[string]any{
		"data_source": map[string]any{
			"type": "upload_file",
			"info_list": map[string]any{
				"file_info_list": map[string]any{
					"file_ids": []string{documentUpload.ID},
				},
			},
		},
		"indexing_technique": "high_quality",
	}, true, http.StatusCreated)
	if len(documentCreate.Documents) != 1 {
		t.Fatalf("expected one created document, got %+v", documentCreate)
	}
	documentID := documentCreate.Documents[0].ID

	metadataField := postJSON[datasetMetadataFieldResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/metadata", map[string]any{
		"name": "language",
		"type": "string",
	}, true, http.StatusCreated)
	if metadataField.ID == "" {
		t.Fatal("expected metadata field id")
	}

	renamedField := postJSON[datasetMetadataFieldResponse](env, http.MethodPatch, "/console/api/datasets/"+dataset.ID+"/metadata/"+metadataField.ID, map[string]any{
		"name": "lang",
	}, true, http.StatusOK)
	if renamedField.Name != "lang" {
		t.Fatalf("unexpected renamed field: %+v", renamedField)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents/metadata", map[string]any{
		"operation_data": []map[string]any{
			{
				"document_id":    documentID,
				"partial_update": true,
				"metadata_list": []map[string]any{
					{
						"id":    metadataField.ID,
						"name":  "lang",
						"type":  "string",
						"value": "en",
					},
				},
			},
		},
	}, true, http.StatusOK)

	metadataList := getJSON[datasetMetadataListResponse](env, "/console/api/datasets/"+dataset.ID+"/metadata", true, http.StatusOK)
	if len(metadataList.DocMetadata) != 1 || metadataList.DocMetadata[0].Count != 1 {
		t.Fatalf("unexpected metadata counts: %+v", metadataList.DocMetadata)
	}

	documentDetail := getJSON[datasetDocumentMetadataOnlyResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"?metadata=only", true, http.StatusOK)
	if len(documentDetail.DocMetadata) == 0 || documentDetail.DocMetadata[0].Value != "en" {
		t.Fatalf("unexpected document metadata detail: %+v", documentDetail.DocMetadata)
	}

	imageUpload := env.uploadFile("/console/api/files/upload", true, "pixel.png", "image/png", mustPNG(t))
	segment := postJSON[datasetSegmentEnvelope](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segment", map[string]any{
		"content":        "Segment created via API",
		"attachment_ids": []string{imageUpload.ID},
	}, true, http.StatusCreated)
	if segment.Data.ID == "" {
		t.Fatal("expected segment id")
	}
	if len(segment.Data.Attachments) != 1 || segment.Data.Attachments[0].ID != imageUpload.ID {
		t.Fatalf("unexpected segment attachments: %+v", segment.Data.Attachments)
	}

	childChunk := postJSON[map[string]datasetChildChunkResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/"+segment.Data.ID+"/child_chunks", map[string]any{
		"content": "child chunk v1",
	}, true, http.StatusCreated)
	if childChunk["data"].ID == "" {
		t.Fatalf("unexpected child chunk create response: %+v", childChunk)
	}

	updatedChildChunk := postJSON[map[string]datasetChildChunkResponse](env, http.MethodPatch, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/"+segment.Data.ID+"/child_chunks/"+childChunk["data"].ID, map[string]any{
		"content": "child chunk updated",
	}, true, http.StatusOK)
	if updatedChildChunk["data"].Content != "child chunk updated" {
		t.Fatalf("unexpected updated child chunk: %+v", updatedChildChunk["data"])
	}

	childChunkList := getJSON[datasetChildChunkListResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/"+segment.Data.ID+"/child_chunks?page=1&limit=20", true, http.StatusOK)
	if childChunkList.Total != 1 || childChunkList.Data[0].Content != "child chunk updated" {
		t.Fatalf("unexpected child chunk list: %+v", childChunkList)
	}

	postJSON[map[string]any](env, http.MethodDelete, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/"+segment.Data.ID+"/child_chunks/"+childChunk["data"].ID, nil, true, http.StatusOK)
	childChunkList = getJSON[datasetChildChunkListResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/"+segment.Data.ID+"/child_chunks?page=1&limit=20", true, http.StatusOK)
	if childChunkList.Total != 0 {
		t.Fatalf("expected no child chunks after delete, got %+v", childChunkList)
	}

	batchFile := env.uploadFile("/console/api/files/upload", true, "batch.csv", "text/csv", []byte("content\nhello\n"))
	batchImport := postJSON[batchImportResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments/batch_import", map[string]any{
		"upload_file_id": batchFile.ID,
	}, true, http.StatusCreated)
	if batchImport.JobID == "" || batchImport.JobStatus != "completed" {
		t.Fatalf("unexpected batch import response: %+v", batchImport)
	}

	batchStatus := getJSON[batchImportResponse](env, "/console/api/datasets/batch_import_status/"+batchImport.JobID, true, http.StatusOK)
	if batchStatus.JobStatus != "completed" {
		t.Fatalf("unexpected batch import status: %+v", batchStatus)
	}

	segments := getJSON[datasetSegmentListResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/segments?page=1&limit=20", true, http.StatusOK)
	if segments.Total != 4 {
		t.Fatalf("expected 4 segments after import, got %+v", segments)
	}
}

func TestExternalDatasetHitTestingUsesExternalAPIContract(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	var gotAuth string
	var gotPath string
	var gotDecodeErr error
	var gotRequest struct {
		RetrievalSetting struct {
			TopK           int     `json:"top_k"`
			ScoreThreshold float64 `json:"score_threshold"`
		} `json:"retrieval_setting"`
		Query             string `json:"query"`
		KnowledgeID       string `json:"knowledge_id"`
		MetadataCondition any    `json:"metadata_condition"`
	}

	externalKnowledgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		defer r.Body.Close()
		gotDecodeErr = json.NewDecoder(r.Body).Decode(&gotRequest)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{
					"content": "alpha guide primary result",
					"title":   "alpha.md",
					"score":   0.95,
					"metadata": map[string]any{
						"x-amz-bedrock-kb-source-uri": "s3://kb/alpha.md",
					},
				},
				{
					"content": "alpha guide secondary result",
					"title":   "beta.md",
					"score":   0.72,
					"metadata": map[string]any{
						"x-amz-bedrock-kb-source-uri": "s3://kb/beta.md",
					},
				},
				{
					"content": "alpha guide low score result",
					"title":   "gamma.md",
					"score":   0.41,
					"metadata": map[string]any{
						"x-amz-bedrock-kb-source-uri": "s3://kb/gamma.md",
					},
				},
			},
		})
	}))
	defer externalKnowledgeServer.Close()

	api := postJSON[externalKnowledgeAPIResponse](env, http.MethodPost, "/console/api/datasets/external-knowledge-api", map[string]any{
		"name":        "Mock External KB",
		"description": "integration test",
		"settings": map[string]any{
			"endpoint": externalKnowledgeServer.URL,
			"api_key":  "secret-token",
		},
	}, true, http.StatusCreated)
	if api.ID == "" {
		t.Fatal("expected external knowledge api id")
	}

	dataset := postJSON[datasetCreateResponse](env, http.MethodPost, "/console/api/datasets/external", map[string]any{
		"name":                      "External Dataset",
		"description":               "integration test",
		"external_knowledge_id":     "kb-alpha",
		"external_knowledge_api_id": api.ID,
		"external_retrieval_model": map[string]any{
			"top_k":                   4,
			"score_threshold":         0.5,
			"score_threshold_enabled": false,
		},
	}, true, http.StatusCreated)

	hitResult := postJSON[externalDatasetHitTestingResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/external-hit-testing", map[string]any{
		"query": `guide "alpha"`,
		"external_retrieval_model": map[string]any{
			"top_k":                   2,
			"score_threshold":         0.7,
			"score_threshold_enabled": true,
		},
	}, true, http.StatusOK)

	if gotDecodeErr != nil {
		t.Fatalf("decode external retrieval request: %v", gotDecodeErr)
	}
	if gotPath != "/retrieval" {
		t.Fatalf("unexpected external retrieval path: %q", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if gotRequest.KnowledgeID != "kb-alpha" {
		t.Fatalf("unexpected knowledge id: %+v", gotRequest)
	}
	if gotRequest.Query != `guide \"alpha\"` {
		t.Fatalf("unexpected escaped query: %q", gotRequest.Query)
	}
	if gotRequest.RetrievalSetting.TopK != 2 || gotRequest.RetrievalSetting.ScoreThreshold != 0.7 {
		t.Fatalf("unexpected retrieval setting: %+v", gotRequest.RetrievalSetting)
	}
	if hitResult.Query.Content != `guide "alpha"` {
		t.Fatalf("unexpected hit testing query: %+v", hitResult.Query)
	}
	if len(hitResult.Records) != 2 {
		t.Fatalf("expected threshold + top_k to keep 2 records, got %+v", hitResult.Records)
	}
	if hitResult.Records[0].Title != "alpha.md" || hitResult.Records[1].Title != "beta.md" {
		t.Fatalf("unexpected external hit testing records: %+v", hitResult.Records)
	}
	if hitResult.Records[0].Metadata["x-amz-bedrock-kb-source-uri"] != "s3://kb/alpha.md" {
		t.Fatalf("unexpected metadata passthrough: %+v", hitResult.Records[0].Metadata)
	}

	queryRecords := getJSON[datasetQueriesResponse](env, "/console/api/datasets/"+dataset.ID+"/queries?page=1&limit=1", true, http.StatusOK)
	if len(queryRecords.Data) != 1 || len(queryRecords.Data[0].Queries) != 1 {
		t.Fatalf("unexpected query records: %+v", queryRecords.Data)
	}
	if queryRecords.Data[0].Queries[0].ContentType != "text_query" || queryRecords.Data[0].Queries[0].Content != `guide "alpha"` {
		t.Fatalf("unexpected stored query record: %+v", queryRecords.Data[0].Queries[0])
	}
}

func TestExternalDatasetHitTestingValidatesQuery(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	requests := 0
	externalKnowledgeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"records": []any{}})
	}))
	defer externalKnowledgeServer.Close()

	api := postJSON[externalKnowledgeAPIResponse](env, http.MethodPost, "/console/api/datasets/external-knowledge-api", map[string]any{
		"name": "Validation External KB",
		"settings": map[string]any{
			"endpoint": externalKnowledgeServer.URL,
			"api_key":  "secret-token",
		},
	}, true, http.StatusCreated)

	dataset := postJSON[datasetCreateResponse](env, http.MethodPost, "/console/api/datasets/external", map[string]any{
		"name":                      "External Validation Dataset",
		"external_knowledge_id":     "kb-validation",
		"external_knowledge_api_id": api.ID,
	}, true, http.StatusCreated)

	emptyQuery := postJSON[errorResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/external-hit-testing", map[string]any{
		"query": "   ",
	}, true, http.StatusBadRequest)
	if emptyQuery.Message != "Query is required." {
		t.Fatalf("unexpected empty query error: %+v", emptyQuery)
	}

	longQuery := postJSON[errorResponse](env, http.MethodPost, "/console/api/datasets/"+dataset.ID+"/external-hit-testing", map[string]any{
		"query": strings.Repeat("a", 251),
	}, true, http.StatusBadRequest)
	if longQuery.Message != "Query cannot exceed 250 characters." {
		t.Fatalf("unexpected long query error: %+v", longQuery)
	}

	if requests != 0 {
		t.Fatalf("expected validation to fail before external request, got %d outbound requests", requests)
	}

	queryRecords := getJSON[datasetQueriesResponse](env, "/console/api/datasets/"+dataset.ID+"/queries?page=1&limit=20", true, http.StatusOK)
	if len(queryRecords.Data) != 0 {
		t.Fatalf("expected no stored query records after validation failures, got %+v", queryRecords.Data)
	}
}

func TestCreateEmptyRAGPipelineDatasetAndDraftAliases(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	plugins := getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if len(plugins) != 1 {
		t.Fatalf("expected only local file datasource plugin before installs, got %+v", plugins)
	}

	assertDatasourcePlugin := func(pluginID, providerType, provider, datasourceName string, authorized bool) {
		t.Helper()

		for _, plugin := range plugins {
			if stringFromAny(plugin["plugin_id"]) != pluginID {
				continue
			}

			if stringFromAny(plugin["provider"]) != provider {
				t.Fatalf("unexpected provider for %s: %+v", pluginID, plugin)
			}
			if ragPipelineBoolValue(plugin["is_authorized"]) != authorized {
				t.Fatalf("unexpected authorization state for %s: %+v", pluginID, plugin)
			}

			declaration := mapFromAny(plugin["declaration"])
			if stringFromAny(declaration["provider_type"]) != providerType {
				t.Fatalf("unexpected provider type for %s: %+v", pluginID, declaration)
			}

			identity := mapFromAny(declaration["identity"])
			if stringFromAny(identity["name"]) != provider {
				t.Fatalf("unexpected provider identity for %s: %+v", pluginID, identity)
			}
			if stringFromAny(plugin["plugin_unique_identifier"]) == "" {
				t.Fatalf("expected plugin unique identifier for %s: %+v", pluginID, plugin)
			}

			datasources := objectListFromAny(declaration["datasources"])
			if len(datasources) != 1 {
				t.Fatalf("expected single datasource entry for %s: %+v", pluginID, datasources)
			}
			datasourceIdentity := mapFromAny(datasources[0]["identity"])
			if stringFromAny(datasourceIdentity["name"]) != datasourceName {
				t.Fatalf("unexpected datasource name for %s: %+v", pluginID, datasourceIdentity)
			}
			if stringFromAny(datasourceIdentity["provider"]) != provider {
				t.Fatalf("unexpected datasource provider for %s: %+v", pluginID, datasourceIdentity)
			}
			return
		}

		t.Fatalf("expected datasource plugin %s in %+v", pluginID, plugins)
	}

	assertDatasourcePlugin("langgenius/file", "local_file", "file", "local-file", true)

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	if dataset.ID == "" || dataset.PipelineID == "" {
		t.Fatalf("expected dataset and pipeline ids, got %+v", dataset)
	}
	if dataset.RuntimeMode != "rag_pipeline" {
		t.Fatalf("unexpected runtime mode: %+v", dataset)
	}
	if dataset.Permission != "only_me" {
		t.Fatalf("unexpected permission: %+v", dataset)
	}
	if dataset.DocForm != "text_model" || dataset.IndexingTechnique != "high_quality" {
		t.Fatalf("unexpected dataset defaults: %+v", dataset)
	}

	missingDraft := getJSON[errorResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", true, http.StatusNotFound)
	if missingDraft.Code != "draft_workflow_not_exist" {
		t.Fatalf("unexpected missing draft response: %+v", missingDraft)
	}

	sync := postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{"id": "source-node", "data": map[string]any{"title": "Source"}},
				{"id": "process-node", "data": map[string]any{"title": "Process"}},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{"opening_statement": "hello"},
		"environment_variables": []map[string]any{
			{"id": "env-limit", "name": "limit", "value_type": "number", "value": 5},
		},
		"conversation_variables": []map[string]any{
			{"id": "conv-mode", "name": "mode", "value_type": "string", "value": "draft"},
		},
		"rag_pipeline_variables": []map[string]any{
			{"belong_to_node_id": "shared", "label": "Shared Query", "variable": "shared_query", "type": "text-input", "required": true},
			{"belong_to_node_id": "source-node", "label": "Source URL", "variable": "source_url", "type": "text-input", "required": true},
			{"belong_to_node_id": "process-node", "label": "Chunk Size", "variable": "chunk_size", "type": "number", "required": false},
		},
	}, true, http.StatusOK)
	if sync.Result != "success" || sync.Hash == "" || sync.UpdatedAt == 0 {
		t.Fatalf("unexpected draft sync response: %+v", sync)
	}

	draft := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", true, http.StatusOK)
	if draft.Hash == "" || len(draft.RagPipelineVariables) != 3 {
		t.Fatalf("unexpected draft response: %+v", draft)
	}

	preProcessing := getJSON[ragPipelineParametersResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft/pre-processing/parameters?node_id=source-node", true, http.StatusOK)
	if len(preProcessing.Variables) != 2 {
		t.Fatalf("expected shared + source-node parameters, got %+v", preProcessing.Variables)
	}
	if stringFromAny(preProcessing.Variables[0]["variable"]) != "shared_query" || stringFromAny(preProcessing.Variables[1]["variable"]) != "source_url" {
		t.Fatalf("unexpected pre-processing variables: %+v", preProcessing.Variables)
	}

	processing := getJSON[ragPipelineParametersResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft/processing/parameters?node_id=process-node", true, http.StatusOK)
	if len(processing.Variables) != 2 {
		t.Fatalf("expected shared + process-node parameters, got %+v", processing.Variables)
	}
	if stringFromAny(processing.Variables[0]["variable"]) != "shared_query" || stringFromAny(processing.Variables[1]["variable"]) != "chunk_size" {
		t.Fatalf("unexpected processing variables: %+v", processing.Variables)
	}
}

func TestDatasourceCatalogTracksWorkspacePluginInstallations(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	plugins := getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if len(plugins) != 1 {
		t.Fatalf("expected only local file datasource plugin before installs, got %+v", plugins)
	}
	localFile := findDatasourcePluginItem(t, plugins, "langgenius/file")
	if !ragPipelineBoolValue(localFile["is_installed"]) || !ragPipelineBoolValue(localFile["is_authorized"]) {
		t.Fatalf("expected local file datasource to stay installed and authorized: %+v", localFile)
	}

	authList := getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	if len(authList.Result) != 0 {
		t.Fatalf("expected no remote datasource auth providers before installs, got %+v", authList.Result)
	}
	defaultAuthList := getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/default-list", true, http.StatusOK)
	if len(defaultAuthList.Result) != 0 {
		t.Fatalf("expected empty default datasource auth list before installs, got %+v", defaultAuthList.Result)
	}

	missingProvider := getJSON[errorResponse](env, "/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource", true, http.StatusNotFound)
	if missingProvider.Code != "provider_not_found" {
		t.Fatalf("unexpected missing datasource provider response: %+v", missingProvider)
	}

	firecrawlUID := installDatasourcePluginFromSpec(t, env, "langgenius/firecrawl_datasource", "firecrawl")
	notionUID := installDatasourcePluginFromSpec(t, env, "langgenius/notion_datasource", "notion_datasource")

	plugins = getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if len(plugins) != 3 {
		t.Fatalf("expected local file + 2 installed datasource plugins, got %+v", plugins)
	}

	firecrawlPlugin := findDatasourcePluginItem(t, plugins, "langgenius/firecrawl_datasource")
	if !ragPipelineBoolValue(firecrawlPlugin["is_installed"]) || ragPipelineBoolValue(firecrawlPlugin["is_authorized"]) {
		t.Fatalf("unexpected firecrawl plugin state after install: %+v", firecrawlPlugin)
	}
	if stringFromAny(firecrawlPlugin["plugin_unique_identifier"]) != firecrawlUID {
		t.Fatalf("unexpected firecrawl plugin unique identifier: %+v", firecrawlPlugin)
	}

	notionPlugin := findDatasourcePluginItem(t, plugins, "langgenius/notion_datasource")
	if !ragPipelineBoolValue(notionPlugin["is_installed"]) || ragPipelineBoolValue(notionPlugin["is_authorized"]) {
		t.Fatalf("unexpected notion plugin state after install: %+v", notionPlugin)
	}
	if stringFromAny(notionPlugin["plugin_unique_identifier"]) != notionUID {
		t.Fatalf("unexpected notion plugin unique identifier: %+v", notionPlugin)
	}

	authList = getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	if len(authList.Result) != 2 {
		t.Fatalf("expected 2 installed datasource auth providers, got %+v", authList.Result)
	}
	firecrawlAuth := findDatasourceAuthItem(t, authList.Result, "langgenius/firecrawl_datasource")
	if !ragPipelineBoolValue(firecrawlAuth["is_installed"]) {
		t.Fatalf("expected firecrawl auth provider to be installed: %+v", firecrawlAuth)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", map[string]any{
		"type": "api-key",
		"name": "Crawler Primary",
		"credentials": map[string]any{
			"api_key": "firecrawl-secret",
		},
	}, true, http.StatusOK)

	uninstallWorkspacePluginByID(t, env, "langgenius/notion_datasource")
	uninstallWorkspacePluginByID(t, env, "langgenius/firecrawl_datasource")

	plugins = getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if len(plugins) != 2 {
		t.Fatalf("expected local file + firecrawl fallback datasource plugin after uninstall, got %+v", plugins)
	}
	firecrawlPlugin = findDatasourcePluginItem(t, plugins, "langgenius/firecrawl_datasource")
	if ragPipelineBoolValue(firecrawlPlugin["is_installed"]) || !ragPipelineBoolValue(firecrawlPlugin["is_authorized"]) {
		t.Fatalf("expected firecrawl datasource to stay available via credential state after uninstall: %+v", firecrawlPlugin)
	}
	if stringFromAny(firecrawlPlugin["plugin_unique_identifier"]) != firecrawlUID {
		t.Fatalf("expected firecrawl unique identifier to stay stable after uninstall fallback: %+v", firecrawlPlugin)
	}

	authList = getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	if len(authList.Result) != 1 {
		t.Fatalf("expected only firecrawl auth provider after uninstall fallback, got %+v", authList.Result)
	}
	firecrawlAuth = findDatasourceAuthItem(t, authList.Result, "langgenius/firecrawl_datasource")
	if ragPipelineBoolValue(firecrawlAuth["is_installed"]) {
		t.Fatalf("expected firecrawl auth provider to reflect uninstall state: %+v", firecrawlAuth)
	}
	if stringFromAny(firecrawlAuth["plugin_unique_identifier"]) != firecrawlUID {
		t.Fatalf("expected firecrawl auth unique identifier to stay stable after uninstall fallback: %+v", firecrawlAuth)
	}

	missingProvider = getJSON[errorResponse](env, "/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource", true, http.StatusNotFound)
	if missingProvider.Code != "provider_not_found" {
		t.Fatalf("expected notion datasource provider to disappear after uninstall, got %+v", missingProvider)
	}
}

func TestDatasourceAuthLifecycleAndOAuthCallback(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	installDatasourcePluginFromSpec(t, env, "langgenius/firecrawl_datasource", "firecrawl")
	installDatasourcePluginFromSpec(t, env, "langgenius/notion_datasource", "notion_datasource")

	authList := getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	if len(authList.Result) != 2 {
		t.Fatalf("expected 2 installed datasource auth entries, got %+v", authList.Result)
	}

	firecrawl := findDatasourceAuthItem(t, authList.Result, "langgenius/firecrawl_datasource")
	if len(objectListFromAny(firecrawl["credential_schema"])) != 1 {
		t.Fatalf("expected firecrawl api key schema, got %+v", firecrawl)
	}
	if _, ok := firecrawl["oauth_schema"]; ok {
		t.Fatalf("did not expect firecrawl oauth schema, got %+v", firecrawl["oauth_schema"])
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", map[string]any{
		"type": "api-key",
		"name": "Crawler Primary",
		"credentials": map[string]any{
			"api_key": "firecrawl-secret",
		},
	}, true, http.StatusOK)

	firecrawlCredentials := getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", true, http.StatusOK)
	if len(firecrawlCredentials.Result) != 1 {
		t.Fatalf("expected single firecrawl credential, got %+v", firecrawlCredentials.Result)
	}
	firstCredential := firecrawlCredentials.Result[0]
	if firstCredential.Name != "Crawler Primary" || firstCredential.Type != "api-key" || !firstCredential.IsDefault {
		t.Fatalf("unexpected first firecrawl credential: %+v", firstCredential)
	}
	if stringFromAny(firstCredential.Credential["api_key"]) != hiddenSecretValue {
		t.Fatalf("expected masked firecrawl api key, got %+v", firstCredential.Credential)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl/update", map[string]any{
		"credential_id": firstCredential.ID,
		"name":          "Crawler Primary Renamed",
		"credentials": map[string]any{
			"api_key": hiddenSecretValue,
		},
	}, true, http.StatusOK)

	firecrawlCredentials = getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", true, http.StatusOK)
	if firecrawlCredentials.Result[0].Name != "Crawler Primary Renamed" {
		t.Fatalf("expected renamed firecrawl credential, got %+v", firecrawlCredentials.Result)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", map[string]any{
		"type": "api-key",
		"name": "Crawler Backup",
		"credentials": map[string]any{
			"api_key": "backup-secret",
		},
	}, true, http.StatusOK)

	firecrawlCredentials = getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", true, http.StatusOK)
	if len(firecrawlCredentials.Result) != 2 {
		t.Fatalf("expected two firecrawl credentials, got %+v", firecrawlCredentials.Result)
	}

	secondCredentialID := ""
	for _, item := range firecrawlCredentials.Result {
		if item.Name == "Crawler Backup" {
			secondCredentialID = item.ID
		}
	}
	if secondCredentialID == "" {
		t.Fatalf("expected backup firecrawl credential in %+v", firecrawlCredentials.Result)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl/default", map[string]any{
		"id": secondCredentialID,
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl/delete", map[string]any{
		"credential_id": firstCredential.ID,
	}, true, http.StatusOK)

	firecrawlCredentials = getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", true, http.StatusOK)
	if len(firecrawlCredentials.Result) != 1 || firecrawlCredentials.Result[0].ID != secondCredentialID || !firecrawlCredentials.Result[0].IsDefault {
		t.Fatalf("unexpected firecrawl credential state after default/delete: %+v", firecrawlCredentials.Result)
	}

	plugins := getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if !datasourcePluginAuthorized(t, plugins, "langgenius/firecrawl_datasource") {
		t.Fatalf("expected firecrawl datasource plugin to become authorized: %+v", plugins)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource/custom-client", map[string]any{
		"client_params": map[string]any{
			"client_id":     "notion-client",
			"client_secret": "notion-secret",
		},
		"enable_oauth_custom_client": true,
	}, true, http.StatusOK)

	authList = getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	notion := findDatasourceAuthItem(t, authList.Result, "langgenius/notion_datasource")
	notionOAuth := mapFromAny(notion["oauth_schema"])
	if !ragPipelineBoolValue(notionOAuth["is_oauth_custom_client_enabled"]) {
		t.Fatalf("expected notion custom oauth client to be enabled: %+v", notionOAuth)
	}
	notionClientParams := mapFromAny(notionOAuth["oauth_custom_client_params"])
	if stringFromAny(notionClientParams["client_id"]) != "notion-client" {
		t.Fatalf("unexpected notion oauth client params: %+v", notionClientParams)
	}
	if stringFromAny(notionClientParams["client_secret"]) != hiddenSecretValue {
		t.Fatalf("expected masked notion client secret, got %+v", notionClientParams)
	}

	authorizationReq, err := http.NewRequest(http.MethodGet, env.server.URL+"/console/api/oauth/plugin/langgenius/notion_datasource/notion_datasource/datasource/get-authorization-url", nil)
	if err != nil {
		t.Fatalf("create datasource oauth authorization request: %v", err)
	}
	authorizationReq.Header.Set("Origin", "http://localhost")
	authorizationResp := env.do(authorizationReq, http.StatusOK)
	defer authorizationResp.Body.Close()

	var oauthURL datasourceOAuthURLResponse
	if err := json.NewDecoder(authorizationResp.Body).Decode(&oauthURL); err != nil {
		t.Fatalf("decode datasource oauth url response: %v", err)
	}
	if oauthURL.AuthorizationURL == "" || !strings.Contains(oauthURL.AuthorizationURL, "/datasource/callback") {
		t.Fatalf("unexpected datasource oauth authorization url: %+v", oauthURL)
	}

	callbackReq, err := http.NewRequest(http.MethodGet, oauthURL.AuthorizationURL, nil)
	if err != nil {
		t.Fatalf("create datasource oauth callback request: %v", err)
	}
	callbackReq.Header.Set("Origin", "http://localhost")
	callbackResp := env.doNoRedirect(callbackReq, http.StatusFound)
	defer callbackResp.Body.Close()
	if location := callbackResp.Header.Get("Location"); !strings.HasPrefix(location, "http://localhost/oauth-callback") {
		t.Fatalf("unexpected datasource oauth callback redirect: %s", location)
	}

	notionCredentials := getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource", true, http.StatusOK)
	if len(notionCredentials.Result) != 1 || notionCredentials.Result[0].Type != "oauth2" {
		t.Fatalf("expected notion oauth credential after callback, got %+v", notionCredentials.Result)
	}

	plugins = getJSON[[]map[string]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if !datasourcePluginAuthorized(t, plugins, "langgenius/notion_datasource") {
		t.Fatalf("expected notion datasource plugin to become authorized: %+v", plugins)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, env.server.URL+"/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource/custom-client", nil)
	if err != nil {
		t.Fatalf("create datasource oauth custom client delete request: %v", err)
	}
	deleteReq.Header.Set(csrfHeader, env.csrfToken())
	env.do(deleteReq, http.StatusOK).Body.Close()

	authList = getJSON[datasourceAuthListResponse](env, "/console/api/auth/plugin/datasource/list", true, http.StatusOK)
	notion = findDatasourceAuthItem(t, authList.Result, "langgenius/notion_datasource")
	notionOAuth = mapFromAny(notion["oauth_schema"])
	if ragPipelineBoolValue(notionOAuth["is_oauth_custom_client_enabled"]) {
		t.Fatalf("expected notion custom oauth client to be deleted: %+v", notionOAuth)
	}
	if len(mapFromAny(notionOAuth["oauth_custom_client_params"])) != 0 {
		t.Fatalf("expected cleared notion oauth custom client params, got %+v", notionOAuth["oauth_custom_client_params"])
	}
}

func TestRAGPipelineDatasourceNodeRunCompatibility(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	installDatasourcePluginFromSpec(t, env, "langgenius/firecrawl_datasource", "firecrawl")
	installDatasourcePluginFromSpec(t, env, "langgenius/notion_datasource", "notion_datasource")
	installDatasourcePluginFromSpec(t, env, "langgenius/google_drive", "google_drive")

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{"id": "notion-node", "data": map[string]any{"title": "Notion Source"}},
				{"id": "crawl-node", "data": map[string]any{"title": "Website Source"}},
				{"id": "drive-node", "data": map[string]any{"title": "Drive Source"}},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name":    "compat",
		"marked_comment": "datasource node run coverage",
	}, true, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", map[string]any{
		"type": "api-key",
		"name": "Crawler",
		"credentials": map[string]any{
			"api_key": "firecrawl-secret",
		},
	}, true, http.StatusOK)
	firecrawlCredentials := getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/firecrawl_datasource/firecrawl", true, http.StatusOK)
	if len(firecrawlCredentials.Result) != 1 {
		t.Fatalf("expected firecrawl credential for datasource node run test, got %+v", firecrawlCredentials.Result)
	}

	notionCallbackReq, err := http.NewRequest(http.MethodGet, env.server.URL+"/console/api/oauth/plugin/langgenius/notion_datasource/notion_datasource/datasource/callback?redirect_origin=http://localhost", nil)
	if err != nil {
		t.Fatalf("create notion callback request: %v", err)
	}
	env.doNoRedirect(notionCallbackReq, http.StatusFound).Body.Close()
	notionCredentials := getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/notion_datasource/notion_datasource", true, http.StatusOK)
	if len(notionCredentials.Result) != 1 {
		t.Fatalf("expected notion credential for datasource node run test, got %+v", notionCredentials.Result)
	}

	driveCallbackReq, err := http.NewRequest(http.MethodGet, env.server.URL+"/console/api/oauth/plugin/langgenius/google_drive/google_drive/datasource/callback?redirect_origin=http://localhost", nil)
	if err != nil {
		t.Fatalf("create google drive callback request: %v", err)
	}
	env.doNoRedirect(driveCallbackReq, http.StatusFound).Body.Close()
	driveCredentials := getJSON[datasourceCredentialListResponse](env, "/console/api/auth/plugin/datasource/langgenius/google_drive/google_drive", true, http.StatusOK)
	if len(driveCredentials.Result) != 1 {
		t.Fatalf("expected google drive credential for datasource node run test, got %+v", driveCredentials.Result)
	}

	notionEvents := postStreamJSON(env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft/datasource/nodes/notion-node/run", map[string]any{
		"inputs": map[string]any{
			"workspace": "product",
			"database":  "roadmap",
		},
		"credential_id":   notionCredentials.Result[0].ID,
		"datasource_type": "online_document",
	}, true, http.StatusOK)
	if len(notionEvents) != 1 || stringFromAny(notionEvents[0]["event"]) != "datasource_completed" {
		t.Fatalf("unexpected notion datasource events: %+v", notionEvents)
	}
	notionData := objectListFromAny(notionEvents[0]["data"])
	if len(notionData) != 1 {
		t.Fatalf("expected notion workspace payload, got %+v", notionEvents[0]["data"])
	}
	rawPages, ok := notionData[0]["pages"].([]any)
	if !ok || len(rawPages) != 3 {
		t.Fatalf("expected 3 notion pages, got %+v", notionData[0]["pages"])
	}

	websiteEvents := postStreamJSON(env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/datasource/nodes/crawl-node/run", map[string]any{
		"inputs": map[string]any{
			"url":   "https://docs.example.com/start",
			"depth": 2,
		},
		"credential_id":   firecrawlCredentials.Result[0].ID,
		"datasource_type": "website_crawl",
		"response_mode":   "streaming",
	}, true, http.StatusOK)
	if len(websiteEvents) != 3 {
		t.Fatalf("expected processing + completed website events, got %+v", websiteEvents)
	}
	if stringFromAny(websiteEvents[0]["event"]) != "datasource_processing" || stringFromAny(websiteEvents[2]["event"]) != "datasource_completed" {
		t.Fatalf("unexpected website event sequence: %+v", websiteEvents)
	}
	websiteResults := objectListFromAny(websiteEvents[2]["data"])
	if len(websiteResults) != 3 {
		t.Fatalf("expected 3 website crawl results, got %+v", websiteEvents[2]["data"])
	}
	if stringFromAny(websiteResults[0]["source_url"]) == "" || stringFromAny(websiteResults[0]["markdown"]) == "" {
		t.Fatalf("expected source_url and markdown in website result, got %+v", websiteResults[0])
	}

	driveEvents := postStreamJSON(env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/datasource/nodes/drive-node/run", map[string]any{
		"inputs":          map[string]any{},
		"credential_id":   driveCredentials.Result[0].ID,
		"datasource_type": "online_drive",
	}, true, http.StatusOK)
	if len(driveEvents) != 1 || stringFromAny(driveEvents[0]["event"]) != "datasource_completed" {
		t.Fatalf("unexpected online drive datasource events: %+v", driveEvents)
	}
	driveData := objectListFromAny(driveEvents[0]["data"])
	if len(driveData) != 2 {
		t.Fatalf("expected initial online drive bucket list, got %+v", driveEvents[0]["data"])
	}
	if stringFromAny(driveData[0]["bucket"]) == "" {
		t.Fatalf("expected online drive bucket entries, got %+v", driveData[0])
	}
}

func TestRAGPipelinePublishReflectsDatasetStateAndDeleteCleansUpPipeline(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes":    []map[string]any{{"id": "source-node", "data": map[string]any{"title": "Source"}}},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{},
		"rag_pipeline_variables": []map[string]any{
			{"belong_to_node_id": "shared", "label": "Shared Query", "variable": "shared_query", "type": "text-input", "required": true},
			{"belong_to_node_id": "source-node", "label": "Source URL", "variable": "source_url", "type": "text-input", "required": true},
		},
	}, true, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name":    "v1",
		"marked_comment": "publish for integration test",
	}, true, http.StatusOK)

	published := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", true, http.StatusOK)
	if len(published.RagPipelineVariables) != 2 {
		t.Fatalf("unexpected published workflow response: %+v", published)
	}

	detail := getJSON[ragPipelineDatasetResponse](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if !detail.IsPublished {
		t.Fatalf("expected dataset detail to reflect published pipeline, got %+v", detail)
	}

	publishedParams := getJSON[ragPipelineParametersResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/pre-processing/parameters?node_id=source-node", true, http.StatusOK)
	if len(publishedParams.Variables) != 2 {
		t.Fatalf("expected published parameters to include shared + node variables, got %+v", publishedParams.Variables)
	}

	postJSON[ragPipelineDatasetResponse](env, http.MethodDelete, "/console/api/datasets/"+dataset.ID, nil, true, http.StatusOK)

	missingPipeline := getJSON[errorResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", true, http.StatusNotFound)
	if missingPipeline.Code != "app_not_found" {
		t.Fatalf("unexpected missing pipeline response after delete: %+v", missingPipeline)
	}
}

func TestRAGPipelineDatasetAndAppMetadataStayInSync(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	updatedDataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPatch, "/console/api/datasets/"+dataset.ID, map[string]any{
		"name":        "Synced Knowledge Pipeline",
		"description": "metadata updated from dataset",
		"icon_info": map[string]any{
			"icon":            "file-pipeline-icon",
			"icon_type":       "image",
			"icon_background": "#111827",
			"icon_url":        "https://example.com/pipeline.png",
		},
	}, true, http.StatusOK)
	if updatedDataset.Name != "Synced Knowledge Pipeline" || updatedDataset.Description != "metadata updated from dataset" {
		t.Fatalf("unexpected updated dataset metadata: %+v", updatedDataset)
	}

	appDetail := getJSON[map[string]any](env, "/console/api/apps/"+dataset.PipelineID, true, http.StatusOK)
	if stringFromAny(appDetail["name"]) != updatedDataset.Name || stringFromAny(appDetail["description"]) != updatedDataset.Description {
		t.Fatalf("expected dataset patch to sync app metadata, got %+v", appDetail)
	}
	if stringFromAny(appDetail["icon_type"]) != "image" || stringFromAny(appDetail["icon"]) != "file-pipeline-icon" || stringFromAny(appDetail["icon_background"]) != "#111827" {
		t.Fatalf("expected dataset patch to sync app icon fields, got %+v", appDetail)
	}
	if stringFromAny(appDetail["icon_url"]) != "https://example.com/pipeline.png" {
		t.Fatalf("expected dataset patch to sync app icon url, got %+v", appDetail)
	}
	appSite := mapFromAny(appDetail["site"])
	if stringFromAny(appSite["title"]) != updatedDataset.Name || stringFromAny(appSite["description"]) != updatedDataset.Description {
		t.Fatalf("expected dataset patch to sync app site metadata, got %+v", appSite)
	}
	if stringFromAny(appSite["icon_url"]) != "https://example.com/pipeline.png" {
		t.Fatalf("expected dataset patch to sync app site icon url, got %+v", appSite)
	}

	updatedApp := postJSON[map[string]any](env, http.MethodPut, "/console/api/apps/"+dataset.PipelineID, map[string]any{
		"name":            "App Driven Pipeline",
		"description":     "metadata updated from app",
		"icon_type":       "emoji",
		"icon":            "🧭",
		"icon_background": "#DCFCE7",
	}, true, http.StatusOK)
	if stringFromAny(updatedApp["name"]) != "App Driven Pipeline" || stringFromAny(updatedApp["description"]) != "metadata updated from app" {
		t.Fatalf("unexpected updated app metadata: %+v", updatedApp)
	}

	datasetDetail := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if stringFromAny(datasetDetail["name"]) != "App Driven Pipeline" || stringFromAny(datasetDetail["description"]) != "metadata updated from app" {
		t.Fatalf("expected app update to sync dataset metadata, got %+v", datasetDetail)
	}
	iconInfo := mapFromAny(datasetDetail["icon_info"])
	if stringFromAny(iconInfo["icon_type"]) != "emoji" || stringFromAny(iconInfo["icon"]) != "🧭" || stringFromAny(iconInfo["icon_background"]) != "#DCFCE7" {
		t.Fatalf("expected app update to sync dataset icon info, got %+v", iconInfo)
	}
	if stringFromAny(iconInfo["icon_url"]) != "" {
		t.Fatalf("expected app emoji icon update to clear dataset icon url, got %+v", iconInfo)
	}

	updatedSite := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+dataset.PipelineID+"/site", map[string]any{
		"title":               "Site Driven Pipeline",
		"description":         "metadata updated from site config",
		"icon_type":           "emoji",
		"icon":                "🛰",
		"icon_background":     "#DBEAFE",
		"show_workflow_steps": false,
	}, true, http.StatusOK)
	updatedSiteData := mapFromAny(updatedSite["site"])
	if stringFromAny(updatedSite["name"]) != "Site Driven Pipeline" || stringFromAny(updatedSiteData["title"]) != "Site Driven Pipeline" {
		t.Fatalf("unexpected updated app site metadata: %+v", updatedSite)
	}

	datasetDetailAfterSite := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if stringFromAny(datasetDetailAfterSite["name"]) != "Site Driven Pipeline" || stringFromAny(datasetDetailAfterSite["description"]) != "metadata updated from site config" {
		t.Fatalf("expected app site update to sync dataset metadata, got %+v", datasetDetailAfterSite)
	}
	iconInfoAfterSite := mapFromAny(datasetDetailAfterSite["icon_info"])
	if stringFromAny(iconInfoAfterSite["icon_type"]) != "emoji" || stringFromAny(iconInfoAfterSite["icon"]) != "🛰" || stringFromAny(iconInfoAfterSite["icon_background"]) != "#DBEAFE" {
		t.Fatalf("expected app site update to sync dataset icon info, got %+v", iconInfoAfterSite)
	}
}

func TestRAGPipelinePublishCopyAndDeleteStayLinked(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes":    []map[string]any{{"id": "source-node", "data": map[string]any{"title": "Source", "type": "start"}}},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name": "copied-pipeline-v1",
	}, true, http.StatusOK)

	persistedDataset := findPersistedDatasetByID(t, env, dataset.ID)
	if !ragPipelineBoolValue(persistedDataset["is_published"]) {
		t.Fatalf("expected publish to persist dataset is_published, got %+v", persistedDataset)
	}

	copiedApp := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+dataset.PipelineID+"/copy", map[string]any{
		"name":            "Copied Pipeline App",
		"description":     "copied from app api",
		"mode":            "workflow",
		"icon_type":       "emoji",
		"icon":            "🧪",
		"icon_background": "#FDE68A",
	}, true, http.StatusCreated)
	copiedAppID := stringFromAny(copiedApp["id"])
	if copiedAppID == "" {
		t.Fatalf("expected copied app id, got %+v", copiedApp)
	}

	copiedDataset := findPersistedDatasetByPipelineID(t, env, copiedAppID)
	if stringFromAny(copiedDataset["name"]) != "Copied Pipeline App" || stringFromAny(copiedDataset["description"]) != "copied from app api" {
		t.Fatalf("expected copied linked dataset metadata, got %+v", copiedDataset)
	}
	if len(objectListFromAny(copiedDataset["documents"])) != 0 {
		t.Fatalf("expected copied linked dataset to start empty, got %+v", copiedDataset["documents"])
	}

	copiedDatasetID := stringFromAny(copiedDataset["id"])
	copiedDatasetDetail := getJSON[map[string]any](env, "/console/api/datasets/"+copiedDatasetID, true, http.StatusOK)
	if stringFromAny(copiedDatasetDetail["pipeline_id"]) != copiedAppID || stringFromAny(copiedDatasetDetail["name"]) != "Copied Pipeline App" {
		t.Fatalf("expected copied dataset detail to stay linked to copied app, got %+v", copiedDatasetDetail)
	}

	copiedDraft := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+copiedAppID+"/workflows/draft", true, http.StatusOK)
	if len(anySlice(copiedDraft.Graph["nodes"])) != 1 || copiedDraft.ID == "" {
		t.Fatalf("expected copied pipeline draft to remain accessible, got %+v", copiedDraft)
	}

	postJSON[map[string]any](env, http.MethodDelete, "/console/api/apps/"+copiedAppID, nil, true, http.StatusNoContent)

	missingDataset := getJSON[errorResponse](env, "/console/api/datasets/"+copiedDatasetID, true, http.StatusNotFound)
	if missingDataset.Code != "dataset_not_found" {
		t.Fatalf("expected linked dataset to be deleted with app, got %+v", missingDataset)
	}
}

func TestRAGPipelineDraftSyncAndRestoreKeepLinkedDatasetSettingsInSync(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)

	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "knowledge-node-v1",
					"data": map[string]any{
						"title":                    "Knowledge Index V1",
						"type":                     "knowledge-index",
						"chunk_structure":          "qa_model",
						"indexing_technique":       "high_quality",
						"embedding_model":          "text-embedding-3-large",
						"embedding_model_provider": "openai",
						"retrieval_model": map[string]any{
							"search_method":           "semantic_search",
							"top_k":                   6,
							"score_threshold":         0.61,
							"score_threshold_enabled": true,
						},
						"summary_index_setting": map[string]any{
							"enable":              true,
							"model_name":          "gpt-4o-mini",
							"model_provider_name": "openai",
							"summary_prompt":      "Summarize knowledge chunks",
						},
					},
				},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{},
	}, true, http.StatusOK)

	detailV1 := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if stringFromAny(detailV1["doc_form"]) != "qa_model" || stringFromAny(detailV1["indexing_technique"]) != "high_quality" {
		t.Fatalf("expected first draft sync to update dataset chunk settings, got %+v", detailV1)
	}
	if stringFromAny(detailV1["embedding_model"]) != "text-embedding-3-large" || stringFromAny(detailV1["embedding_model_provider"]) != "openai" {
		t.Fatalf("expected first draft sync to update embedding config, got %+v", detailV1)
	}
	retrievalV1 := mapFromAny(detailV1["retrieval_model"])
	if topK, ok := ragPipelineIntValue(retrievalV1["top_k"]); !ok || topK != 6 || !ragPipelineBoolValue(retrievalV1["score_threshold_enabled"]) {
		t.Fatalf("expected first draft sync to update retrieval model, got %+v", retrievalV1)
	}
	summaryV1 := mapFromAny(detailV1["summary_index_setting"])
	if !ragPipelineBoolValue(summaryV1["enable"]) || stringFromAny(summaryV1["model_name"]) != "gpt-4o-mini" {
		t.Fatalf("expected first draft sync to update summary index settings, got %+v", summaryV1)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name": "v1",
	}, true, http.StatusOK)

	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "knowledge-node-v2",
					"data": map[string]any{
						"title":                    "Knowledge Index V2",
						"type":                     "knowledge-index",
						"chunk_structure":          "hierarchical_model",
						"indexing_technique":       "economy",
						"embedding_model":          "text-embedding-3-small",
						"embedding_model_provider": "openai",
						"retrieval_model": map[string]any{
							"search_method":           "keyword_search",
							"top_k":                   3,
							"score_threshold":         0.2,
							"score_threshold_enabled": false,
						},
						"summary_index_setting": map[string]any{
							"enable":              false,
							"model_name":          "gpt-4.1-mini",
							"model_provider_name": "openai",
							"summary_prompt":      "Unused summary prompt",
						},
					},
				},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 10, "y": 20, "zoom": 0.9},
		},
		"features": map[string]any{},
	}, true, http.StatusOK)

	detailV2 := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if stringFromAny(detailV2["doc_form"]) != "hierarchical_model" || stringFromAny(detailV2["indexing_technique"]) != "economy" {
		t.Fatalf("expected second draft sync to update dataset chunk settings, got %+v", detailV2)
	}
	retrievalV2 := mapFromAny(detailV2["retrieval_model"])
	if topK, ok := ragPipelineIntValue(retrievalV2["top_k"]); !ok || topK != 3 || ragPipelineBoolValue(retrievalV2["score_threshold_enabled"]) {
		t.Fatalf("expected second draft sync to update retrieval model, got %+v", retrievalV2)
	}
	summaryV2 := mapFromAny(detailV2["summary_index_setting"])
	if ragPipelineBoolValue(summaryV2["enable"]) {
		t.Fatalf("expected second draft sync to disable summary index, got %+v", summaryV2)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name": "v2",
	}, true, http.StatusOK)

	versionList := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows?page=1&limit=10", true, http.StatusOK)
	versionV1 := findWorkflowVersionByMarkedName(t, objectListFromAny(versionList["items"]), "v1")
	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/"+stringFromAny(versionV1["id"])+"/restore", nil, true, http.StatusOK)

	restoredDraft := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", true, http.StatusOK)
	restoredNodeData := ragPipelineKnowledgeIndexNodeData(restoredDraft.Graph)
	if stringFromAny(restoredNodeData["chunk_structure"]) != "qa_model" {
		t.Fatalf("expected restored draft graph to roll back to v1 knowledge settings, got %+v", restoredDraft.Graph)
	}

	restoredDetail := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID, true, http.StatusOK)
	if stringFromAny(restoredDetail["doc_form"]) != "qa_model" || stringFromAny(restoredDetail["embedding_model"]) != "text-embedding-3-large" {
		t.Fatalf("expected restore to sync dataset back to v1 settings, got %+v", restoredDetail)
	}
	restoredRetrieval := mapFromAny(restoredDetail["retrieval_model"])
	if topK, ok := ragPipelineIntValue(restoredRetrieval["top_k"]); !ok || topK != 6 || !ragPipelineBoolValue(restoredRetrieval["score_threshold_enabled"]) {
		t.Fatalf("expected restore to sync retrieval model back to v1, got %+v", restoredRetrieval)
	}
	restoredSummary := mapFromAny(restoredDetail["summary_index_setting"])
	if !ragPipelineBoolValue(restoredSummary["enable"]) || stringFromAny(restoredSummary["model_name"]) != "gpt-4o-mini" {
		t.Fatalf("expected restore to sync summary index settings back to v1, got %+v", restoredSummary)
	}
}

func TestRAGPipelineExportAndImportRoundTrip(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[ragPipelineDatasetResponse](env, http.MethodPatch, "/console/api/datasets/"+dataset.ID, map[string]any{
		"name":        "Imported FAQ Pipeline",
		"description": "round trip export",
		"icon_info": map[string]any{
			"icon":            "🧠",
			"icon_type":       "emoji",
			"icon_background": "#FED7AA",
		},
	}, true, http.StatusOK)

	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "knowledge-node",
					"data": map[string]any{
						"title":                    "Knowledge Index",
						"type":                     "knowledge-index",
						"chunk_structure":          "qa_model",
						"indexing_technique":       "high_quality",
						"embedding_model":          "text-embedding-3-large",
						"embedding_model_provider": "openai",
						"retrieval_model": map[string]any{
							"search_method":           "semantic_search",
							"top_k":                   6,
							"score_threshold":         0.61,
							"score_threshold_enabled": true,
						},
					},
				},
				{
					"id": "tool-node",
					"data": map[string]any{
						"title":                    "Plugin Tool",
						"type":                     "tool",
						"plugin_unique_identifier": "langgenius/demo:tool@1.0.0",
					},
				},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 10, "y": 20, "zoom": 0.9},
		},
		"features": map[string]any{"opening_statement": "hello"},
		"environment_variables": []map[string]any{
			{"id": "env-limit", "name": "limit", "value_type": "number", "value": 5},
		},
		"conversation_variables": []map[string]any{
			{"id": "conv-mode", "name": "mode", "value_type": "string", "value": "draft"},
		},
		"rag_pipeline_variables": []map[string]any{
			{"belong_to_node_id": "shared", "label": "Shared Query", "variable": "shared_query", "type": "text-input", "required": true},
			{"belong_to_node_id": "knowledge-node", "label": "Question", "variable": "question", "type": "paragraph", "required": true},
		},
	}, true, http.StatusOK)

	exported := getJSON[ragPipelineExportResponse](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/exports?include_secret=false", true, http.StatusOK)
	if !strings.Contains(exported.Data, "kind: rag_pipeline") {
		t.Fatalf("expected exported DSL kind, got %q", exported.Data)
	}
	if !strings.Contains(exported.Data, "Imported FAQ Pipeline") || !strings.Contains(exported.Data, "langgenius/demo:tool@1.0.0") {
		t.Fatalf("expected exported DSL metadata and dependencies, got %q", exported.Data)
	}

	imported := postJSON[ragPipelineImportResponse](env, http.MethodPost, "/console/api/rag/pipelines/imports", map[string]any{
		"mode":         "yaml-content",
		"yaml_content": exported.Data,
	}, true, http.StatusOK)
	if imported.Status != "completed" || imported.DatasetID == "" || imported.PipelineID == "" {
		t.Fatalf("unexpected pipeline import response: %+v", imported)
	}
	if imported.CurrentDSLVersion == "" || imported.ImportedDSLVersion == "" {
		t.Fatalf("expected import versions, got %+v", imported)
	}

	importedDataset := getJSON[ragPipelineDatasetResponse](env, "/console/api/datasets/"+imported.DatasetID, true, http.StatusOK)
	if importedDataset.Name != "Imported FAQ Pipeline" || importedDataset.Description != "round trip export" {
		t.Fatalf("unexpected imported dataset metadata: %+v", importedDataset)
	}
	if importedDataset.DocForm != "qa_model" || importedDataset.IndexingTechnique != "high_quality" {
		t.Fatalf("expected imported dataset settings from knowledge node, got %+v", importedDataset)
	}
	if importedDataset.EmbeddingModel != "text-embedding-3-large" || importedDataset.EmbeddingProvider != "openai" {
		t.Fatalf("expected imported embedding settings, got %+v", importedDataset)
	}

	importedDraft := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+imported.PipelineID+"/workflows/draft", true, http.StatusOK)
	if len(anySlice(importedDraft.Graph["nodes"])) != 2 {
		t.Fatalf("expected imported graph nodes, got %+v", importedDraft.Graph)
	}
	if len(importedDraft.EnvironmentVariables) != 1 || len(importedDraft.ConversationVariables) != 1 || len(importedDraft.RagPipelineVariables) != 2 {
		t.Fatalf("expected imported workflow variables, got %+v", importedDraft)
	}
	if stringFromAny(importedDraft.Features["opening_statement"]) != "hello" {
		t.Fatalf("expected imported features, got %+v", importedDraft.Features)
	}
}

func TestRAGPipelineDatasetCreateFromDSL(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	yamlContent := strings.TrimSpace(`
version: 0.1.0
kind: rag_pipeline
rag_pipeline:
  name: Ingest QA Pipeline
  icon: "🧪"
  icon_type: emoji
  icon_background: "#123456"
  description: created from yaml
workflow:
  graph:
    nodes:
      - id: knowledge-node
        data:
          title: Knowledge Index
          type: knowledge-index
          chunk_structure: qa_model
          indexing_technique: high_quality
          embedding_model: text-embedding-3-large
          embedding_model_provider: openai
          retrieval_model:
            search_method: semantic_search
            top_k: 6
            score_threshold: 0.61
            score_threshold_enabled: true
          summary_index_setting:
            enable: true
            model_name: gpt-4o-mini
            model_provider_name: openai
            summary_prompt: Summarize chunks
    edges: []
    viewport:
      x: 12
      y: 34
      zoom: 0.8
  features:
    opening_statement: hello
  environment_variables:
    - id: env-lang
      name: language
      value_type: string
      value: zh-CN
  conversation_variables:
    - id: conv-channel
      name: channel
      value_type: string
      value: web
  rag_pipeline_variables:
    - belong_to_node_id: shared
      label: Shared Query
      variable: shared_query
      type: text-input
      required: true
    - belong_to_node_id: knowledge-node
      label: Upload
      variable: upload
      type: file
      required: false
`)

	created := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/dataset", map[string]any{
		"yaml_content": yamlContent,
	}, true, http.StatusCreated)
	if created.ID == "" || created.PipelineID == "" {
		t.Fatalf("expected created dataset ids, got %+v", created)
	}
	if created.Name != "Ingest QA Pipeline" || created.Description != "created from yaml" {
		t.Fatalf("unexpected created dataset metadata: %+v", created)
	}
	if created.DocForm != "qa_model" || created.IndexingTechnique != "high_quality" {
		t.Fatalf("expected created dataset settings from DSL, got %+v", created)
	}
	if created.EmbeddingModel != "text-embedding-3-large" || created.EmbeddingProvider != "openai" {
		t.Fatalf("expected created embedding settings, got %+v", created)
	}

	draft := getJSON[workflowDraftResponse](env, "/console/api/rag/pipelines/"+created.PipelineID+"/workflows/draft", true, http.StatusOK)
	if len(anySlice(draft.Graph["nodes"])) != 1 {
		t.Fatalf("expected imported draft node, got %+v", draft.Graph)
	}
	if len(draft.EnvironmentVariables) != 1 || len(draft.ConversationVariables) != 1 || len(draft.RagPipelineVariables) != 2 {
		t.Fatalf("expected imported variable lists, got %+v", draft)
	}
	if stringFromAny(draft.Features["opening_statement"]) != "hello" {
		t.Fatalf("unexpected imported features: %+v", draft.Features)
	}
}

func TestRAGPipelineBuiltInTemplateListAndDetail(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	list := getJSON[pipelineTemplateListResponse](env, "/console/api/rag/pipeline/templates?type=built-in&language=zh-Hans", true, http.StatusOK)
	if len(list.PipelineTemplates) == 0 {
		t.Fatal("expected built-in pipeline templates")
	}

	first := list.PipelineTemplates[0]
	if first.ID == "" || first.Name == "" || first.ChunkStructure == "" {
		t.Fatalf("unexpected built-in pipeline template summary: %+v", first)
	}

	detail := getJSON[pipelineTemplateDetailResponse](env, "/console/api/rag/pipeline/templates/"+first.ID+"?type=built-in&language=zh-Hans", true, http.StatusOK)
	if detail.ID != first.ID || detail.Name == "" || detail.ExportData == "" {
		t.Fatalf("unexpected built-in pipeline template detail: %+v", detail)
	}
	if !strings.Contains(detail.ExportData, "kind: rag_pipeline") {
		t.Fatalf("expected built-in export data to be rag pipeline DSL, got %q", detail.ExportData)
	}
	if len(anySlice(detail.Graph["nodes"])) == 0 {
		t.Fatalf("expected built-in graph nodes, got %+v", detail.Graph)
	}
}

func TestRAGPipelineCustomizedTemplateCRUD(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "knowledge-node",
					"data": map[string]any{
						"title":                    "Knowledge Index",
						"type":                     "knowledge-index",
						"chunk_structure":          "hierarchical_model",
						"indexing_technique":       "high_quality",
						"embedding_model":          "text-embedding-3-large",
						"embedding_model_provider": "openai",
						"retrieval_model": map[string]any{
							"search_method":           "semantic_search",
							"top_k":                   6,
							"score_threshold":         0.61,
							"score_threshold_enabled": true,
						},
					},
				},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{"opening_statement": "template workflow"},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name": "v1",
	}, true, http.StatusOK)

	publishResult := postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/customized/publish", map[string]any{
		"name":        "My Customized Template",
		"description": "saved from pipeline",
		"icon_info": map[string]any{
			"icon_type":       "emoji",
			"icon":            "🧪",
			"icon_background": "#123456",
		},
	}, true, http.StatusOK)
	if stringFromAny(publishResult["result"]) != "success" {
		t.Fatalf("unexpected publish-as-template result: %+v", publishResult)
	}

	list := getJSON[pipelineTemplateListResponse](env, "/console/api/rag/pipeline/templates?type=customized", true, http.StatusOK)
	if len(list.PipelineTemplates) != 1 {
		t.Fatalf("expected one customized template, got %+v", list.PipelineTemplates)
	}
	templateID := list.PipelineTemplates[0].ID
	if list.PipelineTemplates[0].Name != "My Customized Template" || list.PipelineTemplates[0].ChunkStructure != "hierarchical_model" {
		t.Fatalf("unexpected customized template summary: %+v", list.PipelineTemplates[0])
	}

	detail := getJSON[pipelineTemplateDetailResponse](env, "/console/api/rag/pipeline/templates/"+templateID+"?type=customized", true, http.StatusOK)
	if detail.CreatedBy != "Tester" {
		t.Fatalf("expected created_by to resolve to user name, got %+v", detail)
	}
	if !strings.Contains(detail.ExportData, "My Customized Template") || !strings.Contains(detail.ExportData, "saved from pipeline") {
		t.Fatalf("expected stored DSL metadata in customized template, got %q", detail.ExportData)
	}
	if len(anySlice(detail.Graph["nodes"])) != 1 {
		t.Fatalf("expected customized template graph to persist, got %+v", detail.Graph)
	}

	updateResult := postJSON[map[string]any](env, http.MethodPatch, "/console/api/rag/pipeline/customized/templates/"+templateID, map[string]any{
		"name":        "Updated Template",
		"description": "updated description",
		"icon_info": map[string]any{
			"icon_type":       "emoji",
			"icon":            "🛠",
			"icon_background": "#654321",
		},
	}, true, http.StatusOK)
	if stringFromAny(updateResult["name"]) != "Updated Template" {
		t.Fatalf("unexpected update template response: %+v", updateResult)
	}

	exported := postJSON[ragPipelineExportResponse](env, http.MethodPost, "/console/api/rag/pipeline/customized/templates/"+templateID, nil, true, http.StatusOK)
	if !strings.Contains(exported.Data, "Updated Template") || !strings.Contains(exported.Data, "updated description") {
		t.Fatalf("expected exported customized template DSL to reflect updated metadata, got %q", exported.Data)
	}

	postJSON[map[string]any](env, http.MethodDelete, "/console/api/rag/pipeline/customized/templates/"+templateID, nil, true, http.StatusOK)

	listAfterDelete := getJSON[pipelineTemplateListResponse](env, "/console/api/rag/pipeline/templates?type=customized", true, http.StatusOK)
	if len(listAfterDelete.PipelineTemplates) != 0 {
		t.Fatalf("expected customized templates to be deleted, got %+v", listAfterDelete.PipelineTemplates)
	}
}

func TestDatasetIndexingEstimatePreviewModes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	hierarchical := postJSON[indexingEstimatePreviewResponse](env, http.MethodPost, "/console/api/datasets/indexing-estimate", map[string]any{
		"dataset_id":   "preview-dataset",
		"doc_form":     "hierarchical_model",
		"doc_language": "English",
		"process_rule": map[string]any{
			"mode": "custom",
			"rules": map[string]any{
				"parent_mode": "full-doc",
			},
		},
		"info_list": map[string]any{
			"data_source_type": "website_crawl",
			"website_info_list": map[string]any{
				"provider": "firecrawl",
				"job_id":   "job-preview",
				"urls": []string{
					"https://example.com/alpha",
					"https://example.com/beta",
				},
			},
		},
	}, true, http.StatusOK)
	if hierarchical.TotalNodes != 2 || hierarchical.ChunkStructure != "hierarchical_model" || hierarchical.ParentMode != "full-doc" {
		t.Fatalf("unexpected hierarchical estimate payload: %+v", hierarchical)
	}
	if len(hierarchical.Preview) != 2 || len(hierarchical.Preview[0].ChildChunks) == 0 {
		t.Fatalf("expected hierarchical preview with child chunks, got %+v", hierarchical.Preview)
	}

	qa := postJSON[indexingEstimatePreviewResponse](env, http.MethodPost, "/console/api/datasets/indexing-estimate", map[string]any{
		"dataset_id":   "preview-dataset",
		"doc_form":     "qa_model",
		"doc_language": "English",
		"process_rule": map[string]any{
			"mode": "custom",
			"rules": map[string]any{
				"parent_mode": "paragraph",
			},
		},
		"info_list": map[string]any{
			"data_source_type": "notion_import",
			"notion_info_list": []map[string]any{
				{
					"workspace_id":  "workspace-1",
					"credential_id": "credential-1",
					"pages": []map[string]any{
						{
							"page_id":   "page-1",
							"page_name": "Architecture Notes",
							"type":      "page",
						},
						{
							"page_id":   "page-2",
							"page_name": "Runbook",
							"type":      "page",
						},
					},
				},
			},
		},
	}, true, http.StatusOK)
	if qa.TotalNodes != 2 || qa.ChunkStructure != "qa_model" {
		t.Fatalf("unexpected qa estimate payload: %+v", qa)
	}
	if len(qa.QAPreview) != 4 || len(qa.Preview) != len(qa.QAPreview) {
		t.Fatalf("expected qa preview pairs for each datasource item, got %+v", qa)
	}
	if qa.QAPreview[0].Question == "" || qa.QAPreview[0].Answer == "" {
		t.Fatalf("expected qa preview entries to be populated, got %+v", qa.QAPreview)
	}
}

func TestRAGPipelinePublishedRunCreatePreviewAndReprocess(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	dataset := postJSON[ragPipelineDatasetResponse](env, http.MethodPost, "/console/api/rag/pipeline/empty-dataset", nil, true, http.StatusCreated)
	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{
					"id": "knowledge-node",
					"data": map[string]any{
						"title":                    "Knowledge Index",
						"type":                     "knowledge-index",
						"chunk_structure":          "hierarchical_model",
						"indexing_technique":       "high_quality",
						"embedding_model":          "text-embedding-3-large",
						"embedding_model_provider": "openai",
						"retrieval_model": map[string]any{
							"search_method":           "semantic_search",
							"top_k":                   4,
							"score_threshold_enabled": false,
							"score_threshold":         0.5,
						},
						"summary_index_setting": map[string]any{
							"enable": false,
						},
					},
				},
			},
			"edges":    []any{},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/publish", map[string]any{
		"marked_name": "published-v1",
	}, true, http.StatusOK)

	upload := env.uploadFile("/console/api/files/upload", true, "pipeline-source.md", "text/markdown", []byte("# Pipeline Source\n\nhello rag pipeline\n"))
	previewPayload := map[string]any{
		"pipeline_id":     dataset.PipelineID,
		"inputs":          ragPipelineRunInputs("Chinese", 800, true),
		"start_node_id":   "datasource-local-file",
		"datasource_type": "local_file",
		"datasource_info_list": []map[string]any{{
			"related_id": upload.ID,
			"name":       upload.Name,
			"extension":  "md",
		}},
		"is_preview": true,
	}

	preview := postJSON[publishedPipelineRunPreviewResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/run", previewPayload, true, http.StatusOK)
	if preview.TaskIOD == "" || preview.WorkflowRunID == "" || preview.Data.WorkflowID == "" {
		t.Fatalf("expected preview workflow ids, got %+v", preview)
	}
	if preview.Data.Status != "succeeded" {
		t.Fatalf("unexpected preview run status: %+v", preview.Data)
	}
	if totalNodes, ok := preview.Data.Outputs["total_nodes"].(float64); !ok || int(totalNodes) != 1 {
		t.Fatalf("unexpected preview outputs: %+v", preview.Data.Outputs)
	}
	if stringFromAny(preview.Data.Outputs["chunk_structure"]) != "hierarchical_model" || stringFromAny(preview.Data.Outputs["parent_mode"]) != "paragraph" {
		t.Fatalf("expected hierarchical preview metadata, got %+v", preview.Data.Outputs)
	}
	previewChunks := anySlice(preview.Data.Outputs["preview"])
	if len(previewChunks) == 0 || len(anySlice(mapFromAny(previewChunks[0])["child_chunks"])) == 0 {
		t.Fatalf("expected preview output to include parent-child chunk slices, got %+v", preview.Data.Outputs)
	}

	previewDetail := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+preview.WorkflowRunID, true, http.StatusOK)
	if !strings.Contains(stringFromAny(previewDetail["inputs"]), "\"datasource_type\":\"local_file\"") || !strings.Contains(stringFromAny(previewDetail["inputs"]), "\"is_preview\":true") {
		t.Fatalf("expected preview run detail to persist pipeline inputs, got %+v", previewDetail)
	}
	if !strings.Contains(stringFromAny(previewDetail["outputs"]), "\"mode\":\"preview\"") || !strings.Contains(stringFromAny(previewDetail["outputs"]), "\"preview_result\"") || !strings.Contains(stringFromAny(previewDetail["outputs"]), "\"chunk_structure\":\"hierarchical_model\"") {
		t.Fatalf("expected preview run detail to persist preview outputs, got %+v", previewDetail)
	}

	previewNodeExecutions := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+preview.WorkflowRunID+"/node-executions", true, http.StatusOK)
	previewNodes := objectListFromAny(previewNodeExecutions["data"])
	if len(previewNodes) != 1 {
		t.Fatalf("expected one preview node execution, got %+v", previewNodes)
	}
	previewNode := previewNodes[0]
	if stringFromAny(previewNode["node_id"]) != "knowledge-node" || stringFromAny(previewNode["node_type"]) != "knowledge-index" {
		t.Fatalf("unexpected preview node execution: %+v", previewNode)
	}
	if stringFromAny(mapFromAny(previewNode["inputs"])["datasource_type"]) != "local_file" {
		t.Fatalf("expected preview node execution to persist datasource context, got %+v", previewNode["inputs"])
	}
	if stringFromAny(mapFromAny(previewNode["outputs"])["mode"]) != "preview" || len(mapFromAny(mapFromAny(previewNode["outputs"])["preview_result"])) == 0 {
		t.Fatalf("expected preview node execution to persist preview outputs, got %+v", previewNode["outputs"])
	}
	if stringFromAny(mapFromAny(mapFromAny(previewNode["outputs"])["preview_result"])["chunk_structure"]) != "hierarchical_model" {
		t.Fatalf("expected preview node execution to persist chunk structure, got %+v", previewNode["outputs"])
	}

	createPayload := map[string]any{
		"pipeline_id":     dataset.PipelineID,
		"inputs":          ragPipelineRunInputs("Chinese", 800, true),
		"start_node_id":   "datasource-local-file",
		"datasource_type": "local_file",
		"datasource_info_list": []map[string]any{{
			"related_id": upload.ID,
			"name":       upload.Name,
			"extension":  "md",
		}},
		"is_preview": false,
	}
	created := postJSON[publishedPipelineRunResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/run", createPayload, true, http.StatusOK)
	if created.Batch == "" || created.Dataset.ID != dataset.ID {
		t.Fatalf("unexpected published run create response: %+v", created)
	}
	if len(created.Documents) != 1 || created.Documents[0].ID == "" {
		t.Fatalf("expected one created document, got %+v", created.Documents)
	}
	if created.Documents[0].DataSourceType != "local_file" || !created.Documents[0].Enable {
		t.Fatalf("unexpected initial document payload: %+v", created.Documents[0])
	}
	if created.Documents[0].Position != 1 {
		t.Fatalf("expected first created document position to be 1, got %+v", created.Documents[0])
	}
	if created.Documents[0].IndexingStatus != "waiting" {
		t.Fatalf("expected newly created document to start in waiting status, got %+v", created.Documents[0])
	}

	runHistory := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs", true, http.StatusOK)
	createRun := findWorkflowRunByBatch(t, objectListFromAny(runHistory["data"]), created.Batch)
	if stringFromAny(mapFromAny(createRun["inputs"])["datasource_type"]) != "local_file" {
		t.Fatalf("expected create run history to persist datasource type, got %+v", createRun)
	}
	if stringFromAny(mapFromAny(createRun["outputs"])["mode"]) != "create" {
		t.Fatalf("expected create run history to persist create mode, got %+v", createRun)
	}
	documentIDs, ok := mapFromAny(createRun["outputs"])["document_ids"].([]any)
	if !ok || len(documentIDs) != 1 {
		t.Fatalf("expected create run history to persist document ids, got %+v", createRun)
	}

	createRunID := stringFromAny(createRun["id"])
	createRunDetail := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+createRunID, true, http.StatusOK)
	if !strings.Contains(stringFromAny(createRunDetail["inputs"]), "\"pipeline_id\":\""+dataset.PipelineID+"\"") || !strings.Contains(stringFromAny(createRunDetail["inputs"]), "\"start_node_id\":\"datasource-local-file\"") {
		t.Fatalf("expected create run detail to persist pipeline inputs, got %+v", createRunDetail)
	}
	if !strings.Contains(stringFromAny(createRunDetail["outputs"]), "\"mode\":\"create\"") || !strings.Contains(stringFromAny(createRunDetail["outputs"]), "\"batch\":\""+created.Batch+"\"") {
		t.Fatalf("expected create run detail to persist batch outputs, got %+v", createRunDetail)
	}

	createNodeExecutions := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+createRunID+"/node-executions", true, http.StatusOK)
	createNodes := objectListFromAny(createNodeExecutions["data"])
	if len(createNodes) != 1 {
		t.Fatalf("expected one create node execution, got %+v", createNodes)
	}
	createNode := createNodes[0]
	if stringFromAny(mapFromAny(createNode["outputs"])["batch"]) != created.Batch {
		t.Fatalf("expected create node execution to persist batch id, got %+v", createNode["outputs"])
	}
	if stringFromAny(mapFromAny(createNode["process_data"])["mode"]) != "create" {
		t.Fatalf("expected create node execution to persist create mode, got %+v", createNode["process_data"])
	}

	documentID := created.Documents[0].ID
	documentStatus := getJSON[indexingStatusResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/indexing-status", true, http.StatusOK)
	if documentStatus.IndexingStatus != "parsing" {
		t.Fatalf("expected first indexing poll to move into parsing, got %+v", documentStatus)
	}
	if documentStatus.CompletedSegments != 0 || documentStatus.TotalSegments < 2 {
		t.Fatalf("unexpected initial indexing progress after parsing poll: %+v", documentStatus)
	}

	expectedBatchStatuses := []string{"cleaning", "splitting", "indexing", "completed"}
	lastBatchStatus := indexingStatusResponse{}
	for i, expectedStatus := range expectedBatchStatuses {
		batchStatus := getJSON[indexingStatusBatchResponse](env, "/console/api/datasets/"+dataset.ID+"/batch/"+created.Batch+"/indexing-status", true, http.StatusOK)
		if len(batchStatus.Data) != 1 {
			t.Fatalf("expected single batch indexing status item, got %+v", batchStatus.Data)
		}
		lastBatchStatus = batchStatus.Data[0]
		if lastBatchStatus.IndexingStatus != expectedStatus {
			t.Fatalf("unexpected batch status at step %d: got %+v want %s", i, lastBatchStatus, expectedStatus)
		}
	}
	if lastBatchStatus.CompletedSegments != lastBatchStatus.TotalSegments || lastBatchStatus.CompletedAt == nil {
		t.Fatalf("expected completed batch status to finish all segments, got %+v", lastBatchStatus)
	}

	documentDetail := getJSON[map[string]any](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"?metadata=without", true, http.StatusOK)
	if stringFromAny(documentDetail["display_status"]) != "available" || stringFromAny(documentDetail["indexing_status"]) != "completed" {
		t.Fatalf("expected document detail to become available after indexing completion, got %+v", documentDetail)
	}

	executionLog := getJSON[pipelineExecutionLogResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/pipeline-execution-log", true, http.StatusOK)
	if executionLog.DatasourceType != "local_file" || executionLog.DatasourceNodeID != "datasource-local-file" {
		t.Fatalf("unexpected execution log metadata: %+v", executionLog)
	}
	if stringFromAny(executionLog.DatasourceInfo["related_id"]) != upload.ID {
		t.Fatalf("expected execution log to store uploaded file id, got %+v", executionLog.DatasourceInfo)
	}
	if stringFromAny(executionLog.InputData["doc_language"]) != "Chinese" {
		t.Fatalf("expected execution log to persist doc language, got %+v", executionLog.InputData)
	}
	if enabled, ok := executionLog.InputData["remove_extra_spaces"].(bool); !ok || !enabled {
		t.Fatalf("expected execution log to persist preprocessing flags, got %+v", executionLog.InputData)
	}

	reprocessPayload := map[string]any{
		"pipeline_id":     dataset.PipelineID,
		"inputs":          ragPipelineRunInputs("Japanese", 640, false),
		"start_node_id":   "datasource-local-file-reprocess",
		"datasource_type": "local_file",
		"datasource_info_list": []map[string]any{{
			"related_id": upload.ID,
			"name":       upload.Name,
			"extension":  "md",
		}},
		"original_document_id": documentID,
		"is_preview":           false,
	}
	reprocessed := postJSON[publishedPipelineRunResponse](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflows/published/run", reprocessPayload, true, http.StatusOK)
	if len(reprocessed.Documents) != 1 || reprocessed.Documents[0].ID != documentID {
		t.Fatalf("expected reprocess to update existing document, got %+v", reprocessed.Documents)
	}
	if reprocessed.Documents[0].IndexingStatus != "waiting" {
		t.Fatalf("expected reprocessed document to re-enter waiting status, got %+v", reprocessed.Documents[0])
	}

	runHistory = getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs", true, http.StatusOK)
	reprocessRun := findWorkflowRunByOriginalDocumentID(t, objectListFromAny(runHistory["data"]), documentID)
	if stringFromAny(mapFromAny(reprocessRun["outputs"])["mode"]) != "reprocess" {
		t.Fatalf("expected reprocess run history to persist reprocess mode, got %+v", reprocessRun)
	}
	reprocessTaskID := stringFromAny(reprocessRun["task_id"])
	if reprocessTaskID == "" {
		t.Fatalf("expected reprocess run history to expose task id, got %+v", reprocessRun)
	}

	reprocessRunID := stringFromAny(reprocessRun["id"])
	reprocessRunDetail := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+reprocessRunID, true, http.StatusOK)
	if stringFromAny(reprocessRunDetail["task_id"]) != reprocessTaskID {
		t.Fatalf("expected reprocess run detail to expose matching task id, got %+v", reprocessRunDetail)
	}
	if !strings.Contains(stringFromAny(reprocessRunDetail["inputs"]), "\"original_document_id\":\""+documentID+"\"") {
		t.Fatalf("expected reprocess run detail to persist original document id, got %+v", reprocessRunDetail)
	}
	if !strings.Contains(stringFromAny(reprocessRunDetail["outputs"]), "\"mode\":\"reprocess\"") || !strings.Contains(stringFromAny(reprocessRunDetail["outputs"]), "\"batch\":\""+reprocessed.Batch+"\"") {
		t.Fatalf("expected reprocess run detail to persist reprocess outputs, got %+v", reprocessRunDetail)
	}

	reprocessNodeExecutions := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+reprocessRunID+"/node-executions", true, http.StatusOK)
	reprocessNodes := objectListFromAny(reprocessNodeExecutions["data"])
	if len(reprocessNodes) != 1 {
		t.Fatalf("expected one reprocess node execution, got %+v", reprocessNodes)
	}
	reprocessNode := reprocessNodes[0]
	if stringFromAny(mapFromAny(reprocessNode["outputs"])["batch"]) != reprocessed.Batch {
		t.Fatalf("expected reprocess node execution to persist updated batch id, got %+v", reprocessNode["outputs"])
	}
	if stringFromAny(mapFromAny(reprocessNode["process_data"])["mode"]) != "reprocess" {
		t.Fatalf("expected reprocess node execution to persist reprocess mode, got %+v", reprocessNode["process_data"])
	}

	reprocessedLog := getJSON[pipelineExecutionLogResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/pipeline-execution-log", true, http.StatusOK)
	if reprocessedLog.DatasourceNodeID != "datasource-local-file-reprocess" {
		t.Fatalf("expected reprocess node id to persist, got %+v", reprocessedLog)
	}
	if stringFromAny(reprocessedLog.InputData["doc_language"]) != "Japanese" {
		t.Fatalf("expected reprocess to update input data, got %+v", reprocessedLog.InputData)
	}
	if enabled, ok := reprocessedLog.InputData["remove_extra_spaces"].(bool); !ok || enabled {
		t.Fatalf("expected updated preprocessing flag, got %+v", reprocessedLog.InputData)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/tasks/"+reprocessTaskID+"/stop", nil, true, http.StatusOK)

	stoppedRunDetail := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+reprocessRunID, true, http.StatusOK)
	if stringFromAny(stoppedRunDetail["status"]) != "stopped" {
		t.Fatalf("expected stopped reprocess run detail status, got %+v", stoppedRunDetail)
	}

	stoppedNodeExecutions := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+reprocessRunID+"/node-executions", true, http.StatusOK)
	stoppedNodes := objectListFromAny(stoppedNodeExecutions["data"])
	if len(stoppedNodes) != 1 || stringFromAny(stoppedNodes[0]["status"]) != "stopped" {
		t.Fatalf("expected stopped node execution status, got %+v", stoppedNodes)
	}

	stoppedDocumentStatus := getJSON[indexingStatusResponse](env, "/console/api/datasets/"+dataset.ID+"/documents/"+documentID+"/indexing-status", true, http.StatusOK)
	if stoppedDocumentStatus.IndexingStatus != "paused" || stoppedDocumentStatus.StoppedAt == nil {
		t.Fatalf("expected stopped document indexing status to become paused with stopped_at, got %+v", stoppedDocumentStatus)
	}

	stoppedBatchStatus := getJSON[indexingStatusBatchResponse](env, "/console/api/datasets/"+dataset.ID+"/batch/"+reprocessed.Batch+"/indexing-status", true, http.StatusOK)
	if len(stoppedBatchStatus.Data) != 1 {
		t.Fatalf("expected single stopped batch document, got %+v", stoppedBatchStatus.Data)
	}
	if stoppedBatchStatus.Data[0].IndexingStatus != "paused" || stoppedBatchStatus.Data[0].StoppedAt == nil {
		t.Fatalf("expected stopped batch indexing status to become paused with stopped_at, got %+v", stoppedBatchStatus.Data[0])
	}
}

func TestPublicWebAppBootstrapRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	app := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps", map[string]any{
		"name":            "Public Workflow",
		"description":     "public bootstrap",
		"mode":            "workflow",
		"icon_type":       "emoji",
		"icon":            "🧪",
		"icon_background": "#E5F4FF",
	}, true, http.StatusCreated)
	appID := stringFromAny(app["id"])
	appCode := stringFromAny(mapFromAny(app["site"])["access_token"])
	if appID == "" || appCode == "" {
		t.Fatalf("expected created app id and site access token, got %+v", app)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/site-enable", map[string]any{
		"enable_site": true,
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/site", map[string]any{
		"title":                   "Public Workflow App",
		"description":             "workflow share bootstrap",
		"default_language":        "ja-JP",
		"copyright":               "open source",
		"privacy_policy":          "https://example.com/privacy",
		"custom_disclaimer":       "Generated in Go",
		"show_workflow_steps":     true,
		"use_icon_as_answer_icon": true,
	}, true, http.StatusOK)

	modelConfig := cloneJSONObject(mapFromAny(app["model_config"]))
	modelConfig["opening_statement"] = "Hello from Go public runtime"
	modelConfig["suggested_questions_after_answer"] = map[string]any{"enabled": true}
	modelConfig["speech_to_text"] = map[string]any{"enabled": true}
	modelConfig["retriever_resource"] = map[string]any{"enabled": true}
	modelConfig["annotation_reply"] = map[string]any{"enabled": true}
	modelConfig["more_like_this"] = map[string]any{"enabled": true}
	modelConfig["user_input_form"] = []any{
		map[string]any{
			"paragraph": map[string]any{
				"label":    "Query",
				"variable": "query",
				"required": true,
				"default":  "",
			},
		},
	}
	modelConfig["file_upload"] = map[string]any{
		"image": map[string]any{
			"enabled":          true,
			"number_limits":    2,
			"detail":           "high",
			"transfer_methods": []any{"remote_url", "local_file"},
		},
	}
	modelConfig["system_parameters"] = map[string]any{
		"audio_file_size_limit":      4096,
		"file_size_limit":            2048,
		"image_file_size_limit":      1024,
		"video_file_size_limit":      8192,
		"workflow_file_upload_limit": 5,
	}
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/model-config", modelConfig, true, http.StatusOK)

	loginStatus := getJSON[map[string]any](env, "/api/login/status?app_code="+url.QueryEscape(appCode), false, http.StatusOK)
	if loggedIn, ok := loginStatus["app_logged_in"].(bool); !ok || !loggedIn {
		t.Fatalf("expected public app login status to resolve app_logged_in, got %+v", loginStatus)
	}

	accessMode := getJSON[map[string]any](env, "/api/webapp/access-mode?appCode="+url.QueryEscape(appCode), false, http.StatusOK)
	if stringFromAny(accessMode["accessMode"]) != "public" {
		t.Fatalf("expected public access mode response, got %+v", accessMode)
	}

	headers := map[string]string{webAppCodeHeader: appCode}
	passport := getJSONWithHeaders[map[string]any](env, "/api/passport", headers, http.StatusOK)
	if stringFromAny(passport["access_token"]) == "" {
		t.Fatalf("expected passport access token, got %+v", passport)
	}

	appInfo := getJSONWithHeaders[publicWebAppInfoResponse](env, "/api/site", headers, http.StatusOK)
	if appInfo.AppID != appID || !appInfo.EnableSite {
		t.Fatalf("unexpected public app info response: %+v", appInfo)
	}
	if stringFromAny(appInfo.Site["title"]) != "Public Workflow App" || stringFromAny(appInfo.Site["default_language"]) != "ja-JP" {
		t.Fatalf("expected public site payload to mirror site settings, got %+v", appInfo.Site)
	}
	if showWorkflowSteps, ok := appInfo.Site["show_workflow_steps"].(bool); !ok || !showWorkflowSteps {
		t.Fatalf("expected workflow steps to be enabled in public site response, got %+v", appInfo.Site)
	}

	parameters := getJSONWithHeaders[map[string]any](env, "/api/parameters", headers, http.StatusOK)
	if stringFromAny(parameters["opening_statement"]) != "Hello from Go public runtime" {
		t.Fatalf("expected public parameters to expose updated opening statement, got %+v", parameters)
	}
	if _, hasModel := parameters["model"]; hasModel {
		t.Fatalf("expected public parameters response to omit model payload, got %+v", parameters)
	}
	if enabled, ok := mapFromAny(parameters["speech_to_text"])["enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected speech_to_text config in public parameters, got %+v", parameters)
	}
	if enabled, ok := mapFromAny(parameters["more_like_this"])["enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected more_like_this config in public parameters, got %+v", parameters)
	}
	forms := anySlice(parameters["user_input_form"])
	if len(forms) != 1 || stringFromAny(mapFromAny(forms[0])["paragraph"].(map[string]any)["variable"]) != "query" {
		t.Fatalf("expected public parameters to expose user input form, got %+v", parameters["user_input_form"])
	}

	meta := getJSONWithHeaders[map[string]any](env, "/api/meta", headers, http.StatusOK)
	if toolIcons := mapFromAny(meta["tool_icons"]); len(toolIcons) != 0 {
		t.Fatalf("expected empty tool icon map for bootstrap meta response, got %+v", meta)
	}
}

func TestPublicWorkflowRunUsesPublishedWorkflowAndStop(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	app := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps", map[string]any{
		"name":            "Public Workflow Runtime",
		"description":     "public workflow runtime",
		"mode":            "workflow",
		"icon_type":       "emoji",
		"icon":            "🛠",
		"icon_background": "#D1FAE5",
	}, true, http.StatusCreated)
	appID := stringFromAny(app["id"])
	appCode := stringFromAny(mapFromAny(app["site"])["access_token"])
	if appID == "" || appCode == "" {
		t.Fatalf("expected created workflow app id and app code, got %+v", app)
	}

	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/apps/"+appID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{"id": "start-node", "data": map[string]any{"type": "start", "title": "Start"}},
				{"id": "answer-node", "data": map[string]any{"type": "answer", "title": "Published Answer"}},
			},
			"edges": []map[string]any{
				{"source": "start-node", "target": "answer-node"},
			},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/workflows/publish", map[string]any{
		"marked_name":    "published-v1",
		"marked_comment": "public workflow runtime",
	}, true, http.StatusOK)

	postJSON[workflowSyncResponse](env, http.MethodPost, "/console/api/apps/"+appID+"/workflows/draft", map[string]any{
		"graph": map[string]any{
			"nodes": []map[string]any{
				{"id": "start-node", "data": map[string]any{"type": "start", "title": "Start"}},
				{"id": "draft-llm", "data": map[string]any{"type": "llm", "title": "Unpublished LLM"}},
				{"id": "answer-node", "data": map[string]any{"type": "answer", "title": "Unpublished Answer"}},
			},
			"edges": []map[string]any{
				{"source": "start-node", "target": "draft-llm"},
				{"source": "draft-llm", "target": "answer-node"},
			},
			"viewport": map[string]any{"x": 0, "y": 0, "zoom": 1},
		},
		"features": map[string]any{},
	}, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/site-enable", map[string]any{
		"enable_site": true,
	}, true, http.StatusOK)

	headers := map[string]string{
		webAppCodeHeader:     appCode,
		webAppPassportHeader: "passport_" + appID,
	}
	events := postStreamJSONWithHeaders(env, "/api/workflows/run", headers, map[string]any{
		"inputs": map[string]any{
			"topic": "public workflow",
		},
		"response_mode": "streaming",
	}, http.StatusOK)
	if len(events) < 4 {
		t.Fatalf("expected public workflow stream events, got %+v", events)
	}
	if stringFromAny(events[0]["event"]) != "workflow_started" {
		t.Fatalf("expected workflow_started first, got %+v", events)
	}

	runID := stringFromAny(events[0]["workflow_run_id"])
	taskID := stringFromAny(events[0]["task_id"])
	if runID == "" || taskID == "" {
		t.Fatalf("expected public workflow stream ids, got %+v", events[0])
	}

	finished := events[len(events)-1]
	if stringFromAny(finished["event"]) != "workflow_finished" {
		t.Fatalf("expected workflow_finished last, got %+v", events)
	}
	if _, exists := mapFromAny(finished["data"])["node_count"]; exists {
		t.Fatalf("expected workflow_finished data to nest outputs instead of raw node_count, got %+v", finished["data"])
	}
	finishedOutputs := mapFromAny(mapFromAny(finished["data"])["outputs"])
	nodeCount, ok := finishedOutputs["node_count"].(float64)
	if !ok || int(nodeCount) != 2 {
		t.Fatalf("expected public workflow run to use published graph with 2 nodes, got %+v", finishedOutputs)
	}

	nodeFinishedTitles := []string{}
	for _, event := range events {
		if stringFromAny(event["event"]) != "node_finished" {
			continue
		}
		nodeFinishedTitles = append(nodeFinishedTitles, stringFromAny(mapFromAny(event["data"])["title"]))
	}
	if len(nodeFinishedTitles) != 2 {
		t.Fatalf("expected 2 node_finished events from published workflow, got %+v", nodeFinishedTitles)
	}
	if slices.Contains(nodeFinishedTitles, "Unpublished LLM") || slices.Contains(nodeFinishedTitles, "Unpublished Answer") {
		t.Fatalf("expected public workflow run to ignore latest draft graph, got %+v", nodeFinishedTitles)
	}
	if !slices.Contains(nodeFinishedTitles, "Published Answer") {
		t.Fatalf("expected public workflow run to expose published workflow node title, got %+v", nodeFinishedTitles)
	}

	postJSONWithHeaders[map[string]any](env, http.MethodPost, "/api/workflows/tasks/"+taskID+"/stop", headers, nil, http.StatusOK)

	runDetail := getJSON[map[string]any](env, "/console/api/apps/"+appID+"/workflow-runs/"+runID, true, http.StatusOK)
	if stringFromAny(runDetail["status"]) != "stopped" {
		t.Fatalf("expected public workflow stop to persist stopped status, got %+v", runDetail)
	}
	nodeExecutions := getJSON[map[string]any](env, "/console/api/apps/"+appID+"/workflow-runs/"+runID+"/node-executions", true, http.StatusOK)
	for _, item := range objectListFromAny(nodeExecutions["data"]) {
		if stringFromAny(item["status"]) != "stopped" {
			t.Fatalf("expected public workflow stop to mark node execution stopped, got %+v", item)
		}
	}
}

func TestPublicChatRuntimeRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	app := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps", map[string]any{
		"name":            "Public Chat Runtime",
		"description":     "public chat runtime",
		"mode":            "chat",
		"icon_type":       "emoji",
		"icon":            "💬",
		"icon_background": "#FDE68A",
	}, true, http.StatusCreated)
	appID := stringFromAny(app["id"])
	appCode := stringFromAny(mapFromAny(app["site"])["access_token"])
	if appID == "" || appCode == "" {
		t.Fatalf("expected created chat app id and app code, got %+v", app)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/site-enable", map[string]any{
		"enable_site": true,
	}, true, http.StatusOK)

	modelConfig := cloneJSONObject(mapFromAny(app["model_config"]))
	modelConfig["opening_statement"] = "Welcome to the public chat runtime"
	modelConfig["suggested_questions_after_answer"] = map[string]any{"enabled": true}
	modelConfig["speech_to_text"] = map[string]any{"enabled": true}
	modelConfig["text_to_speech"] = map[string]any{"enabled": true, "voice": "alloy", "language": "en-US"}
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/model-config", modelConfig, true, http.StatusOK)

	headers := map[string]string{webAppCodeHeader: appCode}
	events := postStreamJSONWithHeaders(env, "/api/chat-messages", headers, map[string]any{
		"query":         "Explain the migration status",
		"inputs":        map[string]any{"audience": "maintainers"},
		"response_mode": "streaming",
	}, http.StatusOK)
	if len(events) < 2 {
		t.Fatalf("expected public chat events, got %+v", events)
	}
	if stringFromAny(events[0]["event"]) != "message" {
		t.Fatalf("expected first public chat event to be message, got %+v", events)
	}
	conversationID := stringFromAny(events[0]["conversation_id"])
	messageID := stringFromAny(events[0]["id"])
	taskID := stringFromAny(events[0]["task_id"])
	if conversationID == "" || messageID == "" || taskID == "" {
		t.Fatalf("expected chat identifiers in first event, got %+v", events[0])
	}
	if stringFromAny(events[len(events)-1]["event"]) != "message_end" {
		t.Fatalf("expected public chat stream to end with message_end, got %+v", events)
	}

	messages := getJSONWithHeaders[map[string]any](env, "/api/messages?conversation_id="+url.QueryEscape(conversationID), headers, http.StatusOK)
	messageItems := objectListFromAny(messages["data"])
	if len(messageItems) != 1 {
		t.Fatalf("expected a single public chat message, got %+v", messages)
	}
	if stringFromAny(messageItems[0]["query"]) != "Explain the migration status" {
		t.Fatalf("expected stored public chat query, got %+v", messageItems[0])
	}
	if stringFromAny(messageItems[0]["status"]) != "normal" {
		t.Fatalf("expected initial public chat status to be normal, got %+v", messageItems[0])
	}
	if feedback := messageItems[0]["feedback"]; feedback != nil {
		t.Fatalf("expected no feedback before feedback submission, got %+v", feedback)
	}
	if len(anySlice(messageItems[0]["extra_contents"])) != 0 {
		t.Fatalf("expected no extra contents for simple public chat message, got %+v", messageItems[0]["extra_contents"])
	}

	postJSONWithHeaders[map[string]any](env, http.MethodPost, "/api/chat-messages/"+taskID+"/stop", headers, nil, http.StatusOK)

	stoppedMessages := getJSONWithHeaders[map[string]any](env, "/api/messages?conversation_id="+url.QueryEscape(conversationID), headers, http.StatusOK)
	stoppedItems := objectListFromAny(stoppedMessages["data"])
	if len(stoppedItems) != 1 || stringFromAny(stoppedItems[0]["status"]) != "stopped" {
		t.Fatalf("expected stopped public chat message, got %+v", stoppedMessages)
	}

	postJSONWithHeaders[map[string]any](env, http.MethodPost, "/api/messages/"+messageID+"/feedbacks", headers, map[string]any{
		"rating":  "like",
		"content": "helpful",
	}, http.StatusOK)

	feedbackMessages := getJSONWithHeaders[map[string]any](env, "/api/messages?conversation_id="+url.QueryEscape(conversationID), headers, http.StatusOK)
	feedbackItems := objectListFromAny(feedbackMessages["data"])
	feedback := mapFromAny(feedbackItems[0]["feedback"])
	if stringFromAny(feedback["rating"]) != "like" || stringFromAny(feedback["content"]) != "helpful" {
		t.Fatalf("expected stored public feedback, got %+v", feedbackItems[0]["feedback"])
	}

	suggested := getJSONWithHeaders[map[string]any](env, "/api/messages/"+messageID+"/suggested-questions", headers, http.StatusOK)
	if len(anySlice(suggested["data"])) != 3 {
		t.Fatalf("expected generated suggested questions, got %+v", suggested)
	}

	moreLikeThis := getJSONWithHeaders[map[string]any](env, "/api/messages/"+messageID+"/more-like-this", headers, http.StatusOK)
	if stringFromAny(moreLikeThis["id"]) == "" || stringFromAny(moreLikeThis["id"]) == messageID {
		t.Fatalf("expected more-like-this to create a new public message, got %+v", moreLikeThis)
	}
	if stringFromAny(moreLikeThis["answer"]) == "" {
		t.Fatalf("expected more-like-this answer, got %+v", moreLikeThis)
	}

	conversations := getJSONWithHeaders[map[string]any](env, "/api/conversations?pinned=false&limit=20", headers, http.StatusOK)
	conversationItems := objectListFromAny(conversations["data"])
	if len(conversationItems) != 1 {
		t.Fatalf("expected one public conversation, got %+v", conversations)
	}
	if stringFromAny(conversationItems[0]["id"]) != conversationID {
		t.Fatalf("expected public conversation id to match chat stream, got %+v", conversationItems[0])
	}
	if stringFromAny(conversationItems[0]["introduction"]) != "Welcome to the public chat runtime" {
		t.Fatalf("expected conversation introduction to reflect opening statement, got %+v", conversationItems[0])
	}

	postJSONWithHeaders[map[string]any](env, http.MethodPatch, "/api/conversations/"+conversationID+"/pin", headers, nil, http.StatusOK)
	pinnedConversations := getJSONWithHeaders[map[string]any](env, "/api/conversations?pinned=true&limit=20", headers, http.StatusOK)
	if len(objectListFromAny(pinnedConversations["data"])) != 1 {
		t.Fatalf("expected pinned public conversation, got %+v", pinnedConversations)
	}

	renamed := postJSONWithHeaders[map[string]any](env, http.MethodPost, "/api/conversations/"+conversationID+"/name", headers, map[string]any{
		"name": "Migration Thread",
	}, http.StatusOK)
	if stringFromAny(renamed["name"]) != "Migration Thread" {
		t.Fatalf("expected renamed public conversation, got %+v", renamed)
	}

	postJSONWithHeaders[map[string]any](env, http.MethodPost, "/api/saved-messages", headers, map[string]any{
		"message_id": messageID,
	}, http.StatusOK)
	savedMessages := getJSONWithHeaders[map[string]any](env, "/api/saved-messages", headers, http.StatusOK)
	savedItems := objectListFromAny(savedMessages["data"])
	if len(savedItems) != 1 || stringFromAny(savedItems[0]["id"]) != messageID {
		t.Fatalf("expected saved public message, got %+v", savedMessages)
	}

	postJSONWithHeaders[map[string]any](env, http.MethodDelete, "/api/saved-messages/"+messageID, headers, nil, http.StatusOK)
	afterDeleteSaved := getJSONWithHeaders[map[string]any](env, "/api/saved-messages", headers, http.StatusOK)
	if len(objectListFromAny(afterDeleteSaved["data"])) != 0 {
		t.Fatalf("expected saved messages to be empty after deletion, got %+v", afterDeleteSaved)
	}

	postJSONWithHeaders[map[string]any](env, http.MethodDelete, "/api/conversations/"+conversationID, headers, nil, http.StatusOK)
	afterDeleteConversations := getJSONWithHeaders[map[string]any](env, "/api/conversations?pinned=false&limit=20", headers, http.StatusOK)
	if len(objectListFromAny(afterDeleteConversations["data"])) != 0 {
		t.Fatalf("expected public conversations to be empty after deletion, got %+v", afterDeleteConversations)
	}
}

func TestPublicCompletionAndAudioRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	app := postJSON[map[string]any](env, http.MethodPost, "/console/api/apps", map[string]any{
		"name":            "Public Completion Runtime",
		"description":     "public completion runtime",
		"mode":            "completion",
		"icon_type":       "emoji",
		"icon":            "📝",
		"icon_background": "#DBEAFE",
	}, true, http.StatusCreated)
	appID := stringFromAny(app["id"])
	appCode := stringFromAny(mapFromAny(app["site"])["access_token"])
	if appID == "" || appCode == "" {
		t.Fatalf("expected created completion app id and app code, got %+v", app)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/site-enable", map[string]any{
		"enable_site": true,
	}, true, http.StatusOK)

	modelConfig := cloneJSONObject(mapFromAny(app["model_config"]))
	modelConfig["speech_to_text"] = map[string]any{"enabled": true}
	modelConfig["text_to_speech"] = map[string]any{"enabled": true, "voice": "alloy", "language": "en-US"}
	postJSON[map[string]any](env, http.MethodPost, "/console/api/apps/"+appID+"/model-config", modelConfig, true, http.StatusOK)

	headers := map[string]string{webAppCodeHeader: appCode}
	events := postStreamJSONWithHeaders(env, "/api/completion-messages", headers, map[string]any{
		"inputs": map[string]any{
			"topic": "Go compatibility layer",
		},
		"response_mode": "streaming",
	}, http.StatusOK)
	if len(events) < 2 || stringFromAny(events[0]["event"]) != "message" {
		t.Fatalf("expected public completion stream message event, got %+v", events)
	}
	messageID := stringFromAny(events[0]["id"])
	if messageID == "" {
		t.Fatalf("expected public completion message id, got %+v", events[0])
	}

	var audioBody bytes.Buffer
	audioWriter := multipart.NewWriter(&audioBody)
	part, err := audioWriter.CreateFormFile("file", "voice.mp3")
	if err != nil {
		t.Fatalf("create audio form file: %v", err)
	}
	if _, err := part.Write([]byte("fake audio bytes")); err != nil {
		t.Fatalf("write audio form file: %v", err)
	}
	if err := audioWriter.Close(); err != nil {
		t.Fatalf("close audio form writer: %v", err)
	}

	audioReq, err := http.NewRequest(http.MethodPost, env.server.URL+"/api/audio-to-text", &audioBody)
	if err != nil {
		t.Fatalf("create audio-to-text request: %v", err)
	}
	audioReq.Header.Set("Content-Type", audioWriter.FormDataContentType())
	for key, value := range headers {
		audioReq.Header.Set(key, value)
	}
	audioResp := env.do(audioReq, http.StatusOK)
	defer audioResp.Body.Close()

	var audioToText map[string]any
	if err := json.NewDecoder(audioResp.Body).Decode(&audioToText); err != nil {
		t.Fatalf("decode audio-to-text response: %v", err)
	}
	if !strings.Contains(stringFromAny(audioToText["text"]), "voice.mp3") {
		t.Fatalf("expected audio-to-text transcription to reference uploaded file, got %+v", audioToText)
	}

	ttsPayload, err := json.Marshal(map[string]any{
		"message_id": messageID,
		"streaming":  true,
	})
	if err != nil {
		t.Fatalf("marshal text-to-audio payload: %v", err)
	}
	ttsReq, err := http.NewRequest(http.MethodPost, env.server.URL+"/api/text-to-audio", bytes.NewReader(ttsPayload))
	if err != nil {
		t.Fatalf("create text-to-audio request: %v", err)
	}
	ttsReq.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		ttsReq.Header.Set(key, value)
	}
	ttsResp := env.do(ttsReq, http.StatusOK)
	defer ttsResp.Body.Close()

	if contentType := ttsResp.Header.Get("Content-Type"); contentType != "audio/mpeg" {
		t.Fatalf("expected text-to-audio response to be audio/mpeg, got %q", contentType)
	}
	ttsBody, err := io.ReadAll(ttsResp.Body)
	if err != nil {
		t.Fatalf("read text-to-audio body: %v", err)
	}
	if len(ttsBody) == 0 {
		t.Fatalf("expected text-to-audio body to contain mp3 bytes")
	}
}

func TestWorkspaceMemberInvitationActivationAndFeatureRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	features := getJSON[map[string]any](env, "/console/api/features", true, http.StatusOK)
	if enabled, ok := features["dataset_operator_enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected dataset_operator_enabled feature, got %+v", features)
	}
	if size := nestedInt(features, "workspace_members", "size"); size != 1 {
		t.Fatalf("expected one initial workspace member, got %+v", features["workspace_members"])
	}

	permissions := getJSON[map[string]any](env, "/console/api/workspaces/current/permission", true, http.StatusOK)
	if allowed, ok := permissions["allow_member_invite"].(bool); !ok || !allowed {
		t.Fatalf("expected allow_member_invite=true, got %+v", permissions)
	}

	invite := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/members/invite-email", map[string]any{
		"emails":   []string{"member@example.com", "pending@example.com", "tester@example.com"},
		"role":     "editor",
		"language": "en-US",
	}, true, http.StatusOK)
	results, ok := invite["invitation_results"].([]any)
	if !ok || len(results) != 3 {
		t.Fatalf("expected three invitation results, got %+v", invite)
	}

	var activationURL string
	successCount := 0
	failedCount := 0
	for _, raw := range results {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected invitation result type: %#v", raw)
		}
		switch item["status"] {
		case "success":
			successCount++
			if email, _ := item["email"].(string); email == "member@example.com" {
				activationURL, _ = item["url"].(string)
			}
		case "failed":
			failedCount++
		}
	}
	if successCount != 2 || failedCount != 1 || activationURL == "" {
		t.Fatalf("unexpected invitation result summary: %+v", results)
	}

	members := getJSON[map[string]any](env, "/console/api/workspaces/current/members", true, http.StatusOK)
	accounts, ok := members["accounts"].([]any)
	if !ok || len(accounts) != 3 {
		t.Fatalf("expected owner + two pending members, got %+v", members)
	}
	pendingID := accountIDByEmail(accounts, "pending@example.com")
	if pendingID == "" {
		t.Fatalf("expected pending invitation account, got %+v", accounts)
	}

	postJSON[map[string]any](env, http.MethodPut, "/console/api/workspaces/current/members/"+pendingID+"/update-role", map[string]any{
		"role": "dataset_operator",
	}, true, http.StatusOK)
	updatedMembers := getJSON[map[string]any](env, "/console/api/workspaces/current/members", true, http.StatusOK)
	updatedAccounts, _ := updatedMembers["accounts"].([]any)
	if role := accountFieldByEmail(updatedAccounts, "pending@example.com", "role"); role != "dataset_operator" {
		t.Fatalf("expected pending invitation role update, got %+v", updatedAccounts)
	}

	postJSON[map[string]any](env, http.MethodDelete, "/console/api/workspaces/current/members/"+pendingID, nil, true, http.StatusOK)
	prunedMembers := getJSON[map[string]any](env, "/console/api/workspaces/current/members", true, http.StatusOK)
	prunedAccounts, _ := prunedMembers["accounts"].([]any)
	if len(prunedAccounts) != 2 {
		t.Fatalf("expected one pending invitation to be removed, got %+v", prunedAccounts)
	}

	token := invitationToken(t, activationURL)
	check := getJSON[map[string]any](env, "/console/api/activate/check?token="+url.QueryEscape(token), false, http.StatusOK)
	if valid, ok := check["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected invitation token to be valid, got %+v", check)
	}
	if workspaceName := nestedString(check, "data", "workspace_name"); workspaceName != "Default Workspace" {
		t.Fatalf("unexpected invitation workspace name: %+v", check)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/activate", map[string]any{
		"token":              token,
		"name":               "Invited Member",
		"interface_language": "zh-Hans",
		"timezone":           "Asia/Shanghai",
	}, false, http.StatusOK)

	profile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := profile["email"].(string); email != "member@example.com" {
		t.Fatalf("expected activated member profile, got %+v", profile)
	}
	if isPasswordSet, ok := profile["is_password_set"].(bool); !ok || isPasswordSet {
		t.Fatalf("expected invited member to have no password set, got %+v", profile)
	}

	currentWorkspace := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current", map[string]any{}, true, http.StatusOK)
	if role, _ := currentWorkspace["role"].(string); role != "editor" {
		t.Fatalf("expected activated member workspace role editor, got %+v", currentWorkspace)
	}

	education := getJSON[map[string]any](env, "/console/api/account/education", true, http.StatusOK)
	if isStudent, ok := education["is_student"].(bool); !ok || isStudent {
		t.Fatalf("expected non-student education status, got %+v", education)
	}
	integrates := getJSON[map[string]any](env, "/console/api/account/integrates", true, http.StatusOK)
	if data, ok := integrates["data"].([]any); !ok || len(data) != 0 {
		t.Fatalf("expected no bound account integrations, got %+v", integrates)
	}
}

func TestWorkspaceOwnershipTransferAPIs(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	invite := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/members/invite-email", map[string]any{
		"emails": []string{"new-owner@example.com"},
		"role":   "admin",
	}, true, http.StatusOK)
	results, ok := invite["invitation_results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected one invitation result, got %+v", invite)
	}
	result, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected invitation result payload: %#v", results[0])
	}
	token := invitationToken(t, stringFromAny(result["url"]))

	postJSON[map[string]any](env, http.MethodPost, "/console/api/activate", map[string]any{
		"token":              token,
		"name":               "Next Owner",
		"interface_language": "en-US",
		"timezone":           "UTC",
	}, false, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)

	members := getJSON[map[string]any](env, "/console/api/workspaces/current/members", true, http.StatusOK)
	accounts, _ := members["accounts"].([]any)
	newOwnerID := accountIDByEmail(accounts, "new-owner@example.com")
	if newOwnerID == "" {
		t.Fatalf("expected activated member in workspace list, got %+v", accounts)
	}

	send := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/members/send-owner-transfer-confirm-email", map[string]any{}, true, http.StatusOK)
	transferToken, _ := send["data"].(string)
	if transferToken == "" {
		t.Fatalf("expected owner transfer token, got %+v", send)
	}

	verify := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/members/owner-transfer-check", map[string]any{
		"code":  "123456",
		"token": transferToken,
	}, true, http.StatusOK)
	if valid, ok := verify["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected owner transfer verification to succeed, got %+v", verify)
	}
	verifiedToken, _ := verify["token"].(string)
	if verifiedToken == "" {
		t.Fatalf("expected verified owner transfer token, got %+v", verify)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/members/"+newOwnerID+"/owner-transfer", map[string]any{
		"token": verifiedToken,
	}, true, http.StatusOK)

	currentWorkspace := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current", map[string]any{}, true, http.StatusOK)
	if role, _ := currentWorkspace["role"].(string); role != "admin" {
		t.Fatalf("expected previous owner to become admin, got %+v", currentWorkspace)
	}

	updatedMembers := getJSON[map[string]any](env, "/console/api/workspaces/current/members", true, http.StatusOK)
	updatedAccounts, _ := updatedMembers["accounts"].([]any)
	if role := accountFieldByEmail(updatedAccounts, "tester@example.com", "role"); role != "admin" {
		t.Fatalf("expected original owner role admin, got %+v", updatedAccounts)
	}
	if role := accountFieldByEmail(updatedAccounts, "new-owner@example.com", "role"); role != "owner" {
		t.Fatalf("expected new owner role owner, got %+v", updatedAccounts)
	}

	permissions := getJSON[map[string]any](env, "/console/api/workspaces/current/permission", true, http.StatusOK)
	if allowed, ok := permissions["allow_owner_transfer"].(bool); !ok || allowed {
		t.Fatalf("expected transferred admin to lose owner transfer permission, got %+v", permissions)
	}
}

func TestAccountRegistrationInitAndForgotPasswordRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	features := getJSON[map[string]any](env, "/console/api/system-features", false, http.StatusOK)
	if allowed, ok := features["is_allow_register"].(bool); !ok || !allowed {
		t.Fatalf("expected registration feature enabled, got %+v", features)
	}
	if enabled, ok := features["enable_change_email"].(bool); !ok || !enabled {
		t.Fatalf("expected change email feature enabled, got %+v", features)
	}
	if setup, ok := features["is_email_setup"].(bool); !ok || !setup {
		t.Fatalf("expected email setup feature enabled, got %+v", features)
	}

	send := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register/send-email", map[string]any{
		"email":    "signup@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	registerToken, _ := send["data"].(string)
	if registerToken == "" {
		t.Fatalf("expected registration token, got %+v", send)
	}

	validity := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register/validity", map[string]any{
		"email": "signup@example.com",
		"code":  "123456",
		"token": registerToken,
	}, false, http.StatusOK)
	if valid, ok := validity["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected registration verification to succeed, got %+v", validity)
	}
	verifiedRegisterToken, _ := validity["token"].(string)
	if verifiedRegisterToken == "" {
		t.Fatalf("expected verified registration token, got %+v", validity)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register", map[string]any{
		"token":            verifiedRegisterToken,
		"new_password":     "Signup#123",
		"password_confirm": "Signup#123",
	}, false, http.StatusOK)

	profile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := profile["email"].(string); email != "signup@example.com" {
		t.Fatalf("expected newly registered account profile, got %+v", profile)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/account/init", map[string]any{
		"invitation_code":    "dify-go",
		"interface_language": "zh-Hans",
		"timezone":           "Asia/Shanghai",
	}, true, http.StatusOK)
	updatedProfile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if language, _ := updatedProfile["interface_language"].(string); language != "zh-Hans" {
		t.Fatalf("expected account init to update interface_language, got %+v", updatedProfile)
	}
	if timezone, _ := updatedProfile["timezone"].(string); timezone != "Asia/Shanghai" {
		t.Fatalf("expected account init to update timezone, got %+v", updatedProfile)
	}

	sendForgot := postJSON[map[string]any](env, http.MethodPost, "/console/api/forgot-password", map[string]any{
		"email":    "signup@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	forgotToken, _ := sendForgot["data"].(string)
	if forgotToken == "" {
		t.Fatalf("expected forgot-password token, got %+v", sendForgot)
	}

	validateForgot := postJSON[map[string]any](env, http.MethodPost, "/console/api/forgot-password/validity", map[string]any{
		"token": forgotToken,
	}, false, http.StatusOK)
	if valid, ok := validateForgot["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected forgot-password token validation, got %+v", validateForgot)
	}
	if email, _ := validateForgot["email"].(string); email != "signup@example.com" {
		t.Fatalf("expected forgot-password token email, got %+v", validateForgot)
	}

	verifyForgot := postJSON[map[string]any](env, http.MethodPost, "/console/api/forgot-password/validity", map[string]any{
		"email": "signup@example.com",
		"code":  "123456",
		"token": forgotToken,
	}, false, http.StatusOK)
	if valid, ok := verifyForgot["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected forgot-password code verification, got %+v", verifyForgot)
	}
	resetToken, _ := verifyForgot["token"].(string)
	if resetToken == "" {
		t.Fatalf("expected reset token, got %+v", verifyForgot)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/forgot-password/resets", map[string]any{
		"token":            resetToken,
		"new_password":     "Reset#123",
		"password_confirm": "Reset#123",
	}, false, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "signup@example.com",
		"password": "Reset#123",
	}, false, http.StatusOK)

	reloginProfile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := reloginProfile["email"].(string); email != "signup@example.com" {
		t.Fatalf("expected relogin with reset password to succeed, got %+v", reloginProfile)
	}

	publicForgot := postJSON[map[string]any](env, http.MethodPost, "/api/forgot-password", map[string]any{
		"email":    "signup@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	publicForgotToken, _ := publicForgot["data"].(string)
	if publicForgotToken == "" {
		t.Fatalf("expected public forgot-password token, got %+v", publicForgot)
	}
	publicValidity := postJSON[map[string]any](env, http.MethodPost, "/api/forgot-password/validity", map[string]any{
		"token": publicForgotToken,
	}, false, http.StatusOK)
	if valid, ok := publicValidity["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected public forgot-password token validation, got %+v", publicValidity)
	}
	postJSON[map[string]any](env, http.MethodPost, "/api/forgot-password/resets", map[string]any{
		"token":            publicForgotToken,
		"new_password":     "Reset#456",
		"password_confirm": "Reset#456",
	}, false, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "signup@example.com",
		"password": "Reset#456",
	}, false, http.StatusOK)
}

func TestAccountChangeEmailRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	signupSend := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register/send-email", map[string]any{
		"email":    "occupied@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	signupToken, _ := signupSend["data"].(string)
	signupValidity := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register/validity", map[string]any{
		"email": "occupied@example.com",
		"code":  "123456",
		"token": signupToken,
	}, false, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/email-register", map[string]any{
		"token":            stringFromAny(signupValidity["token"]),
		"new_password":     "Occupied#123",
		"password_confirm": "Occupied#123",
	}, false, http.StatusOK)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)

	duplicate := postJSON[errorResponse](env, http.MethodPost, "/console/api/account/change-email/check-email-unique", map[string]any{
		"email": "occupied@example.com",
	}, true, http.StatusBadRequest)
	if duplicate.Code != "email_already_in_use" {
		t.Fatalf("expected duplicate email check code, got %+v", duplicate)
	}

	sendOld := postJSON[map[string]any](env, http.MethodPost, "/console/api/account/change-email", map[string]any{
		"email": "tester@example.com",
		"phase": "old_email",
	}, true, http.StatusOK)
	oldToken, _ := sendOld["data"].(string)
	if oldToken == "" {
		t.Fatalf("expected old email token, got %+v", sendOld)
	}

	verifyOld := postJSON[map[string]any](env, http.MethodPost, "/console/api/account/change-email/validity", map[string]any{
		"email": "tester@example.com",
		"code":  "123456",
		"token": oldToken,
	}, true, http.StatusOK)
	if valid, ok := verifyOld["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected old email verification to succeed, got %+v", verifyOld)
	}
	oldVerifiedToken, _ := verifyOld["token"].(string)

	sendNew := postJSON[map[string]any](env, http.MethodPost, "/console/api/account/change-email", map[string]any{
		"email": "tester+new@example.com",
		"phase": "new_email",
		"token": oldVerifiedToken,
	}, true, http.StatusOK)
	newToken, _ := sendNew["data"].(string)
	if newToken == "" {
		t.Fatalf("expected new email token, got %+v", sendNew)
	}

	verifyNew := postJSON[map[string]any](env, http.MethodPost, "/console/api/account/change-email/validity", map[string]any{
		"email": "tester+new@example.com",
		"code":  "123456",
		"token": newToken,
	}, true, http.StatusOK)
	if valid, ok := verifyNew["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected new email verification to succeed, got %+v", verifyNew)
	}
	readyToken, _ := verifyNew["token"].(string)

	postJSON[map[string]any](env, http.MethodPost, "/console/api/account/change-email/reset", map[string]any{
		"new_email": "tester+new@example.com",
		"token":     readyToken,
	}, true, http.StatusOK)

	profile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := profile["email"].(string); email != "tester+new@example.com" {
		t.Fatalf("expected updated profile email, got %+v", profile)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)
	postJSON[map[string]any](env, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester+new@example.com",
		"password": "password123",
	}, false, http.StatusOK)
}

func TestAccountEmailCodeEducationAndOAuthRoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	features := getJSON[map[string]any](env, "/console/api/system-features", false, http.StatusOK)
	if enabled, ok := features["enable_email_code_login"].(bool); !ok || !enabled {
		t.Fatalf("expected email code login feature enabled, got %+v", features)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)

	sendConsole := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-code-login", map[string]any{
		"email":    "tester@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	consoleToken, _ := sendConsole["data"].(string)
	if consoleToken == "" {
		t.Fatalf("expected console email login token, got %+v", sendConsole)
	}

	verifyConsole := postJSON[map[string]any](env, http.MethodPost, "/console/api/email-code-login/validity", map[string]any{
		"email":    "tester@example.com",
		"code":     base64.StdEncoding.EncodeToString([]byte("123456")),
		"token":    consoleToken,
		"language": "en-US",
	}, false, http.StatusOK)
	if result, _ := verifyConsole["result"].(string); result != "success" {
		t.Fatalf("expected successful email code login, got %+v", verifyConsole)
	}
	if accessToken := nestedString(verifyConsole, "data", "access_token"); accessToken == "" {
		t.Fatalf("expected email code login to issue access token, got %+v", verifyConsole)
	}

	profile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := profile["email"].(string); email != "tester@example.com" {
		t.Fatalf("expected profile after email code login, got %+v", profile)
	}

	postJSON[map[string]any](env, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)

	sendPublic := postJSON[map[string]any](env, http.MethodPost, "/api/email-code-login", map[string]any{
		"email":    "tester@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	publicToken, _ := sendPublic["data"].(string)
	if publicToken == "" {
		t.Fatalf("expected public email login token, got %+v", sendPublic)
	}

	verifyPublic := postJSON[map[string]any](env, http.MethodPost, "/api/email-code-login/validity", map[string]any{
		"email": "tester@example.com",
		"code":  base64.StdEncoding.EncodeToString([]byte("123456")),
		"token": publicToken,
	}, false, http.StatusOK)
	if result, _ := verifyPublic["result"].(string); result != "success" {
		t.Fatalf("expected successful public email code login, got %+v", verifyPublic)
	}

	publicProfile := getJSON[map[string]any](env, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := publicProfile["email"].(string); email != "tester@example.com" {
		t.Fatalf("expected console session after public email code login, got %+v", publicProfile)
	}

	verifyEducation := getJSON[map[string]any](env, "/console/api/account/education/verify", true, http.StatusOK)
	educationToken, _ := verifyEducation["token"].(string)
	if educationToken == "" {
		t.Fatalf("expected education verification token, got %+v", verifyEducation)
	}

	addEducation := postJSON[map[string]any](env, http.MethodPost, "/console/api/account/education", map[string]any{
		"token":       educationToken,
		"institution": "Stanford University",
		"role":        "Student",
	}, true, http.StatusOK)
	if message, _ := addEducation["message"].(string); message != "success" {
		t.Fatalf("expected education add success, got %+v", addEducation)
	}

	educationStatus := getJSON[map[string]any](env, "/console/api/account/education", true, http.StatusOK)
	if isStudent, ok := educationStatus["is_student"].(bool); !ok || !isStudent {
		t.Fatalf("expected student education status, got %+v", educationStatus)
	}
	if allowRefresh, ok := educationStatus["allow_refresh"].(bool); !ok || !allowRefresh {
		t.Fatalf("expected education refresh enabled, got %+v", educationStatus)
	}
	if _, ok := educationStatus["expire_at"].(float64); !ok {
		t.Fatalf("expected education expiry timestamp, got %+v", educationStatus)
	}

	workspaceFeatures := getJSON[map[string]any](env, "/console/api/features", true, http.StatusOK)
	if enabled, ok := mapFromAny(workspaceFeatures["education"])["enabled"].(bool); !ok || !enabled {
		t.Fatalf("expected education feature enabled, got %+v", workspaceFeatures)
	}
	if activated, ok := mapFromAny(workspaceFeatures["education"])["activated"].(bool); !ok || !activated {
		t.Fatalf("expected education feature activated after verification, got %+v", workspaceFeatures)
	}

	autocomplete := getJSON[map[string]any](env, "/console/api/account/education/autocomplete?keywords=stan&page=0&limit=5", true, http.StatusOK)
	options, _ := autocomplete["data"].([]any)
	if len(options) == 0 || stringFromAny(options[0]) != "Stanford University" {
		t.Fatalf("expected Stanford University autocomplete match, got %+v", autocomplete)
	}

	oauthInfo := postJSON[map[string]any](env, http.MethodPost, "/console/api/oauth/provider", map[string]any{
		"client_id":    "demo-client",
		"redirect_uri": "https://example.com/callback",
	}, false, http.StatusOK)
	if scope, _ := oauthInfo["scope"].(string); !strings.Contains(scope, "read:email") {
		t.Fatalf("expected oauth provider scope payload, got %+v", oauthInfo)
	}
	if label := nestedString(oauthInfo, "app_label", "en_US"); label != "demo-client" {
		t.Fatalf("expected oauth provider label to reflect client_id, got %+v", oauthInfo)
	}

	oauthAuthorize := postJSON[map[string]any](env, http.MethodPost, "/console/api/oauth/provider/authorize", map[string]any{
		"client_id": "demo-client",
	}, true, http.StatusOK)
	if code, _ := oauthAuthorize["code"].(string); code == "" {
		t.Fatalf("expected oauth authorize code, got %+v", oauthAuthorize)
	}
}

func TestPersistentAuthFlowsAndSSORoutes(t *testing.T) {
	env := newServerTestEnv(t)
	env.setupAndLogin()

	forgot := postJSON[map[string]any](env, http.MethodPost, "/console/api/forgot-password", map[string]any{
		"email":    "tester@example.com",
		"language": "en-US",
	}, false, http.StatusOK)
	forgotToken, _ := forgot["data"].(string)
	if forgotToken == "" {
		t.Fatalf("expected forgot-password token, got %+v", forgot)
	}

	restarted := env.restart()
	forgotValidity := postJSON[map[string]any](restarted, http.MethodPost, "/console/api/forgot-password/validity", map[string]any{
		"token": forgotToken,
	}, false, http.StatusOK)
	if valid, ok := forgotValidity["is_valid"].(bool); !ok || !valid {
		t.Fatalf("expected persisted forgot-password token to validate after restart, got %+v", forgotValidity)
	}

	verifyForgot := postJSON[map[string]any](restarted, http.MethodPost, "/console/api/forgot-password/validity", map[string]any{
		"email": "tester@example.com",
		"code":  base64.StdEncoding.EncodeToString([]byte("123456")),
		"token": forgotToken,
	}, false, http.StatusOK)
	resetToken, _ := verifyForgot["token"].(string)
	if resetToken == "" {
		t.Fatalf("expected persisted forgot-password flow to promote after restart, got %+v", verifyForgot)
	}

	postJSON[map[string]any](restarted, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)
	invite := postJSON[map[string]any](restarted, http.MethodPost, "/console/api/workspaces/current/members/invite-email", map[string]any{
		"emails": []string{"persistent-owner@example.com"},
		"role":   "admin",
	}, true, http.StatusOK)
	results, _ := invite["invitation_results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected invitation result, got %+v", invite)
	}
	token := invitationToken(t, stringFromAny(mapFromAny(results[0])["url"]))
	postJSON[map[string]any](restarted, http.MethodPost, "/console/api/activate", map[string]any{
		"token":              token,
		"name":               "Persistent Owner",
		"interface_language": "en-US",
		"timezone":           "UTC",
	}, false, http.StatusOK)
	postJSON[map[string]any](restarted, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)

	members := getJSON[map[string]any](restarted, "/console/api/workspaces/current/members", true, http.StatusOK)
	newOwnerID := accountIDByEmail(anySlice(members["accounts"]), "persistent-owner@example.com")
	if newOwnerID == "" {
		t.Fatalf("expected persistent owner member, got %+v", members)
	}

	sendTransfer := postJSON[map[string]any](restarted, http.MethodPost, "/console/api/workspaces/current/members/send-owner-transfer-confirm-email", map[string]any{}, true, http.StatusOK)
	transferToken, _ := sendTransfer["data"].(string)
	if transferToken == "" {
		t.Fatalf("expected owner transfer token, got %+v", sendTransfer)
	}

	restartedAgain := restarted.restart()
	postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)
	transferCheck := postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/workspaces/current/members/owner-transfer-check", map[string]any{
		"code":  base64.StdEncoding.EncodeToString([]byte("123456")),
		"token": transferToken,
	}, true, http.StatusOK)
	verifiedTransferToken, _ := transferCheck["token"].(string)
	if valid, ok := transferCheck["is_valid"].(bool); !ok || !valid || verifiedTransferToken == "" {
		t.Fatalf("expected persisted owner transfer token to verify after restart, got %+v", transferCheck)
	}
	postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/workspaces/current/members/"+newOwnerID+"/owner-transfer", map[string]any{
		"token": verifiedTransferToken,
	}, true, http.StatusOK)

	postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/logout", nil, true, http.StatusOK)
	consoleSSO := getJSON[map[string]any](restartedAgain, "/console/api/enterprise/sso/oidc/login", false, http.StatusOK)
	if url, _ := consoleSSO["url"].(string); url != "/apps" {
		t.Fatalf("expected console SSO local redirect, got %+v", consoleSSO)
	}
	if state, _ := consoleSSO["state"].(string); state == "" {
		t.Fatalf("expected console SSO state, got %+v", consoleSSO)
	}
	profile := getJSON[map[string]any](restartedAgain, "/console/api/account/profile", true, http.StatusOK)
	if email, _ := profile["email"].(string); email == "" {
		t.Fatalf("expected console SSO to establish session, got %+v", profile)
	}

	app := postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/apps", map[string]any{
		"name": "SSO Webapp",
		"mode": "advanced-chat",
		"icon": "🤝",
	}, true, http.StatusCreated)
	appCode := stringFromAny(mapFromAny(app["site"])["access_token"])
	if appCode == "" {
		t.Fatalf("expected app code, got %+v", app)
	}
	postJSON[map[string]any](restartedAgain, http.MethodPost, "/console/api/apps/"+stringFromAny(app["id"])+"/site-enable", map[string]any{
		"enable_site": true,
	}, true, http.StatusOK)
	redirectURL := "/chat/" + appCode
	publicSSO := getJSON[map[string]any](restartedAgain, "/api/enterprise/sso/oidc/login?app_code="+url.QueryEscape(appCode)+"&redirect_url="+url.QueryEscape(redirectURL), false, http.StatusOK)
	publicRedirect, _ := publicSSO["url"].(string)
	if !strings.Contains(publicRedirect, "web_sso_token=") {
		t.Fatalf("expected public SSO redirect to carry web_sso_token, got %+v", publicSSO)
	}
	parsedRedirect, err := url.Parse(publicRedirect)
	if err != nil {
		t.Fatalf("parse public SSO redirect: %v", err)
	}
	webSSOToken := parsedRedirect.Query().Get("web_sso_token")
	if webSSOToken == "" {
		t.Fatalf("expected web_sso_token in redirect URL, got %s", publicRedirect)
	}
	status := getJSONWithHeaders[map[string]any](restartedAgain, "/api/login/status?app_code="+url.QueryEscape(appCode), map[string]string{
		"Authorization": "Bearer " + webSSOToken,
	}, http.StatusOK)
	if loggedIn, ok := status["logged_in"].(bool); !ok || !loggedIn {
		t.Fatalf("expected public SSO bearer token to be accepted, got %+v", status)
	}
}

func ragPipelineRunInputs(language string, chunkSize int, removeExtraSpaces bool) map[string]any {
	return map[string]any{
		"separator":              "\n\n",
		"chunk_size":             chunkSize,
		"chunk_overlap":          20,
		"subchunk_separator":     "\n",
		"subchunk_chunk_size":    200,
		"subchunk_chunk_overlap": 20,
		"parent_mode":            "paragraph",
		"doc_language":           language,
		"doc_form":               "hierarchical_model",
		"summary_index_enabled":  false,
		"remove_extra_spaces":    removeExtraSpaces,
		"remove_urls_emails":     false,
	}
}

func newServerTestEnv(t *testing.T) *serverTestEnv {
	t.Helper()

	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	handler, err := New(config.Config{
		Addr:                 ":0",
		AppVersion:           "test",
		AppTitle:             "dify-go-test",
		Edition:              "SELF_HOSTED",
		EnvName:              "TEST",
		StateFile:            stateFile,
		UploadDir:            filepath.Join(tmpDir, "uploads"),
		DefaultWorkspaceName: "Default Workspace",
		WebOrigins:           []string{"http://localhost"},
		AccessTokenTTL:       time.Hour,
		RefreshTokenTTL:      24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("create server handler: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &serverTestEnv{
		t:         t,
		server:    server,
		client:    &http.Client{Jar: jar},
		stateFile: stateFile,
	}
}

func (e *serverTestEnv) setupAndLogin() {
	postJSON[map[string]any](e, http.MethodPost, "/console/api/setup", map[string]any{
		"email":    "tester@example.com",
		"name":     "Tester",
		"password": "password123",
		"language": "en-US",
	}, false, http.StatusCreated)
	postJSON[map[string]any](e, http.MethodPost, "/console/api/login", map[string]any{
		"email":    "tester@example.com",
		"password": "password123",
	}, false, http.StatusOK)
}

func (e *serverTestEnv) restart() *serverTestEnv {
	e.t.Helper()

	handler, err := New(config.Config{
		Addr:                 ":0",
		AppVersion:           "test",
		AppTitle:             "dify-go-test",
		Edition:              "SELF_HOSTED",
		EnvName:              "TEST",
		StateFile:            e.stateFile,
		UploadDir:            filepath.Join(filepath.Dir(e.stateFile), "uploads-restarted"),
		DefaultWorkspaceName: "Default Workspace",
		WebOrigins:           []string{"http://localhost"},
		AccessTokenTTL:       time.Hour,
		RefreshTokenTTL:      24 * time.Hour,
	})
	if err != nil {
		e.t.Fatalf("restart server handler: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatalf("create restarted cookie jar: %v", err)
	}
	server := httptest.NewServer(handler)
	e.t.Cleanup(server.Close)

	return &serverTestEnv{
		t:         e.t,
		server:    server,
		client:    &http.Client{Jar: jar},
		stateFile: e.stateFile,
	}
}

func (e *serverTestEnv) uploadFile(path string, auth bool, filename string, contentType string, content []byte) uploadResponse {
	e.t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		e.t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+path, &body)
	if err != nil {
		e.t.Fatalf("create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if auth {
		req.Header.Set(csrfHeader, e.csrfToken())
	}
	resp := e.do(req, http.StatusCreated)
	defer resp.Body.Close()

	var uploaded uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploaded); err != nil {
		e.t.Fatalf("decode upload response: %v", err)
	}
	if uploaded.MimeType == "" {
		uploaded.MimeType = contentType
	}
	return uploaded
}

func postJSON[T any](e *serverTestEnv, method string, path string, payload any, auth bool, wantStatus int) T {
	e.t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, e.server.URL+path, body)
	if err != nil {
		e.t.Fatalf("create json request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth && requiresCSRF(method) {
		req.Header.Set(csrfHeader, e.csrfToken())
	}
	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	if isNoContentStatus(wantStatus) {
		var zero T
		return zero
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		e.t.Fatalf("decode json response for %s %s: %v", method, path, err)
	}
	return result
}

func getJSON[T any](e *serverTestEnv, path string, auth bool, wantStatus int) T {
	e.t.Helper()

	req, err := http.NewRequest(http.MethodGet, e.server.URL+path, nil)
	if err != nil {
		e.t.Fatalf("create get request: %v", err)
	}
	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		e.t.Fatalf("decode get response for %s: %v", path, err)
	}
	return result
}

func getJSONWithHeaders[T any](e *serverTestEnv, path string, headers map[string]string, wantStatus int) T {
	e.t.Helper()

	req, err := http.NewRequest(http.MethodGet, e.server.URL+path, nil)
	if err != nil {
		e.t.Fatalf("create get request with headers: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		e.t.Fatalf("decode get response with headers for %s: %v", path, err)
	}
	return result
}

func postJSONWithHeaders[T any](e *serverTestEnv, method string, path string, headers map[string]string, payload any, wantStatus int) T {
	e.t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("marshal payload with headers: %v", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, e.server.URL+path, body)
	if err != nil {
		e.t.Fatalf("create json request with headers: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	if isNoContentStatus(wantStatus) {
		var zero T
		return zero
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		e.t.Fatalf("decode json response with headers for %s %s: %v", method, path, err)
	}
	return result
}

func postStreamJSON(e *serverTestEnv, path string, payload any, auth bool, wantStatus int) []map[string]any {
	e.t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("marshal stream payload: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+path, body)
	if err != nil {
		e.t.Fatalf("create stream request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		req.Header.Set(csrfHeader, e.csrfToken())
	}

	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read stream body for %s: %v", path, err)
	}

	events := []map[string]any{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
			e.t.Fatalf("decode stream event for %s: %v raw=%q", path, err, line)
		}
		events = append(events, payload)
	}
	return events
}

func postStreamJSONWithHeaders(e *serverTestEnv, path string, headers map[string]string, payload any, wantStatus int) []map[string]any {
	e.t.Helper()

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("marshal stream payload with headers: %v", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+path, body)
	if err != nil {
		e.t.Fatalf("create stream request with headers: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read stream body with headers for %s: %v", path, err)
	}

	events := []map[string]any{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
			e.t.Fatalf("decode stream event with headers for %s: %v raw=%q", path, err, line)
		}
		events = append(events, payload)
	}
	return events
}

func (e *serverTestEnv) getBytes(path string, auth bool, wantStatus int) []byte {
	e.t.Helper()

	fullURL := path
	if strings.HasPrefix(path, "/") {
		fullURL = e.server.URL + path
	}
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		e.t.Fatalf("create byte request: %v", err)
	}
	resp := e.do(req, wantStatus)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read response body for %s: %v", path, err)
	}
	return data
}

func (e *serverTestEnv) do(req *http.Request, wantStatus int) *http.Response {
	e.t.Helper()

	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("do request %s %s: %v", req.Method, req.URL.String(), err)
	}
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		e.t.Fatalf("unexpected status for %s %s: got %d want %d body=%s", req.Method, req.URL.Path, resp.StatusCode, wantStatus, string(body))
	}
	return resp
}

func (e *serverTestEnv) doNoRedirect(req *http.Request, wantStatus int) *http.Response {
	e.t.Helper()

	client := &http.Client{
		Jar: e.client.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		e.t.Fatalf("do request without redirect %s %s: %v", req.Method, req.URL.String(), err)
	}
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		e.t.Fatalf("unexpected status for %s %s without redirect: got %d want %d body=%s", req.Method, req.URL.Path, resp.StatusCode, wantStatus, string(body))
	}
	return resp
}

func (e *serverTestEnv) csrfToken() string {
	e.t.Helper()

	parsedURL, err := url.Parse(e.server.URL)
	if err != nil {
		e.t.Fatalf("parse server url: %v", err)
	}
	for _, cookie := range e.client.Jar.Cookies(parsedURL) {
		if cookie.Name == csrfTokenCookie {
			return cookie.Value
		}
	}
	e.t.Fatal("csrf token cookie not found")
	return ""
}

func nestedInt(payload map[string]any, parent, field string) int {
	parentMap, _ := payload[parent].(map[string]any)
	value, _ := parentMap[field].(float64)
	return int(value)
}

func nestedString(payload map[string]any, parent, field string) string {
	parentMap, _ := payload[parent].(map[string]any)
	value, _ := parentMap[field].(string)
	return value
}

func invitationToken(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse invitation url %q: %v", raw, err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatalf("expected invitation token in %q", raw)
	}
	return token
}

func accountIDByEmail(accounts []any, email string) string {
	for _, raw := range accounts {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(stringFromAny(item["email"]), email) {
			return stringFromAny(item["id"])
		}
	}
	return ""
}

func accountFieldByEmail(accounts []any, email, field string) string {
	for _, raw := range accounts {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(stringFromAny(item["email"]), email) {
			return stringFromAny(item[field])
		}
	}
	return ""
}

func findDatasourceAuthItem(t *testing.T, items []map[string]any, pluginID string) map[string]any {
	t.Helper()

	for _, item := range items {
		if stringFromAny(item["plugin_id"]) == pluginID {
			return item
		}
	}

	t.Fatalf("expected datasource auth item %s in %+v", pluginID, items)
	return nil
}

func findDatasourcePluginItem(t *testing.T, items []map[string]any, pluginID string) map[string]any {
	t.Helper()

	for _, item := range items {
		if stringFromAny(item["plugin_id"]) == pluginID {
			return item
		}
	}

	t.Fatalf("expected datasource plugin item %s in %+v", pluginID, items)
	return nil
}

func readPersistedState(t *testing.T, env *serverTestEnv) map[string]any {
	t.Helper()

	data, err := os.ReadFile(env.stateFile)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode state file: %v", err)
	}
	return payload
}

func findPersistedDatasetByID(t *testing.T, env *serverTestEnv, datasetID string) map[string]any {
	t.Helper()

	state := readPersistedState(t, env)
	for _, item := range objectListFromAny(state["datasets"]) {
		if stringFromAny(item["id"]) == datasetID {
			return item
		}
	}

	t.Fatalf("expected persisted dataset %s in %+v", datasetID, state["datasets"])
	return nil
}

func findPersistedDatasetByPipelineID(t *testing.T, env *serverTestEnv, pipelineID string) map[string]any {
	t.Helper()

	state := readPersistedState(t, env)
	for _, item := range objectListFromAny(state["datasets"]) {
		if stringFromAny(item["pipeline_id"]) == pipelineID {
			return item
		}
	}

	t.Fatalf("expected persisted dataset with pipeline_id %s in %+v", pipelineID, state["datasets"])
	return nil
}

func datasourcePluginAuthorized(t *testing.T, items []map[string]any, pluginID string) bool {
	t.Helper()

	for _, item := range items {
		if stringFromAny(item["plugin_id"]) == pluginID {
			return ragPipelineBoolValue(item["is_authorized"])
		}
	}

	t.Fatalf("expected datasource plugin %s in %+v", pluginID, items)
	return false
}

func findWorkflowRunByBatch(t *testing.T, items []map[string]any, batchID string) map[string]any {
	t.Helper()

	for _, item := range items {
		if stringFromAny(mapFromAny(item["outputs"])["batch"]) == batchID {
			return item
		}
	}

	t.Fatalf("expected workflow run batch %s in %+v", batchID, items)
	return nil
}

func findWorkflowRunByOriginalDocumentID(t *testing.T, items []map[string]any, documentID string) map[string]any {
	t.Helper()

	for _, item := range items {
		if stringFromAny(mapFromAny(item["outputs"])["original_document_id"]) == documentID {
			return item
		}
	}

	t.Fatalf("expected workflow run original_document_id %s in %+v", documentID, items)
	return nil
}

func findWorkflowVersionByMarkedName(t *testing.T, items []map[string]any, markedName string) map[string]any {
	t.Helper()

	for _, item := range items {
		if stringFromAny(item["marked_name"]) == markedName {
			return item
		}
	}

	t.Fatalf("expected workflow version %s in %+v", markedName, items)
	return nil
}

func installDatasourcePluginFromSpec(t *testing.T, env *serverTestEnv, pluginID, provider string) string {
	t.Helper()

	spec, ok := ragPipelineDatasourceProviderSpecByProvider(pluginID, provider)
	if !ok {
		t.Fatalf("datasource provider spec not found for %s/%s", pluginID, provider)
	}

	response := postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/plugin/install/marketplace", map[string]any{
		"plugin_unique_identifiers": []string{spec.PluginUniqueIdentifier},
	}, true, http.StatusOK)
	if stringFromAny(response["plugin_unique_identifier"]) != spec.PluginUniqueIdentifier {
		t.Fatalf("unexpected plugin install response for %s: %+v", pluginID, response)
	}

	return spec.PluginUniqueIdentifier
}

func uninstallWorkspacePluginByID(t *testing.T, env *serverTestEnv, pluginID string) {
	t.Helper()

	list := getJSON[map[string]any](env, "/console/api/workspaces/current/plugin/list?page=1&limit=100", true, http.StatusOK)
	plugins := objectListFromAny(list["plugins"])
	for _, item := range plugins {
		if stringFromAny(item["plugin_id"]) != pluginID {
			continue
		}
		postJSON[map[string]any](env, http.MethodPost, "/console/api/workspaces/current/plugin/uninstall", map[string]any{
			"plugin_installation_id": stringFromAny(item["installation_id"]),
		}, true, http.StatusOK)
		return
	}

	t.Fatalf("expected installed workspace plugin %s in %+v", pluginID, plugins)
}

func isNoContentStatus(status int) bool {
	return status == http.StatusNoContent
}

func mustPNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+S9kAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	return data
}
