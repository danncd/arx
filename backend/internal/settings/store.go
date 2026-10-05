package settings

import (
	permission "arx/internal/permissions"
	environment "arx/internal/platform/paths"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	mutex     sync.Mutex
	directory string
	values    Values
	views     Views
}

func Open(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	store := &Store{directory: directory, values: Values{Version: 1, AutoContinue: true}, views: Views{Version: 1, Views: map[string]string{}}}
	for name, target := range map[string]any{"settings.json": &store.values, "views.json": &store.views} {
		if err := readJSON(filepath.Join(directory, name), target); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("Could not read %s: %w", name, err)
		}
	}
	if store.values.Version != 1 || store.views.Version != 1 {
		return nil, errors.New("Unsupported settings version")
	}
	if store.values.LocalIdleMinutes == 0 {
		store.values.LocalIdleMinutes = 5
	}
	if !validLocalIdleMinutes(store.values.LocalIdleMinutes) {
		return nil, errors.New("Invalid local model idle timeout")
	}
	if store.values.Directory == "" {
		workingDirectory, err := environment.DefaultDirectory()
		if err != nil {
			return nil, err
		}
		store.values.Directory = workingDirectory
	}
	if store.values.Permissions.Mode == "" {
		store.values.Permissions = permission.Default(store.values.Directory)
	}
	if store.values.Permissions.Mode != permission.Ask && store.values.Permissions.Mode != permission.Folders && store.values.Permissions.Mode != permission.Full {
		return nil, errors.New("Invalid permission settings")
	}
	for _, root := range store.values.Permissions.Roots {
		if !filepath.IsAbs(root) {
			return nil, errors.New("Invalid permission folder")
		}
	}
	if store.values.Permissions.Roots == nil {
		store.values.Permissions.Roots = []string{}
	}
	if store.views.Views == nil {
		store.views.Views = map[string]string{}
	}
	return store, nil
}

func (s *Store) Values() Values {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	values := s.values
	values.ChatPermissions = copyPolicies(s.values.ChatPermissions)
	values.Permissions.Roots = append([]string{}, values.Permissions.Roots...)
	return values
}

func (s *Store) Configure(run Run, directory string, autoContinue ...*bool) (Values, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if run.Provider != "" && run.Provider != "deepseek" && run.Provider != "local" && run.Provider != "network" {
		return s.values, errors.New("Unknown model provider")
	}
	if len(run.Model) > 200 || len(run.Effort) > 80 {
		return s.values, errors.New("Invalid model settings")
	}
	next := s.values
	next.Run = run
	if len(autoContinue) > 0 && autoContinue[0] != nil {
		next.AutoContinue = *autoContinue[0]
	}
	if directory != next.Directory {
		resolved, err := environment.Directory(directory)
		if err != nil {
			return s.values, err
		}
		next.Directory = resolved
	}
	if err := writeJSON(filepath.Join(s.directory, "settings.json"), next); err != nil {
		return s.values, err
	}
	s.values = next
	return next, nil
}

func (s *Store) ConfigureLocalIdleMinutes(minutes int) (Values, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !validLocalIdleMinutes(minutes) {
		return s.values, errors.New("Choose 5, 10, or 15 minutes")
	}
	next := s.values
	next.LocalIdleMinutes = minutes
	if err := writeJSON(filepath.Join(s.directory, "settings.json"), next); err != nil {
		return s.values, err
	}
	s.values = next
	return next, nil
}

func validLocalIdleMinutes(minutes int) bool {
	return minutes == 5 || minutes == 10 || minutes == 15
}

func (s *Store) Views() map[string]string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return copyViews(s.views.Views)
}

func (s *Store) SaveView(key, value string) error {
	return s.SaveViews(map[string]string{key: value})
}

func (s *Store) SaveViews(values map[string]string) error {
	if len(values) > 1000 {
		return errors.New("Too many view preferences")
	}
	for key, value := range values {
		if strings.TrimSpace(key) == "" || len(key) > 200 || len(value) > 256<<10 {
			return errors.New("Invalid view preference")
		}
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	next := Views{Version: 1, Views: copyViews(s.views.Views)}
	for key, value := range values {
		if value == "" {
			delete(next.Views, key)
		} else {
			next.Views[key] = value
		}
	}
	if err := writeJSON(filepath.Join(s.directory, "views.json"), next); err != nil {
		return err
	}
	s.views = next
	return nil
}

func copyViews(views map[string]string) map[string]string {
	copy := make(map[string]string, len(views))
	for key, value := range views {
		copy[key] = value
	}
	return copy
}
