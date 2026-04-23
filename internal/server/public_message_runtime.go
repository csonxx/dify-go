package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/langgenius/dify-go/internal/state"
)

const publicRuntimeSessionCookie = "dify_go_public_session"

var publicAudioMP3Base64 = "SUQzBAAAAAAAI1RTU0UAAAAPAAADTGF2ZjYyLjEyLjEwMAAAAAAAAAAAAAAA//NYwAAAAAAAAAAAAEluZm8AAAAPAAAAEAAAAvQASUlJSUlJVVVVVVVVYWFhYWFhbW1tbW1teXl5eXl5eYaGhoaGhpKSkpKSkp6enp6enqqqqqqqqqq2tra2trbDw8PDw8PPz8/Pz8/b29vb29vb5+fn5+fn8/Pz8/Pz////////AAAAAExhdmM2Mi4yOAAAAAAAAAAAAAAAACQCgAAAAAAAAAL0CuwmeQAAAAAAAAAAAAAA//MYxAAAAANIAAAAAExBTUUzLjEwMFVVVVVVVVVVVVVVVUxB//MYxBcAAANIAAAAAE1FMy4xMDBVVVVVVVVVVVVVVVVVVUxB//MYxC4AAANIAAAAAE1FMy4xMDBVVVVVVVVVVVVVVVVVVUxB//MYxEUAAANIAAAAAE1FMy4xMDBVVVVVVVVVVVVVVVVVVUxB//MYxFwAAANIAAAAAE1FMy4xMDBVVVVVVVVVVVVVVVVVVUxB//MYxHMAAANIAAAAAE1FMy4xMDBVVVVVVVVVVVVVVVVVVVVV//MYxIoAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxKEAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxLgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxM8AAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOYAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV//MYxOgAAANIAAAAAFVVVVVVVVVVVVVVVVVVVVVVVVVVVVVV"

func (s *server) handlePublicChatMessages(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}
	if !isPublicChatApp(app) {
		writeError(w, http.StatusBadRequest, "invalid_request", "Chat is not enabled for this app.")
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	conversation, message, err := s.persistPublicGeneratedMessage(app, actorID, payload, "chat")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if strings.ToLower(strings.TrimSpace(stringFromAny(payload["response_mode"]))) != "streaming" {
		writeJSON(w, http.StatusOK, s.publicMessageResponse(app, message))
		return
	}

	if err := s.streamWorkflowEvents(w, r, s.publicMessageEvents(app, conversation, message)); err != nil {
		return
	}
}

func (s *server) handlePublicCompletionMessages(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(app.Mode) != "completion" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Completion is not enabled for this app.")
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	conversation, message, err := s.persistPublicGeneratedMessage(app, actorID, payload, "completion")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if strings.ToLower(strings.TrimSpace(stringFromAny(payload["response_mode"]))) != "streaming" {
		writeJSON(w, http.StatusOK, s.publicMessageResponse(app, message))
		return
	}

	if err := s.streamWorkflowEvents(w, r, s.publicMessageEvents(app, conversation, message)); err != nil {
		return
	}
}

func (s *server) handlePublicChatMessageStop(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	if _, _, err := s.store.StopAppPublicMessageTask(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "taskID"), s.publicWorkflowActor(app), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handlePublicMessages(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	conversationID := strings.TrimSpace(r.URL.Query().Get("conversation_id"))
	if conversationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "conversation_id is required.")
		return
	}

	messages, ok := s.store.ListAppPublicMessages(app.ID, app.WorkspaceID, actorID, conversationID)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	slices.SortFunc(messages, func(a, b state.AppPublicMessage) int {
		if a.CreatedAt == b.CreatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.CreatedAt < b.CreatedAt {
			return -1
		}
		return 1
	})

	lastID := strings.TrimSpace(r.URL.Query().Get("last_id"))
	if lastID != "" {
		cursor := slices.IndexFunc(messages, func(item state.AppPublicMessage) bool {
			return item.ID == lastID
		})
		if cursor >= 0 && cursor+1 < len(messages) {
			messages = messages[cursor+1:]
		}
	}

	limit := intQuery(r, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	data := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		data = append(data, s.publicMessageResponse(app, message))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": hasMore,
		"limit":    limit,
	})
}

