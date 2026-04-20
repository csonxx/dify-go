package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiCors "github.com/go-chi/cors"
	"golang.org/x/crypto/bcrypt"

	"github.com/langgenius/dify-go/internal/config"
	"github.com/langgenius/dify-go/internal/state"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
	csrfTokenCookie    = "csrf_token"
	initCookie         = "dify_go_init_validated"
	csrfHeader         = "X-CSRF-Token"
)

type ctxKey string

const userContextKey ctxKey = "user"

type server struct {
	cfg      config.Config
	store    *state.Store
	sessions *sessionManager
	legacy   *legacyProxy
}

func New(cfg config.Config) (http.Handler, error) {
	store, err := state.Open(cfg.StateFile)
	if err != nil {
		return nil, err
	}

	legacy, err := newLegacyProxy(cfg.LegacyAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("build legacy proxy: %w", err)
	}

	s := &server{
		cfg:      cfg,
		store:    store,
		sessions: newSessionManager(cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		legacy:   legacy,
	}

	r := chi.NewRouter()
	r.Use(chiCors.Handler(chiCors.Options{
		AllowedOrigins:   cfg.WebOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", csrfHeader, "X-App-Code", "X-App-Passport"},
		ExposedHeaders:   []string{"X-Version", "X-Env", "X-Trace-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(s.versionHeaders)

	r.Get("/", s.handleRoot)
	r.Get("/healthz", s.handleHealthz)

	r.Mount("/console/api", s.consoleRoutes())
	r.Mount("/api", s.publicRoutes())
	r.Mount("/files", s.fileRoutes())
	r.Mount("/inner/api", s.compatOnlyRoutes())
	r.Mount("/mcp", s.compatOnlyRoutes())
	r.Mount("/trigger", s.triggerRoutes())

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("route %s was not found", r.URL.Path))
	})

	return r, nil
}

func (s *server) consoleRoutes() http.Handler {
	r := chi.NewRouter()

	r.Get("/system-features", s.handleConsoleSystemFeatures)
	r.Get("/setup", s.handleSetupStatus)
	r.Post("/setup", s.handleSetup)
	r.Get("/init", s.handleInitStatus)
	r.Post("/init", s.handleInitValidate)
	r.Get("/version", s.handleVersion)
	r.Post("/login", s.handleLogin)
	r.Post("/refresh-token", s.handleRefreshToken)
	r.With(s.withAuth).Post("/logout", s.handleLogout)

	r.Group(func(r chi.Router) {
		r.Use(s.withAuth)
		r.Get("/account/profile", s.handleAccountProfile)
		r.Get("/account/avatar", s.handleAccountAvatar)
		r.Get("/workflow/{workflowRunID}/pause-details", s.handleWorkflowPauseDetails)
		s.mountAppRoutes(r)
		s.mountDatasetRoutes(r)
		r.Post("/workspaces/current", s.handleCurrentWorkspace)
		r.Get("/workspaces", s.handleWorkspaces)
		r.Get("/workspaces/current/permission", s.handleWorkspacePermission)
		s.mountWorkspaceModelRoutes(r)
		s.mountWorkspaceToolRoutes(r)
		s.mountWorkspaceExtensionRoutes(r)
		s.mountWorkspacePluginRoutes(r)
		r.Get("/files/upload", s.handleUploadConfig)
		r.Post("/files/upload", s.handleFileUpload)
		r.Get("/files/support-type", s.handleFileSupportTypes)
		r.Get("/files/{fileID}/preview", s.handleFilePreview)
		r.Post("/remote-files/upload", s.handleRemoteFileUpload)
		r.Get("/spec/schema-definitions", s.handleSchemaDefinitions)
	})

	r.NotFound(s.compatFallback)
	return r
}

func (s *server) publicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/system-features", s.handlePublicSystemFeatures)
	r.Get("/login/status", s.handlePublicLoginStatus)
	r.Post("/logout", s.handlePublicLogout)
	r.Get("/files/upload", s.handleUploadConfig)
	r.Post("/files/upload", s.handlePublicFileUpload)
	r.Get("/files/support-type", s.handleFileSupportTypes)
	r.Post("/remote-files/upload", s.handlePublicRemoteFileUpload)
	r.NotFound(s.compatFallback)
	return r
}

func (s *server) compatOnlyRoutes() http.Handler {
	r := chi.NewRouter()
	r.NotFound(s.compatFallback)
	return r
}

func (s *server) compatFallback(w http.ResponseWriter, r *http.Request) {
	if s.legacy != nil {
		s.legacy.ServeHTTP(w, r)
		return
	}

	writeError(
		w,
		http.StatusNotImplemented,
		"route_not_ported",
		fmt.Sprintf("The route %s is not ported to Go yet. Set DIFY_GO_LEGACY_API_BASE_URL to proxy unported APIs to the original Dify backend during migration.", r.URL.Path),
	)
}

func (s *server) versionHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version", s.cfg.AppVersion)
		w.Header().Set("X-Env", s.cfg.EnvName)
		next.ServeHTTP(w, r)
	})
}

