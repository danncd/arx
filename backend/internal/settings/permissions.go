package settings

import (
	permission "arx/internal/permissions"
	"path/filepath"
)

func (s *Store) SavePermissions(policy permission.Policy) (Values, error) {
	normalized, err := permission.Normalize(policy)
	if err != nil {
		return s.Values(), err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	next := s.values
	next.Permissions = normalized
	if err := writeJSON(filepath.Join(s.directory, "settings.json"), next); err != nil {
		return s.values, err
	}
	s.values = next
	return next, nil
}
