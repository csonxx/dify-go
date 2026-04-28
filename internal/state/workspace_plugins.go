package state

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	WorkspacePluginInstallPermissionEveryone = "everyone"
	WorkspacePluginInstallPermissionAdmins   = "admins"
	WorkspacePluginInstallPermissionNoOne    = "noone"
	WorkspacePluginDebugPermissionEveryone   = "everyone"
	WorkspacePluginDebugPermissionAdmins     = "admins"
	WorkspacePluginDebugPermissionNoOne      = "noone"
	WorkspacePluginAutoUpgradeDisabled       = "disabled"
	WorkspacePluginAutoUpgradeFixOnly        = "fix_only"
	WorkspacePluginAutoUpgradeLatest         = "latest"
	WorkspacePluginAutoUpgradeExclude        = "exclude"
	WorkspacePluginAutoUpgradePartial        = "partial"
	WorkspacePluginAutoUpgradeAll            = "all"
	WorkspacePluginStatusActive              = "active"
	WorkspacePluginStatusDeleted             = "deleted"
	WorkspacePluginTaskStatusRunning         = "running"
	WorkspacePluginTaskStatusSuccess         = "success"
	WorkspacePluginTaskStatusFailed          = "failed"
)

type WorkspacePluginPermissionSettings struct {
	InstallPermission string `json:"install_permission"`
	DebugPermission   string `json:"debug_permission"`
}

type WorkspacePluginAutoUpgradeSettings struct {
	StrategySetting  string   `json:"strategy_setting"`
	UpgradeTimeOfDay int      `json:"upgrade_time_of_day"`
	UpgradeMode      string   `json:"upgrade_mode"`
	ExcludePlugins   []string `json:"exclude_plugins"`
	IncludePlugins   []string `json:"include_plugins"`
}

type WorkspacePluginPreferences struct {
	Permission  WorkspacePluginPermissionSettings  `json:"permission"`
	AutoUpgrade WorkspacePluginAutoUpgradeSettings `json:"auto_upgrade"`
}

type WorkspacePluginMeta struct {
	Repo    string `json:"repo,omitempty"`
	Version string `json:"version,omitempty"`
	Package string `json:"package,omitempty"`
}

type WorkspacePluginInstallation struct {
	ID                     string              `json:"id"`
	Name                   string              `json:"name"`
	PluginID               string              `json:"plugin_id"`
	PluginUniqueIdentifier string              `json:"plugin_unique_identifier"`
	Author                 string              `json:"author"`
	Category               string              `json:"category"`
	Version                string              `json:"version"`
	LatestVersion          string              `json:"latest_version"`
	LatestUniqueIdentifier string              `json:"latest_unique_identifier"`
	Source                 string              `json:"source"`
	Verified               bool                `json:"verified"`
	Label                  string              `json:"label"`
	Description            string              `json:"description"`
	Icon                   string              `json:"icon"`
	IconDark               string              `json:"icon_dark"`
	MinimumDifyVersion     string              `json:"minimum_dify_version"`
	Tags                   []string            `json:"tags,omitempty"`
	Status                 string              `json:"status"`
	DeprecatedReason       string              `json:"deprecated_reason,omitempty"`
	AlternativePluginID    string              `json:"alternative_plugin_id,omitempty"`
	Meta                   WorkspacePluginMeta `json:"meta,omitempty"`
	CreatedAt              int64               `json:"created_at"`
	UpdatedAt              int64               `json:"updated_at"`
}

type WorkspacePluginTaskPlugin struct {
	PluginUniqueIdentifier string `json:"plugin_unique_identifier"`
	PluginID               string `json:"plugin_id"`
	Source                 string `json:"source"`
	Status                 string `json:"status"`
	Message                string `json:"message"`
	Icon                   string `json:"icon"`
	Label                  string `json:"label"`
}

