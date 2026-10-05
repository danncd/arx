package library

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var modelID = regexp.MustCompile(`^local:[a-f0-9]{32}$`)

func ManagedDirectory(directory, id string) (string, error) {
	if !modelID.MatchString(id) {
		return "", errors.New("Invalid local model ID")
	}
	target := filepath.Join(directory, "models", strings.TrimPrefix(id, "local:"))
	return target, checkPath(directory, target)
}

func ManagedPath(directory, id, name string) (string, error) {
	root, err := ManagedDirectory(directory, id)
	if err != nil {
		return "", err
	}
	if !filepath.IsLocal(name) || filepath.Clean(name) != name {
		return "", errors.New("Invalid model file path")
	}
	target := filepath.Join(root, name)
	return target, checkPath(directory, target)
}

func checkPath(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || !filepath.IsLocal(relative) {
		return errors.New("Model file is outside its managed directory")
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("Managed model paths cannot contain symbolic links")
		}
	}
	return nil
}

func ValidateEntry(directory string, entry Entry) error {
	if !modelID.MatchString(entry.ID) || !filepath.IsAbs(entry.Path) || entry.Size < 0 || entry.Received < 0 {
		return errors.New("Invalid saved model entry")
	}
	if entry.Imported {
		if entry.Projector != "" && !filepath.IsAbs(entry.Projector) {
			return errors.New("Invalid projector path")
		}
		return nil
	}
	root, err := ManagedDirectory(directory, entry.ID)
	if err != nil {
		return err
	}
	for _, target := range []string{entry.Path, entry.Projector} {
		if target == "" {
			continue
		}
		if err := checkPath(root, target); err != nil {
			return err
		}
	}
	for _, file := range entry.Files {
		if _, err := ManagedPath(directory, entry.ID, file.Name); err != nil {
			return err
		}
		if file.Size < 0 {
			return errors.New("Invalid saved model size")
		}
	}
	return nil
}
