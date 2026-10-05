package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func DefaultDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	directory := filepath.Join(home, "Documents", "Arx")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	return directory, nil
}

func Directory(path string) (string, error) {
	if path == "" {
		return "", errors.New("Choose a working directory")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("Working directory must be an absolute path")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("Working directory is unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("Choose an existing directory")
	}
	return resolved, nil
}

func Resolve(directory, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("A path is required")
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	root, err := Directory(directory)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, path), nil
}
