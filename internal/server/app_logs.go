package server

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

type appConversationMessage struct {
	ID              string
	ConversationID  string
	Query           string
	Inputs          map[string]any
	Answer          string
	Status          string
	CreatedAt       int64
	UpdatedAt       int64
	WorkflowRunID   string
	ParentMessageID string
}

type appConversation struct {
	ID           string
	CreatedBy    string
	CreatedAt    int64
	UpdatedAt    int64
	LatestStatus string
	Messages     []appConversationMessage
	StatusCount  map[string]int
}

func (s *server) handleAnnotationsCount(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"count": len(app.Annotations),
	})
}

func (s *server) handleChatConversations(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	conversations := s.filteredAppConversations(app, r)
	sortField, desc := sortFieldAndDirection(strings.TrimSpace(r.URL.Query().Get("sort_by")))
	slices.SortFunc(conversations, func(a, b appConversation) int {
		left := a.CreatedAt
		right := b.CreatedAt
		if sortField == "updated_at" {
			left = a.UpdatedAt
			right = b.UpdatedAt
		}
		if left == right {
			return strings.Compare(a.ID, b.ID)
		}
		if desc {
			if left > right {
				return -1
			}
			return 1
		}
		if left < right {
			return -1
		}
		return 1
	})

	page, limit, pageItems := paginateConversations(conversations, intQuery(r, "page", 1), intQuery(r, "limit", 20))
	data := make([]map[string]any, 0, len(pageItems))
	for _, conversation := range pageItems {
		data = append(data, s.chatConversationSummaryResponse(app, conversation))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": page*limit < len(conversations),
		"limit":    limit,
		"total":    len(conversations),
		"page":     page,
	})
}

func (s *server) handleChatConversationDetail(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	conversation, found := s.findAppConversation(app, chi.URLParam(r, "conversationID"))
	if !found {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.chatConversationDetailResponse(app, conversation))
}

func (s *server) handleCompletionConversations(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	conversations := s.filteredAppConversations(app, r)
	slices.SortFunc(conversations, func(a, b appConversation) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})

	page, limit, pageItems := paginateConversations(conversations, intQuery(r, "page", 1), intQuery(r, "limit", 20))
	data := make([]map[string]any, 0, len(pageItems))
	for _, conversation := range pageItems {
		data = append(data, s.completionConversationSummaryResponse(app, conversation))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": page*limit < len(conversations),
		"limit":    limit,
		"total":    len(conversations),
		"page":     page,
	})
}

func (s *server) handleCompletionConversationDetail(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	conversation, found := s.findAppConversation(app, chi.URLParam(r, "conversationID"))
	if !found {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.completionConversationDetailResponse(app, conversation))
}

func (s *server) handleWorkflowAppLogs(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	after, hasAfter := parseConsoleTime(r.URL.Query().Get("created_at__after"))
	before, hasBefore := parseConsoleTime(r.URL.Query().Get("created_at__before"))
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	filtered := make([]state.WorkflowRun, 0, len(app.WorkflowRuns))
	for _, run := range app.WorkflowRuns {
		if status != "" && run.Status != status {
			continue
		}
		if hasAfter && run.CreatedAt < after {
			continue
		}
		if hasBefore && run.CreatedAt > before {
			continue
		}
		if keyword != "" && !workflowRunMatchesKeyword(run, keyword) {
			continue
		}
		filtered = append(filtered, run)
	}

	slices.SortFunc(filtered, func(a, b state.WorkflowRun) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt > b.CreatedAt {
			return -1
		}
		return 1
	})

	page := intQuery(r, "page", 1)
	limit := intQuery(r, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	data := make([]map[string]any, 0, end-start)
	for _, run := range filtered[start:end] {
		data = append(data, s.workflowAppLogResponse(app, run))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": end < len(filtered),
		"limit":    limit,
		"total":    len(filtered),
		"page":     page,
	})
}

func (s *server) handleWorkflowPauseDetails(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	app, run, found := s.store.FindWorkflowRun(workspace.ID, chi.URLParam(r, "workflowRunID"))
	if !found {
		writeError(w, http.StatusNotFound, "workflow_run_not_found", "Workflow run does not exist.")
		return
	}

	writeJSON(w, http.StatusOK, s.workflowPauseDetailsResponse(app, run))
}