type WorkspacePluginTask struct {
	ID               string                      `json:"id"`
	CreatedAt        int64                       `json:"created_at"`
	UpdatedAt        int64                       `json:"updated_at"`
	Status           string                      `json:"status"`
	TotalPlugins     int                         `json:"total_plugins"`
	CompletedPlugins int                         `json:"completed_plugins"`
	Plugins          []WorkspacePluginTaskPlugin `json:"plugins,omitempty"`
}

type UpsertWorkspacePluginInput struct {
	OriginalPluginUniqueIdentifier string
	PluginUniqueIdentifier         string
	PluginID                       string
	Name                           string
	Author                         string
	Category                       string
	Version                        string
	LatestVersion                  string
	LatestUniqueIdentifier         string
	Source                         string
	Verified                       bool
	Label                          string
	Description                    string
	Icon                           string
	IconDark                       string
	MinimumDifyVersion             string
	Tags                           []string
	Status                         string
	DeprecatedReason               string
	AlternativePluginID            string
	Meta                           WorkspacePluginMeta
	TaskStatus                     string
	TaskMessage                    string
}

func normalizeWorkspacePluginPreferences(preferences *WorkspacePluginPreferences) {
	if preferences == nil {
		return
	}
	if strings.TrimSpace(preferences.Permission.InstallPermission) == "" {
		preferences.Permission.InstallPermission = WorkspacePluginInstallPermissionEveryone
	}
	if strings.TrimSpace(preferences.Permission.DebugPermission) == "" {
		preferences.Permission.DebugPermission = WorkspacePluginDebugPermissionEveryone
	}
	if strings.TrimSpace(preferences.AutoUpgrade.StrategySetting) == "" {
		preferences.AutoUpgrade.StrategySetting = WorkspacePluginAutoUpgradeDisabled
	}
	if strings.TrimSpace(preferences.AutoUpgrade.UpgradeMode) == "" {
		preferences.AutoUpgrade.UpgradeMode = WorkspacePluginAutoUpgradeExclude
	}
	if preferences.AutoUpgrade.ExcludePlugins == nil {
		preferences.AutoUpgrade.ExcludePlugins = []string{}
	}
	if preferences.AutoUpgrade.IncludePlugins == nil {
		preferences.AutoUpgrade.IncludePlugins = []string{}
	}
}

func normalizeWorkspacePluginInstallation(plugin *WorkspacePluginInstallation) {
	if plugin == nil {
		return
	}
	if strings.TrimSpace(plugin.Name) == "" {
		plugin.Name = workspacePluginNameFromPluginID(plugin.PluginID)
	}
	if strings.TrimSpace(plugin.Author) == "" {
		plugin.Author = workspacePluginAuthorFromPluginID(plugin.PluginID)
	}
	if strings.TrimSpace(plugin.Label) == "" {
		plugin.Label = workspacePluginHumanize(firstNonEmpty(plugin.Name, plugin.PluginID))
	}
	if strings.TrimSpace(plugin.Description) == "" {
		plugin.Description = plugin.Label
	}
	if strings.TrimSpace(plugin.Category) == "" {
		plugin.Category = "tool"
	}
	if strings.TrimSpace(plugin.Source) == "" {
		plugin.Source = "marketplace"
	}
	if strings.TrimSpace(plugin.Version) == "" {
		plugin.Version = firstNonEmpty(plugin.Meta.Version, workspacePluginVersionFromUniqueIdentifier(plugin.PluginUniqueIdentifier), "1.0.0")
	}
	if strings.TrimSpace(plugin.LatestVersion) == "" {
		plugin.LatestVersion = plugin.Version
	}
	if strings.TrimSpace(plugin.LatestUniqueIdentifier) == "" {
		plugin.LatestUniqueIdentifier = plugin.PluginUniqueIdentifier
	}
	if strings.TrimSpace(plugin.MinimumDifyVersion) == "" {
		plugin.MinimumDifyVersion = "0.0.0"
	}
	if strings.TrimSpace(plugin.Status) == "" {
		plugin.Status = WorkspacePluginStatusActive
	}
	if strings.TrimSpace(plugin.Icon) == "" {
		plugin.Icon = workspacePluginIconFilename(plugin.PluginID, false)
	}
	if strings.TrimSpace(plugin.IconDark) == "" {
		plugin.IconDark = workspacePluginIconFilename(plugin.PluginID, true)
	}
	if plugin.Tags == nil {
		plugin.Tags = []string{}
	}
}

