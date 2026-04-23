package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

func (s *server) handlePublicWorkflowRun(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	app, workflow, actor, ok := s.currentPublicWorkflowApp(w, r)
	if !ok {
		return
	}

	now := time.Now()
	run := s.buildWorkflowRunForState(app, workflow, actor, payload, workflowRunOptions{Mode: "published"}, now)
	if _, err := s.store.SaveWorkflowRun(app.ID, app.WorkspaceID, actor, run, now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist workflow run.")
		return
	}

	responseMode := strings.ToLower(strings.TrimSpace(stringFromAny(payload["response_mode"])))
	if responseMode != "" && responseMode != "streaming" {
		writeJSON(w, http.StatusOK, map[string]any{
			"task_id":         run.TaskID,
			"workflow_run_id": run.ID,
			"data":            s.workflowFinishedEvent(app, run)["data"],
		})
		return
	}

	if err := s.streamWorkflowEvents(w, r, s.workflowRunEvents(app, run, false)); err != nil {
		return
	}
}

func (s *server) handlePublicWorkflowRunStop(w http.ResponseWriter, r *http.Request) {
	app, _, actor, ok := s.currentPublicWorkflowApp(w, r)
	if !ok {
		return
	}

	if _, err := s.store.StopWorkflowRun(app.ID, app.WorkspaceID, chi.URLParam(r, "taskID"), actor, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) currentPublicWorkflowApp(w http.ResponseWriter, r *http.Request) (state.App, state.WorkflowState, state.User, bool) {
	app, _, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return state.App{}, state.WorkflowState{}, state.User{}, false
	}
	if !isWorkflowApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Workflow is not enabled for this app.")
		return state.App{}, state.WorkflowState{}, state.User{}, false
	}
	if strings.TrimSpace(app.AccessMode) != "public" {
		writeError(w, http.StatusUnauthorized, "web_app_access_denied", "Web app access denied.")
		return state.App{}, state.WorkflowState{}, state.User{}, false
	}
	if app.WorkflowPublished == nil {
		writeError(w, http.StatusNotFound, "published_workflow_not_exist", "Published workflow does not exist.")
		return state.App{}, state.WorkflowState{}, state.User{}, false
	}

	return app, *app.WorkflowPublished, s.publicWorkflowActor(app), true
}

func (s *server) publicWorkflowActor(app state.App) state.User {
	if user, ok := s.store.GetUser(app.CreatedBy); ok {
		return user
	}

	return state.User{
		ID:          app.CreatedBy,
		Name:        app.AuthorName,
		WorkspaceID: app.WorkspaceID,
	}
}
