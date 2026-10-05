package permissions

import (
	environment "arx/internal/platform/paths"
	"errors"
	"path/filepath"
	"strings"
)

type Mode string

const (
	Ask     Mode = "ask"
	Folders Mode = "folders"
	Full    Mode = "full"
)

type Policy struct {
	Mode  Mode     `json:"mode"`
	Roots []string `json:"roots"`
}

func Default(directory string) Policy {
	return Policy{Mode: Folders, Roots: []string{directory}}
}

func Normalize(policy Policy) (Policy, error) {
	if policy.Mode == "" {
		policy.Mode = Ask
	}
	if policy.Mode != Ask && policy.Mode != Folders && policy.Mode != Full {
		return Policy{}, errors.New("Unknown permission mode")
	}
	if len(policy.Roots) > 32 {
		return Policy{}, errors.New("Choose up to 32 folders")
	}
	roots := []string{}
	for _, path := range policy.Roots {
		root, err := environment.Directory(path)
		if err != nil {
			return Policy{}, err
		}
		duplicate := false
		for _, held := range roots {
			if held == root {
				duplicate = true
			}
		}
		if !duplicate {
			roots = append(roots, root)
		}
	}
	if policy.Mode == Folders && len(roots) == 0 {
		return Policy{}, errors.New("Choose at least one folder")
	}
	policy.Roots = roots
	return policy, nil
}

func Contains(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func (p Policy) Root(path string) (string, error) {
	for _, root := range p.Roots {
		if Contains(root, path) {
			return root, nil
		}
	}
	return "", errors.New("This path is outside the selected folders")
}