func normalizeWorkspacePluginTask(task *WorkspacePluginTask) {
	if task == nil {
		return
	}
	if task.Plugins == nil {
		task.Plugins = []WorkspacePluginTaskPlugin{}
	}
	for i := range task.Plugins {
		if strings.TrimSpace(task.Plugins[i].Status) == "" {
			task.Plugins[i].Status = WorkspacePluginTaskStatusSuccess
		}
		if strings.TrimSpace(task.Plugins[i].Label) == "" {
			task.Plugins[i].Label = workspacePluginHumanize(firstNonEmpty(task.Plugins[i].PluginID, task.Plugins[i].PluginUniqueIdentifier))
		}
		if strings.TrimSpace(task.Plugins[i].Icon) == "" {
			task.Plugins[i].Icon = workspacePluginIconFilename(task.Plugins[i].PluginID, false)
		}
	}
	recalculateWorkspacePluginTask(task)
}

func recalculateWorkspacePluginTask(task *WorkspacePluginTask) {
	if task == nil {
		return
	}
	task.TotalPlugins = len(task.Plugins)
	completed := 0
	hasFailed := false
	hasRunning := false
	for _, item := range task.Plugins {
		switch item.Status {
		case WorkspacePluginTaskStatusFailed:
			completed++
			hasFailed = true
		case WorkspacePluginTaskStatusSuccess:
			completed++
		default:
			hasRunning = true
		}
	}
	task.CompletedPlugins = completed
	switch {
	case hasRunning:
		task.Status = WorkspacePluginTaskStatusRunning
	case hasFailed:
		task.Status = WorkspacePluginTaskStatusFailed
	default:
		task.Status = WorkspacePluginTaskStatusSuccess
	}
}

func cloneWorkspacePluginPreferences(src WorkspacePluginPreferences) WorkspacePluginPreferences {
	out := WorkspacePluginPreferences{
		Permission: WorkspacePluginPermissionSettings{
			InstallPermission: src.Permission.InstallPermission,
			DebugPermission:   src.Permission.DebugPermission,
		},
		AutoUpgrade: WorkspacePluginAutoUpgradeSettings{
			StrategySetting:  src.AutoUpgrade.StrategySetting,
			UpgradeTimeOfDay: src.AutoUpgrade.UpgradeTimeOfDay,
			UpgradeMode:      src.AutoUpgrade.UpgradeMode,
			ExcludePlugins:   cloneStringSlice(src.AutoUpgrade.ExcludePlugins),
			IncludePlugins:   cloneStringSlice(src.AutoUpgrade.IncludePlugins),
		},
	}
	normalizeWorkspacePluginPreferences(&out)
	return out
}

func cloneWorkspacePluginInstallation(src WorkspacePluginInstallation) WorkspacePluginInstallation {
	out := WorkspacePluginInstallation{
		ID:                     src.ID,
		Name:                   src.Name,
		PluginID:               src.PluginID,
		PluginUniqueIdentifier: src.PluginUniqueIdentifier,
		Author:                 src.Author,
		Category:               src.Category,
		Version:                src.Version,
		LatestVersion:          src.LatestVersion,
		LatestUniqueIdentifier: src.LatestUniqueIdentifier,
		Source:                 src.Source,
		Verified:               src.Verified,
		Label:                  src.Label,
		Description:            src.Description,
		Icon:                   src.Icon,
		IconDark:               src.IconDark,
		MinimumDifyVersion:     src.MinimumDifyVersion,
		Tags:                   cloneStringSlice(src.Tags),
		Status:                 src.Status,
		DeprecatedReason:       src.DeprecatedReason,
		AlternativePluginID:    src.AlternativePluginID,
		Meta: WorkspacePluginMeta{
			Repo:    src.Meta.Repo,
			Version: src.Meta.Version,
			Package: src.Meta.Package,
		},
		CreatedAt: src.CreatedAt,
		UpdatedAt: src.UpdatedAt,
	}
	normalizeWorkspacePluginInstallation(&out)
	return out
}

