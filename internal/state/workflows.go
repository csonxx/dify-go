package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) GetWorkflowDraft(appID, workspaceID string) (WorkflowState, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return WorkflowState{}, false, false
	}
	if app.WorkflowDraft == nil {
		return WorkflowState{}, false, true
	}
	return cloneWorkflowState(*app.WorkflowDraft), true, true
}

func (s *Store) GetPublishedWorkflow(appID, workspaceID string) (WorkflowState, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return WorkflowState{}, false, false
	}
	if app.WorkflowPublished != nil {
		return cloneWorkflowState(*app.WorkflowPublished), true, true
	}
	if app.WorkflowDraft != nil {
		fallback := cloneWorkflowState(*app.WorkflowDraft)
		fallback.ToolPublished = false
		return fallback, true, true
	}
	return WorkflowState{}, false, true
}

func (s *Store) SyncWorkflowDraft(appID, workspaceID string, user User, graph, features map[string]any, environmentVariables, conversationVariables []map[string]any, hash string, now time.Time) (WorkflowState, error) {
	return s.mutateWorkflow(appID, workspaceID, now, func(app *App) error {
		if app.Workflow == nil {
			return fmt.Errorf("workflow is not enabled for this app")
		}

		if app.WorkflowDraft != nil && hash != "" && app.WorkflowDraft.Hash != "" && app.WorkflowDraft.Hash != hash {
			return fmt.Errorf("draft_workflow_not_sync")
		}

		draft := &WorkflowState{}
		if app.WorkflowDraft != nil {
			copied := cloneWorkflowState(*app.WorkflowDraft)
			draft = &copied
		}
		if draft.ID == "" {
			draft.ID = firstNonEmpty(app.Workflow.ID, generateID("wf"))
			draft.CreatedAt = now.UTC().Unix()
			draft.CreatedBy = user.ID
			draft.Version = "draft"
		}

		draft.Graph = cloneMap(graph)
		draft.Features = cloneMap(features)
		draft.EnvironmentVariables = cloneObjectList(environmentVariables)
		draft.ConversationVariables = cloneObjectList(conversationVariables)
		draft.UpdatedAt = now.UTC().Unix()
		draft.UpdatedBy = user.ID
		draft.ToolPublished = app.WorkflowPublished != nil
		draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables)
		app.WorkflowDraft = draft
		if app.Workflow != nil {
			app.Workflow.UpdatedAt = draft.UpdatedAt
			app.Workflow.UpdatedBy = user.ID
		}
		return nil
	})
}

func (s *Store) UpdateWorkflowEnvironmentVariables(appID, workspaceID string, user User, environmentVariables []map[string]any, now time.Time) (WorkflowState, error) {
	return s.mutateWorkflow(appID, workspaceID, now, func(app *App) error {
		draft := ensureWorkflowDraft(app, user, now)
		draft.EnvironmentVariables = cloneObjectList(environmentVariables)
		draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables)
		app.WorkflowDraft = draft
		return nil
	})
}

func (s *Store) UpdateWorkflowConversationVariables(appID, workspaceID string, user User, conversationVariables []map[string]any, now time.Time) (WorkflowState, error) {
	return s.mutateWorkflow(appID, workspaceID, now, func(app *App) error {
		draft := ensureWorkflowDraft(app, user, now)
		draft.ConversationVariables = cloneObjectList(conversationVariables)
		draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables)
		app.WorkflowDraft = draft
		return nil
	})
}

func (s *Store) UpdateWorkflowFeatures(appID, workspaceID string, user User, features map[string]any, now time.Time) (WorkflowState, error) {
	return s.mutateWorkflow(appID, workspaceID, now, func(app *App) error {
		draft := ensureWorkflowDraft(app, user, now)
		draft.Features = cloneMap(features)
		draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables)
		app.WorkflowDraft = draft
		return nil
	})
}

func (s *Store) WorkflowEnvironmentVariables(appID, workspaceID string) ([]map[string]any, bool, bool) {
	draft, ok, exists := s.GetWorkflowDraft(appID, workspaceID)
	if !exists {
		return nil, false, false
	}
	if !ok {
		return []map[string]any{}, false, true
	}
	return cloneObjectList(draft.EnvironmentVariables), true, true
}