func (s *server) handlePublicConversations(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	conversations, ok := s.store.ListAppPublicConversations(app.ID, app.WorkspaceID, actorID)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	pinnedFilter := strings.TrimSpace(r.URL.Query().Get("pinned"))
	filtered := make([]state.AppPublicConversation, 0, len(conversations))
	for _, conversation := range conversations {
		if pinnedFilter == "true" && !conversation.Pinned {
			continue
		}
		if pinnedFilter == "false" && conversation.Pinned {
			continue
		}
		filtered = append(filtered, conversation)
	}

	slices.SortFunc(filtered, func(a, b state.AppPublicConversation) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})

	lastID := strings.TrimSpace(r.URL.Query().Get("last_id"))
	if lastID != "" {
		cursor := slices.IndexFunc(filtered, func(item state.AppPublicConversation) bool {
			return item.ID == lastID
		})
		if cursor >= 0 && cursor+1 < len(filtered) {
			filtered = filtered[cursor+1:]
		}
	}

	limit := intQuery(r, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	hasMore := len(filtered) > limit
	if hasMore {
		filtered = filtered[:limit]
	}

	data := make([]map[string]any, 0, len(filtered))
	for _, conversation := range filtered {
		data = append(data, s.publicConversationResponse(conversation))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":     data,
		"has_more": hasMore,
		"limit":    limit,
	})
}

func (s *server) handlePublicConversationPin(w http.ResponseWriter, r *http.Request) {
	s.handlePublicConversationMutation(w, r, boolPtr(true), nil, false)
}

func (s *server) handlePublicConversationUnpin(w http.ResponseWriter, r *http.Request) {
	s.handlePublicConversationMutation(w, r, boolPtr(false), nil, false)
}

func (s *server) handlePublicConversationDelete(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	if err := s.store.DeleteAppPublicConversation(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "conversationID"), s.publicWorkflowActor(app), time.Now()); err != nil {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handlePublicConversationName(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	var name *string
	if raw := strings.TrimSpace(stringFromAny(payload["name"])); raw != "" {
		name = &raw
	}
	autoGenerate := false
	if flag, ok := payload["auto_generate"].(bool); ok {
		autoGenerate = flag
	}
	s.handlePublicConversationMutation(w, r, nil, name, autoGenerate)
}

func (s *server) handlePublicConversationMutation(w http.ResponseWriter, r *http.Request, pinned *bool, name *string, autoGenerate bool) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	conversation, err := s.store.UpdateAppPublicConversation(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "conversationID"), s.publicWorkflowActor(app), name, pinned, autoGenerate, time.Now())
	if err != nil {
		writeError(w, http.StatusNotFound, "conversation_not_found", "Conversation not found.")
		return
	}

	writeJSON(w, http.StatusOK, s.publicConversationResponse(conversation))
}

func (s *server) handlePublicMessageFeedback(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	messageID := chi.URLParam(r, "messageID")
	if _, found, exists := s.store.GetAppPublicMessage(app.ID, app.WorkspaceID, actorID, messageID); !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	} else if !found {
		writeError(w, http.StatusNotFound, "message_not_found", "Message not found.")
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	feedback, err := s.store.SaveAppMessageFeedback(app.ID, app.WorkspaceID, s.publicWorkflowActor(app), state.SaveAppMessageFeedbackInput{
		MessageID:     messageID,
		Rating:        stringFromAny(payload["rating"]),
		Content:       stringFromAny(payload["content"]),
		FromSource:    "user",
		FromEndUserID: actorID,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result": map[string]any{
			"rating":      nullIfEmpty(feedback.Rating),
			"content":     nullIfEmpty(feedback.Content),
			"from_source": feedback.FromSource,
		},
	})
}

func (s *server) handlePublicMessageSuggestedQuestions(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	message, found, exists := s.store.GetAppPublicMessage(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "messageID"))
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "message_not_found", "Message not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": cloneStringList(message.SuggestedQuestions),
	})
}

