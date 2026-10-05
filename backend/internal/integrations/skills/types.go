package skills

import (
	"arx/internal/integrations"
	"errors"
	"gopkg.in/yaml.v3"
	"strings"
)

const MaxManifest = 64 << 10

type Skill struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Path         string             `json:"path"`
	Enabled      bool               `json:"enabled"`
	Activation   string             `json:"activation"`
	Dependencies []string           `json:"dependencies"`
	Usage        integrations.Usage `json:"usage"`
	Error        string             `json:"error,omitempty"`
}
type Detail struct {
	Skill
	Instructions string   `json:"instructions"`
	Files        []string `json:"files"`
}
type Edit struct {
	ID           string   `json:"id"`
	Path         string   `json:"path"`
	Instructions string   `json:"instructions"`
	Enabled      bool     `json:"enabled"`
	Activation   string   `json:"activation"`
	Dependencies []string `json:"dependencies"`
}
type manifest struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func Validate(body string) (string, string, error) {
	if len(body) > MaxManifest {
		return "", "", errors.New("SKILL.md exceeds 64 KiB")
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if !strings.HasPrefix(body, "---\n") {
		return "", "", errors.New("SKILL.md needs YAML front matter with name and description")
	}
	_, tail, _ := strings.Cut(body, "---\n")
	header, instructions, ok := strings.Cut(tail, "\n---\n")
	if !ok {
		return "", "", errors.New("Close the YAML front matter with ---")
	}
	var m manifest
	if err := yaml.Unmarshal([]byte(header), &m); err != nil {
		return "", "", errors.New("Invalid SKILL.md YAML front matter")
	}
	if !integrations.Identifier.MatchString(m.Name) {
		return "", "", errors.New("Skill name must use lowercase letters, numbers, hyphens, or underscores")
	}
	if strings.TrimSpace(m.Description) == "" || len(m.Description) > 1024 {
		return "", "", errors.New("Enter a skill description up to 1024 characters")
	}
	if strings.TrimSpace(instructions) == "" {
		return "", "", errors.New("Add instructions after the front matter")
	}
	return m.Name, m.Description, nil
}
