package server

import (
	"strings"
)

type ragPipelineDatasourceProviderSpec struct {
	PluginID               string
	PluginUniqueIdentifier string
	Provider               string
	ProviderType           string
	Author                 string
	Label                  string
	Description            string
	Icon                   string
	Tags                   []string
	DatasourceName         string
	DatasourceLabel        string
	DatasourceDescription  string
	IncludeInAuthCatalog   bool
	SystemOAuth            bool
	CredentialSchema       []map[string]any
	OAuthClientSchema      []map[string]any
	OAuthCredentialSchema  []map[string]any
}

func ragPipelineDatasourceProviderSpecs() []ragPipelineDatasourceProviderSpec {
	return []ragPipelineDatasourceProviderSpec{
		{
			PluginID:               "langgenius/file",
			PluginUniqueIdentifier: "langgenius/file:0.0.1@dify-go",
			Provider:               "file",
			ProviderType:           "local_file",
			Author:                 "langgenius",
			Label:                  "File Source",
			Description:            "Upload a local file and use it as the pipeline datasource.",
			Icon:                   providerIconDataURI("F", "#DBEAFE", "#1D4ED8"),
			Tags:                   []string{"file", "builtin"},
			DatasourceName:         "local-file",
			DatasourceLabel:        "Local File",
			DatasourceDescription:  "Upload and process a local file.",
			IncludeInAuthCatalog:   false,
			SystemOAuth:            false,
			CredentialSchema:       []map[string]any{},
			OAuthClientSchema:      []map[string]any{},
			OAuthCredentialSchema:  []map[string]any{},
		},
		{
			PluginID:               "langgenius/notion_datasource",
			PluginUniqueIdentifier: "langgenius/notion_datasource:0.1.12@dify-go",
			Provider:               "notion_datasource",
			ProviderType:           "online_document",
			Author:                 "langgenius",
			Label:                  "Notion",
			Description:            "Browse authorized Notion pages and use them as pipeline input.",
			Icon:                   providerIconDataURI("N", "#111827", "#FFFFFF"),
			Tags:                   []string{"notion", "document"},
			DatasourceName:         "notion_datasource",
			DatasourceLabel:        "Notion",
			DatasourceDescription:  "Select pages from an authorized Notion workspace.",
			IncludeInAuthCatalog:   true,
			SystemOAuth:            true,
			CredentialSchema:       []map[string]any{},
			OAuthClientSchema: []map[string]any{
				datasourceCredentialField("client_id", "Client ID", "text-input", true, "", ""),
				datasourceCredentialField("client_secret", "Client Secret", "secret-input", true, "", ""),
				datasourceCredentialField("authorization_url", "Authorization URL", "text-input", false, "https://api.notion.com/v1/oauth/authorize", ""),
				datasourceCredentialField("token_url", "Token URL", "text-input", false, "https://api.notion.com/v1/oauth/token", ""),
				datasourceCredentialField("scope", "Scope", "text-input", false, "read:content", ""),
			},
			OAuthCredentialSchema: []map[string]any{
				datasourceCredentialField("access_token", "Access Token", "secret-input", true, "", ""),
				datasourceCredentialField("refresh_token", "Refresh Token", "secret-input", false, "", ""),
			},
		},
		{
			PluginID:               "langgenius/firecrawl_datasource",
			PluginUniqueIdentifier: "langgenius/firecrawl_datasource:0.2.4@dify-go",
			Provider:               "firecrawl",
			ProviderType:           "website_crawl",
			Author:                 "langgenius",
			Label:                  "Firecrawl",
			Description:            "Crawl websites, preview the discovered pages, and import the selected content.",
			Icon:                   providerIconDataURI("W", "#FEF3C7", "#92400E"),
			Tags:                   []string{"website", "crawl"},
			DatasourceName:         "crawl",
			DatasourceLabel:        "Firecrawl",
			DatasourceDescription:  "Crawl a website and select the pages to process.",
			IncludeInAuthCatalog:   true,
			SystemOAuth:            false,
			CredentialSchema: []map[string]any{
				datasourceCredentialField("api_key", "API Key", "secret-input", true, "", "Paste the Firecrawl API key for website crawling."),
			},
			OAuthClientSchema:     []map[string]any{},
			OAuthCredentialSchema: []map[string]any{},
		},
		{
			PluginID:               "langgenius/google_drive",
			PluginUniqueIdentifier: "langgenius/google_drive:0.1.6@dify-go",
			Provider:               "google_drive",
			ProviderType:           "online_drive",
			Author:                 "langgenius",
			Label:                  "Google Drive",
			Description:            "Browse files from Google Drive and use the selected file as pipeline input.",
			Icon:                   providerIconDataURI("D", "#DCFCE7", "#166534"),
			Tags:                   []string{"drive", "cloud"},
			DatasourceName:         "google_drive",
			DatasourceLabel:        "Google Drive",
			DatasourceDescription:  "Browse Google Drive folders and select files to process.",
			IncludeInAuthCatalog:   true,
			SystemOAuth:            true,
			CredentialSchema:       []map[string]any{},
			OAuthClientSchema: []map[string]any{
				datasourceCredentialField("client_id", "Client ID", "text-input", true, "", ""),
				datasourceCredentialField("client_secret", "Client Secret", "secret-input", true, "", ""),
				datasourceCredentialField("authorization_url", "Authorization URL", "text-input", false, "https://accounts.google.com/o/oauth2/v2/auth", ""),
				datasourceCredentialField("token_url", "Token URL", "text-input", false, "https://oauth2.googleapis.com/token", ""),
				datasourceCredentialField("scope", "Scope", "text-input", false, "https://www.googleapis.com/auth/drive.readonly", ""),
			},
			OAuthCredentialSchema: []map[string]any{
				datasourceCredentialField("access_token", "Access Token", "secret-input", true, "", ""),
				datasourceCredentialField("refresh_token", "Refresh Token", "secret-input", false, "", ""),
			},
		},
	}
}

