package state

import (
	"strings"
	"time"
)

type AuthFlow struct {
	Token     string `json:"token"`
	Kind      string `json:"kind"`
	Email     string `json:"email"`
	UserID    string `json:"user_id"`
	NewEmail  string `json:"new_email"`
	ExpiresAt int64  `json:"expires_at"`
}

type AuthFlowInput struct {
	Kind      string
	Email     string
	UserID    string
	NewEmail  string
	ExpiresAt time.Time
}

func normalizeAuthFlow(flow *AuthFlow) {
	flow.Token = strings.TrimSpace(flow.Token)
	flow.Kind = strings.TrimSpace(flow.Kind)
	flow.Email = strings.ToLower(strings.TrimSpace(flow.Email))
	flow.UserID = strings.TrimSpace(flow.UserID)
	flow.NewEmail = strings.ToLower(strings.TrimSpace(flow.NewEmail))
}

func (s *Store) IssueAuthFlow(input AuthFlowInput, now time.Time) (AuthFlow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneExpiredAuthFlowsLocked(now)
	flow := AuthFlow{
		Token:     generateID("auth"),
		Kind:      strings.TrimSpace(input.Kind),
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		UserID:    strings.TrimSpace(input.UserID),
		NewEmail:  strings.ToLower(strings.TrimSpace(input.NewEmail)),
		ExpiresAt: input.ExpiresAt.UTC().Unix(),
	}
	s.state.AuthFlows = append(s.state.AuthFlows, flow)
	if err := s.saveLocked(); err != nil {
		return AuthFlow{}, err
	}
	return flow, nil
}

func (s *Store) PromoteAuthFlow(token, expectKind, nextKind, newEmail string, expiresAt time.Time, now time.Time) (AuthFlow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token = strings.TrimSpace(token)
	pruned := s.pruneExpiredAuthFlowsLocked(now)
	for i, flow := range s.state.AuthFlows {
		if flow.Token != token || flow.Kind != strings.TrimSpace(expectKind) {
			continue
		}

		next := AuthFlow{
			Token:     generateID("auth"),
			Kind:      strings.TrimSpace(nextKind),
			Email:     flow.Email,
			UserID:    flow.UserID,
			NewEmail:  firstNonEmpty(strings.ToLower(strings.TrimSpace(newEmail)), flow.NewEmail),
			ExpiresAt: expiresAt.UTC().Unix(),
		}
		s.state.AuthFlows = append(s.state.AuthFlows[:i], s.state.AuthFlows[i+1:]...)
		s.state.AuthFlows = append(s.state.AuthFlows, next)
		if err := s.saveLocked(); err != nil {
			return AuthFlow{}, false, err
		}
		return next, true, nil
	}
	if pruned {
		if err := s.saveLocked(); err != nil {
			return AuthFlow{}, false, err
		}
	}
	return AuthFlow{}, false, nil
}

func (s *Store) GetAuthFlow(token, expectKind string, now time.Time) (AuthFlow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token = strings.TrimSpace(token)
	if s.pruneExpiredAuthFlowsLocked(now) {
		if err := s.saveLocked(); err != nil {
			return AuthFlow{}, false, err
		}
	}
	for _, flow := range s.state.AuthFlows {
		if flow.Token == token && flow.Kind == strings.TrimSpace(expectKind) {
			return flow, true, nil
		}
	}
	return AuthFlow{}, false, nil
}

func (s *Store) ConsumeAuthFlow(token, expectKind string, now time.Time) (AuthFlow, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token = strings.TrimSpace(token)
	pruned := s.pruneExpiredAuthFlowsLocked(now)
	for i, flow := range s.state.AuthFlows {
		if flow.Token != token || flow.Kind != strings.TrimSpace(expectKind) {
			continue
		}
		s.state.AuthFlows = append(s.state.AuthFlows[:i], s.state.AuthFlows[i+1:]...)
		if err := s.saveLocked(); err != nil {
			return AuthFlow{}, false, err
		}
		return flow, true, nil
	}
	if pruned {
		if err := s.saveLocked(); err != nil {
			return AuthFlow{}, false, err
		}
	}
	return AuthFlow{}, false, nil
}

func (s *Store) pruneExpiredAuthFlowsLocked(now time.Time) bool {
	if len(s.state.AuthFlows) == 0 {
		return false
	}

	cutoff := now.UTC().Unix()
	kept := s.state.AuthFlows[:0]
	changed := false
	for _, flow := range s.state.AuthFlows {
		if flow.ExpiresAt > 0 && flow.ExpiresAt <= cutoff {
			changed = true
			continue
		}
		kept = append(kept, flow)
	}
	s.state.AuthFlows = kept
	return changed
}
