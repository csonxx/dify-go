package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) mountRAGPipelineRoutes(r chi.Router) {
	r.Route("/rag/pipeline", func(r chi.Router) {
		r.Post("/empty-dataset", s.handleRAGPipelineEmptyDatasetCreate)
		r.Post("/dataset", s.handleRAGPipelineDatasetCreateFromDSL)
		r.NotFound(s.compatFallback)
	})

	r.Get("/rag/pipelines/datasource-plugins", s.handleRAGPipelineDatasourcePlugins)
	r.Post("/rag/pipelines/imports", s.handleRAGPipelineImport)
	r.Post("/rag/pipelines/imports/{importID}/confirm", s.handleRAGPipelineImportConfirm)
	r.Route("/rag/pipelines/{pipelineID}", func(r chi.Router) {
		r.Use(s.withResolvedPipelineApp)
		r.Get("/exports", s.handleRAGPipelineExport)
		r.Get("/workflows/draft", s.handleWorkflowDraftGet)
		r.Post("/workflows/draft", s.handleWorkflowDraftSync)
		r.Get("/workflows/draft/environment-variables", s.handleWorkflowDraftEnvironmentVariables)
		r.Post("/workflows/draft/environment-variables", s.handleWorkflowDraftEnvironmentVariablesUpdate)
		r.Get("/workflows/draft/conversation-variables", s.handleWorkflowDraftConversationVariables)
		r.Post("/workflows/draft/conversation-variables", s.handleWorkflowDraftConversationVariablesUpdate)
		r.Get("/workflows/draft/system-variables", s.handleWorkflowDraftSystemVariables)
		r.Get("/workflows/draft/variables", s.handleWorkflowDraftVariables)
		r.Delete("/workflows/draft/variables", s.handleWorkflowDraftVariablesDelete)
		r.Delete("/workflows/draft/variables/{varID}", s.handleWorkflowDraftVariableDelete)
		r.Put("/workflows/draft/variables/{varID}/reset", s.handleWorkflowDraftVariableReset)
		r.Get("/workflows/draft/nodes/{nodeID}/variables", s.handleWorkflowDraftNodeVariables)
		r.Delete("/workflows/draft/nodes/{nodeID}/variables", s.handleWorkflowDraftNodeVariablesDelete)
		r.Get("/workflows/draft/nodes/{nodeID}/last-run", s.handleWorkflowDraftNodeLastRun)
		r.Post("/workflows/draft/nodes/{nodeID}/run", s.handleWorkflowDraftNodeRun)
		r.Post("/workflows/draft/nodes/{nodeID}/trigger/run", s.handleWorkflowDraftNodeTriggerRun)
		r.Post("/workflows/draft/iteration/nodes/{nodeID}/run", s.handleWorkflowDraftIterationNodeRun)
		r.Post("/workflows/draft/loop/nodes/{nodeID}/run", s.handleWorkflowDraftLoopNodeRun)
		r.Post("/workflows/draft/features", s.handleWorkflowDraftFeaturesUpdate)
		r.Post("/workflows/draft/run", s.handleWorkflowDraftRun)
		r.Post("/workflows/draft/trigger/run", s.handleWorkflowDraftTriggerRun)
		r.Post("/workflows/draft/trigger/run-all", s.handleWorkflowDraftTriggerRunAll)
		r.Get("/workflows/draft/pre-processing/parameters", s.handleRAGPipelineDraftPreProcessingParameters)
		r.Get("/workflows/draft/processing/parameters", s.handleRAGPipelineDraftProcessingParameters)
		r.Get("/workflows/default-workflow-block-configs", s.handleWorkflowDefaultBlockConfigs)
		r.Get("/workflows/default-workflow-block-configs/{blockType}", s.handleWorkflowDefaultBlockConfig)
		r.Get("/workflows/publish", s.handleWorkflowPublishedGet)
		r.Post("/workflows/publish", s.handleWorkflowPublish)
		r.Get("/workflows/published/pre-processing/parameters", s.handleRAGPipelinePublishedPreProcessingParameters)
		r.Get("/workflows/published/processing/parameters", s.handleRAGPipelinePublishedProcessingParameters)
		r.Get("/workflows", s.handleWorkflowVersionList)
		r.Post("/workflows/{versionID}/restore", s.handleWorkflowVersionRestore)
		r.Patch("/workflows/{versionID}", s.handleWorkflowVersionUpdate)
		r.Delete("/workflows/{versionID}", s.handleWorkflowVersionDelete)
		r.Get("/workflow-runs", s.handleWorkflowRunHistory)
		r.Post("/workflow-runs/tasks/{taskID}/stop", s.handleWorkflowRunStop)
		r.Get("/workflow-runs/{runID}/node-executions", s.handleWorkflowRunNodeExecutions)
		r.Get("/workflow-runs/{runID}", s.handleWorkflowRunDetail)
		r.NotFound(s.compatFallback)
	})
}

func (s *server) withResolvedPipelineApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspace, ok := s.currentUserWorkspace(r)
		if !ok {
			writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
			return
		}

		pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineID"))
		if pipelineID == "" {
			writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
			return
		}

		app, found := s.findPipelineDependencyApp(workspace.ID, pipelineID)
		if !found {
			if s.legacy != nil {
				s.compatFallback(w, r)
				return
			}
			writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
			return
		}

		ctx := context.WithValue(r.Context(), resolvedAppContextKey, app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) handleRAGPipelineEmptyDatasetCreate(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.currentUserWorkspace(r)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	dataset, _, err := s.store.CreateRAGPipelineDataset(workspace.ID, user, state.CreateRAGPipelineDatasetInput{}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	response := s.datasetResponse(dataset)
	response["dataset_id"] = dataset.ID
	writeJSON(w, http.StatusCreated, response)
}

func (s *server) handleRAGPipelineDatasourcePlugins(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentUserWorkspace(r); !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	writeJSON(w, http.StatusOK, []any{})
}

func (s *server) handleRAGPipelineDraftPreProcessingParameters(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineParameters(w, r, true)
}

func (s *server) handleRAGPipelineDraftProcessingParameters(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineParameters(w, r, true)
}

func (s *server) handleRAGPipelinePublishedPreProcessingParameters(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineParameters(w, r, false)
}

func (s *server) handleRAGPipelinePublishedProcessingParameters(w http.ResponseWriter, r *http.Request) {
	s.handleRAGPipelineParameters(w, r, false)
}

func (s *server) handleRAGPipelineParameters(w http.ResponseWriter, r *http.Request, isDraft bool) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var workflow *state.WorkflowState
	if isDraft {
		workflow = app.WorkflowDraft
	} else {
		workflow = app.WorkflowPublished
	}
	if workflow == nil {
		writeJSON(w, http.StatusOK, map[string]any{"variables": []any{}})
		return
	}

	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))
	variables := filterRAGPipelineVariables(workflow.RagPipelineVariables, nodeID)
	writeJSON(w, http.StatusOK, map[string]any{"variables": variables})
}

func filterRAGPipelineVariables(items []map[string]any, nodeID string) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	if nodeID == "" {
		return cloneMapList(items)
	}

	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		belongToNodeID := strings.TrimSpace(stringFromAny(item["belong_to_node_id"]))
		if belongToNodeID == "" || belongToNodeID == "shared" || belongToNodeID == nodeID {
			filtered = append(filtered, cloneJSONObject(item))
		}
	}
	return filtered
}
