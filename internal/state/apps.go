package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Workflow struct {
	ID        string `json:"id"`
	CreatedBy string `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt int64  `json:"updated_at"`
}

type WorkflowState struct {
	ID                    string           `json:"id"`
	Graph                 map[string]any   `json:"graph"`
	Features              map[string]any   `json:"features"`
	CreatedBy             string           `json:"created_by"`
	CreatedAt             int64            `json:"created_at"`
	UpdatedBy             string           `json:"updated_by"`
	UpdatedAt             int64            `json:"updated_at"`
	Hash                  string           `json:"hash"`
	ToolPublished         bool             `json:"tool_published"`
	EnvironmentVariables  []map[string]any `json:"environment_variables"`
	ConversationVariables []map[string]any `json:"conversation_variables"`
	RagPipelineVariables  []map[string]any `json:"rag_pipeline_variables"`
	Version               string           `json:"version"`
	MarkedName            string           `json:"marked_name"`
	MarkedComment         string           `json:"marked_comment"`
}

type Site struct {
	AccessToken            string  `json:"access_token"`
	Title                  string  `json:"title"`
	Description            string  `json:"description"`
	ChatColorTheme         string  `json:"chat_color_theme"`
	ChatColorThemeInverted bool    `json:"chat_color_theme_inverted"`
	Author                 string  `json:"author"`
	SupportEmail           string  `json:"support_email"`
	DefaultLanguage        string  `json:"default_language"`
	CustomizeDomain        string  `json:"customize_domain"`
	Theme                  string  `json:"theme"`
	CustomizeTokenStrategy string  `json:"customize_token_strategy"`
	PromptPublic           bool    `json:"prompt_public"`
	AppBaseURL             string  `json:"app_base_url"`
	Copyright              string  `json:"copyright"`
	PrivacyPolicy          string  `json:"privacy_policy"`
	CustomDisclaimer       string  `json:"custom_disclaimer"`
	IconType               string  `json:"icon_type"`
	Icon                   string  `json:"icon"`
	IconBackground         string  `json:"icon_background"`
	IconURL                *string `json:"icon_url"`
	ShowWorkflowSteps      bool    `json:"show_workflow_steps"`
	UseIconAsAnswerIcon    bool    `json:"use_icon_as_answer_icon"`
}

type Tracing struct {
	Enabled  bool                      `json:"enabled"`
	Provider string                    `json:"provider"`
	Configs  map[string]map[string]any `json:"configs"`
}

type App struct {
	ID                  string                           `json:"id"`
	WorkspaceID         string                           `json:"workspace_id"`
	Name                string                           `json:"name"`
	Description         string                           `json:"description"`
	Mode                string                           `json:"mode"`
	IconType            string                           `json:"icon_type"`
	Icon                string                           `json:"icon"`
	IconBackground      string                           `json:"icon_background"`
	UseIconAsAnswerIcon bool                             `json:"use_icon_as_answer_icon"`
	EnableSite          bool                             `json:"enable_site"`
	EnableAPI           bool                             `json:"enable_api"`
	APIRPM              int                              `json:"api_rpm"`
	APIRPH              int                              `json:"api_rph"`
	IsDemo              bool                             `json:"is_demo"`
	AuthorName          string                           `json:"author_name"`
	CreatedBy           string                           `json:"created_by"`
	UpdatedBy           string                           `json:"updated_by"`
	CreatedAt           int64                            `json:"created_at"`
	UpdatedAt           int64                            `json:"updated_at"`
	AccessMode          string                           `json:"access_mode"`
	MaxActiveRequests   *int                             `json:"max_active_requests,omitempty"`
	ModelConfig         map[string]any                   `json:"model_config"`
	Site                Site                             `json:"site"`
	Workflow            *Workflow                        `json:"workflow,omitempty"`
	WorkflowDraft       *WorkflowState                   `json:"workflow_draft,omitempty"`
	WorkflowPublished   *WorkflowState                   `json:"workflow_published,omitempty"`
	WorkflowVersions    []WorkflowState                  `json:"workflow_versions,omitempty"`
	WorkflowRuns        []WorkflowRun                    `json:"workflow_runs,omitempty"`
	WorkflowNodeRuns    map[string]WorkflowNodeExecution `json:"workflow_node_runs,omitempty"`
	MCPServer           *AppMCPServer                    `json:"mcp_server,omitempty"`
	Annotations         []AppAnnotation                  `json:"annotations,omitempty"`
	MessageFeedbacks    []AppMessageFeedback             `json:"message_feedbacks,omitempty"`
	Tracing             Tracing                          `json:"tracing"`
}

type AppListFilters struct {
	Page          int
	Limit         int
	Mode          string
	Name          string
	IsCreatedByMe bool
	CurrentUserID string
}

type AppPage struct {
	Page    int
	Limit   int
	Total   int
	HasMore bool
	Data    []App
}

type CreateAppInput struct {
	Name           string
	Description    string
	Mode           string
	IconType       string
	Icon           string
	IconBackground string
}

type UpdateAppInput struct {
	Name                string
	Description         string
	IconType            string
	Icon                string
	IconBackground      string
	UseIconAsAnswerIcon *bool
	MaxActiveRequests   *int
}

type CopyAppInput struct {
	Name           string
	Description    string
	Mode           string
	IconType       string
	Icon           string
	IconBackground string
}

var allowedAppModes = map[string]struct{}{
	"chat":          {},
	"agent-chat":    {},
	"advanced-chat": {},
	"workflow":      {},
	"completion":    {},
}

func (s *Store) ListApps(workspaceID string, filters AppListFilters) AppPage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page := filters.Page
	if page < 1 {
		page = 1
	}
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var apps []App
	for _, app := range s.state.Apps {
		if app.WorkspaceID != workspaceID {
			continue
		}
		if filters.Mode != "" && filters.Mode != "all" && app.Mode != filters.Mode {
			continue
		}
		if filters.IsCreatedByMe && app.CreatedBy != filters.CurrentUserID {
			continue
		}
		if filters.Name != "" && !strings.Contains(strings.ToLower(app.Name), strings.ToLower(filters.Name)) {
			continue
		}
		apps = append(apps, app)
	}

	slices.SortFunc(apps, func(a, b App) int {
		if a.UpdatedAt == b.UpdatedAt {
			if a.CreatedAt == b.CreatedAt {
				return strings.Compare(a.ID, b.ID)
			}
			if a.CreatedAt > b.CreatedAt {
				return -1
			}
			return 1
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})

	total := len(apps)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	pageApps := make([]App, end-start)
	copy(pageApps, apps[start:end])

	return AppPage{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: end < total,
		Data:    pageApps,
	}
}

func (s *Store) GetApp(id, workspaceID string) (App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, app := range s.state.Apps {
		if app.ID == id && app.WorkspaceID == workspaceID {
			return app, true
		}
	}
	return App{}, false
}

func (s *Store) CreateApp(workspaceID string, owner User, input CreateAppInput, now time.Time) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode := normalizeAppMode(input.Mode)
	if mode == "" {
		return App{}, fmt.Errorf("unsupported app mode")
	}

	timestamp := now.UTC().Unix()
	appID := generateID("app")
	app := App{
		ID:                  appID,
		WorkspaceID:         workspaceID,
		Name:                strings.TrimSpace(input.Name),
		Description:         strings.TrimSpace(input.Description),
		Mode:                mode,
		IconType:            defaultIconType(input.IconType),
		Icon:                strings.TrimSpace(input.Icon),
		IconBackground:      strings.TrimSpace(input.IconBackground),
		UseIconAsAnswerIcon: false,
		EnableSite:          false,
		EnableAPI:           false,
		APIRPM:              60,
		APIRPH:              3600,
		IsDemo:              false,
		AuthorName:          owner.Name,
		CreatedBy:           owner.ID,
		UpdatedBy:           owner.ID,
		CreatedAt:           timestamp,
		UpdatedAt:           timestamp,
		AccessMode:          "public",
		ModelConfig:         defaultAppModelConfig(mode, timestamp),
		Site:                defaultSiteConfig(owner, input, appID),
		Tracing: Tracing{
			Enabled: false,
			Configs: map[string]map[string]any{},
		},
	}
	if app.IconType == "emoji" && app.Icon == "" {
		app.Icon = "🤖"
	}
	if app.IconType == "emoji" && app.IconBackground == "" {
		app.IconBackground = "#FFEAD5"
	}
	if mode == "workflow" || mode == "advanced-chat" {
		app.Workflow = &Workflow{
			ID:        generateID("wf"),
			CreatedBy: owner.ID,
			CreatedAt: timestamp,
			UpdatedBy: owner.ID,
			UpdatedAt: timestamp,
		}
		app.Site.ShowWorkflowSteps = true
	}

	s.state.Apps = append(s.state.Apps, app)
	if err := s.saveLocked(); err != nil {
		return App{}, err
	}
	return app, nil
}

func (s *Store) UpdateApp(id, workspaceID string, input UpdateAppInput, user User, now time.Time) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(id, workspaceID)
	if index < 0 {
		return App{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	app.Name = strings.TrimSpace(input.Name)
	app.Description = strings.TrimSpace(input.Description)
	app.IconType = defaultIconType(input.IconType)
	app.Icon = strings.TrimSpace(input.Icon)
	app.IconBackground = strings.TrimSpace(input.IconBackground)
	if input.UseIconAsAnswerIcon != nil {
		app.UseIconAsAnswerIcon = *input.UseIconAsAnswerIcon
		app.Site.UseIconAsAnswerIcon = *input.UseIconAsAnswerIcon
	}
	app.MaxActiveRequests = input.MaxActiveRequests
	app.Site.Title = app.Name
	app.Site.Description = app.Description
	app.Site.IconType = app.IconType
	app.Site.Icon = app.Icon
	app.Site.IconBackground = app.IconBackground
	if app.IconType != "image" && app.IconType != "link" {
		app.Site.IconURL = nil
	}
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.syncLinkedRAGPipelineDatasetFromAppLocked(&app, user, now)

	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return App{}, err
	}
	return app, nil
}

func (s *Store) syncLinkedRAGPipelineDatasetFromAppLocked(app *App, user User, now time.Time) {
	if app == nil {
		return
	}

	for i := range s.state.Datasets {
		dataset := &s.state.Datasets[i]
		if dataset.WorkspaceID != app.WorkspaceID || strings.TrimSpace(dataset.PipelineID) != app.ID {
			continue
		}

		dataset.Name = app.Name
		dataset.Description = app.Description
		dataset.IconInfo = normalizeDatasetIconInfo(DatasetIconInfo{
			Icon:           app.Icon,
			IconBackground: app.IconBackground,
			IconType:       app.IconType,
			IconURL:        appIconURL(app),
		}, app.Name)
		dataset.UpdatedAt = now.UTC().Unix()
		dataset.UpdatedBy = user.ID
		return
	}
}

func appIconURL(app *App) string {
	if app == nil || app.Site.IconURL == nil {
		return ""
	}
	return strings.TrimSpace(*app.Site.IconURL)
}

func (s *Store) DeleteApp(id, workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(id, workspaceID)
	if index < 0 {
		return fmt.Errorf("app not found")
	}

	s.removeLinkedRAGPipelineDatasetsLocked(id, workspaceID)
	s.state.Apps = append(s.state.Apps[:index], s.state.Apps[index+1:]...)
	return s.saveLocked()
}

func (s *Store) CopyApp(id, workspaceID string, owner User, input CopyAppInput, now time.Time) (App, error) {
	original, ok := s.GetApp(id, workspaceID)
	if !ok {
		return App{}, fmt.Errorf("app not found")
	}
	linkedDataset, hasLinkedDataset := s.FindRAGPipelineDataset(workspaceID, id)

	createInput := CreateAppInput{
		Name:           firstNonEmpty(input.Name, original.Name+" Copy"),
		Description:    firstNonEmpty(input.Description, original.Description),
		Mode:           firstNonEmpty(input.Mode, original.Mode),
		IconType:       firstNonEmpty(input.IconType, original.IconType),
		Icon:           firstNonEmpty(input.Icon, original.Icon),
		IconBackground: firstNonEmpty(input.IconBackground, original.IconBackground),
	}

	app, err := s.CreateApp(workspaceID, owner, createInput, now)
	if err != nil {
		return App{}, err
	}
	app.EnableSite = original.EnableSite
	app.EnableAPI = original.EnableAPI
	app.AccessMode = original.AccessMode
	app.UseIconAsAnswerIcon = original.UseIconAsAnswerIcon
	app.Site.UseIconAsAnswerIcon = original.Site.UseIconAsAnswerIcon
	app.ModelConfig = cloneMap(original.ModelConfig)
	app.Tracing = Tracing{
		Enabled:  false,
		Provider: "",
		Configs:  cloneTracingConfigs(original.Tracing.Configs),
	}
	if original.WorkflowDraft != nil {
		draft := cloneWorkflowState(*original.WorkflowDraft)
		draft.ID = firstNonEmpty(draft.ID, app.Workflow.ID)
		draft.CreatedAt = now.UTC().Unix()
		draft.CreatedBy = owner.ID
		draft.UpdatedAt = draft.CreatedAt
		draft.UpdatedBy = owner.ID
		draft.ToolPublished = original.WorkflowPublished != nil
		draft.Hash = workflowHash(draft.Graph, draft.Features, draft.EnvironmentVariables, draft.ConversationVariables, draft.RagPipelineVariables)
		app.WorkflowDraft = &draft
	}
	if original.WorkflowPublished != nil {
		published := cloneWorkflowState(*original.WorkflowPublished)
		published.ID = firstNonEmpty(published.ID, app.Workflow.ID)
		published.CreatedAt = now.UTC().Unix()
		published.CreatedBy = owner.ID
		published.UpdatedAt = published.CreatedAt
		published.UpdatedBy = owner.ID
		published.Hash = workflowHash(published.Graph, published.Features, published.EnvironmentVariables, published.ConversationVariables, published.RagPipelineVariables)
		app.WorkflowPublished = &published
	}
	if len(original.WorkflowVersions) > 0 {
		app.WorkflowVersions = cloneWorkflowStateList(original.WorkflowVersions)
		for i := range app.WorkflowVersions {
			app.WorkflowVersions[i].CreatedAt = now.UTC().Unix()
			app.WorkflowVersions[i].CreatedBy = owner.ID
			app.WorkflowVersions[i].UpdatedAt = app.WorkflowVersions[i].CreatedAt
			app.WorkflowVersions[i].UpdatedBy = owner.ID
		}
	}
	if app.WorkflowPublished != nil && len(app.WorkflowVersions) == 0 {
		app.WorkflowVersions = []WorkflowState{cloneWorkflowState(*app.WorkflowPublished)}
	}
	if app.Workflow != nil {
		if app.WorkflowPublished != nil {
			app.Workflow.ID = app.WorkflowPublished.ID
		} else if app.WorkflowDraft != nil {
			app.Workflow.ID = app.WorkflowDraft.ID
		}
	}

	var copiedDataset *Dataset
	if hasLinkedDataset && app.Workflow != nil && strings.TrimSpace(app.Mode) == "workflow" {
		dataset := copiedLinkedRAGPipelineDataset(linkedDataset, app, owner, now)
		copiedDataset = &dataset
	}

	return s.replaceAppAndLinkedDataset(app, copiedDataset)
}

func (s *Store) UpdateAppSiteStatus(id, workspaceID string, enabled bool, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		app.EnableSite = enabled
	})
}

func (s *Store) UpdateAppAPIStatus(id, workspaceID string, enabled bool, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		app.EnableAPI = enabled
	})
}

func (s *Store) UpdateAppSite(id, workspaceID string, updates map[string]any, user User, now time.Time) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(id, workspaceID)
	if index < 0 {
		return App{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	for key, value := range updates {
		switch key {
		case "title":
			app.Site.Title = stringValue(value, app.Site.Title)
		case "description":
			app.Site.Description = stringValue(value, app.Site.Description)
		case "chat_color_theme":
			app.Site.ChatColorTheme = stringValue(value, app.Site.ChatColorTheme)
		case "chat_color_theme_inverted":
			app.Site.ChatColorThemeInverted = boolValue(value, app.Site.ChatColorThemeInverted)
		case "default_language":
			app.Site.DefaultLanguage = stringValue(value, app.Site.DefaultLanguage)
		case "customize_domain":
			app.Site.CustomizeDomain = stringValue(value, app.Site.CustomizeDomain)
		case "copyright":
			app.Site.Copyright = stringValue(value, app.Site.Copyright)
		case "privacy_policy":
			app.Site.PrivacyPolicy = stringValue(value, app.Site.PrivacyPolicy)
		case "custom_disclaimer":
			app.Site.CustomDisclaimer = stringValue(value, app.Site.CustomDisclaimer)
		case "customize_token_strategy":
			app.Site.CustomizeTokenStrategy = stringValue(value, app.Site.CustomizeTokenStrategy)
		case "prompt_public":
			app.Site.PromptPublic = boolValue(value, app.Site.PromptPublic)
		case "app_base_url":
			app.Site.AppBaseURL = stringValue(value, app.Site.AppBaseURL)
		case "icon_type":
			app.Site.IconType = defaultIconType(stringValue(value, app.Site.IconType))
			app.IconType = app.Site.IconType
		case "icon":
			app.Site.Icon = stringValue(value, app.Site.Icon)
			app.Icon = app.Site.Icon
		case "icon_background":
			app.Site.IconBackground = stringValue(value, app.Site.IconBackground)
			app.IconBackground = app.Site.IconBackground
		case "show_workflow_steps":
			app.Site.ShowWorkflowSteps = boolValue(value, app.Site.ShowWorkflowSteps)
		case "use_icon_as_answer_icon":
			app.Site.UseIconAsAnswerIcon = boolValue(value, app.Site.UseIconAsAnswerIcon)
			app.UseIconAsAnswerIcon = app.Site.UseIconAsAnswerIcon
		}
	}
	app.Name = app.Site.Title
	app.Description = app.Site.Description
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = user.ID
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
		app.Workflow.UpdatedBy = user.ID
	}
	s.syncLinkedRAGPipelineDatasetFromAppLocked(&app, user, now)

	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return App{}, err
	}
	return app, nil
}

func (s *Store) ResetAppSiteAccessToken(id, workspaceID string, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		app.Site.AccessToken = generateID("site")
	})
}

func (s *Store) UpdateTracingStatus(id, workspaceID string, enabled bool, provider string, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		app.Tracing.Enabled = enabled
		if provider != "" {
			app.Tracing.Provider = provider
		}
		if !enabled {
			app.Tracing.Provider = ""
		}
	})
}

func (s *Store) GetTracingStatus(id, workspaceID string) (Tracing, bool) {
	app, ok := s.GetApp(id, workspaceID)
	if !ok {
		return Tracing{}, false
	}
	if app.Tracing.Configs == nil {
		app.Tracing.Configs = map[string]map[string]any{}
	}
	return app.Tracing, true
}

func (s *Store) GetTracingConfig(id, workspaceID, provider string) (map[string]any, bool, bool) {
	app, ok := s.GetApp(id, workspaceID)
	if !ok {
		return nil, false, false
	}
	if app.Tracing.Configs == nil {
		return map[string]any{}, false, true
	}
	config, configured := app.Tracing.Configs[provider]
	if !configured {
		return map[string]any{}, false, true
	}
	return cloneMap(config), true, true
}

func (s *Store) SaveTracingConfig(id, workspaceID, provider string, config map[string]any, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		if app.Tracing.Configs == nil {
			app.Tracing.Configs = map[string]map[string]any{}
		}
		app.Tracing.Configs[provider] = cloneMap(config)
	})
}

func (s *Store) RemoveTracingConfig(id, workspaceID, provider string, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		if app.Tracing.Configs != nil {
			delete(app.Tracing.Configs, provider)
		}
		if app.Tracing.Provider == provider {
			app.Tracing.Provider = ""
			app.Tracing.Enabled = false
		}
	})
}

func (s *Store) UpdateModelConfig(id, workspaceID string, config map[string]any, now time.Time) (App, error) {
	return s.mutateApp(id, workspaceID, now, func(app *App) {
		app.ModelConfig = cloneMap(config)
	})
}

func (s *Store) replaceApp(app App) (App, error) {
	return s.replaceAppAndLinkedDataset(app, nil)
}

func (s *Store) replaceAppAndLinkedDataset(app App, linkedDataset *Dataset) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(app.ID, app.WorkspaceID)
	if index < 0 {
		return App{}, fmt.Errorf("app not found")
	}
	s.state.Apps[index] = app
	if linkedDataset != nil {
		dataset := cloneDataset(*linkedDataset)
		s.state.Datasets = append(s.state.Datasets, dataset)
	}
	if err := s.saveLocked(); err != nil {
		return App{}, err
	}
	return app, nil
}

func (s *Store) mutateApp(id, workspaceID string, now time.Time, apply func(app *App)) (App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findAppIndexLocked(id, workspaceID)
	if index < 0 {
		return App{}, fmt.Errorf("app not found")
	}

	app := s.state.Apps[index]
	apply(&app)
	app.UpdatedAt = now.UTC().Unix()
	app.UpdatedBy = app.CreatedBy
	if app.Workflow != nil {
		app.Workflow.UpdatedAt = app.UpdatedAt
	}
	s.state.Apps[index] = app
	if err := s.saveLocked(); err != nil {
		return App{}, err
	}
	return app, nil
}

func (s *Store) findAppIndexLocked(id, workspaceID string) int {
	for i, app := range s.state.Apps {
		if app.ID == id && app.WorkspaceID == workspaceID {
			return i
		}
	}
	return -1
}

func (s *Store) removeLinkedRAGPipelineDatasetsLocked(appID, workspaceID string) {
	filtered := s.state.Datasets[:0]
	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID == workspaceID && strings.TrimSpace(dataset.PipelineID) == appID {
			continue
		}
		filtered = append(filtered, dataset)
	}
	s.state.Datasets = filtered
}

func copiedLinkedRAGPipelineDataset(source Dataset, app App, owner User, now time.Time) Dataset {
	dataset := cloneDataset(source)
	timestamp := now.UTC().Unix()
	dataset.ID = generateID("dataset")
	dataset.WorkspaceID = app.WorkspaceID
	dataset.Name = app.Name
	dataset.Description = app.Description
	dataset.AuthorName = ownerDisplayName(owner)
	dataset.CreatedBy = owner.ID
	dataset.UpdatedBy = owner.ID
	dataset.CreatedAt = timestamp
	dataset.UpdatedAt = timestamp
	dataset.IsPublished = app.WorkflowPublished != nil
	dataset.PipelineID = app.ID
	dataset.IconInfo = normalizeDatasetIconInfo(DatasetIconInfo{
		Icon:           app.Icon,
		IconBackground: app.IconBackground,
		IconType:       app.IconType,
		IconURL:        appIconURL(&app),
	}, app.Name)
	dataset.Documents = []DatasetDocument{}
	dataset.Queries = []DatasetQueryRecord{}
	dataset.BatchImportJobs = []DatasetBatchImportJob{}
	normalizeDataset(&dataset)
	return dataset
}

func defaultAppModelConfig(mode string, timestamp int64) map[string]any {
	modelMode := "chat"
	if mode == "completion" {
		modelMode = "completion"
	}

	return map[string]any{
		"opening_statement":                "",
		"suggested_questions":              []string{},
		"suggested_questions_after_answer": map[string]any{"enabled": false},
		"speech_to_text":                   map[string]any{"enabled": false},
		"text_to_speech":                   map[string]any{"enabled": false, "voice": "", "language": ""},
		"retriever_resource":               map[string]any{"enabled": false},
		"annotation_reply": map[string]any{
			"id":              "",
			"enabled":         false,
			"score_threshold": 0.9,
			"embedding_model": map[string]any{"embedding_provider_name": "", "embedding_model_name": ""},
		},
		"more_like_this": map[string]any{"enabled": false},
		"sensitive_word_avoidance": map[string]any{
			"enabled": false,
		},
		"external_data_tools":    []any{},
		"model":                  map[string]any{"provider": "langgenius/openai/openai", "name": "gpt-4o-mini", "mode": modelMode, "completion_params": defaultCompletionParams()},
		"user_input_form":        []any{},
		"dataset_query_variable": "",
		"pre_prompt":             "",
		"agent_mode": map[string]any{
			"enabled":       mode == "agent-chat",
			"strategy":      "function_call",
			"max_iteration": 10,
			"tools":         []any{},
		},
		"prompt_type":        "simple",
		"chat_prompt_config": map[string]any{"prompt": []map[string]any{{"role": "system", "text": ""}}},
		"completion_prompt_config": map[string]any{
			"prompt":                      map[string]any{"text": ""},
			"conversation_histories_role": map[string]any{"user_prefix": "", "assistant_prefix": ""},
		},
		"dataset_configs": map[string]any{
			"retrieval_model":         "multiple",
			"reranking_model":         map[string]any{"reranking_provider_name": "", "reranking_model_name": ""},
			"top_k":                   4,
			"score_threshold_enabled": false,
			"score_threshold":         0.8,
			"datasets":                map[string]any{"datasets": []any{}},
		},
		"file_upload": map[string]any{
			"image": map[string]any{
				"enabled":          false,
				"number_limits":    3,
				"detail":           "low",
				"transfer_methods": []string{"local_file", "remote_url"},
			},
		},
		"created_at": timestamp,
		"updated_at": timestamp,
		"system_parameters": map[string]any{
			"audio_file_size_limit":      50 * 1024 * 1024,
			"file_size_limit":            15 * 1024 * 1024,
			"image_file_size_limit":      10 * 1024 * 1024,
			"video_file_size_limit":      100 * 1024 * 1024,
			"workflow_file_upload_limit": 10,
		},
	}
}

func defaultCompletionParams() map[string]any {
	return map[string]any{
		"max_tokens":        512,
		"temperature":       0.7,
		"top_p":             1.0,
		"echo":              false,
		"stop":              []any{},
		"presence_penalty":  0.0,
		"frequency_penalty": 0.0,
	}
}

func defaultSiteConfig(owner User, input CreateAppInput, appID string) Site {
	return Site{
		AccessToken:            generateID("site"),
		Title:                  strings.TrimSpace(input.Name),
		Description:            strings.TrimSpace(input.Description),
		ChatColorTheme:         "#155AEF",
		ChatColorThemeInverted: false,
		Author:                 owner.Name,
		SupportEmail:           owner.Email,
		DefaultLanguage:        firstNonEmpty(owner.InterfaceLanguage, "en-US"),
		CustomizeDomain:        "",
		Theme:                  "light",
		CustomizeTokenStrategy: "not_allow",
		PromptPublic:           false,
		AppBaseURL:             "http://localhost:3000",
		Copyright:              "",
		PrivacyPolicy:          "",
		CustomDisclaimer:       "",
		IconType:               defaultIconType(input.IconType),
		Icon:                   strings.TrimSpace(input.Icon),
		IconBackground:         strings.TrimSpace(input.IconBackground),
		ShowWorkflowSteps:      false,
		UseIconAsAnswerIcon:    false,
	}
}

func normalizeAppMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if _, ok := allowedAppModes[mode]; ok {
		return mode
	}
	return ""
}

func defaultIconType(iconType string) string {
	iconType = strings.TrimSpace(iconType)
	switch iconType {
	case "image", "link", "emoji":
		return iconType
	default:
		return "emoji"
	}
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		switch typed := v.(type) {
		case map[string]any:
			dst[k] = cloneMap(typed)
		case []any:
			dst[k] = cloneSlice(typed)
		default:
			dst[k] = typed
		}
	}
	return dst
}

func cloneSlice(src []any) []any {
	out := make([]any, len(src))
	for i, item := range src {
		switch typed := item.(type) {
		case map[string]any:
			out[i] = cloneMap(typed)
		case []any:
			out[i] = cloneSlice(typed)
		default:
			out[i] = typed
		}
	}
	return out
}

func cloneTracingConfigs(src map[string]map[string]any) map[string]map[string]any {
	if src == nil {
		return map[string]map[string]any{}
	}
	dst := make(map[string]map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneMap(v)
	}
	return dst
}

func stringValue(value any, fallback string) string {
	if str, ok := value.(string); ok {
		return str
	}
	return fallback
}

func boolValue(value any, fallback bool) bool {
	if boolean, ok := value.(bool); ok {
		return boolean
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