func (s *server) handleMCPServerGet(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.mcpServerResponse(app.MCPServer))
}

func (s *server) handleMCPServerCreate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		Description string            `json:"description"`
		Parameters  map[string]string `json:"parameters"`
		Headers     map[string]string `json:"headers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	server, err := s.store.UpsertAppMCPServer(app.ID, app.WorkspaceID, currentUser(r), state.UpsertAppMCPServerInput{
		Description: payload.Description,
		Status:      "inactive",
		Parameters:  payload.Parameters,
		Headers:     payload.Headers,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, s.mcpServerResponse(&server))
}

func (s *server) handleMCPServerUpdate(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		ID          string            `json:"id"`
		Description string            `json:"description"`
		Status      string            `json:"status"`
		Parameters  map[string]string `json:"parameters"`
		Headers     map[string]string `json:"headers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	server, err := s.store.UpsertAppMCPServer(app.ID, app.WorkspaceID, currentUser(r), state.UpsertAppMCPServerInput{
		ID:          payload.ID,
		Description: payload.Description,
		Status:      payload.Status,
		Parameters:  payload.Parameters,
		Headers:     payload.Headers,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.mcpServerResponse(&server))
}

func (s *server) handleMCPServerRefresh(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	appID := chi.URLParam(r, "appID")
	app, found := s.store.GetApp(appID, workspace.ID)
	if !found {
		app, found = s.store.FindAppByMCPServer(workspace.ID, appID)
		if !found {
			writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
			return
		}
	}

	server, err := s.store.RefreshAppMCPServer(app.ID, app.WorkspaceID, user, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, s.mcpServerResponse(&server))
}

func (s *server) handleAppChatMessages(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "conversation_id is required.")
		return
	}

	conversation, found := s.findAppConversation(app, conversationID)
	if !found {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found.")
		return
	}

	messages := slices.Clone(conversation.Messages)
	slices.SortFunc(messages, func(a, b appConversationMessage) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt > b.CreatedAt {
			return -1
		}
		return 1
	})

	firstID := strings.TrimSpace(r.URL.Query().Get("first_id"))
	start := 0
	if firstID != "" {
		index := slices.IndexFunc(messages, func(item appConversationMessage) bool {
			return item.ID == firstID
		})
		if index >= 0 {
			start = index + 1
		}
	}

	limit := intQuery(r, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if start > len(messages) {
		start = len(messages)
	}
	end := start + limit
	if end > len(messages) {
		end = len(messages)
	}

	data := make([]map[string]any, 0, end-start)
	for _, message := range messages[start:end] {
		data = append(data, s.chatMessageResponse(app, message))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": end < len(messages),
		"limit":    limit,
	})
}