func cloneWorkspacePluginTaskPlugin(src WorkspacePluginTaskPlugin) WorkspacePluginTaskPlugin {
	return WorkspacePluginTaskPlugin{
		PluginUniqueIdentifier: src.PluginUniqueIdentifier,
		PluginID:               src.PluginID,
		Source:                 src.Source,
		Status:                 src.Status,
		Message:                src.Message,
		Icon:                   src.Icon,
		Label:                  src.Label,
	}
}

func cloneWorkspacePluginTask(src WorkspacePluginTask) WorkspacePluginTask {
	out := WorkspacePluginTask{
		ID:               src.ID,
		CreatedAt:        src.CreatedAt,
		UpdatedAt:        src.UpdatedAt,
		Status:           src.Status,
		TotalPlugins:     src.TotalPlugins,
		CompletedPlugins: src.CompletedPlugins,
		Plugins:          make([]WorkspacePluginTaskPlugin, len(src.Plugins)),
	}
	for i, item := range src.Plugins {
		out.Plugins[i] = cloneWorkspacePluginTaskPlugin(item)
	}
	normalizeWorkspacePluginTask(&out)
	return out
}

func (s *Store) GetWorkspacePluginPreferences(workspaceID string) (WorkspacePluginPreferences, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspacePluginPreferences{}, false
	}
	return cloneWorkspacePluginPreferences(settings.PluginPreferences), true
}

func (s *Store) UpdateWorkspacePluginPreferences(workspaceID string, preferences WorkspacePluginPreferences) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	settings.PluginPreferences = cloneWorkspacePluginPreferences(preferences)
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ExcludeWorkspacePluginAutoUpgrade(workspaceID, pluginID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return fmt.Errorf("plugin_id is required")
	}
	if !slices.Contains(settings.PluginPreferences.AutoUpgrade.ExcludePlugins, pluginID) {
		settings.PluginPreferences.AutoUpgrade.ExcludePlugins = append(settings.PluginPreferences.AutoUpgrade.ExcludePlugins, pluginID)
	}
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListWorkspaceInstalledPlugins(workspaceID string) []WorkspacePluginInstallation {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspacePluginInstallation{}
	}
	items := make([]WorkspacePluginInstallation, len(settings.Plugins))
	for i, item := range settings.Plugins {
		items[i] = cloneWorkspacePluginInstallation(item)
	}
	slices.SortFunc(items, func(a, b WorkspacePluginInstallation) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.PluginID, b.PluginID)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})
	return items
}

func (s *Store) ListWorkspaceInstalledPluginsByPluginIDs(workspaceID string, pluginIDs []string) []WorkspacePluginInstallation {
	items := s.ListWorkspaceInstalledPlugins(workspaceID)
	if len(pluginIDs) == 0 {
		return items
	}
	lookup := make(map[string]WorkspacePluginInstallation, len(items))
	for _, item := range items {
		lookup[item.PluginID] = item
	}
	filtered := make([]WorkspacePluginInstallation, 0, len(pluginIDs))
	for _, pluginID := range pluginIDs {
		if item, ok := lookup[strings.TrimSpace(pluginID)]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (s *Store) GetWorkspaceInstalledPluginByUniqueIdentifier(workspaceID, uniqueIdentifier string) (WorkspacePluginInstallation, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspacePluginInstallation{}, false
	}
	uniqueIdentifier = strings.TrimSpace(uniqueIdentifier)
	for _, item := range settings.Plugins {
		if item.PluginUniqueIdentifier == uniqueIdentifier {
			return cloneWorkspacePluginInstallation(item), true
		}
	}
	return WorkspacePluginInstallation{}, false
}

func (s *Store) GetWorkspaceInstalledPluginByInstallationID(workspaceID, installationID string) (WorkspacePluginInstallation, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspacePluginInstallation{}, false
	}
	installationID = strings.TrimSpace(installationID)
	for _, item := range settings.Plugins {
		if item.ID == installationID {
			return cloneWorkspacePluginInstallation(item), true
		}
	}
	return WorkspacePluginInstallation{}, false
}

