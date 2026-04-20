package state

import (
	"fmt"
	"strings"
	"time"
)

type WorkflowRun struct {
	ID             string                  `json:"id"`
	TaskID         string                  `json:"task_id"`
	Version        string                  `json:"version"`
	Graph          map[string]any          `json:"graph"`
	Inputs         map[string]any          `json:"inputs"`
	Status         string                  `json:"status"`
	Outputs        map[string]any          `json:"outputs"`
	Error          string                  `json:"error,omitempty"`
	ElapsedTime    float64                 `json:"elapsed_time"`
	TotalTokens    int                     `json:"total_tokens"`
	TotalPrice     float64                 `json:"total_price"`
	Currency       string                  `json:"currency"`
	TotalSteps     int                     `json:"total_steps"`
	CreatedAt      int64                   `json:"created_at"`
	FinishedAt     int64                   `json:"finished_at"`
	CreatedBy      string                  `json:"created_by"`
	ConversationID string                  `json:"conversation_id,omitempty"`
	MessageID      string                  `json:"message_id,omitempty"`
	NodeExecutions []WorkflowNodeExecution `json:"node_executions,omitempty"`
}

type WorkflowNodeExecution struct {
	ID                string         `json:"id"`
	Index             int            `json:"index"`
	PredecessorNodeID string         `json:"predecessor_node_id"`
	NodeID            string         `json:"node_id"`
	NodeType          string         `json:"node_type"`
	Title             string         `json:"title"`
	Inputs            map[string]any `json:"inputs"`
	ProcessData       map[string]any `json:"process_data"`
	Outputs           map[string]any `json:"outputs"`
	Status            string         `json:"status"`
	Error             string         `json:"error,omitempty"`
	ElapsedTime       float64        `json:"elapsed_time"`
	TotalTokens       int            `json:"total_tokens"`
	TotalPrice        float64        `json:"total_price"`
	Currency          string         `json:"currency"`
	CreatedAt         int64          `json:"created_at"`
	FinishedAt        int64          `json:"finished_at"`
	CreatedBy         string         `json:"created_by"`
}

func (s *Store) ListWorkflowVersions(appID, workspaceID string, page, limit int, userID string, namedOnly bool) ([]WorkflowState, bool, int, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return nil, false, page, false
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	versions := cloneWorkflowStateList(app.WorkflowVersions)
	if len(versions) == 0 && app.WorkflowPublished != nil {
		versions = []WorkflowState{cloneWorkflowState(*app.WorkflowPublished)}
	}

	filtered := make([]WorkflowState, 0, len(versions)+1)
	if app.WorkflowDraft != nil {
		filtered = append(filtered, cloneWorkflowState(*app.WorkflowDraft))
	}
	for _, version := range versions {
		if userID != "" && version.CreatedBy != userID {
			continue
		}
		if namedOnly && strings.TrimSpace(version.MarkedName) == "" {
			continue
		}
		filtered = append(filtered, version)
	}

	start := (page - 1) * limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], end < len(filtered), page, true
}

func (s *Store) UpdateWorkflowVersionMetadata(appID, workspaceID, versionID string, user User, markedName, markedComment string, now time.Time) (WorkflowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowState{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	ensureWorkflowVersions(&app)
	versionIndex := workflowVersionIndex(app.WorkflowVersions, versionID)
	if versionIndex < 0 {
		return WorkflowState{}, fmt.Errorf("workflow_version_not_found")
	}

	version := cloneWorkflowState(app.WorkflowVersions[versionIndex])
	version.MarkedName = strings.TrimSpace(markedName)
	version.MarkedComment = strings.TrimSpace(markedComment)
	version.UpdatedAt = now.UTC().Unix()
	version.UpdatedBy = user.ID
	app.WorkflowVersions[versionIndex] = version
	if app.WorkflowPublished != nil && app.WorkflowPublished.ID == versionID {
		published := cloneWorkflowState(*app.WorkflowPublished)
		published.MarkedName = version.MarkedName
		published.MarkedComment = version.MarkedComment
		published.UpdatedAt = version.UpdatedAt
		published.UpdatedBy = version.UpdatedBy
		app.WorkflowPublished = &published
	}

	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return WorkflowState{}, err
	}
	return cloneWorkflowState(version), nil
}