func ragPipelineDatasourceProviderSpecByProvider(pluginID, provider string) (ragPipelineDatasourceProviderSpec, bool) {
	pluginID = strings.TrimSpace(pluginID)
	provider = strings.TrimSpace(provider)
	for _, item := range ragPipelineDatasourceProviderSpecs() {
		if item.PluginID == pluginID && item.Provider == provider {
			return item, true
		}
	}
	return ragPipelineDatasourceProviderSpec{}, false
}

func ragPipelineDatasourceProviderSpecByType(datasourceType string) (ragPipelineDatasourceProviderSpec, bool) {
	switch strings.TrimSpace(datasourceType) {
	case "online_document":
		return ragPipelineDatasourceProviderSpecByProvider("langgenius/notion_datasource", "notion_datasource")
	case "website_crawl":
		return ragPipelineDatasourceProviderSpecByProvider("langgenius/firecrawl_datasource", "firecrawl")
	case "online_drive":
		return ragPipelineDatasourceProviderSpecByProvider("langgenius/google_drive", "google_drive")
	case "local_file":
		return ragPipelineDatasourceProviderSpecByProvider("langgenius/file", "file")
	default:
		return ragPipelineDatasourceProviderSpec{}, false
	}
}

func (s *server) ragPipelineDatasourcePlugins(workspaceID string) []map[string]any {
	specs := ragPipelineDatasourceProviderSpecs()
	plugins := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		plugins = append(plugins, s.ragPipelineDatasourcePluginPayload(workspaceID, spec))
	}
	return plugins
}

func (s *server) ragPipelineDatasourcePluginPayload(workspaceID string, spec ragPipelineDatasourceProviderSpec) map[string]any {
	return map[string]any{
		"plugin_id":                spec.PluginID,
		"plugin_unique_identifier": spec.PluginUniqueIdentifier,
		"provider":                 spec.Provider,
		"is_authorized":            s.isRAGPipelineDatasourceAuthorized(workspaceID, spec),
		"declaration": map[string]any{
			"credentials_schema": []any{},
			"provider_type":      spec.ProviderType,
			"identity": map[string]any{
				"author":      spec.Author,
				"name":        spec.Provider,
				"label":       localizedText(spec.Label),
				"icon":        spec.Icon,
				"description": localizedText(spec.Description),
				"tags":        append([]string{}, spec.Tags...),
			},
			"datasources": []map[string]any{
				{
					"description": localizedText(spec.DatasourceDescription),
					"identity": map[string]any{
						"author":   spec.Author,
						"icon":     spec.Icon,
						"label":    localizedText(spec.DatasourceLabel),
						"name":     spec.DatasourceName,
						"provider": spec.Provider,
					},
					"parameters":    []any{},
					"output_schema": map[string]any{},
				},
			},
		},
	}
}

func (s *server) isRAGPipelineDatasourceAuthorized(workspaceID string, spec ragPipelineDatasourceProviderSpec) bool {
	if spec.ProviderType == "local_file" {
		return true
	}
	return len(s.store.ListWorkspaceDatasourceCredentials(workspaceID, spec.PluginID, spec.Provider)) > 0
}

func datasourceCredentialField(name, label, kind string, required bool, defaultValue any, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"label":       localizedText(label),
		"description": localizedText(description),
		"type":        kind,
		"required":    required,
		"default":     defaultValue,
		"multiple":    false,
	}
}