func (s *Store) UpsertWorkspacePlugin(workspaceID string, input UpsertWorkspacePluginInput, now time.Time) (WorkspacePluginInstallation, WorkspacePluginTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return WorkspacePluginInstallation{}, WorkspacePluginTask{}, fmt.Errorf("workspace not found")
	}

	uniqueIdentifier := strings.TrimSpace(input.PluginUniqueIdentifier)
	if uniqueIdentifier == "" {
		return WorkspacePluginInstallation{}, WorkspacePluginTask{}, fmt.Errorf("plugin_unique_identifier is required")
	}

	pluginID := firstNonEmpty(strings.TrimSpace(input.PluginID), workspacePluginIDFromUniqueIdentifier(uniqueIdentifier))
	if pluginID == "" {
		return WorkspacePluginInstallation{}, WorkspacePluginTask{}, fmt.Errorf("plugin_id is required")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)

	index := -1
	if original := strings.TrimSpace(input.OriginalPluginUniqueIdentifier); original != "" {
		index = slices.IndexFunc(settings.Plugins, func(item WorkspacePluginInstallation) bool {
			return item.PluginUniqueIdentifier == original
		})
	}
	if index < 0 {
		index = slices.IndexFunc(settings.Plugins, func(item WorkspacePluginInstallation) bool {
			return item.PluginID == pluginID
		})
	}

	existing := WorkspacePluginInstallation{}
	createdAt := now.UTC().Unix()
	installationID := generateID("plug")
	if index >= 0 {
		existing = cloneWorkspacePluginInstallation(settings.Plugins[index])
		createdAt = existing.CreatedAt
		installationID = firstNonEmpty(existing.ID, installationID)
	}

	plugin := WorkspacePluginInstallation{
		ID:                     installationID,
		Name:                   firstNonEmpty(strings.TrimSpace(input.Name), workspacePluginNameFromPluginID(pluginID)),
		PluginID:               pluginID,
		PluginUniqueIdentifier: uniqueIdentifier,
		Author:                 firstNonEmpty(strings.TrimSpace(input.Author), workspacePluginAuthorFromPluginID(pluginID)),
		Category:               firstNonEmpty(strings.TrimSpace(input.Category), existing.Category, "tool"),
		Version:                firstNonEmpty(strings.TrimSpace(input.Version), strings.TrimSpace(input.Meta.Version), workspacePluginVersionFromUniqueIdentifier(uniqueIdentifier), existing.Version, "1.0.0"),
		LatestVersion:          firstNonEmpty(strings.TrimSpace(input.LatestVersion), strings.TrimSpace(input.Version), strings.TrimSpace(input.Meta.Version), workspacePluginVersionFromUniqueIdentifier(uniqueIdentifier), existing.LatestVersion, "1.0.0"),
		LatestUniqueIdentifier: firstNonEmpty(strings.TrimSpace(input.LatestUniqueIdentifier), uniqueIdentifier),
		Source:                 firstNonEmpty(strings.TrimSpace(input.Source), existing.Source, "marketplace"),
		Verified:               input.Verified || existing.Verified,
		Label:                  firstNonEmpty(strings.TrimSpace(input.Label), existing.Label, workspacePluginHumanize(workspacePluginNameFromPluginID(pluginID))),
		Description:            firstNonEmpty(strings.TrimSpace(input.Description), existing.Description, workspacePluginHumanize(workspacePluginNameFromPluginID(pluginID))+" plugin"),
		Icon:                   firstNonEmpty(strings.TrimSpace(input.Icon), existing.Icon, workspacePluginIconFilename(pluginID, false)),
		IconDark:               firstNonEmpty(strings.TrimSpace(input.IconDark), existing.IconDark, workspacePluginIconFilename(pluginID, true)),
		MinimumDifyVersion:     firstNonEmpty(strings.TrimSpace(input.MinimumDifyVersion), existing.MinimumDifyVersion, "0.0.0"),
		Tags:                   cloneStringSlice(input.Tags),
		Status:                 firstNonEmpty(strings.TrimSpace(input.Status), WorkspacePluginStatusActive),
		DeprecatedReason:       strings.TrimSpace(input.DeprecatedReason),
		AlternativePluginID:    strings.TrimSpace(input.AlternativePluginID),
		Meta: WorkspacePluginMeta{
			Repo:    firstNonEmpty(strings.TrimSpace(input.Meta.Repo), existing.Meta.Repo),
			Version: firstNonEmpty(strings.TrimSpace(input.Meta.Version), strings.TrimSpace(input.Version), existing.Meta.Version),
			Package: firstNonEmpty(strings.TrimSpace(input.Meta.Package), existing.Meta.Package),
		},
		CreatedAt: createdAt,
		UpdatedAt: now.UTC().Unix(),
	}
	if len(plugin.Tags) == 0 && len(existing.Tags) > 0 {
		plugin.Tags = cloneStringSlice(existing.Tags)
	}
	normalizeWorkspacePluginInstallation(&plugin)

	if index >= 0 {
		settings.Plugins[index] = plugin
	} else {
		settings.Plugins = append(settings.Plugins, plugin)
	}

	task := WorkspacePluginTask{
		ID:        generateID("ptask"),
		CreatedAt: now.UTC().Unix(),
		UpdatedAt: now.UTC().Unix(),
		Plugins: []WorkspacePluginTaskPlugin{
			{
				PluginUniqueIdentifier: plugin.PluginUniqueIdentifier,
				PluginID:               plugin.PluginID,
				Source:                 plugin.Source,
				Status:                 firstNonEmpty(strings.TrimSpace(input.TaskStatus), WorkspacePluginTaskStatusSuccess),
				Message:                strings.TrimSpace(input.TaskMessage),
				Icon:                   plugin.Icon,
				Label:                  plugin.Label,
			},
		},
	}
	normalizeWorkspacePluginTask(&task)
	settings.PluginTasks = append(settings.PluginTasks, task)

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	if err := s.saveLocked(); err != nil {
		return WorkspacePluginInstallation{}, WorkspacePluginTask{}, err
	}
	return cloneWorkspacePluginInstallation(plugin), cloneWorkspacePluginTask(task), nil
}

