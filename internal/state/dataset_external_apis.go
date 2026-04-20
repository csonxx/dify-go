package state

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

const externalKnowledgeAPISecretPlaceholder = "[__HIDDEN__]"

type ExternalKnowledgeAPI struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
	APIKey      string `json:"api_key"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedBy   string `json:"updated_by"`
	UpdatedAt   string `json:"updated_at"`
}

type ExternalKnowledgeAPIInput struct {
	Name        string
	Description string
	Endpoint    string
	APIKey      string
}

type ExternalKnowledgeAPIPage struct {
	Page    int
	Limit   int
	Total   int
	HasMore bool
	Data    []ExternalKnowledgeAPI
}

type DatasetBindingSummary struct {
	ID   string
	Name string
}

type CreateExternalDatasetInput struct {
	Name                   string
	Description            string
	ExternalKnowledgeID    string
	ExternalKnowledgeAPIID string
	ExternalRetrievalModel DatasetExternalRetrievalModel
}

func normalizeExternalKnowledgeAPI(api *ExternalKnowledgeAPI) {
	api.Name = strings.TrimSpace(api.Name)
	api.Description = strings.TrimSpace(api.Description)
	api.Endpoint = strings.TrimSpace(api.Endpoint)
	api.APIKey = strings.TrimSpace(api.APIKey)
	api.CreatedBy = strings.TrimSpace(api.CreatedBy)
	api.CreatedAt = strings.TrimSpace(api.CreatedAt)
	api.UpdatedBy = strings.TrimSpace(api.UpdatedBy)
	api.UpdatedAt = strings.TrimSpace(api.UpdatedAt)
}

func (s *Store) ListExternalKnowledgeAPIs(workspaceID string, page, limit int) ExternalKnowledgeAPIPage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items := make([]ExternalKnowledgeAPI, 0)
	for _, item := range s.state.ExternalKnowledgeAPIs {
		if item.WorkspaceID != workspaceID {
			continue
		}
		items = append(items, cloneExternalKnowledgeAPI(item))
	}

	slices.SortFunc(items, func(a, b ExternalKnowledgeAPI) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(b.ID, a.ID)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})

	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return ExternalKnowledgeAPIPage{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: end < total,
		Data:    cloneExternalKnowledgeAPIList(items[start:end]),
	}
}

func (s *Store) GetExternalKnowledgeAPI(workspaceID, apiID string) (ExternalKnowledgeAPI, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	index := s.findExternalKnowledgeAPIIndexLocked(workspaceID, apiID)
	if index < 0 {
		return ExternalKnowledgeAPI{}, false
	}
	return cloneExternalKnowledgeAPI(s.state.ExternalKnowledgeAPIs[index]), true
}

func (s *Store) CreateExternalKnowledgeAPI(workspaceID string, user User, input ExternalKnowledgeAPIInput, now time.Time) (ExternalKnowledgeAPI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api name is required")
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api endpoint is required")
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api key is required")
	}
	if s.externalKnowledgeAPINameExistsLocked(workspaceID, name, "") {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api name already exists")
	}

	nowString := now.UTC().Format(time.RFC3339)
	api := ExternalKnowledgeAPI{
		ID:          generateID("extapi"),
		WorkspaceID: workspaceID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Endpoint:    endpoint,
		APIKey:      apiKey,
		CreatedBy:   user.ID,
		CreatedAt:   nowString,
		UpdatedBy:   user.ID,
		UpdatedAt:   nowString,
	}
	normalizeExternalKnowledgeAPI(&api)

	s.state.ExternalKnowledgeAPIs = append(s.state.ExternalKnowledgeAPIs, api)
	if err := s.saveLocked(); err != nil {
		return ExternalKnowledgeAPI{}, err
	}
	return cloneExternalKnowledgeAPI(api), nil
}

func (s *Store) UpdateExternalKnowledgeAPI(workspaceID, apiID string, input ExternalKnowledgeAPIInput, user User, now time.Time) (ExternalKnowledgeAPI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findExternalKnowledgeAPIIndexLocked(workspaceID, apiID)
	if index < 0 {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api not found")
	}

	api := s.state.ExternalKnowledgeAPIs[index]
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api name is required")
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api endpoint is required")
	}
	if s.externalKnowledgeAPINameExistsLocked(workspaceID, name, apiID) {
		return ExternalKnowledgeAPI{}, fmt.Errorf("external knowledge api name already exists")
	}

	api.Name = name
	api.Description = strings.TrimSpace(input.Description)
	api.Endpoint = endpoint
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey != "" && apiKey != externalKnowledgeAPISecretPlaceholder {
		api.APIKey = apiKey
	}
	api.UpdatedBy = user.ID
	api.UpdatedAt = now.UTC().Format(time.RFC3339)
	normalizeExternalKnowledgeAPI(&api)
	s.state.ExternalKnowledgeAPIs[index] = api
	s.syncExternalKnowledgeAPIDatasetsLocked(workspaceID, api, false, user, now)

	if err := s.saveLocked(); err != nil {
		return ExternalKnowledgeAPI{}, err
	}
	return cloneExternalKnowledgeAPI(api), nil
}

func (s *Store) DeleteExternalKnowledgeAPI(workspaceID, apiID string, user User, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.findExternalKnowledgeAPIIndexLocked(workspaceID, apiID)
	if index < 0 {
		return fmt.Errorf("external knowledge api not found")
	}

	api := s.state.ExternalKnowledgeAPIs[index]
	s.state.ExternalKnowledgeAPIs = append(s.state.ExternalKnowledgeAPIs[:index], s.state.ExternalKnowledgeAPIs[index+1:]...)
	s.syncExternalKnowledgeAPIDatasetsLocked(workspaceID, api, true, user, now)
	return s.saveLocked()
}

func (s *Store) ExternalKnowledgeAPIDatasetBindings(workspaceID, apiID string) []DatasetBindingSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]DatasetBindingSummary, 0)
	for _, dataset := range s.state.Datasets {
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		if dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID != apiID {
			continue
		}
		items = append(items, DatasetBindingSummary{
			ID:   dataset.ID,
			Name: dataset.Name,
		})
	}

	slices.SortFunc(items, func(a, b DatasetBindingSummary) int {
		nameCmp := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		if nameCmp != 0 {
			return nameCmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	return items
}

func (s *Store) ExternalKnowledgeAPIUseCount(workspaceID, apiID string) int {
	return len(s.ExternalKnowledgeAPIDatasetBindings(workspaceID, apiID))
}

func (s *Store) CreateExternalDataset(workspaceID string, user User, input CreateExternalDatasetInput, now time.Time) (Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return Dataset{}, fmt.Errorf("dataset name is required")
	}
	externalKnowledgeID := strings.TrimSpace(input.ExternalKnowledgeID)
	if externalKnowledgeID == "" {
		return Dataset{}, fmt.Errorf("external knowledge id is required")
	}
	apiIndex := s.findExternalKnowledgeAPIIndexLocked(workspaceID, strings.TrimSpace(input.ExternalKnowledgeAPIID))
	if apiIndex < 0 {
		return Dataset{}, fmt.Errorf("external knowledge api not found")
	}

	api := s.state.ExternalKnowledgeAPIs[apiIndex]
	timestamp := now.UTC().Unix()
	dataset := Dataset{
		ID:                  generateID("dataset"),
		WorkspaceID:         workspaceID,
		Name:                name,
		Description:         strings.TrimSpace(input.Description),
		Permission:          datasetPermissionAllTeamMembers,
		AuthorName:          ownerDisplayName(user),
		CreatedBy:           user.ID,
		UpdatedBy:           user.ID,
		CreatedAt:           timestamp,
		UpdatedAt:           timestamp,
		Provider:            "external",
		EmbeddingAvailable:  true,
		IconInfo:            normalizeDatasetIconInfo(DatasetIconInfo{}, name),
		RetrievalModel:      normalizeDatasetRetrievalModel(DatasetRetrievalModel{}),
		BuiltInFieldEnabled: false,
		PartialMemberList:   []string{},
		RuntimeMode:         datasetRuntimeModeGeneral,
		ExternalKnowledgeInfo: DatasetExternalKnowledgeInfo{
			ExternalKnowledgeID:          externalKnowledgeID,
			ExternalKnowledgeAPIID:       api.ID,
			ExternalKnowledgeAPIName:     api.Name,
			ExternalKnowledgeAPIEndpoint: api.Endpoint,
		},
		ExternalRetrievalModel: normalizeDatasetExternalRetrievalModel(input.ExternalRetrievalModel),
		MetadataFields:         []DatasetMetadataField{},
		BatchImportJobs:        []DatasetBatchImportJob{},
		Documents:              []DatasetDocument{},
		Queries:                []DatasetQueryRecord{},
	}
	normalizeDataset(&dataset)

	s.state.Datasets = append(s.state.Datasets, dataset)
	if err := s.saveLocked(); err != nil {
		return Dataset{}, err
	}
	return cloneDataset(dataset), nil
}

func (s *Store) findExternalKnowledgeAPIIndexLocked(workspaceID, apiID string) int {
	for i, api := range s.state.ExternalKnowledgeAPIs {
		if api.WorkspaceID == workspaceID && api.ID == apiID {
			return i
		}
	}
	return -1
}

func (s *Store) externalKnowledgeAPINameExistsLocked(workspaceID, name, ignoreID string) bool {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for _, api := range s.state.ExternalKnowledgeAPIs {
		if api.WorkspaceID != workspaceID {
			continue
		}
		if api.ID == ignoreID {
			continue
		}
		if strings.ToLower(api.Name) == normalizedName {
			return true
		}
	}
	return false
}

func (s *Store) hydrateDatasetExternalKnowledgeInfoLocked(dataset *Dataset) {
	apiID := strings.TrimSpace(dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID)
	if apiID == "" {
		dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName = ""
		dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint = ""
		return
	}

	index := s.findExternalKnowledgeAPIIndexLocked(dataset.WorkspaceID, apiID)
	if index < 0 {
		dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName = ""
		dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint = ""
		return
	}

	api := s.state.ExternalKnowledgeAPIs[index]
	dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName = api.Name
	dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint = api.Endpoint
}

func (s *Store) syncExternalKnowledgeAPIDatasetsLocked(workspaceID string, api ExternalKnowledgeAPI, clear bool, user User, now time.Time) {
	timestamp := now.UTC().Unix()
	for i := range s.state.Datasets {
		dataset := &s.state.Datasets[i]
		if dataset.WorkspaceID != workspaceID {
			continue
		}
		if dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID != api.ID {
			continue
		}
		if clear {
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIID = ""
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName = ""
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint = ""
		} else {
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIName = api.Name
			dataset.ExternalKnowledgeInfo.ExternalKnowledgeAPIEndpoint = api.Endpoint
		}
		dataset.UpdatedAt = timestamp
		dataset.UpdatedBy = user.ID
	}
}

func cloneExternalKnowledgeAPI(src ExternalKnowledgeAPI) ExternalKnowledgeAPI {
	data, err := json.Marshal(src)
	if err != nil {
		return ExternalKnowledgeAPI{}
	}
	var out ExternalKnowledgeAPI
	if err := json.Unmarshal(data, &out); err != nil {
		return ExternalKnowledgeAPI{}
	}
	normalizeExternalKnowledgeAPI(&out)
	return out
}

func cloneExternalKnowledgeAPIList(src []ExternalKnowledgeAPI) []ExternalKnowledgeAPI {
	out := make([]ExternalKnowledgeAPI, len(src))
	for i, item := range src {
		out[i] = cloneExternalKnowledgeAPI(item)
	}
	return out
}