func (s *Store) DeleteWorkflowVersion(appID, workspaceID, versionID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	ensureWorkflowVersions(&app)
	if app.WorkflowPublished != nil && app.WorkflowPublished.ID == versionID {
		return fmt.Errorf("latest_workflow_version_cannot_delete")
	}

	versionIndex := workflowVersionIndex(app.WorkflowVersions, versionID)
	if versionIndex < 0 {
		return fmt.Errorf("workflow_version_not_found")
	}

	app.WorkflowVersions = append(app.WorkflowVersions[:versionIndex], app.WorkflowVersions[versionIndex+1:]...)
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return s.saveLocked()
}

func (s *Store) RestoreWorkflowVersion(appID, workspaceID, versionID string, user User, now time.Time) (WorkflowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowState{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	ensureWorkflowVersions(&app)
	versionIndex := workflowVersionIndex(app.WorkflowVersions, versionID)
	if versionIndex < 0 {
		return WorkflowState{}, fmt.Errorf("workflow_version_not_found")
	}

	draft := cloneWorkflowState(app.WorkflowVersions[versionIndex])
	if app.WorkflowDraft != nil && strings.TrimSpace(app.WorkflowDraft.ID) != "" {
		draft.ID = app.WorkflowDraft.ID
	} else {
		draft.ID = generateID("wf")
	}
	draft.Version = "draft"
	draft.UpdatedAt = now.UTC().Unix()
	draft.UpdatedBy = user.ID
	draft.ToolPublished = app.WorkflowPublished != nil
	draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables, draft.RagPipelineVariables)
	app.WorkflowDraft = &draft
	app.UpdatedAt = draft.UpdatedAt
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return WorkflowState{}, err
	}
	return cloneWorkflowState(draft), nil
}

func (s *Store) SaveWorkflowRun(appID, workspaceID string, user User, run WorkflowRun, now time.Time) (WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowRun{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	app.WorkflowRuns = append([]WorkflowRun{cloneWorkflowRun(run)}, app.WorkflowRuns...)
	if len(app.WorkflowRuns) > 100 {
		app.WorkflowRuns = app.WorkflowRuns[:100]
	}
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return WorkflowRun{}, err
	}
	return cloneWorkflowRun(run), nil
}

func (s *Store) ListWorkflowRuns(appID, workspaceID string) ([]WorkflowRun, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return nil, false
	}

	runs := cloneWorkflowRunList(app.WorkflowRuns)
	return runs, true
}

func (s *Store) GetWorkflowRun(appID, workspaceID, runID string) (WorkflowRun, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return WorkflowRun{}, false, false
	}

	for _, run := range app.WorkflowRuns {
		if run.ID == runID {
			return cloneWorkflowRun(run), true, true
		}
	}
	return WorkflowRun{}, false, true
}

func (s *Store) StopWorkflowRun(appID, workspaceID, taskID string, user User, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return false, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	updated := false
	for i := range app.WorkflowRuns {
		if app.WorkflowRuns[i].TaskID != taskID {
			continue
		}
		if app.WorkflowRuns[i].Status == "running" {
			app.WorkflowRuns[i].Status = "stopped"
			app.WorkflowRuns[i].FinishedAt = now.UTC().Unix()
		}
		for j := range app.WorkflowRuns[i].NodeExecutions {
			if app.WorkflowRuns[i].NodeExecutions[j].Status == "running" {
				app.WorkflowRuns[i].NodeExecutions[j].Status = "stopped"
				app.WorkflowRuns[i].NodeExecutions[j].FinishedAt = now.UTC().Unix()
			}
		}
		updated = true
		break
	}
	if !updated {
		return false, nil
	}

	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return true, s.saveLocked()
}