func (s *Store) DeleteWorkspacePluginInstallation(workspaceID, installationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.Plugins, func(item WorkspacePluginInstallation) bool {
		return item.ID == strings.TrimSpace(installationID)
	})
	if index < 0 {
		return fmt.Errorf("plugin installation %s not found", installationID)
	}
	settings.Plugins = append(settings.Plugins[:index], settings.Plugins[index+1:]...)
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) ListWorkspacePluginTasks(workspaceID string) []WorkspacePluginTask {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return []WorkspacePluginTask{}
	}
	items := make([]WorkspacePluginTask, len(settings.PluginTasks))
	for i, item := range settings.PluginTasks {
		items[i] = cloneWorkspacePluginTask(item)
	}
	slices.SortFunc(items, func(a, b WorkspacePluginTask) int {
		if a.UpdatedAt == b.UpdatedAt {
			return strings.Compare(a.ID, b.ID)
		}
		if a.UpdatedAt > b.UpdatedAt {
			return -1
		}
		return 1
	})
	return items
}

func (s *Store) GetWorkspacePluginTask(workspaceID, taskID string) (WorkspacePluginTask, bool) {
	settings, ok := s.GetWorkspaceToolSettings(workspaceID)
	if !ok {
		return WorkspacePluginTask{}, false
	}
	taskID = strings.TrimSpace(taskID)
	for _, item := range settings.PluginTasks {
		if item.ID == taskID {
			return cloneWorkspacePluginTask(item), true
		}
	}
	return WorkspacePluginTask{}, false
}

