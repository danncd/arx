package settings

import permission "arx/internal/permissions"

type Run struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model"`
	Effort   string `json:"effort"`
}

type Values struct {
	AutoContinue     bool                         `json:"auto_continue"`
	ChatPermissions  map[string]permission.Policy `json:"chat_permissions,omitempty"`
	Permissions      permission.Policy            `json:"permissions"`
	Version          int                          `json:"version"`
	Run              Run                          `json:"run"`
	Directory        string                       `json:"directory,omitempty"`
	LocalIdleMinutes int                          `json:"local_idle_minutes"`
}

type Views struct {
	Version int               `json:"version"`
	Views   map[string]string `json:"views"`
}