func (s *server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.store.IsSetupComplete() {
			writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
			return
		}

		token := readCookie(r, accessTokenCookie)
		if token == "" {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				token = strings.TrimSpace(authHeader[7:])
			}
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required.")
			return
		}

		session, ok := s.sessions.Get(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Session expired or invalid.")
			return
		}

		cookieCSRF := readCookie(r, csrfTokenCookie)
		if requiresCSRF(r.Method) {
			headerCSRF := strings.TrimSpace(r.Header.Get(csrfHeader))
			if headerCSRF == "" || cookieCSRF == "" || headerCSRF != cookieCSRF || headerCSRF != session.CSRFToken {
				writeError(w, http.StatusUnauthorized, "unauthorized", "CSRF token is missing or invalid.")
				return
			}
		}

		user, ok := s.store.GetUser(session.UserID)
		if !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Account not found.")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *server) handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":        "dify-go",
		"version":     s.cfg.AppVersion,
		"description": "Go backend compatibility layer for Dify. Run the unchanged frontend from ./web and point it at this API on port 5001.",
	})
}

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	setupDone, _ := s.store.SetupStatus()
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"setup":  setupDone,
	})
}

func (s *server) handleConsoleSystemFeatures(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.systemFeaturesPayload())
}

func (s *server) handlePublicSystemFeatures(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.systemFeaturesPayload())
}

