package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type AppPublicConversation struct {
	ID           string         `json:"id"`
	ActorID      string         `json:"actor_id"`
	Name         string         `json:"name"`
	Inputs       map[string]any `json:"inputs,omitempty"`
	Introduction string         `json:"introduction"`
	Pinned       bool           `json:"pinned"`
	CreatedAt    int64          `json:"created_at"`
	UpdatedAt    int64          `json:"updated_at"`
}

type AppPublicMessage struct {
	ID                 string           `json:"id"`
	ActorID            string           `json:"actor_id"`
	TaskID             string           `json:"task_id"`
	ConversationID     string           `json:"conversation_id"`
	Query              string           `json:"query"`
	Inputs             map[string]any   `json:"inputs,omitempty"`
	Answer             string           `json:"answer"`
	Status             string           `json:"status"`
	CreatedAt          int64            `json:"created_at"`
	UpdatedAt          int64            `json:"updated_at"`
	ParentMessageID    string           `json:"parent_message_id,omitempty"`
	WorkflowRunID      string           `json:"workflow_run_id,omitempty"`
	SuggestedQuestions []string         `json:"suggested_questions,omitempty"`
	RetrieverResources []map[string]any `json:"retriever_resources,omitempty"`
	AgentThoughts      []map[string]any `json:"agent_thoughts,omitempty"`
	MessageFiles       []map[string]any `json:"message_files,omitempty"`
	ExtraContents      []map[string]any `json:"extra_contents,omitempty"`
	ProviderLatency    float64          `json:"provider_latency"`
	MessageTokens      int              `json:"message_tokens"`
	AnswerTokens       int              `json:"answer_tokens"`
}

type AppPublicSavedMessage struct {
	MessageID  string `json:"message_id"`
	ActorID    string `json:"actor_id"`
	CreatedAt  int64  `json:"created_at"`
	ModifiedAt int64  `json:"modified_at"`
}

type SaveAppPublicMessageInput struct {
	ActorID          string
	ConversationID   string
	ConversationName string
	ConversationInfo string
	ConversationData map[string]any
	TaskID           string
	Query            string
	Inputs           map[string]any
	Answer           string
	Status           string
	ParentMessageID  string
	WorkflowRunID    string
	Suggested        []string
	Retriever        []map[string]any
	Thoughts         []map[string]any
	Files            []map[string]any
	ExtraContents    []map[string]any
	ProviderLatency  float64
}

