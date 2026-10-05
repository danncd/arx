package contract

import (
	"arx/internal/generation/jobs"
	generation "arx/internal/generation/models"
	mcp "arx/internal/integrations/mcp"
	"arx/internal/media/artifacts"
	permission "arx/internal/permissions"
	settings "arx/internal/settings"
	tool "arx/internal/tools"
)

type GenerationConfigureParams struct {
	Category string `json:"category"`
	ID       string `json:"id"`
}

type GenerationActionParams struct {
	Action string `json:"action"`
	ID     string `json:"id"`
}

type MediaResolveParams struct {
	ID string `json:"id"`
}

type WebImageParams struct {
	URL          string `json:"url"`
	Conversation string `json:"conversation"`
}

type ToolsRunParams struct {
	tool.Call
	Conversation string `json:"conversation,omitempty"`
}

type ToolsCancelParams struct {
	ID string `json:"id"`
}

type PermissionsRespondParams struct {
	ID    string `json:"id"`
	Allow bool   `json:"allow"`
}

type PermissionsConfigureParams struct {
	permission.Policy
	Conversation string `json:"conversation,omitempty"`
}

type NetworkConnectParams struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type NetworkRemoveParams struct {
	ID string `json:"id"`
}

type LocalConfigureParams struct {
	IdleMinutes int `json:"idle_minutes"`
}

type LocalRepositoryParams struct {
	ID string `json:"id"`
}

type LocalDownloadParams struct {
	Repository string `json:"repository"`
	Variant    string `json:"variant"`
}

type LocalImportParams struct {
	Path      string `json:"path"`
	Projector string `json:"projector,omitempty"`
}

type LocalActionParams struct {
	Action      string `json:"action"`
	ID          string `json:"id"`
	DeleteFiles bool   `json:"deleteFiles,omitempty"`
}

type ChatContextParams struct {
	Conversation string `json:"conversation"`
	Model        string `json:"model"`
}

type ChatStopParams struct {
	ID string `json:"id"`
}

type DeepseekConnectParams struct {
	Key string `json:"key"`
}

type AttachmentsImportParams struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type AttachmentsReadParams struct {
	ID string `json:"id"`
}

type HistoryParams struct {
	Conversation string `json:"conversation"`
	Before       string `json:"before,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type ConfigureParams struct {
	Run          settings.Run `json:"run"`
	AutoContinue *bool        `json:"auto_continue"`
	Directory    string       `json:"directory"`
}

type SaveViewsParams struct {
	Views map[string]string `json:"views"`
}

type SaveViewParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type McpTestParams struct {
	ID            string             `json:"id"`
	Configuration *mcp.Configuration `json:"configuration"`
}

type McpRemoveParams struct {
	ID string `json:"id"`
}

type SkillsDetailParams struct {
	ID string `json:"id"`
}

type SkillsReadParams struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type SkillsValidateParams struct {
	Instructions string `json:"instructions"`
}

type NoParams struct{}
type Accepted struct {
	Accepted bool `json:"accepted"`
}
type Pending struct {
	Pending *permission.Request `json:"pending"`
}
type IDResult struct {
	ID string `json:"id"`
}
type SkillValidation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type GenerationState struct {
	Library generation.State   `json:"library"`
	Jobs    []jobs.Job         `json:"jobs"`
	Catalog []generation.Model `json:"catalog"`
}
type ResolvedMedia struct {
	Artifact artifacts.Artifact `json:"artifact"`
	Path     string             `json:"path"`
}