func (s *server) handlePublicMessageMoreLikeThis(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	original, found, exists := s.store.GetAppPublicMessage(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "messageID"))
	if !exists {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "message_not_found", "Message not found.")
		return
	}

	answer := strings.TrimSpace(original.Answer)
	if answer == "" {
		answer = "Alternative response"
	}
	answer = answer + "\n\nAlternative version from dify-go public runtime."

	conversation, message, err := s.store.SaveAppPublicMessage(app.ID, app.WorkspaceID, s.publicWorkflowActor(app), state.SaveAppPublicMessageInput{
		ActorID:          actorID,
		ConversationID:   original.ConversationID,
		ConversationName: publicConversationTitle(original.Query, answer),
		ConversationInfo: stringFromAny(app.ModelConfig["opening_statement"]),
		ConversationData: cloneJSONObject(original.Inputs),
		Query:            original.Query,
		Inputs:           cloneJSONObject(original.Inputs),
		Answer:           answer,
		ParentMessageID:  original.ID,
		Status:           "normal",
		Suggested:        s.publicSuggestedQuestions(app, original.Query, answer),
		ProviderLatency:  0.09,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	_ = conversation
	writeJSON(w, http.StatusOK, map[string]any{
		"id":              message.ID,
		"task_id":         message.TaskID,
		"conversation_id": message.ConversationID,
		"answer":          message.Answer,
	})
}

func (s *server) handlePublicSavedMessagesCreate(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}
	messageID := strings.TrimSpace(stringFromAny(payload["message_id"]))
	if messageID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "message_id is required.")
		return
	}

	if err := s.store.SaveAppPublicSavedMessage(app.ID, app.WorkspaceID, actorID, messageID, s.publicWorkflowActor(app), time.Now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handlePublicSavedMessagesList(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	messages, ok := s.store.ListAppPublicSavedMessages(app.ID, app.WorkspaceID, actorID)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	data := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		data = append(data, map[string]any{
			"id":     message.ID,
			"answer": message.Answer,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *server) handlePublicSavedMessageDelete(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	if err := s.store.DeleteAppPublicSavedMessage(app.ID, app.WorkspaceID, actorID, chi.URLParam(r, "messageID"), s.publicWorkflowActor(app), time.Now()); err != nil {
		writeError(w, http.StatusNotFound, "saved_message_not_found", "Saved message not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handlePublicAudioToText(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.currentPublicRuntimeApp(w, r); !ok {
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid multipart form.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "File is required.")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Unable to read uploaded audio.")
		return
	}

	name := "audio"
	if header != nil && strings.TrimSpace(header.Filename) != "" {
		name = strings.TrimSpace(header.Filename)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"text": fmt.Sprintf("Transcribed audio from %s (%d bytes)", name, len(content)),
	})
}

func (s *server) handlePublicTextToAudio(w http.ResponseWriter, r *http.Request) {
	app, actorID, ok := s.currentPublicRuntimeApp(w, r)
	if !ok {
		return
	}

	payload, err := decodeJSONObjectBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	text := strings.TrimSpace(stringFromAny(payload["text"]))
	if text == "" {
		if messageID := strings.TrimSpace(stringFromAny(payload["message_id"])); messageID != "" {
			message, found, exists := s.store.GetAppPublicMessage(app.ID, app.WorkspaceID, actorID, messageID)
			if !exists {
				writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
				return
			}
			if found {
				text = strings.TrimSpace(message.Answer)
			}
		}
	}
	if text == "" {
		text = "Generated audio from dify-go public runtime."
	}

	audio, err := base64.StdEncoding.DecodeString(publicAudioMP3Base64)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to encode audio response.")
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(audio)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audio)
}

func (s *server) currentPublicRuntimeApp(w http.ResponseWriter, r *http.Request) (state.App, string, bool) {
	app, _, ok := s.currentPublicApp(r)
	if !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return state.App{}, "", false
	}
	if !app.EnableSite || strings.TrimSpace(app.AccessMode) != "public" {
		writeError(w, http.StatusUnauthorized, "web_app_access_denied", "Web app access denied.")
		return state.App{}, "", false
	}

	actorID := strings.TrimSpace(r.Header.Get(webAppPassportHeader))
	if actorID == "" {
		actorID = strings.TrimSpace(readCookie(r, publicRuntimeSessionCookie))
	}
	if actorID == "" {
		actorID = accessTokenFromRequest(r)
	}
	if actorID == "" {
		actorID = runtimeID("public")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     publicRuntimeSessionCookie,
		Value:    actorID,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})

	return app, actorID, true
}

func isPublicChatApp(app state.App) bool {
	switch strings.TrimSpace(app.Mode) {
	case "chat", "agent-chat", "advanced-chat":
		return true
	default:
		return false
	}
}

