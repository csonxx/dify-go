package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type AppMCPServer struct {
	ID          string            `json:"id"`
	ServerCode  string            `json:"server_code"`
	Description string            `json:"description"`
	Status      string            `json:"status"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
}

type AppAnnotation struct {
	ID        string `json:"id"`
	MessageID string `json:"message_id,omitempty"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	AccountID string `json:"account_id"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type AppMessageFeedback struct {
	MessageID     string `json:"message_id"`
	Rating        string `json:"rating"`
	Content       string `json:"content"`
	FromSource    string `json:"from_source"`
	FromEndUserID string `json:"from_end_user_id,omitempty"`
	UpdatedAt     int64  `json:"updated_at"`
}

type UpsertAppMCPServerInput struct {
	ID          string
	Description string
	Status      string
	Parameters  map[string]string
	Headers     map[string]string
}

type SaveAppAnnotationInput struct {
	ID        string
	MessageID string
	Question  string
	Answer    string
}

type SaveAppMessageFeedbackInput struct {
	MessageID     string
	Rating        string
	Content       string
	FromSource    string
	FromEndUserID string
}

func (s *Store) GetAppMCPServer(appID, workspaceID string) (AppMCPServer, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return AppMCPServer{}, false, false
	}
	if app.MCPServer == nil {
		return AppMCPServer{}, false, true
	}
	return cloneAppMCPServer(*app.MCPServer), true, true
}

func (s *Store) FindAppByMCPServer(workspaceID, serverID string) (App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, app := range s.state.Apps {
		if app.WorkspaceID != workspaceID || app.MCPServer == nil {
			continue
		}
		if app.MCPServer.ID == serverID {
			return app, true
		}
	}
	return App{}, false
}

func (s *Store) FindWorkflowRun(workspaceID, runID string) (App, WorkflowRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, app := range s.state.Apps {
		if app.WorkspaceID != workspaceID {
			continue
		}
		for _, run := range app.WorkflowRuns {
			if run.ID == runID {
				return app, cloneWorkflowRun(run), true
			}
		}
	}
	return App{}, WorkflowRun{}, false
}

func (s *Store) UpsertAppMCPServer(appID, workspaceID string, user User, input UpsertAppMCPServerInput, now time.Time) (AppMCPServer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppMCPServer{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	timestamp := now.UTC().Unix()
	server := AppMCPServer{
		ID:          generateID("mcp_server"),
		ServerCode:  generateID("mcp"),
		Description: strings.TrimSpace(input.Description),
		Status:      normalizeMCPServerStatus(input.Status),
		Parameters:  cloneStringMap(input.Parameters),
		Headers:     cloneStringMap(input.Headers),
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
	}
	if app.MCPServer != nil {
		server = cloneAppMCPServer(*app.MCPServer)
	}
	if strings.TrimSpace(input.ID) != "" {
		server.ID = strings.TrimSpace(input.ID)
	}
	if strings.TrimSpace(server.ID) == "" {
		server.ID = generateID("mcp_server")
	}
	if strings.TrimSpace(server.ServerCode) == "" {
		server.ServerCode = generateID("mcp")
	}
	if input.Description != "" || app.MCPServer == nil {
		server.Description = strings.TrimSpace(input.Description)
	}
	if input.Parameters != nil {
		server.Parameters = cloneStringMap(input.Parameters)
	}
	if input.Headers != nil {
		server.Headers = cloneStringMap(input.Headers)
	}
	if strings.TrimSpace(input.Status) != "" || app.MCPServer == nil {
		server.Status = normalizeMCPServerStatus(input.Status)
	}
	server.UpdatedAt = timestamp
	if server.CreatedAt == 0 {
		server.CreatedAt = timestamp
	}

	app.MCPServer = &server
	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppMCPServer{}, err
	}
	return cloneAppMCPServer(server), nil
}

func (s *Store) RefreshAppMCPServer(appID, workspaceID string, user User, now time.Time) (AppMCPServer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppMCPServer{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	timestamp := now.UTC().Unix()
	server := AppMCPServer{
		ID:         generateID("mcp_server"),
		Status:     "inactive",
		Parameters: map[string]string{},
		Headers:    map[string]string{},
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp,
	}
	if app.MCPServer != nil {
		server = cloneAppMCPServer(*app.MCPServer)
	}
	if strings.TrimSpace(server.ID) == "" {
		server.ID = generateID("mcp_server")
	}
	server.ServerCode = generateID("mcp")
	server.UpdatedAt = timestamp
	if server.CreatedAt == 0 {
		server.CreatedAt = timestamp
	}

	app.MCPServer = &server
	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppMCPServer{}, err
	}
	return cloneAppMCPServer(server), nil
}

func (s *Store) SaveAppAnnotation(appID, workspaceID string, user User, input SaveAppAnnotationInput, now time.Time) (AppAnnotation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppAnnotation{}, fmt.Errorf("app not found")
	}

	answer := strings.TrimSpace(input.Answer)
	if answer == "" {
		return AppAnnotation{}, fmt.Errorf("annotation answer is required")
	}

	app := s.state.Apps[index]
	timestamp := now.UTC().Unix()
	annotationIndex := slices.IndexFunc(app.Annotations, func(item AppAnnotation) bool {
		if strings.TrimSpace(input.ID) != "" && item.ID == input.ID {
			return true
		}
		return strings.TrimSpace(input.MessageID) != "" && item.MessageID == input.MessageID
	})

	annotation := AppAnnotation{
		ID:        generateID("annotation"),
		MessageID: strings.TrimSpace(input.MessageID),
		Question:  strings.TrimSpace(input.Question),
		Answer:    answer,
		AccountID: user.ID,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}
	if annotationIndex >= 0 {
		annotation = cloneAppAnnotation(app.Annotations[annotationIndex])
		annotation.MessageID = firstNonEmpty(strings.TrimSpace(input.MessageID), annotation.MessageID)
		annotation.Question = firstNonEmpty(strings.TrimSpace(input.Question), annotation.Question)
		annotation.Answer = answer
		annotation.AccountID = user.ID
		annotation.UpdatedAt = timestamp
		app.Annotations[annotationIndex] = annotation
	} else {
		app.Annotations = append(app.Annotations, annotation)
	}

	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppAnnotation{}, err
	}
	return cloneAppAnnotation(annotation), nil
}

func (s *Store) DeleteAppAnnotation(appID, workspaceID, annotationID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	annotationIndex := slices.IndexFunc(app.Annotations, func(item AppAnnotation) bool {
		return item.ID == annotationID
	})
	if annotationIndex < 0 {
		return fmt.Errorf("annotation not found")
	}

	app.Annotations = append(app.Annotations[:annotationIndex], app.Annotations[annotationIndex+1:]...)
	timestamp := now.UTC().Unix()
	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return s.saveLocked()
}

func (s *Store) SaveAppMessageFeedback(appID, workspaceID string, user User, input SaveAppMessageFeedbackInput, now time.Time) (AppMessageFeedback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppMessageFeedback{}, fmt.Errorf("app not found")
	}

	messageID := strings.TrimSpace(input.MessageID)
	if messageID == "" {
		return AppMessageFeedback{}, fmt.Errorf("message_id is required")
	}

	app := s.state.Apps[index]
	timestamp := now.UTC().Unix()
	feedback := AppMessageFeedback{
		MessageID:     messageID,
		Rating:        strings.TrimSpace(input.Rating),
		Content:       strings.TrimSpace(input.Content),
		FromSource:    firstNonEmpty(strings.TrimSpace(input.FromSource), "admin"),
		FromEndUserID: strings.TrimSpace(input.FromEndUserID),
		UpdatedAt:     timestamp,
	}

	feedbackIndex := slices.IndexFunc(app.MessageFeedbacks, func(item AppMessageFeedback) bool {
		return item.MessageID == feedback.MessageID &&
			item.FromSource == feedback.FromSource &&
			item.FromEndUserID == feedback.FromEndUserID
	})
	if feedbackIndex >= 0 {
		app.MessageFeedbacks[feedbackIndex] = feedback
	} else {
		app.MessageFeedbacks = append(app.MessageFeedbacks, feedback)
	}

	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppMessageFeedback{}, err
	}
	return cloneAppMessageFeedback(feedback), nil
}

func cloneAppMCPServer(src AppMCPServer) AppMCPServer {
	return AppMCPServer{
		ID:          src.ID,
		ServerCode:  src.ServerCode,
		Description: src.Description,
		Status:      src.Status,
		Parameters:  cloneStringMap(src.Parameters),
		Headers:     cloneStringMap(src.Headers),
		CreatedAt:   src.CreatedAt,
		UpdatedAt:   src.UpdatedAt,
	}
}

func cloneAppAnnotation(src AppAnnotation) AppAnnotation {
	return AppAnnotation{
		ID:        src.ID,
		MessageID: src.MessageID,
		Question:  src.Question,
		Answer:    src.Answer,
		AccountID: src.AccountID,
		CreatedAt: src.CreatedAt,
		UpdatedAt: src.UpdatedAt,
	}
}

func cloneAppAnnotationList(src []AppAnnotation) []AppAnnotation {
	if src == nil {
		return []AppAnnotation{}
	}
	out := make([]AppAnnotation, len(src))
	for i, item := range src {
		out[i] = cloneAppAnnotation(item)
	}
	return out
}

func cloneAppMessageFeedback(src AppMessageFeedback) AppMessageFeedback {
	return AppMessageFeedback{
		MessageID:     src.MessageID,
		Rating:        src.Rating,
		Content:       src.Content,
		FromSource:    src.FromSource,
		FromEndUserID: src.FromEndUserID,
		UpdatedAt:     src.UpdatedAt,
	}
}

func cloneAppMessageFeedbackList(src []AppMessageFeedback) []AppMessageFeedback {
	if src == nil {
		return []AppMessageFeedback{}
	}
	out := make([]AppMessageFeedback, len(src))
	for i, item := range src {
		out[i] = cloneAppMessageFeedback(item)
	}
	return out
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func normalizeMCPServerStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "active":
		return "active"
	default:
		return "inactive"
	}
}