func (s *Store) PublishWorkflow(appID, workspaceID string, user User, markedName, markedComment string, now time.Time) (WorkflowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowState{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	if app.WorkflowDraft == nil {
		return WorkflowState{}, fmt.Errorf("draft_workflow_not_exist")
	}

	published := cloneWorkflowState(*app.WorkflowDraft)
	published.MarkedName = markedName
	published.MarkedComment = markedComment
	published.CreatedAt = now.UTC().Unix()
	published.CreatedBy = user.ID
	published.UpdatedAt = now.UTC().Unix()
	published.UpdatedBy = user.ID
	published.ToolPublished = true
	published.Version = "published"
	published.Hash = workflowHash(published.Graph, published.Features, published.EnvironmentVariables, published.ConversationVariables)
	app.WorkflowPublished = &published

	draft := cloneWorkflowState(*app.WorkflowDraft)
	draft.ToolPublished = true
	draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables)
	app.WorkflowDraft = &draft

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
	return cloneWorkflowState(published), nil
}

func (s *Store) mutateWorkflow(appID, workspaceID string, now time.Time, apply func(app *App) error) (WorkflowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return WorkflowState{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	if err := apply(&app); err != nil {
		return WorkflowState{}, err
	}
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = app.CreatedBy
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return WorkflowState{}, err
	}
	if app.WorkflowDraft != nil {
		return cloneWorkflowState(*app.WorkflowDraft), nil
	}
	if app.WorkflowPublished != nil {
		return cloneWorkflowState(*app.WorkflowPublished), nil
	}
	return WorkflowState{}, nil
}

func ensureWorkflowDraft(app *App, user User, now time.Time) *WorkflowState {
	if app.WorkflowDraft != nil {
		draft := cloneWorkflowState(*app.WorkflowDraft)
		draft.UpdatedAt = now.UTC().Unix()
		draft.UpdatedBy = user.ID
		return &draft
	}

	return &WorkflowState{
		ID:                    firstNonEmpty(stringValueFromWorkflow(app.Workflow), generateID("wf")),
		Graph:                 map[string]any{"nodes": []any{}, "edges": []any{}, "viewport": map[string]any{"x": 0, "y": 0, "zoom": 1}},
		Features:              map[string]any{},
		CreatedAt:             now.UTC().Unix(),
		CreatedBy:             user.ID,
		UpdatedAt:             now.UTC().Unix(),
		UpdatedBy:             user.ID,
		EnvironmentVariables:  []map[string]any{},
		ConversationVariables: []map[string]any{},
		Version:               "draft",
	}
}

func stringValueFromWorkflow(workflow *Workflow) string {
	if workflow == nil {
		return ""
	}
	return workflow.ID
}

func workflowHash(graph, features map[string]any, env, conv []map[string]any) string {
	payload := map[string]any{
		"graph":                  graph,
		"features":               features,
		"environment_variables":  env,
		"conversation_variables": conv,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:16])
}

func cloneWorkflowState(src WorkflowState) WorkflowState {
	return WorkflowState{
		ID:                    src.ID,
		Graph:                 cloneMap(src.Graph),
		Features:              cloneMap(src.Features),
		CreatedBy:             src.CreatedBy,
		CreatedAt:             src.CreatedAt,
		UpdatedBy:             src.UpdatedBy,
		UpdatedAt:             src.UpdatedAt,
		Hash:                  src.Hash,
		ToolPublished:         src.ToolPublished,
		EnvironmentVariables:  cloneObjectList(src.EnvironmentVariables),
		ConversationVariables: cloneObjectList(src.ConversationVariables),
		Version:               src.Version,
		MarkedName:            src.MarkedName,
		MarkedComment:         src.MarkedComment,
	}
}

func cloneObjectList(src []map[string]any) []map[string]any {
	if src == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, len(src))
	for i, item := range src {
		out[i] = cloneMap(item)
	}
	return out
}
