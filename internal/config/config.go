package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr                 string
	AppVersion           string
	AppTitle             string
	Edition              string
	EnvName              string
	StateFile            string
	UploadDir            string
	InitPassword         string
	CookieDomain         string
	SecureCookies        bool
	LegacyAPIBaseURL     string
	CheckUpdateURL       string
	DefaultWorkspaceName string
	WebOrigins           []string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:                 envOrDefault("DIFY_GO_ADDR", ":5001"),
		AppVersion:           envOrDefault("DIFY_GO_VERSION", "0.1.0"),
		AppTitle:             envOrDefault("DIFY_GO_APP_TITLE", "Dify Go"),
		Edition:              envOrDefault("DIFY_GO_EDITION", "SELF_HOSTED"),
		EnvName:              envOrDefault("DIFY_GO_ENV", "DEVELOPMENT"),
		StateFile:            envOrDefault("DIFY_GO_STATE_FILE", "var/state.json"),
		UploadDir:            envOrDefault("DIFY_GO_UPLOAD_DIR", "var/uploads"),
		InitPassword:         os.Getenv("INIT_PASSWORD"),
		CookieDomain:         strings.TrimSpace(os.Getenv("DIFY_GO_COOKIE_DOMAIN")),
		SecureCookies:        envOrDefault("DIFY_GO_SECURE_COOKIES", "false") == "true",
		LegacyAPIBaseURL:     strings.TrimSpace(os.Getenv("DIFY_GO_LEGACY_API_BASE_URL")),
		CheckUpdateURL:       strings.TrimSpace(os.Getenv("DIFY_GO_CHECK_UPDATE_URL")),
		DefaultWorkspaceName: envOrDefault("DIFY_GO_DEFAULT_WORKSPACE_NAME", "Default Workspace"),
		WebOrigins:           splitCSV(envOrDefault("DIFY_GO_WEB_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001")),
		AccessTokenTTL:       60 * time.Minute,
		RefreshTokenTTL:      30 * 24 * time.Hour,
	}

	if cfg.Addr == "" {
		return Config{}, fmt.Errorf("empty listen address")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