func (s *server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	completed, setupAt := s.store.SetupStatus()
	if completed {
		writeJSON(w, http.StatusOK, map[string]any{
			"step":     "finished",
			"setup_at": setupAt,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": "not_started",
	})
}

func (s *server) handleInitStatus(w http.ResponseWriter, r *http.Request) {
	if s.initValidated(r) || s.cfg.InitPassword == "" || s.store.IsSetupComplete() {
		writeJSON(w, http.StatusOK, map[string]any{"status": "finished"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "not_started"})
}

func (s *server) handleInitValidate(w http.ResponseWriter, r *http.Request) {
	if s.store.IsSetupComplete() {
		writeError(w, http.StatusForbidden, "already_setup", "Dify Go is already initialized.")
		return
	}

	var payload struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if s.cfg.InitPassword != "" && payload.Password != s.cfg.InitPassword {
		writeError(w, http.StatusUnauthorized, "init_validate_failed", "The initialization password is incorrect.")
		return
	}

	http.SetCookie(w, s.cookie(initCookie, "1", 30*time.Minute, false))
	writeJSON(w, http.StatusCreated, map[string]any{"result": "success"})
}

func (s *server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if s.store.IsSetupComplete() {
		writeError(w, http.StatusForbidden, "already_setup", "Dify Go is already initialized.")
		return
	}
	if s.cfg.InitPassword != "" && !s.initValidated(r) {
		writeError(w, http.StatusUnauthorized, "not_init_validated", "Initialization password validation is required before setup.")
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	if !strings.Contains(payload.Email, "@") || strings.TrimSpace(payload.Name) == "" || len(payload.Password) < 8 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Email, name, or password is invalid.")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password.")
		return
	}

	if _, _, err := s.store.CreateInitialSetup(
		payload.Email,
		payload.Name,
		string(passwordHash),
		payload.Language,
		s.cfg.DefaultWorkspaceName,
		time.Now(),
	); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to persist initial setup.")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"result": "success"})
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.store.IsSetupComplete() {
		writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
		return
	}

	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON payload.")
		return
	}

	user, ok := s.store.FindUserByEmail(payload.Email)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_failed", "Invalid email or password.")
		return
	}

	password := decodeMaybeBase64(payload.Password)
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		writeError(w, http.StatusUnauthorized, "authentication_failed", "Invalid email or password.")
		return
	}

	if _, err := s.store.TouchLogin(user.ID, time.Now()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to update login metadata.")
		return
	}

	session := s.sessions.Issue(user.ID)
	s.setAuthCookies(w, session)
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := readCookie(r, refreshTokenCookie)
	if refreshToken == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Refresh token is missing.")
		return
	}

	session, ok := s.sessions.Refresh(refreshToken)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Refresh token is invalid or expired.")
		return
	}

	s.setAuthCookies(w, session)
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Delete(readCookie(r, accessTokenCookie), readCookie(r, refreshTokenCookie))
	s.clearAuthCookies(w)
	writeJSON(w, http.StatusOK, map[string]any{"result": "success"})
}

func (s *server) handlePublicLogout(w http.ResponseWriter, r *http.Request) {
	s.handleLogout(w, r)
}

func (s *server) handleAccountProfile(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 user.ID,
		"name":               user.Name,
		"email":              user.Email,
		"avatar":             user.Avatar,
		"avatar_url":         nil,
		"is_password_set":    true,
		"interface_language": user.InterfaceLanguage,
		"interface_theme":    user.InterfaceTheme,
		"timezone":           user.Timezone,
		"last_login_at":      user.LastLoginAt,
		"last_active_at":     user.LastActiveAt,
		"created_at":         user.CreatedAt,
	})
}

func (s *server) handleAccountAvatar(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"avatar_url": "",
	})
}

func (s *server) handleCurrentWorkspace(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                     workspace.ID,
		"name":                   workspace.Name,
		"plan":                   workspace.Plan,
		"status":                 workspace.Status,
		"created_at":             workspace.CreatedAt,
		"role":                   user.Role,
		"providers":              []any{},
		"trial_credits":          workspace.TrialCredits,
		"trial_credits_used":     workspace.TrialCreditsUsed,
		"next_credit_reset_date": workspace.NextCreditResetDate,
	})
}

func (s *server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspaces := s.store.ListWorkspacesForUser(user.ID)
	items := make([]map[string]any, 0, len(workspaces))
	for _, workspace := range workspaces {
		items = append(items, map[string]any{
			"id":         workspace.ID,
			"name":       workspace.Name,
			"plan":       workspace.Plan,
			"status":     workspace.Status,
			"created_at": workspace.CreatedAt,
			"current":    true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (s *server) handleWorkspacePermission(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	workspace, ok := s.store.UserWorkspace(user.ID)
	if !ok {
		writeError(w, http.StatusNotFound, "workspace_not_found", "Workspace not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workspace_id":         workspace.ID,
		"allow_member_invite":  false,
		"allow_owner_transfer": false,
	})
}

func (s *server) handleUploadConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"batch_count_limit":                uploadBatchCountLimit,
		"image_file_size_limit":            uploadImageFileSizeLimitMB,
		"image_file_batch_limit":           uploadImageFileBatchLimit,
		"single_chunk_attachment_limit":    uploadSingleChunkAttachmentLimit,
		"attachment_image_file_size_limit": uploadAttachmentImageFileSizeLimitMB,
		"file_size_limit":                  uploadDocumentFileSizeLimitMB,
		"audio_file_size_limit":            uploadAudioFileSizeLimitMB,
		"video_file_size_limit":            uploadVideoFileSizeLimitMB,
		"workflow_file_upload_limit":       uploadWorkflowFileUploadLimit,
		"file_upload_limit":                uploadFileUploadLimit,
	})
}

func (s *server) handleFileSupportTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"allowed_extensions": []string{"txt", "markdown", "md", "mdx", "pdf", "html", "htm", "xlsx", "xls", "docx", "csv", "vtt", "properties"},
	})
}

