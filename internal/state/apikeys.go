package state

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"time"
)

type APIKey struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	ResourceID string `json:"resource_id"`
	Token      string `json:"token"`
	LastUsedAt *int64 `json:"last_used_at,omitempty"`
	CreatedAt  int64  `json:"created_at"`
}

const (
	apiKeyTypeApp = "app"
	maxAppAPIKeys = 10
)

func (s *Store) ListAppAPIKeys(appID, workspaceID string) ([]APIKey, error) {
	if _, ok := s.GetApp(appID, workspaceID); !ok {
		return nil, fmt.Errorf("app not found")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]APIKey, 0, len(s.state.APIKeys))
	for _, key := range s.state.APIKeys {
		if key.Type != apiKeyTypeApp || key.ResourceID != appID {
			continue
		}
		keys = append(keys, key)
	}

	slices.SortFunc(keys, func(a, b APIKey) int {
		if a.CreatedAt == b.CreatedAt {
			return bcmp(a.ID, b.ID)
		}
		if a.CreatedAt > b.CreatedAt {
			return -1
		}
		return 1
	})

	return keys, nil
}

func (s *Store) CreateAppAPIKey(appID, workspaceID string, now time.Time) (APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findAppIndexLocked(appID, workspaceID) < 0 {
		return APIKey{}, fmt.Errorf("app not found")
	}

	count := 0
	for _, key := range s.state.APIKeys {
		if key.Type == apiKeyTypeApp && key.ResourceID == appID {
			count++
		}
	}
	if count >= maxAppAPIKeys {
		return APIKey{}, fmt.Errorf("cannot create more than %d API keys for this app", maxAppAPIKeys)
	}

	key := APIKey{
		ID:         generateID("key"),
		Type:       apiKeyTypeApp,
		ResourceID: appID,
		Token:      generateAPIKeyToken("app-"),
		CreatedAt:  now.UTC().Unix(),
	}

	s.state.APIKeys = append(s.state.APIKeys, key)
	if err := s.saveLocked(); err != nil {
		return APIKey{}, err
	}
	return key, nil
}

func (s *Store) DeleteAppAPIKey(appID, workspaceID, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.findAppIndexLocked(appID, workspaceID) < 0 {
		return fmt.Errorf("app not found")
	}

	for i, key := range s.state.APIKeys {
		if key.Type == apiKeyTypeApp && key.ResourceID == appID && key.ID == keyID {
			s.state.APIKeys = append(s.state.APIKeys[:i], s.state.APIKeys[i+1:]...)
			return s.saveLocked()
		}
	}

	return fmt.Errorf("api key not found")
}

func generateAPIKeyToken(prefix string) string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("generate api key: %w", err))
	}
	return prefix + hex.EncodeToString(buf)
}

func bcmp(a, b string) int {
	switch {
	case a > b:
		return -1
	case a < b:
		return 1
	default:
		return 0
	}
}
