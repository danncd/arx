package settings

import (
	permission "arx/internal/permissions"
	"errors"
	"path/filepath"
	"strings"
)

func copyPolicies(policies map[string]permission.Policy) map[string]permission.Policy {
	result := make(map[string]permission.Policy, len(policies))
	for id, policy := range policies {
		policy.Roots = append([]string{}, policy.Roots...)
		result[id] = policy
	}
	return result
}

func (s *Store) ChatPermissions(conversation string) permission.Policy {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	policy, exists := s.values.ChatPermissions[conversation]
	if !exists {
		return permission.Default(s.values.Directory)
	}
	policy.Roots = append([]string{}, policy.Roots...)
	return policy
}

func (s *Store) SaveChatPermissions(conversation string, policy permission.Policy) (Values, error) {
	if strings.TrimSpace(conversation) == "" || len(conversation) > 128 {
		return Values{}, errors.New("A chat is required")
	}
	normalized, err := permission.Normalize(policy)
	if err != nil {
		return Values{}, err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	next := s.values
	next.ChatPermissions = copyPolicies(next.ChatPermissions)
	next.ChatPermissions[conversation] = normalized
	if err := writeJSON(filepath.Join(s.directory, "settings.json"), next); err != nil {
		return Values{}, err
	}
	s.values = next
	next.ChatPermissions = copyPolicies(next.ChatPermissions)
	return next, nil
}
