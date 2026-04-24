package server

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *server) handleConsoleSSOLogin(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var userID string

	inviteToken := strings.TrimSpace(r.URL.Query().Get("invite_token"))
	if inviteToken != "" {
		user, _, err := s.store.ActivateWorkspaceInvitation(inviteToken, "", "en-US", "UTC", now)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_invitation", "The invitation token is invalid.")
			return
		}
		userID = user.ID
	} else {
		user, ok := s.store.PrimaryUser()
		if !ok {
			writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
			return
		}
		userID = user.ID
	}

	if _, err := s.issueAuthSession(w, userID, now); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to complete SSO sign in.")
		return
	}

	writeJSON(w, http.StatusOK, s.ssoLoginResponse("/apps", "", r.URL.Query().Get("redirect_url")))
}

func (s *server) handlePublicWebSSOLogin(w http.ResponseWriter, r *http.Request) {
	appCode := strings.TrimSpace(r.URL.Query().Get("app_code"))
	if appCode == "" {
		appCode = strings.TrimSpace(r.URL.Query().Get("appCode"))
	}
	if appCode == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "App code is required.")
		return
	}
	if _, ok := s.store.FindAppBySiteAccessToken(appCode); !ok {
		writeError(w, http.StatusNotFound, "app_not_found", "App not found.")
		return
	}

	redirectURL := strings.TrimSpace(r.URL.Query().Get("redirect_url"))
	if redirectURL == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Redirect URL is required.")
		return
	}

	user, ok := s.store.PrimaryUser()
	if !ok {
		writeError(w, http.StatusUnauthorized, "not_setup", "Dify Go has not been initialized yet.")
		return
	}

	session, err := s.issueAuthSession(w, user.ID, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to complete webapp SSO sign in.")
		return
	}

	writeJSON(w, http.StatusOK, s.ssoLoginResponse(redirectURL, session.AccessToken, redirectURL))
}

func (s *server) ssoLoginResponse(rawURL, accessToken, stateSeed string) map[string]any {
	redirectURL := appendQueryParam(rawURL, "web_sso_token", accessToken)
	return map[string]any{
		"url":   redirectURL,
		"state": generateRuntimeID("sso_" + normalizeSSOStatePrefix(stateSeed)),
	}
}

func appendQueryParam(rawURL, key, value string) string {
	if strings.TrimSpace(rawURL) == "" || strings.TrimSpace(value) == "" {
		return rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		separator := "?"
		if strings.Contains(rawURL, "?") {
			separator = "&"
		}
		return rawURL + separator + url.QueryEscape(key) + "=" + url.QueryEscape(value)
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func normalizeSSOStatePrefix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "state"
	}
	builder := strings.Builder{}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
		if builder.Len() >= 24 {
			break
		}
	}
	if builder.Len() == 0 {
		return "state"
	}
	return builder.String()
}
