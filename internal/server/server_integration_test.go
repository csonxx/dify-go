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

	reprocessRunID := stringFromAny(reprocessRun["id"])
	reprocessRunDetail := getJSON[map[string]any](env, "/console/api/rag/pipelines/"+dataset.PipelineID+"/workflow-runs/"+reprocessRunID, true, http.StatusOK)
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
