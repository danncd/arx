package skills

import (
	"arx/internal/permissions"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func root(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("Choose an absolute skill folder")
	}
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return "", errors.New("Skill folder is unavailable")
	}
	return p, nil
}
func readFile(directory, path string) (string, error) {
	if filepath.IsAbs(path) || path == "" {
		return "", errors.New("Choose a file relative to the skill folder")
	}
	if !permissions.Contains(directory, filepath.Join(directory, path)) {
		return "", errors.New("Skill reference is outside its folder")
	}
	boundary, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer boundary.Close()
	f, err := boundary.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("Choose a regular skill file")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxManifest+1))
	if err != nil {
		return "", err
	}
	if len(data) > MaxManifest {
		return "", errors.New("Skill reference exceeds 64 KiB")
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return "", errors.New("Skill reference is not a text file")
	}
	return string(data), nil
}
func files(directory string) ([]string, error) {
	out := []string{}
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == directory {
			return nil
		}
		if len(out) >= 200 {
			return filepath.SkipAll
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() {
			rel, _ := filepath.Rel(directory, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}