func (s *server) persistPublicGeneratedMessage(app state.App, actorID string, payload map[string]any, mode string) (state.AppPublicConversation, state.AppPublicMessage, error) {
	inputs := workflowRunInputs(payload)
	query := strings.TrimSpace(stringFromAny(payload["query"]))
	if query == "" {
		query = publicQueryFromInputs(inputs)
	}

	files := s.publicMessageFiles(app.WorkspaceID, objectListFromAny(payload["files"]))
	answer := s.publicGeneratedAnswer(app, query, inputs, files, mode)
	conversationName := strings.TrimSpace(stringFromAny(payload["conversation_name"]))
	if conversationName == "" {
		conversationName = publicConversationTitle(query, answer)
	}

	return s.store.SaveAppPublicMessage(app.ID, app.WorkspaceID, s.publicWorkflowActor(app), state.SaveAppPublicMessageInput{
		ActorID:          actorID,
		ConversationID:   strings.TrimSpace(stringFromAny(payload["conversation_id"])),
		ConversationName: conversationName,
		ConversationInfo: stringFromAny(app.ModelConfig["opening_statement"]),
		ConversationData: cloneJSONObject(inputs),
		TaskID:           runtimeID("task"),
		Query:            query,
		Inputs:           cloneJSONObject(inputs),
		Answer:           answer,
		Status:           "normal",
		ParentMessageID:  strings.TrimSpace(stringFromAny(payload["parent_message_id"])),
		Suggested:        s.publicSuggestedQuestions(app, query, answer),
		Retriever:        []map[string]any{},
		Thoughts:         []map[string]any{},
		Files:            files,
		ExtraContents:    []map[string]any{},
		ProviderLatency:  0.08,
	}, time.Now())
}

func (s *server) publicMessageEvents(app state.App, conversation state.AppPublicConversation, message state.AppPublicMessage) []map[string]any {
	chunks := publicAnswerChunks(message.Answer)
	events := make([]map[string]any, 0, len(chunks)+1)
	for _, chunk := range chunks {
		events = append(events, map[string]any{
			"event":           "message",
			"task_id":         message.TaskID,
			"conversation_id": conversation.ID,
			"id":              message.ID,
			"answer":          chunk,
			"created_at":      message.CreatedAt,
		})
	}
	events = append(events, map[string]any{
		"event":           "message_end",
		"task_id":         message.TaskID,
		"conversation_id": conversation.ID,
		"id":              message.ID,
		"metadata": map[string]any{
			"annotation_reply":    nil,
			"retriever_resources": publicCloneMapList(message.RetrieverResources),
		},
		"files": []any{},
	})
	return events
}

func (s *server) publicConversationResponse(conversation state.AppPublicConversation) map[string]any {
	inputs := any(nil)
	if len(conversation.Inputs) > 0 {
		inputs = cloneJSONObject(conversation.Inputs)
	}

	return map[string]any{
		"id":           conversation.ID,
		"name":         firstImportValue(strings.TrimSpace(conversation.Name), "New Conversation"),
		"inputs":       inputs,
		"introduction": conversation.Introduction,
		"pinned":       conversation.Pinned,
	}
}

func (s *server) publicMessageResponse(app state.App, message state.AppPublicMessage) map[string]any {
	return map[string]any{
		"id":                        message.ID,
		"conversation_id":           message.ConversationID,
		"query":                     message.Query,
		"inputs":                    cloneJSONObject(message.Inputs),
		"message":                   s.publicMessageLog(message),
		"message_tokens":            message.MessageTokens,
		"answer_tokens":             message.AnswerTokens,
		"answer":                    message.Answer,
		"provider_response_latency": message.ProviderLatency,
		"created_at":                message.CreatedAt,
		"feedback":                  s.publicUserFeedback(app, message.ID),
		"feedbacks":                 s.messageFeedbackResponse(app, message.ID),
		"retriever_resources":       publicCloneMapList(message.RetrieverResources),
		"annotation":                nil,
		"annotation_hit_history":    nil,
		"message_files":             publicCloneMapList(message.MessageFiles),
		"metadata":                  map[string]any{},
		"agent_thoughts":            publicCloneMapList(message.AgentThoughts),
		"workflow_run_id":           nullIfEmpty(message.WorkflowRunID),
		"parent_message_id":         nullIfEmpty(message.ParentMessageID),
		"status":                    message.Status,
		"extra_contents":            publicCloneMapList(message.ExtraContents),
	}
}