func (s *Store) SaveAppPublicMessage(appID, workspaceID string, user User, input SaveAppPublicMessageInput, now time.Time) (AppPublicConversation, AppPublicMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppPublicConversation{}, AppPublicMessage{}, fmt.Errorf("app not found")
	}

	actorID := strings.TrimSpace(input.ActorID)
	if actorID == "" {
		return AppPublicConversation{}, AppPublicMessage{}, fmt.Errorf("actor_id is required")
	}

	timestamp := now.UTC().Unix()
	app := s.state.Apps[index]
	normalizeApp(&app)

	conversationID := firstNonEmpty(strings.TrimSpace(input.ConversationID), generateID("conv"))
	conversationIndex := slices.IndexFunc(app.PublicConversations, func(item AppPublicConversation) bool {
		return item.ID == conversationID && item.ActorID == actorID
	})

	conversation := AppPublicConversation{
		ID:           conversationID,
		ActorID:      actorID,
		Name:         strings.TrimSpace(input.ConversationName),
		Inputs:       cloneMap(input.ConversationData),
		Introduction: strings.TrimSpace(input.ConversationInfo),
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}
	if conversationIndex >= 0 {
		conversation = cloneAppPublicConversation(app.PublicConversations[conversationIndex])
	}
	if strings.TrimSpace(input.ConversationName) != "" {
		conversation.Name = strings.TrimSpace(input.ConversationName)
	}
	if input.ConversationData != nil {
		conversation.Inputs = cloneMap(input.ConversationData)
	}
	if strings.TrimSpace(input.ConversationInfo) != "" || conversationIndex < 0 {
		conversation.Introduction = strings.TrimSpace(input.ConversationInfo)
	}
	conversation.UpdatedAt = timestamp
	if conversation.CreatedAt == 0 {
		conversation.CreatedAt = timestamp
	}

	parentMessageID := strings.TrimSpace(input.ParentMessageID)
	if parentMessageID == "" {
		for i := len(app.PublicMessages) - 1; i >= 0; i-- {
			message := app.PublicMessages[i]
			if message.ActorID == actorID && message.ConversationID == conversationID {
				parentMessageID = message.ID
				break
			}
		}
	}

	message := AppPublicMessage{
		ID:                 generateID("msg"),
		ActorID:            actorID,
		TaskID:             firstNonEmpty(strings.TrimSpace(input.TaskID), generateID("task")),
		ConversationID:     conversationID,
		Query:              strings.TrimSpace(input.Query),
		Inputs:             cloneMap(input.Inputs),
		Answer:             strings.TrimSpace(input.Answer),
		Status:             normalizeAppPublicMessageStatus(input.Status),
		CreatedAt:          timestamp,
		UpdatedAt:          timestamp,
		ParentMessageID:    parentMessageID,
		WorkflowRunID:      strings.TrimSpace(input.WorkflowRunID),
		SuggestedQuestions: cloneStringSlice(input.Suggested),
		RetrieverResources: cloneObjectList(input.Retriever),
		AgentThoughts:      cloneObjectList(input.Thoughts),
		MessageFiles:       cloneObjectList(input.Files),
		ExtraContents:      cloneObjectList(input.ExtraContents),
		ProviderLatency:    input.ProviderLatency,
		MessageTokens:      len(strings.Fields(strings.TrimSpace(input.Query))),
		AnswerTokens:       len(strings.Fields(strings.TrimSpace(input.Answer))),
	}

	if conversation.Name == "" {
		conversation.Name = publicConversationName(message.Query, message.Answer)
	}

	if conversationIndex >= 0 {
		app.PublicConversations[conversationIndex] = conversation
	} else {
		app.PublicConversations = append(app.PublicConversations, conversation)
	}
	app.PublicMessages = append(app.PublicMessages, message)
	if len(app.PublicMessages) > 1000 {
		app.PublicMessages = app.PublicMessages[len(app.PublicMessages)-1000:]
	}
	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = timestamp
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppPublicConversation{}, AppPublicMessage{}, err
	}

	return cloneAppPublicConversation(conversation), cloneAppPublicMessage(message), nil
}

func (s *Store) ListAppPublicConversations(appID, workspaceID, actorID string) ([]AppPublicConversation, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return nil, false
	}

	filtered := make([]AppPublicConversation, 0, len(app.PublicConversations))
	for _, conversation := range app.PublicConversations {
		if conversation.ActorID != strings.TrimSpace(actorID) {
			continue
		}
		filtered = append(filtered, cloneAppPublicConversation(conversation))
	}
	return filtered, true
}

func (s *Store) GetAppPublicConversation(appID, workspaceID, actorID, conversationID string) (AppPublicConversation, bool, bool) {
	conversations, ok := s.ListAppPublicConversations(appID, workspaceID, actorID)
	if !ok {
		return AppPublicConversation{}, false, false
	}

	for _, conversation := range conversations {
		if conversation.ID == strings.TrimSpace(conversationID) {
			return cloneAppPublicConversation(conversation), true, true
		}
	}
	return AppPublicConversation{}, false, true
}

func (s *Store) UpdateAppPublicConversation(appID, workspaceID, actorID, conversationID string, user User, name *string, pinned *bool, autoGenerate bool, now time.Time) (AppPublicConversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppPublicConversation{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	normalizeApp(&app)
	conversationIndex := slices.IndexFunc(app.PublicConversations, func(item AppPublicConversation) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.ID == strings.TrimSpace(conversationID)
	})
	if conversationIndex < 0 {
		return AppPublicConversation{}, fmt.Errorf("conversation not found")
	}

	conversation := cloneAppPublicConversation(app.PublicConversations[conversationIndex])
	if autoGenerate {
		latest := latestPublicMessage(app.PublicMessages, actorID, conversation.ID)
		conversation.Name = publicConversationName(latest.Query, latest.Answer)
	}
	if name != nil {
		conversation.Name = strings.TrimSpace(*name)
	}
	if pinned != nil {
		conversation.Pinned = *pinned
	}
	if strings.TrimSpace(conversation.Name) == "" {
		conversation.Name = "New Conversation"
	}
	conversation.UpdatedAt = now.UTC().Unix()
	app.PublicConversations[conversationIndex] = conversation
	app.UpdatedAt = conversation.UpdatedAt
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppPublicConversation{}, err
	}

	return cloneAppPublicConversation(conversation), nil
}