func (s *Store) DeleteWorkspacePluginTask(workspaceID, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	index := slices.IndexFunc(settings.PluginTasks, func(item WorkspacePluginTask) bool {
		return item.ID == strings.TrimSpace(taskID)
	})
	if index < 0 {
		return fmt.Errorf("plugin task %s not found", taskID)
	}
	settings.PluginTasks = append(settings.PluginTasks[:index], settings.PluginTasks[index+1:]...)
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) DeleteAllWorkspacePluginTasks(workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	settings.PluginTasks = []WorkspacePluginTask{}
	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func (s *Store) DeleteWorkspacePluginTaskItem(workspaceID, taskID, identifier string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaceIndex := s.findWorkspaceIndexLocked(workspaceID)
	if workspaceIndex < 0 {
		return fmt.Errorf("workspace not found")
	}

	settings := cloneWorkspaceToolSettings(s.state.Workspaces[workspaceIndex].ToolSettings)
	normalizeWorkspaceToolSettings(&settings)
	taskIndex := slices.IndexFunc(settings.PluginTasks, func(item WorkspacePluginTask) bool {
		return item.ID == strings.TrimSpace(taskID)
	})
	if taskIndex < 0 {
		return fmt.Errorf("plugin task %s not found", taskID)
	}

	task := cloneWorkspacePluginTask(settings.PluginTasks[taskIndex])
	itemIndex := slices.IndexFunc(task.Plugins, func(item WorkspacePluginTaskPlugin) bool {
		return item.PluginUniqueIdentifier == strings.TrimSpace(identifier) || item.PluginID == strings.TrimSpace(identifier)
	})
	if itemIndex < 0 {
		return fmt.Errorf("plugin task item %s not found", identifier)
	}

	task.Plugins = append(task.Plugins[:itemIndex], task.Plugins[itemIndex+1:]...)
	if len(task.Plugins) == 0 {
		settings.PluginTasks = append(settings.PluginTasks[:taskIndex], settings.PluginTasks[taskIndex+1:]...)
	} else {
		task.UpdatedAt = time.Now().UTC().Unix()
		recalculateWorkspacePluginTask(&task)
		settings.PluginTasks[taskIndex] = task
	}

	s.state.Workspaces[workspaceIndex].ToolSettings = settings
	return s.saveLocked()
}

func workspacePluginIDFromUniqueIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.LastIndex(value, "@"); index > 0 {
		return value[:index]
	}
	lastSlash := strings.LastIndex(value, "/")
	if index := strings.LastIndex(value, ":"); index > lastSlash {
		return value[:index]
	}
	return value
}

func workspacePluginVersionFromUniqueIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if index := strings.LastIndex(value, "@"); index > 0 && index+1 < len(value) {
		return strings.TrimSpace(value[index+1:])
	}
	lastSlash := strings.LastIndex(value, "/")
	if index := strings.LastIndex(value, ":"); index > lastSlash && index+1 < len(value) {
		return strings.TrimSpace(value[index+1:])
	}
	return ""
}

func workspacePluginAuthorFromPluginID(pluginID string) string {
	parts := strings.Split(strings.TrimSpace(pluginID), "/")
	if len(parts) > 1 {
		return parts[0]
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0]
	}
	return "local"
}

func workspacePluginNameFromPluginID(pluginID string) string {
	parts := strings.Split(strings.TrimSpace(pluginID), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func workspacePluginIconFilename(pluginID string, dark bool) string {
	base := strings.Trim(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(pluginID), "/", "-"), "_", "-"), "-")
	if base == "" {
		base = "plugin"
	}
	if dark {
		return base + "-dark.svg"
	}
	return base + ".svg"
}

func workspacePluginHumanize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == '/' || r == '.'
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}
