package server

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
	IsAuthorized           bool
	DatasourceName         string
	DatasourceLabel        string
	DatasourceDescription  string
}

func (s *server) ragPipelineDatasourcePlugins() []map[string]any {
	specs := []ragPipelineDatasourceProviderSpec{
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
			IsAuthorized:           true,
			DatasourceName:         "local-file",
			DatasourceLabel:        "Local File",
			DatasourceDescription:  "Upload and process a local file.",
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
			IsAuthorized:           false,
			DatasourceName:         "notion_datasource",
			DatasourceLabel:        "Notion",
			DatasourceDescription:  "Select pages from an authorized Notion workspace.",
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
			IsAuthorized:           false,
			DatasourceName:         "crawl",
			DatasourceLabel:        "Firecrawl",
			DatasourceDescription:  "Crawl a website and select the pages to process.",
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
			IsAuthorized:           false,
			DatasourceName:         "google_drive",
			DatasourceLabel:        "Google Drive",
			DatasourceDescription:  "Browse Google Drive folders and select files to process.",
		},
	}

	plugins := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		plugins = append(plugins, s.ragPipelineDatasourcePluginPayload(spec))
	}
	return plugins
}

func (s *server) ragPipelineDatasourcePluginPayload(spec ragPipelineDatasourceProviderSpec) map[string]any {
	return map[string]any{
		"plugin_id":                spec.PluginID,
		"plugin_unique_identifier": spec.PluginUniqueIdentifier,
		"provider":                 spec.Provider,
		"is_authorized":            spec.IsAuthorized,
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