func (s *Store) DeleteAppPublicConversation(appID, workspaceID, actorID, conversationID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	normalizeApp(&app)

	conversationIndex := slices.IndexFunc(app.PublicConversations, func(item AppPublicConversation) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.ID == strings.TrimSpace(conversationID)
	})
	if conversationIndex < 0 {
		return fmt.Errorf("conversation not found")
	}

	messageIDs := map[string]struct{}{}
	filteredMessages := app.PublicMessages[:0]
	for _, message := range app.PublicMessages {
		if message.ActorID == strings.TrimSpace(actorID) && message.ConversationID == strings.TrimSpace(conversationID) {
			messageIDs[message.ID] = struct{}{}
			continue
		}
		filteredMessages = append(filteredMessages, message)
	}
	app.PublicMessages = filteredMessages
	app.PublicConversations = append(app.PublicConversations[:conversationIndex], app.PublicConversations[conversationIndex+1:]...)

	filteredSaved := app.SavedMessages[:0]
	for _, item := range app.SavedMessages {
		if _, ok := messageIDs[item.MessageID]; ok {
			continue
		}
		filteredSaved = append(filteredSaved, item)
	}
	app.SavedMessages = filteredSaved

	filteredFeedbacks := app.MessageFeedbacks[:0]
	for _, item := range app.MessageFeedbacks {
		if _, ok := messageIDs[item.MessageID]; ok {
			continue
		}
		filteredFeedbacks = append(filteredFeedbacks, item)
	}
	app.MessageFeedbacks = filteredFeedbacks

	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return s.saveLocked()
}

func (s *Store) ListAppPublicMessages(appID, workspaceID, actorID, conversationID string) ([]AppPublicMessage, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return nil, false
	}

	filtered := make([]AppPublicMessage, 0, len(app.PublicMessages))
	for _, message := range app.PublicMessages {
		if message.ActorID != strings.TrimSpace(actorID) {
			continue
		}
		if strings.TrimSpace(conversationID) != "" && message.ConversationID != strings.TrimSpace(conversationID) {
			continue
		}
		filtered = append(filtered, cloneAppPublicMessage(message))
	}
	return filtered, true
}

func (s *Store) GetAppPublicMessage(appID, workspaceID, actorID, messageID string) (AppPublicMessage, bool, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return AppPublicMessage{}, false, false
	}

	for _, message := range app.PublicMessages {
		if message.ActorID == strings.TrimSpace(actorID) && message.ID == strings.TrimSpace(messageID) {
			return cloneAppPublicMessage(message), true, true
		}
	}
	return AppPublicMessage{}, false, true
}

func (s *Store) StopAppPublicMessageTask(appID, workspaceID, actorID, taskID string, user User, now time.Time) (AppPublicMessage, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return AppPublicMessage{}, false, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	normalizeApp(&app)
	messageIndex := slices.IndexFunc(app.PublicMessages, func(item AppPublicMessage) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.TaskID == strings.TrimSpace(taskID)
	})
	if messageIndex < 0 {
		return AppPublicMessage{}, false, nil
	}

	message := cloneAppPublicMessage(app.PublicMessages[messageIndex])
	message.Status = "stopped"
	message.UpdatedAt = now.UTC().Unix()
	app.PublicMessages[messageIndex] = message
	app.UpdatedAt = message.UpdatedAt
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return AppPublicMessage{}, false, err
	}

	return cloneAppPublicMessage(message), true, nil
}

func (s *Store) SaveAppPublicSavedMessage(appID, workspaceID, actorID, messageID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	normalizeApp(&app)
	messageIndex := slices.IndexFunc(app.PublicMessages, func(item AppPublicMessage) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.ID == strings.TrimSpace(messageID)
	})
	if messageIndex < 0 {
		return fmt.Errorf("message not found")
	}

	timestamp := now.UTC().Unix()
	savedIndex := slices.IndexFunc(app.SavedMessages, func(item AppPublicSavedMessage) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.MessageID == strings.TrimSpace(messageID)
	})
	if savedIndex >= 0 {
		saved := app.SavedMessages[savedIndex]
		saved.ModifiedAt = timestamp
		app.SavedMessages[savedIndex] = saved
	} else {
		app.SavedMessages = append(app.SavedMessages, AppPublicSavedMessage{
			MessageID:  strings.TrimSpace(messageID),
			ActorID:    strings.TrimSpace(actorID),
			CreatedAt:  timestamp,
			ModifiedAt: timestamp,
		})
	}

	app.UpdatedAt = timestamp
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return s.saveLocked()
}

