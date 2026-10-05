package mcp

import (
	"arx/internal/integrations"
	"encoding/json"
	"errors"
	"strings"
)

type Configuration struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Arguments   []string          `json:"arguments"`
	Environment map[string]string `json:"environment"`
	Enabled     bool              `json:"enabled"`
	Policy      string            `json:"policy"`

	ReadOnlyTools []string `json:"readOnlyTools"`
}
type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	ReadOnlyHint bool            `json:"readOnlyHint"`
	ReadOnly     bool            `json:"readOnly"`
	Calls        int             `json:"calls"`
}
type Server struct {
	Configuration
	Usage  integrations.Usage `json:"usage"`
	Tools  []Tool             `json:"tools"`
	Status string             `json:"status"`
	Error  string             `json:"error,omitempty"`
}

func Validate(c Configuration) error {
	if !integrations.Identifier.MatchString(c.ID) {
		return errors.New("Use a server ID with lowercase letters, numbers, hyphens, or underscores")
	}
	if strings.TrimSpace(c.Name) == "" || len(c.Name) > 100 {
		return errors.New("Enter a server name up to 100 characters")
	}
	if strings.TrimSpace(c.Command) == "" || len(c.Command) > 4096 || strings.ContainsRune(c.Command, 0) {
		return errors.New("Enter a valid executable")
	}
	if c.Policy != "ask" && c.Policy != "changes" {
		return errors.New("Choose a call permission policy")
	}
	if len(c.Arguments) > 128 || len(c.Environment) > 100 || len(c.ReadOnlyTools) > 1000 {
		return errors.New("Server configuration is too large")
	}
	for _, a := range c.Arguments {
		if len(a) > 8192 || strings.ContainsRune(a, 0) {
			return errors.New("Invalid executable argument")
		}
	}
	for k, v := range c.Environment {
		if k == "" || strings.ContainsAny(k, "=\x00") || strings.ContainsRune(v, 0) || len(k)+len(v) > 16384 {
			return errors.New("Invalid environment variable")
		}
	}
	for _, n := range c.ReadOnlyTools {
		if n == "" || len(n) > 200 {
			return errors.New("Invalid read-only tool name")
		}
	}
	return nil
}