func (s *Store) SaveWorkflowNodeRun(appID, workspaceID string, user User, execution WorkflowNodeExecution, now time.Time) (WorkflowNodeExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowNodeExecution{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	if app.WorkflowNodeRuns == nil {
		app.WorkflowNodeRuns = map[string]WorkflowNodeExecution{}
	}
	app.WorkflowNodeRuns[execution.NodeID] = cloneWorkflowNodeExecution(execution)
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return WorkflowNodeExecution{}, err
	}
	return cloneWorkflowNodeExecution(execution), nil
}

func (s *Store) GetWorkflowNodeRun(appID, workspaceID, nodeID string) (WorkflowNodeExecution, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return WorkflowNodeExecution{}, false, false
	}
	if execution, ok := app.WorkflowNodeRuns[nodeID]; ok {
		return cloneWorkflowNodeExecution(execution), true, true
	}
	for _, run := range app.WorkflowRuns {
		for _, execution := range run.NodeExecutions {
			if execution.NodeID == nodeID {
				return cloneWorkflowNodeExecution(execution), true, true
			}
		}
	}
	return WorkflowNodeExecution{}, false, true
}

func ensureWorkflowVersions(app *App) {
	if len(app.WorkflowVersions) == 0 && app.WorkflowPublished != nil {
		app.WorkflowVersions = []WorkflowState{cloneWorkflowState(*app.WorkflowPublished)}
	}
}

func workflowVersionIndex(items []WorkflowState, versionID string) int {
	for i, item := range items {
		if item.ID == versionID {
			return i
		}
	}
	return -1
}

func cloneWorkflowStateList(src []WorkflowState) []WorkflowState {
	if src == nil {
		return []WorkflowState{}
	}
	out := make([]WorkflowState, len(src))
	for i, item := range src {
		out[i] = cloneWorkflowState(item)
	}
	return out
}

func cloneWorkflowRun(src WorkflowRun) WorkflowRun {
	return WorkflowRun{
		ID:             src.ID,
		TaskID:         src.TaskID,
		Version:        src.Version,
		Graph:          cloneMap(src.Graph),
		Inputs:         cloneMap(src.Inputs),
		Status:         src.Status,
		Outputs:        cloneMap(src.Outputs),
		Error:          src.Error,
		ElapsedTime:    src.ElapsedTime,
		TotalTokens:    src.TotalTokens,
		TotalPrice:     src.TotalPrice,
		Currency:       src.Currency,
		TotalSteps:     src.TotalSteps,
		CreatedAt:      src.CreatedAt,
		FinishedAt:     src.FinishedAt,
		CreatedBy:      src.CreatedBy,
		ConversationID: src.ConversationID,
		MessageID:      src.MessageID,
		NodeExecutions: cloneWorkflowNodeExecutionList(src.NodeExecutions),
	}
}

func cloneWorkflowRunList(src []WorkflowRun) []WorkflowRun {
	if src == nil {
		return []WorkflowRun{}
	}
	out := make([]WorkflowRun, len(src))
	for i, item := range src {
		out[i] = cloneWorkflowRun(item)
	}
	return out
}

func cloneWorkflowNodeExecution(src WorkflowNodeExecution) WorkflowNodeExecution {
	return WorkflowNodeExecution{
		ID:                src.ID,
		Index:             src.Index,
		PredecessorNodeID: src.PredecessorNodeID,
		NodeID:            src.NodeID,
		NodeType:          src.NodeType,
		Title:             src.Title,
		Inputs:            cloneMap(src.Inputs),
		ProcessData:       cloneMap(src.ProcessData),
		Outputs:           cloneMap(src.Outputs),
		Status:            src.Status,
		Error:             src.Error,
		ElapsedTime:       src.ElapsedTime,
		TotalTokens:       src.TotalTokens,
		TotalPrice:        src.TotalPrice,
		Currency:          src.Currency,
		CreatedAt:         src.CreatedAt,
		FinishedAt:        src.FinishedAt,
		CreatedBy:         src.CreatedBy,
	}
}

func cloneWorkflowNodeExecutionList(src []WorkflowNodeExecution) []WorkflowNodeExecution {
	if src == nil {
		return []WorkflowNodeExecution{}
	}
	out := make([]WorkflowNodeExecution, len(src))
	for i, item := range src {
		out[i] = cloneWorkflowNodeExecution(item)
	}
	return out
}