func (s *server) publicMessageLog(message state.AppPublicMessage) []map[string]any {
	return []map[string]any{
		{"role": "user", "text": message.Query},
		{"role": "assistant", "text": message.Answer},
	}
}

func (s *server) publicUserFeedback(app state.App, messageID string) any {
	for _, item := range feedbacksByMessage(app.MessageFeedbacks)[messageID] {
		if item.FromSource != "user" {
			continue
		}
		return map[string]any{
			"rating":  nullIfEmpty(item.Rating),
			"content": nullIfEmpty(item.Content),
		}
	}
	return nil
}

func (s *server) publicSuggestedQuestions(app state.App, query, answer string) []string {
	if enabled, ok := mapFromAny(app.ModelConfig["suggested_questions_after_answer"])["enabled"].(bool); ok && !enabled {
		return []string{}
	}

	base := publicFirstValue(strings.TrimSpace(query), strings.TrimSpace(answer), "this topic")
	base = strings.TrimSpace(base)
	if len(base) > 48 {
		base = strings.TrimSpace(base[:48])
	}
	return []string{
		"Can you expand on " + base + "?",
		"Give me a shorter version of " + base + ".",
		"Show another example related to " + base + ".",
	}
}

func (s *server) publicGeneratedAnswer(app state.App, query string, inputs map[string]any, files []map[string]any, mode string) string {
	topic := publicFirstValue(strings.TrimSpace(query), publicQueryFromInputs(inputs), "your request")
	prefix := "Go public runtime reply"
	if mode == "completion" {
		prefix = "Generated completion"
	}

	answer := fmt.Sprintf("%s for %s.", prefix, topic)
	if len(files) > 0 {
		answer += fmt.Sprintf("\n\nProcessed %d attached file(s).", len(files))
	}
	if prompt := strings.TrimSpace(stringFromAny(app.ModelConfig["pre_prompt"])); prompt != "" {
		answer += "\n\nPrompt context loaded."
	}
	return answer
}

func publicQueryFromInputs(inputs map[string]any) string {
	if query := appLogQuery(inputs); query != "" {
		return query
	}
	if len(inputs) == 0 {
		return ""
	}
	return stringifyWorkflowValue(inputs)
}

func (s *server) publicMessageFiles(workspaceID string, files []map[string]any) []map[string]any {
	normalized := make([]map[string]any, 0, len(files))
	for _, file := range files {
		item := cloneJSONObject(file)
		if item["belongs_to"] == nil {
			item["belongs_to"] = "user"
		}
		uploadFileID := strings.TrimSpace(stringFromAny(item["upload_file_id"]))
		if uploadFileID != "" {
			if recorded, ok := s.store.GetUploadedFile(workspaceID, uploadFileID); ok {
				item["id"] = firstImportValue(strings.TrimSpace(stringFromAny(item["id"])), recorded.ID)
				item["filename"] = firstImportValue(strings.TrimSpace(stringFromAny(item["filename"])), recorded.Name)
				item["name"] = firstImportValue(strings.TrimSpace(stringFromAny(item["name"])), recorded.Name)
				item["mime_type"] = firstImportValue(strings.TrimSpace(stringFromAny(item["mime_type"])), recorded.MimeType)
				item["size"] = recorded.Size
				item["url"] = firstImportValue(strings.TrimSpace(stringFromAny(item["url"])), recorded.SourceURL)
			}
		}
		if item["transfer_method"] == nil {
			item["transfer_method"] = "local_file"
		}
		if item["type"] == nil {
			item["type"] = "custom"
		}
		normalized = append(normalized, item)
	}
	return normalized
}

func publicAnswerChunks(answer string) []string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return []string{""}
	}
	runes := []rune(answer)
	if len(runes) <= 48 {
		return []string{answer}
	}
	mid := len(runes) / 2
	return []string{
		string(runes[:mid]),
		string(runes[mid:]),
	}
}

func publicConversationTitle(query, answer string) string {
	value := publicFirstValue(strings.TrimSpace(query), strings.TrimSpace(answer), "New Conversation")
	if len(value) > 80 {
		return strings.TrimSpace(value[:80])
	}
	return value
}

func publicCloneMapList(src []map[string]any) []map[string]any {
	if src == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, len(src))
	for i, item := range src {
		out[i] = cloneJSONObject(item)
	}
	return out
}

func publicFirstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneStringList(src []string) []string {
	if src == nil {
		return []string{}
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}