func (s *server) handleAppFeedbacks(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		MessageID string  `json:"message_id"`
		Rating    *string `json:"rating"`
		Content   string  `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	rating := ""
	if payload.Rating != nil {
		rating = strings.TrimSpace(*payload.Rating)
	}
	if rating != "" && rating != "like" && rating != "dislike" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Unsupported rating.")
		return
	}

	if _, err := s.store.SaveAppMessageFeedback(app.ID, app.WorkspaceID, currentUser(r), state.SaveAppMessageFeedbackInput{
		MessageID:  payload.MessageID,
		Rating:     rating,
		Content:    payload.Content,
		FromSource: "admin",
	}, time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleAppAnnotationsUpsert(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	var payload struct {
		MessageID string `json:"message_id"`
		Content   string `json:"content"`
		Question  string `json:"question"`
		Answer    string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	question := strings.TrimSpace(payload.Question)
	answer := strings.TrimSpace(payload.Answer)
	if strings.TrimSpace(payload.MessageID) != "" {
		if message, found := s.findConversationMessage(app, payload.MessageID); found {
			question = firstImportValue(question, message.Query)
		}
		answer = firstImportValue(strings.TrimSpace(payload.Content), answer)
	}

	annotation, err := s.store.SaveAppAnnotation(app.ID, app.WorkspaceID, currentUser(r), state.SaveAppAnnotationInput{
		MessageID: payload.MessageID,
		Question:  question,
		Answer:    answer,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	author := s.workflowActor(annotation.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         annotation.ID,
		"question":   annotation.Question,
		"answer":     annotation.Answer,
		"created_at": annotation.CreatedAt,
		"account": map[string]any{
			"name": author["name"],
		},
		"result": "success",
	})
}

func (s *server) handleAppAnnotationDelete(w http.ResponseWriter, r *http.Request) {
	app, ok := s.currentUserApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	if err := s.store.DeleteAppAnnotation(app.ID, app.WorkspaceID, chi.URLParam(r, "annotationID"), currentUser(r), time.Now()); err != nil {
		if err.Error() == "annotation not found" {
			writeError(w, http.StatusNotFound, "annotation_not_found", "Annotation not found.")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) filteredAppConversations(app state.App, r *http.Request) []appConversation {
	conversations := s.buildAppConversations(app)
	keyword := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("keyword")))
	annotationStatus := strings.TrimSpace(r.URL.Query().Get("annotation_status"))
	start, hasStart := parseConsoleTime(r.URL.Query().Get("start"))
	end, hasEnd := parseConsoleTime(r.URL.Query().Get("end"))

	filtered := make([]appConversation, 0, len(conversations))
	for _, conversation := range conversations {
		if keyword != "" && !conversationMatchesKeyword(conversation, keyword) {
			continue
		}

		hasAnnotation := s.conversationHasAnnotation(app, conversation)
		if annotationStatus == "annotated" && !hasAnnotation {
			continue
		}
		if annotationStatus == "not_annotated" && hasAnnotation {
			continue
		}
		if hasStart && conversation.UpdatedAt < start {
			continue
		}
		if hasEnd && conversation.UpdatedAt > end {
			continue
		}

		filtered = append(filtered, conversation)
	}
	return filtered
}

func (s *server) buildAppConversations(app state.App) []appConversation {
	grouped := make(map[string]*appConversation)

	for _, run := range app.WorkflowRuns {
		conversationID := strings.TrimSpace(run.ConversationID)
		if conversationID == "" {
			conversationID = firstImportValue(strings.TrimSpace(run.MessageID), run.ID)
		}
		if conversationID == "" {
			continue
		}

		messageID := firstImportValue(strings.TrimSpace(run.MessageID), run.ID)
		conversation, ok := grouped[conversationID]
		if !ok {
			conversation = &appConversation{
				ID:          conversationID,
				CreatedBy:   run.CreatedBy,
				Messages:    []appConversationMessage{},
				StatusCount: map[string]int{},
			}
			grouped[conversationID] = conversation
		}

		conversation.Messages = append(conversation.Messages, appConversationMessage{
			ID:             messageID,
			ConversationID: conversationID,
			Query:          appLogQuery(run.Inputs),
			Inputs:         cloneJSONObject(run.Inputs),
			Answer:         appLogAnswer(run.Outputs),
			Status:         run.Status,
			CreatedAt:      run.CreatedAt,
			UpdatedAt:      maxInt64(run.FinishedAt, run.CreatedAt),
			WorkflowRunID:  run.ID,
		})
	}

	conversations := make([]appConversation, 0, len(grouped))
	for _, conversation := range grouped {
		slices.SortFunc(conversation.Messages, func(a, b appConversationMessage) int {
			if a.CreatedAt == b.CreatedAt {
				return strings.Compare(a.ID, b.ID)
			}
			if a.CreatedAt < b.CreatedAt {
				return -1
			}
			return 1
		})

		for index := range conversation.Messages {
			if index > 0 {
				conversation.Messages[index].ParentMessageID = conversation.Messages[index-1].ID
			}
			if conversation.CreatedAt == 0 || conversation.Messages[index].CreatedAt < conversation.CreatedAt {
				conversation.CreatedAt = conversation.Messages[index].CreatedAt
			}
			if conversation.Messages[index].UpdatedAt > conversation.UpdatedAt {
				conversation.UpdatedAt = conversation.Messages[index].UpdatedAt
			}
			conversation.LatestStatus = conversation.Messages[index].Status
			incrementStatusCount(conversation.StatusCount, conversation.Messages[index].Status)
		}
		conversations = append(conversations, *conversation)
	}

	return conversations
}

func (s *server) findAppConversation(app state.App, conversationID string) (appConversation, bool) {
	for _, conversation := range s.buildAppConversations(app) {
		if conversation.ID == conversationID {
			return conversation, true
		}
	}
	return appConversation{}, false
}

func (s *server) findConversationMessage(app state.App, messageID string) (appConversationMessage, bool) {
	for _, conversation := range s.buildAppConversations(app) {
		for _, message := range conversation.Messages {
			if message.ID == messageID {
				return message, true
			}
		}
	}
	return appConversationMessage{}, false
}

func (s *server) conversationHasAnnotation(app state.App, conversation appConversation) bool {
	annotations := annotationsByMessage(app.Annotations)
	for _, message := range conversation.Messages {
		if _, ok := annotations[message.ID]; ok {
			return true
		}
	}
	return false
}

func (s *server) chatConversationSummaryResponse(app state.App, conversation appConversation) map[string]any {
	latest := latestConversationMessage(conversation)
	userStats, adminStats := feedbackStats(feedbacksByMessage(app.MessageFeedbacks), conversation)
	author := s.workflowActor(conversation.CreatedBy)

	return map[string]any{
		"id":                       conversation.ID,
		"status":                   normalizeConversationStatus(conversation.LatestStatus),
		"from_source":              "console",
		"from_end_user_id":         "",
		"from_end_user_session_id": "",
		"from_account_id":          conversation.CreatedBy,
		"from_account_name":        author["name"],
		"read_at":                  conversation.UpdatedAt,
		"created_at":               conversation.CreatedAt,
		"updated_at":               conversation.UpdatedAt,
		"name":                     firstImportValue(latest.Answer, latest.Query),
		"summary":                  firstImportValue(latest.Answer, latest.Query),
		"message_count":            len(conversation.Messages),
		"annotated":                s.conversationHasAnnotation(app, conversation),
		"user_feedback_stats":      userStats,
		"admin_feedback_stats":     adminStats,
		"status_count": map[string]any{
			"paused":          conversation.StatusCount["paused"],
			"success":         conversation.StatusCount["succeeded"],
			"failed":          conversation.StatusCount["failed"] + conversation.StatusCount["stopped"],
			"partial_success": conversation.StatusCount["partial-succeeded"],
		},
		"model_config": s.appLogModelConfig(app),
	}
}

func (s *server) chatConversationDetailResponse(app state.App, conversation appConversation) map[string]any {
	userStats, adminStats := feedbackStats(feedbacksByMessage(app.MessageFeedbacks), conversation)
	author := s.workflowActor(conversation.CreatedBy)

	return map[string]any{
		"id":                       conversation.ID,
		"status":                   normalizeConversationStatus(conversation.LatestStatus),
		"from_source":              "console",
		"from_end_user_id":         "",
		"from_end_user_session_id": "",
		"from_account_id":          conversation.CreatedBy,
		"from_account_name":        author["name"],
		"read_at":                  conversation.UpdatedAt,
		"created_at":               conversation.CreatedAt,
		"updated_at":               conversation.UpdatedAt,
		"message_count":            len(conversation.Messages),
		"user_feedback_stats":      userStats,
		"admin_feedback_stats":     adminStats,
		"model_config":             s.appLogModelConfig(app),
	}
}

func (s *server) completionConversationSummaryResponse(app state.App, conversation appConversation) map[string]any {
	latest := latestConversationMessage(conversation)
	userStats, adminStats := feedbackStats(feedbacksByMessage(app.MessageFeedbacks), conversation)
	author := s.workflowActor(conversation.CreatedBy)

	return map[string]any{
		"id":                       conversation.ID,
		"status":                   normalizeConversationStatus(conversation.LatestStatus),
		"from_source":              "console",
		"from_end_user_id":         "",
		"from_end_user_session_id": "",
		"from_account_id":          conversation.CreatedBy,
		"from_account_name":        author["name"],
		"read_at":                  conversation.UpdatedAt,
		"created_at":               conversation.CreatedAt,
		"updated_at":               conversation.UpdatedAt,
		"annotation":               s.annotationLogResponse(app, latest.ID),
		"user_feedback_stats":      userStats,
		"admin_feedback_stats":     adminStats,
		"model_config": map[string]any{
			"provider": s.appLogModelProvider(app),
			"model_id": s.appLogModelName(app),
			"configs":  map[string]any{"prompt_template": modelPromptTemplate(app.ModelConfig)},
		},
		"message": s.completionMessageResponse(app, latest),
	}
}

func (s *server) completionConversationDetailResponse(app state.App, conversation appConversation) map[string]any {
	latest := latestConversationMessage(conversation)
	author := s.workflowActor(conversation.CreatedBy)

	return map[string]any{
		"id":                       conversation.ID,
		"status":                   normalizeConversationStatus(conversation.LatestStatus),
		"from_source":              "console",
		"from_end_user_id":         "",
		"from_end_user_session_id": "",
		"from_account_id":          conversation.CreatedBy,
		"from_account_name":        author["name"],
		"created_at":               conversation.CreatedAt,
		"updated_at":               conversation.UpdatedAt,
		"model_config":             s.appLogModelConfig(app),
		"message":                  s.completionMessageResponse(app, latest),
	}
}

func (s *server) completionMessageResponse(app state.App, message appConversationMessage) map[string]any {
	return map[string]any{
		"id":              message.ID,
		"conversation_id": message.ConversationID,
		"query":           message.Query,
		"inputs":          message.Inputs,
		"message": []map[string]any{
			{"role": "user", "text": message.Query},
			{"role": "assistant", "text": message.Answer},
		},
		"message_tokens":            appLogMessageTokens(message.Query),
		"answer_tokens":             appLogMessageTokens(message.Answer),
		"answer":                    message.Answer,
		"provider_response_latency": appLogLatency(message),
		"created_at":                message.CreatedAt,
		"annotation":                s.annotationLogResponse(app, message.ID),
		"annotation_hit_history":    nil,
		"feedbacks":                 s.messageFeedbackResponse(app, message.ID),
		"message_files":             []any{},
		"metadata":                  map[string]any{},
		"agent_thoughts":            []any{},
		"workflow_run_id":           message.WorkflowRunID,
		"parent_message_id":         nullIfEmpty(message.ParentMessageID),
	}
}

func (s *server) chatMessageResponse(app state.App, message appConversationMessage) map[string]any {
	return map[string]any{
		"id":              message.ID,
		"conversation_id": message.ConversationID,
		"query":           message.Query,
		"inputs":          message.Inputs,
		"message": []map[string]any{
			{"role": "user", "text": message.Query},
			{"role": "assistant", "text": message.Answer},
		},
		"message_tokens":            appLogMessageTokens(message.Query),
		"answer_tokens":             appLogMessageTokens(message.Answer),
		"answer":                    message.Answer,
		"provider_response_latency": appLogLatency(message),
		"created_at":                message.CreatedAt,
		"annotation":                s.annotationLogResponse(app, message.ID),
		"annotation_hit_history":    nil,
		"feedbacks":                 s.messageFeedbackResponse(app, message.ID),
		"message_files":             []any{},
		"metadata":                  map[string]any{},
		"agent_thoughts":            []any{},
		"workflow_run_id":           message.WorkflowRunID,
		"parent_message_id":         nullIfEmpty(message.ParentMessageID),
	}
}

func (s *server) workflowAppLogResponse(app state.App, run state.WorkflowRun) map[string]any {
	response := map[string]any{
		"id": run.ID,
		"workflow_run": map[string]any{
			"id":             run.ID,
			"version":        run.Version,
			"status":         run.Status,
			"error":          nullIfEmpty(run.Error),
			"triggered_from": s.workflowTriggeredFrom(app, run),
			"elapsed_time":   run.ElapsedTime,
			"total_tokens":   run.TotalTokens,
			"total_price":    run.TotalPrice,
			"currency":       firstImportValue(run.Currency, "USD"),
			"total_steps":    run.TotalSteps,
			"finished_at":    run.FinishedAt,
		},
		"created_from":       workflowCreatedFrom(run),
		"created_by_role":    "account",
		"created_by_account": s.workflowActor(run.CreatedBy),
		"created_at":         run.CreatedAt,
		"read_at":            run.CreatedAt,
	}

	if metadata := s.workflowTriggerMetadata(app, run); len(metadata) > 0 {
		response["details"] = map[string]any{
			"trigger_metadata": metadata,
		}
	}
	return response
}

func (s *server) workflowPauseDetailsResponse(app state.App, run state.WorkflowRun) map[string]any {
	pausedAt := time.Unix(maxInt64(run.FinishedAt, run.CreatedAt), 0).UTC().Format(time.RFC3339)
	pausedNodes := make([]map[string]any, 0)
	for _, execution := range run.NodeExecutions {
		if execution.Status != "paused" && execution.NodeType != "human-input" {
			continue
		}

		pauseType := map[string]any{
			"type": "breakpoint",
		}
		if execution.NodeType == "human-input" {
			pauseType = map[string]any{
				"type":                "human_input",
				"form_id":             execution.NodeID,
				"backstage_input_url": firstImportValue(strings.TrimSpace(app.Site.AppBaseURL), "http://localhost:3000") + "/app/" + app.ID + "/workflow-runs/" + run.ID + "/human-input/" + execution.NodeID,
			}
		}

		pausedNodes = append(pausedNodes, map[string]any{
			"node_id":    execution.NodeID,
			"node_title": execution.Title,
			"pause_type": pauseType,
		})
	}

	if len(pausedNodes) == 0 && run.Status == "paused" {
		title := "Paused Node"
		nodeID := "paused-node"
		if len(run.NodeExecutions) > 0 {
			lastNode := run.NodeExecutions[len(run.NodeExecutions)-1]
			title = firstImportValue(lastNode.Title, title)
			nodeID = firstImportValue(lastNode.NodeID, nodeID)
		}
		pausedNodes = append(pausedNodes, map[string]any{
			"node_id":    nodeID,
			"node_title": title,
			"pause_type": map[string]any{
				"type": "breakpoint",
			},
		})
	}

	return map[string]any{
		"paused_at":    pausedAt,
		"paused_nodes": pausedNodes,
	}
}

func (s *server) mcpServerResponse(server *state.AppMCPServer) map[string]any {
	if server == nil {
		return map[string]any{
			"id":          "",
			"server_code": "",
			"description": "",
			"status":      "inactive",
			"parameters":  map[string]string{},
			"headers":     map[string]string{},
		}
	}

	return map[string]any{
		"id":          server.ID,
		"server_code": server.ServerCode,
		"description": server.Description,
		"status":      server.Status,
		"parameters":  server.Parameters,
		"headers":     server.Headers,
	}
}

func (s *server) annotationLogResponse(app state.App, messageID string) any {
	annotation, ok := annotationsByMessage(app.Annotations)[messageID]
	if !ok {
		return nil
	}

	return map[string]any{
		"id":         annotation.ID,
		"content":    annotation.Answer,
		"question":   annotation.Question,
		"answer":     annotation.Answer,
		"account":    s.workflowActor(annotation.AccountID),
		"created_at": annotation.CreatedAt,
	}
}

func (s *server) messageFeedbackResponse(app state.App, messageID string) []map[string]any {
	items := feedbacksByMessage(app.MessageFeedbacks)[messageID]
	if len(items) == 0 {
		return []map[string]any{}
	}

	response := make([]map[string]any, 0, len(items))
	for _, item := range items {
		rating := any(nil)
		if strings.TrimSpace(item.Rating) != "" {
			rating = item.Rating
		}
		response = append(response, map[string]any{
			"rating":           rating,
			"content":          nullIfEmpty(item.Content),
			"from_source":      item.FromSource,
			"from_end_user_id": nullIfEmpty(item.FromEndUserID),
		})
	}
	return response
}

func (s *server) appLogModelConfig(app state.App) map[string]any {
	completionParams := modelCompletionParams(app.ModelConfig)
	provider := s.appLogModelProvider(app)
	modelName := s.appLogModelName(app)

	return map[string]any{
		"provider": provider,
		"model_id": modelName,
		"configs": map[string]any{
			"introduction":      stringFromAny(app.ModelConfig["opening_statement"]),
			"prompt_template":   modelPromptTemplate(app.ModelConfig),
			"prompt_variables":  appLogPromptVariables(app.ModelConfig),
			"completion_params": completionParams,
		},
		"model": map[string]any{
			"name":              modelName,
			"provider":          provider,
			"completion_params": completionParams,
		},
		"user_input_form": app.ModelConfig["user_input_form"],
	}
}

func (s *server) appLogModelProvider(app state.App) string {
	model := mapFromAny(app.ModelConfig["model"])
	return firstImportValue(stringFromAny(model["provider"]), "langgenius/openai/openai")
}

func (s *server) appLogModelName(app state.App) string {
	model := mapFromAny(app.ModelConfig["model"])
	return firstImportValue(stringFromAny(model["name"]), "gpt-4o-mini")
}

func workflowRunMatchesKeyword(run state.WorkflowRun, keyword string) bool {
	values := []string{
		run.ID,
		run.TaskID,
		run.Version,
		run.Status,
		run.Error,
		appLogQuery(run.Inputs),
		appLogAnswer(run.Outputs),
		stringifyWorkflowValue(run.Inputs),
		stringifyWorkflowValue(run.Outputs),
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func sortFieldAndDirection(sortBy string) (string, bool) {
	field := strings.TrimPrefix(sortBy, "-")
	if field == "" {
		field = "created_at"
	}
	return field, strings.HasPrefix(sortBy, "-")
}

func paginateConversations(items []appConversation, page, limit int) (int, int, []appConversation) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	start := (page - 1) * limit
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return page, limit, items[start:end]
}

func parseConsoleTime(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed.Unix(), true
		}
	}
	if unixValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		return unixValue, true
	}
	return 0, false
}

func feedbacksByMessage(items []state.AppMessageFeedback) map[string][]state.AppMessageFeedback {
	grouped := make(map[string][]state.AppMessageFeedback)
	for _, item := range items {
		grouped[item.MessageID] = append(grouped[item.MessageID], item)
	}
	return grouped
}

func annotationsByMessage(items []state.AppAnnotation) map[string]state.AppAnnotation {
	grouped := make(map[string]state.AppAnnotation)
	for _, item := range items {
		if strings.TrimSpace(item.MessageID) == "" {
			continue
		}
		grouped[item.MessageID] = item
	}
	return grouped
}

func feedbackStats(grouped map[string][]state.AppMessageFeedback, conversation appConversation) (map[string]any, map[string]any) {
	userLike := 0
	userDislike := 0
	adminLike := 0
	adminDislike := 0

	for _, message := range conversation.Messages {
		for _, feedback := range grouped[message.ID] {
			switch feedback.FromSource {
			case "user":
				if feedback.Rating == "like" {
					userLike++
				}
				if feedback.Rating == "dislike" {
					userDislike++
				}
			default:
				if feedback.Rating == "like" {
					adminLike++
				}
				if feedback.Rating == "dislike" {
					adminDislike++
				}
			}
		}
	}

	return map[string]any{
			"like":    userLike,
			"dislike": userDislike,
		}, map[string]any{
			"like":    adminLike,
			"dislike": adminDislike,
		}
}

func latestConversationMessage(conversation appConversation) appConversationMessage {
	if len(conversation.Messages) == 0 {
		return appConversationMessage{}
	}
	return conversation.Messages[len(conversation.Messages)-1]
}

func conversationMatchesKeyword(conversation appConversation, keyword string) bool {
	if strings.Contains(strings.ToLower(conversation.ID), keyword) {
		return true
	}
	for _, message := range conversation.Messages {
		if strings.Contains(strings.ToLower(message.Query), keyword) || strings.Contains(strings.ToLower(message.Answer), keyword) {
			return true
		}
	}
	return false
}

func incrementStatusCount(statusCount map[string]int, status string) {
	if statusCount == nil {
		return
	}
	statusCount[status]++
}

func normalizeConversationStatus(status string) string {
	switch status {
	case "paused":
		return "paused"
	case "running":
		return "normal"
	default:
		return "finished"
	}
}

func workflowCreatedFrom(run state.WorkflowRun) string {
	if strings.TrimSpace(run.ConversationID) != "" {
		return "web-app"
	}
	return "explore"
}

func (s *server) workflowTriggeredFrom(app state.App, run state.WorkflowRun) string {
	if app.Mode == "workflow" && strings.TrimSpace(run.ConversationID) == "" {
		return "debugging"
	}
	if strings.TrimSpace(run.ConversationID) != "" || strings.TrimSpace(run.MessageID) != "" {
		return "app-run"
	}
	return "debugging"
}

func (s *server) workflowTriggerMetadata(app state.App, run state.WorkflowRun) map[string]any {
	if s.workflowTriggeredFrom(app, run) != "app-run" {
		return map[string]any{}
	}
	return map[string]any{
		"type": "web-app",
	}
}

func appLogQuery(inputs map[string]any) string {
	return firstImportValue(
		stringFromAny(inputs["query"]),
		stringFromAny(inputs["default_input"]),
	)
}

func appLogAnswer(outputs map[string]any) string {
	answer := firstImportValue(
		stringFromAny(outputs["answer"]),
		stringFromAny(outputs["text"]),
	)
	if answer != "" {
		return answer
	}
	if len(outputs) == 0 {
		return ""
	}
	return stringifyWorkflowValue(outputs)
}

func appLogMessageTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	return len([]rune(value))
}

func appLogLatency(message appConversationMessage) float64 {
	if message.UpdatedAt <= message.CreatedAt {
		return 0
	}
	return float64(message.UpdatedAt - message.CreatedAt)
}

func modelPromptTemplate(modelConfig map[string]any) string {
	if prompt := strings.TrimSpace(stringFromAny(modelConfig["pre_prompt"])); prompt != "" {
		return prompt
	}
	completionPrompt := mapFromAny(modelConfig["completion_prompt_config"])
	if prompt := strings.TrimSpace(stringFromAny(mapFromAny(completionPrompt["prompt"])["text"])); prompt != "" {
		return prompt
	}
	chatPrompt := mapFromAny(modelConfig["chat_prompt_config"])
	promptItems, ok := chatPrompt["prompt"].([]any)
	if !ok {
		return ""
	}
	lines := make([]string, 0, len(promptItems))
	for _, item := range promptItems {
		line := strings.TrimSpace(stringFromAny(mapFromAny(item)["text"]))
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func modelCompletionParams(modelConfig map[string]any) map[string]any {
	model := mapFromAny(modelConfig["model"])
	params := cloneJSONObject(mapFromAny(model["completion_params"]))
	if len(params) > 0 {
		return params
	}
	return map[string]any{
		"max_tokens":        512,
		"temperature":       0.7,
		"top_p":             1.0,
		"stop":              []any{},
		"presence_penalty":  0.0,
		"frequency_penalty": 0.0,
	}
}

func appLogPromptVariables(modelConfig map[string]any) []map[string]any {
	rawItems, ok := modelConfig["user_input_form"].([]any)
	if !ok {
		return []map[string]any{}
	}

	variables := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item := mapFromAny(rawItem)
		if len(item) == 0 {
			continue
		}
		for inputType, rawConfig := range item {
			config := mapFromAny(rawConfig)
			key := firstImportValue(stringFromAny(config["variable"]), stringFromAny(config["name"]))
			if key == "" {
				continue
			}
			variables = append(variables, map[string]any{
				"key":         key,
				"name":        firstImportValue(stringFromAny(config["label"]), key),
				"description": stringFromAny(config["description"]),
				"type":        firstImportValue(inputType, "text-input"),
				"default":     stringFromAny(config["default"]),
				"options":     config["options"],
			})
			break
		}
	}
	return variables
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