func (s *server) handleSchemaDefinitions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (s *server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":         s.cfg.AppVersion,
		"release_date":    "",
		"release_notes":   "",
		"can_auto_update": false,
		"features": map[string]any{
			"can_replace_logo":             false,
			"model_load_balancing_enabled": false,
		},
	})
}

func (s *server) handlePublicLoginStatus(w http.ResponseWriter, r *http.Request) {
	loggedIn := false
	if token := readCookie(r, accessTokenCookie); token != "" {
		_, loggedIn = s.sessions.Get(token)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logged_in":     loggedIn,
		"app_logged_in": false,
	})
}

func (s *server) systemFeaturesPayload() map[string]any {
	return map[string]any{
		"trial_models": []any{},
		"plugin_installation_permission": map[string]any{
			"plugin_installation_scope":    "all",
			"restrict_to_marketplace_only": false,
		},
		"sso_enforced_for_signin":          false,
		"sso_enforced_for_signin_protocol": "",
		"sso_enforced_for_web":             false,
		"sso_enforced_for_web_protocol":    "",
		"enable_marketplace":               false,
		"enable_change_email":              false,
		"enable_email_code_login":          false,
		"enable_email_password_login":      true,
		"enable_social_oauth_login":        false,
		"enable_collaboration_mode":        false,
		"is_allow_create_workspace":        false,
		"is_allow_register":                false,
		"is_email_setup":                   false,
		"license": map[string]any{
			"status":     "none",
			"expired_at": "",
		},
		"branding": map[string]any{
			"enabled":           false,
			"login_page_logo":   "",
			"workspace_logo":    "",
			"favicon":           "",
			"application_title": s.cfg.AppTitle,
		},
		"webapp_auth": map[string]any{
			"enabled":                    false,
			"allow_sso":                  false,
			"sso_config":                 map[string]any{"protocol": ""},
			"allow_email_code_login":     false,
			"allow_email_password_login": true,
		},
		"enable_trial_app":      false,
		"enable_explore_banner": false,
	}
}

func (s *server) initValidated(r *http.Request) bool {
	return readCookie(r, initCookie) == "1"
}

func (s *server) setAuthCookies(w http.ResponseWriter, session sessionTokens) {
	http.SetCookie(w, s.cookie(accessTokenCookie, session.AccessToken, s.cfg.AccessTokenTTL, true))
	http.SetCookie(w, s.cookie(refreshTokenCookie, session.RefreshToken, s.cfg.RefreshTokenTTL, true))
	http.SetCookie(w, s.cookie(csrfTokenCookie, session.CSRFToken, s.cfg.AccessTokenTTL, false))
}

func (s *server) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, s.expiredCookie(accessTokenCookie, true))
	http.SetCookie(w, s.expiredCookie(refreshTokenCookie, true))
	http.SetCookie(w, s.expiredCookie(csrfTokenCookie, false))
}

func (s *server) cookie(name, value string, ttl time.Duration, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		HttpOnly: httpOnly,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	}
}

func (s *server) expiredCookie(name string, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		HttpOnly: httpOnly,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}

func currentUser(r *http.Request) state.User {
	user, _ := r.Context().Value(userContextKey).(state.User)
	return user
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
		"status":  status,
	})
}

func decodeMaybeBase64(value string) string {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(decoded)
}

func readCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}