func (s *Store) DeleteAppPublicSavedMessage(appID, workspaceID, actorID, messageID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(appID, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	normalizeApp(&app)
	savedIndex := slices.IndexFunc(app.SavedMessages, func(item AppPublicSavedMessage) bool {
		return item.ActorID == strings.TrimSpace(actorID) && item.MessageID == strings.TrimSpace(messageID)
	})
	if savedIndex < 0 {
		return fmt.Errorf("saved message not found")
	}

	app.SavedMessages = append(app.SavedMessages[:savedIndex], app.SavedMessages[savedIndex+1:]...)
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.state.Apps[index] = app
	return s.saveLocked()
}

func (s *Store) ListAppPublicSavedMessages(appID, workspaceID, actorID string) ([]AppPublicMessage, bool) {
	app, ok := s.GetApp(appID, workspaceID)
	if !ok {
		return nil, false
	}

	savedByMessageID := make(map[string]AppPublicSavedMessage, len(app.SavedMessages))
	for _, item := range app.SavedMessages {
		if item.ActorID != strings.TrimSpace(actorID) {
			continue
		}
		savedByMessageID[item.MessageID] = item
	}

	type savedMessageRef struct {
		message AppPublicMessage
		savedAt int64
	}
	refs := make([]savedMessageRef, 0, len(savedByMessageID))
	for _, message := range app.PublicMessages {
		saved, ok := savedByMessageID[message.ID]
		if !ok {
			continue
		}
		refs = append(refs, savedMessageRef{
			message: cloneAppPublicMessage(message),
			savedAt: saved.ModifiedAt,
		})
	}
	slices.SortFunc(refs, func(a, b savedMessageRef) int {
		if a.savedAt == b.savedAt {
			return strings.Compare(a.message.ID, b.message.ID)
		}
		if a.savedAt > b.savedAt {
			return -1
		}
		return 1
	})

	result := make([]AppPublicMessage, 0, len(refs))
	for _, item := range refs {
		result = append(result, item.message)
	}
	return result, true
}

func latestPublicMessage(messages []AppPublicMessage, actorID, conversationID string) AppPublicMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		item := messages[i]
		if item.ActorID == strings.TrimSpace(actorID) && item.ConversationID == strings.TrimSpace(conversationID) {
			return cloneAppPublicMessage(item)
		}
	}
	return AppPublicMessage{}
}

func publicConversationName(query, answer string) string {
	value := firstNonEmpty(strings.TrimSpace(query), strings.TrimSpace(answer), "New Conversation")
	if len(value) > 80 {
		return strings.TrimSpace(value[:80])
	}
	return value
}

func normalizeAppPublicMessageStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "paused", "stopped":
		return strings.TrimSpace(status)
	default:
		return "normal"
	}
}

func cloneAppPublicConversation(src AppPublicConversation) AppPublicConversation {
	return AppPublicConversation{
		ID:           src.ID,
		ActorID:      src.ActorID,
		Name:         src.Name,
		Inputs:       cloneMap(src.Inputs),
		Introduction: src.Introduction,
		Pinned:       src.Pinned,
		CreatedAt:    src.CreatedAt,
		UpdatedAt:    src.UpdatedAt,
	}
}

func cloneAppPublicMessage(src AppPublicMessage) AppPublicMessage {
	return AppPublicMessage{
		ID:                 src.ID,
		ActorID:            src.ActorID,
		TaskID:             src.TaskID,
		ConversationID:     src.ConversationID,
		Query:              src.Query,
		Inputs:             cloneMap(src.Inputs),
		Answer:             src.Answer,
		Status:             src.Status,
		CreatedAt:          src.CreatedAt,
		UpdatedAt:          src.UpdatedAt,
		ParentMessageID:    src.ParentMessageID,
		WorkflowRunID:      src.WorkflowRunID,
		SuggestedQuestions: cloneStringSlice(src.SuggestedQuestions),
		RetrieverResources: cloneObjectList(src.RetrieverResources),
		AgentThoughts:      cloneObjectList(src.AgentThoughts),
		MessageFiles:       cloneObjectList(src.MessageFiles),
		ExtraContents:      cloneObjectList(src.ExtraContents),
		ProviderLatency:    src.ProviderLatency,
		MessageTokens:      src.MessageTokens,
		AnswerTokens:       src.AnswerTokens,
	}
}
