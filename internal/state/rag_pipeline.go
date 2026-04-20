package state

import (
	"fmt"
	"strings"
	"time"
)

const (
	ragPipelineDefaultDatasetName = "Untitled"
	ragPipelineIcon               = "📙"
	ragPipelineIconBackground     = "#FFF4ED"
	ragPipelineDocForm            = "text_model"
	ragPipelineIndexingTechnique  = "high_quality"
)

type CreateRAGPipelineDatasetInput struct {
	Name        string
	Description string
}

func (s *Store) CreateRAGPipelineDataset(workspaceID string, owner User, input CreateRAGPipelineDatasetInput, now time.Time) (Dataset, App, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = s.nextRAGPipelineDatasetNameLocked(workspaceID)
	}

	mode := normalizeAppMode("workflow")
	if mode == "" {
		return Dataset{}, App{}, fmt.Errorf("unsupported app mode")
	}

	timestamp := now.UTC().Unix()
	appID := generateID("app")
	appInput := CreateAppInput{
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		Mode:           mode,
		IconType:       "emoji",
		Icon:           ragPipelineIcon,
		IconBackground: ragPipelineIconBackground,
	}

	app := App{
		ID:                  appID,
		WorkspaceID:         workspaceID,
		Name:                appInput.Name,
		Description:         appInput.Description,
		Mode:                mode,
		IconType:            defaultIconType(appInput.IconType),
		Icon:                appInput.Icon,
		IconBackground:      appInput.IconBackground,
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
		Site:                defaultSiteConfig(owner, appInput, appID),
		Tracing: Tracing{
			Enabled: false,
			Configs: map[string]map[string]any{},
		},
		Workflow: &Workflow{
			ID:        generateID("wf"),
			CreatedBy: owner.ID,
			CreatedAt: timestamp,
			UpdatedBy: owner.ID,
			UpdatedAt: timestamp,
		},
	}
	app.Site.ShowWorkflowSteps = true

	dataset := Dataset{
		ID:                     generateID("dataset"),
		WorkspaceID:            workspaceID,
		Name:                   name,
		Description:            appInput.Description,
		Permission:             datasetPermissionOnlyMe,
		IndexingTechnique:      ragPipelineIndexingTechnique,
		AuthorName:             ownerDisplayName(owner),
		CreatedBy:              owner.ID,
		UpdatedBy:              owner.ID,
		CreatedAt:              timestamp,
		UpdatedAt:              timestamp,
		DocForm:                ragPipelineDocForm,
		Provider:               datasetProviderLocal,
		EmbeddingAvailable:     true,
		IconInfo:               normalizeDatasetIconInfo(DatasetIconInfo{Icon: ragPipelineIcon, IconBackground: ragPipelineIconBackground, IconType: "emoji"}, name),
		RetrievalModel:         normalizeDatasetRetrievalModel(DatasetRetrievalModel{}),
		ExternalRetrievalModel: normalizeDatasetExternalRetrievalModel(DatasetExternalRetrievalModel{}),
		BuiltInFieldEnabled:    false,
		PartialMemberList:      []string{},
		RuntimeMode:            datasetRuntimeModeRAGPipeline,
		EnableAPI:              false,
		IsPublished:            false,
		IsMultimodal:           false,
		PipelineID:             app.ID,
		MetadataFields:         []DatasetMetadataField{},
		BatchImportJobs:        []DatasetBatchImportJob{},
		Documents:              []DatasetDocument{},
		Queries:                []DatasetQueryRecord{},
	}

	s.state.Apps = append(s.state.Apps, app)
	s.state.Datasets = append(s.state.Datasets, dataset)
	if err := s.saveLocked(); err != nil {
		return Dataset{}, App{}, err
	}
	return cloneDataset(dataset), app, nil
}

func (s *Store) nextRAGPipelineDatasetNameLocked(workspaceID string) string {
	base := ragPipelineDefaultDatasetName
	used := make(map[string]struct{})
	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		used[strings.ToLower(strings.TrimSpace(dataset.Name))] = struct{}{}
	}
	if _, ok := used[strings.ToLower(base)]; !ok {
		return base
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s %d", base, i)
		if _, ok := used[strings.ToLower(candidate)]; !ok {
			return candidate
		}
	}
}
