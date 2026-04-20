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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/langgenius/dify-go/internal/config"
)

type serverTestEnv struct {
	t      *testing.T
	server *httptest.Server
	client *http.Client
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
	ID                string `json:"id"`
	PipelineID        string `json:"pipeline_id"`
	RuntimeMode       string `json:"runtime_mode"`
	Permission        string `json:"permission"`
	DocForm           string `json:"doc_form"`
	IndexingTechnique string `json:"indexing_technique"`
	IsPublished       bool   `json:"is_published"`
}

type workflowDraftResponse struct {
	ID                   string           `json:"id"`
	Hash                 string           `json:"hash"`
	RagPipelineVariables []map[string]any `json:"rag_pipeline_variables"`
}

type workflowSyncResponse struct {
	Result    string `json:"result"`
	UpdatedAt int64  `json:"updated_at"`
	Hash      string `json:"hash"`
}

type ragPipelineParametersResponse struct {
	Variables []map[string]any `json:"variables"`
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

	plugins := getJSON[[]any](env, "/console/api/rag/pipelines/datasource-plugins", true, http.StatusOK)
	if len(plugins) != 0 {
		t.Fatalf("expected empty datasource plugin list, got %+v", plugins)
	}

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

func newServerTestEnv(t *testing.T) *serverTestEnv {
	t.Helper()

	tmpDir := t.TempDir()
	handler, err := New(config.Config{
		Addr:                 ":0",
		AppVersion:           "test",
		AppTitle:             "dify-go-test",
		Edition:              "SELF_HOSTED",
		EnvName:              "TEST",
		StateFile:            filepath.Join(tmpDir, "state.json"),
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
		t:      t,
		server: server,
		client: &http.Client{Jar: jar},
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
