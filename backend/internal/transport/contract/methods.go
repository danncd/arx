package contract

import (
	"arx/internal/app"
	"arx/internal/chat"
	generation "arx/internal/generation/models"
	"arx/internal/inference/deepseek"
	"arx/internal/integrations/mcp"
	"arx/internal/integrations/skills"
	"arx/internal/media/attachments"
	"arx/internal/models/local"
	"arx/internal/models/local/catalog"
	"arx/internal/models/network"
	"arx/internal/settings"
	"arx/internal/tools"
	"reflect"
)

type Method struct {
	Name, Group, Error string
	Renderer, Async    bool
	Timeout            int
	Params, Result     reflect.Type
	Alternatives       []reflect.Type
}

func method[P, R any](name, group, failure string, renderer, async bool, timeout int) Method {
	return Method{Name: name, Group: group, Error: failure, Renderer: renderer, Async: async, Timeout: timeout, Params: reflect.TypeFor[P](), Result: reflect.TypeFor[R]()}
}

func union[T any](method Method) Method {
	method.Alternatives = append(method.Alternatives, reflect.TypeFor[T]())
	return method
}

var Methods = []Method{
	method[AttachmentsImportParams, attachments.Image]("attachments.import", "attachment", "attachment_failed", false, false, 5000),
	method[AttachmentsReadParams, string]("attachments.read", "attachment", "attachment_failed", false, false, 5000),
	method[ChatContextParams, chat.ContextReport]("chat.context", "chat", "chat_failed", true, false, 5000),
	method[chat.Send, chat.Run]("chat.send", "chat", "chat_failed", true, false, 0),
	method[NoParams, chat.ChatEvent]("chat.state", "chat", "chat_failed", true, false, 5000),
	method[ChatStopParams, Accepted]("chat.stop", "chat", "chat_failed", true, false, 5000),
	method[ConfigureParams, settings.Values]("configure", "state", "request_failed", true, false, 5000),
	method[DeepseekConnectParams, deepseek.Connection]("deepseek.connect", "connection", "connection_failed", true, true, 60000),
	method[NoParams, deepseek.Connection]("deepseek.disconnect", "connection", "connection_failed", true, true, 60000),
	method[NoParams, deepseek.Connection]("deepseek.refresh", "connection", "connection_failed", true, true, 60000),
	method[NoParams, deepseek.Connection]("deepseek.status", "connection", "connection_failed", true, true, 60000),
	union[bool](method[GenerationActionParams, generation.State]("generation.action", "generation", "generation_failed", true, true, 5000)),
	method[GenerationConfigureParams, generation.State]("generation.configure", "generation", "generation_failed", true, true, 5000),
	method[NoParams, GenerationState]("generation.state", "generation", "generation_failed", true, true, 5000),
	method[HistoryParams, app.History]("history", "state", "request_failed", true, false, 5000),
	method[LocalActionParams, local.State]("local.action", "local", "local_model_failed", true, true, 60000),
	method[LocalConfigureParams, settings.Values]("local.configure", "local", "local_model_failed", true, true, 60000),
	method[LocalDownloadParams, IDResult]("local.download", "local", "local_model_failed", true, true, 60000),
	method[LocalImportParams, IDResult]("local.import", "local", "local_model_failed", true, true, 60000),
	method[LocalRepositoryParams, catalog.Repository]("local.repository", "local", "local_model_failed", true, true, 60000),
	method[catalog.SearchOptions, catalog.SearchPage]("local.search", "local", "local_model_failed", true, true, 60000),
	method[NoParams, local.State]("local.state", "local", "local_model_failed", true, true, 60000),
	method[McpRemoveParams, []mcp.Server]("mcp.remove", "integration", "integration_failed", true, true, 5000),
	method[mcp.Configuration, []mcp.Server]("mcp.save", "integration", "integration_failed", true, true, 5000),
	method[NoParams, []mcp.Server]("mcp.state", "integration", "integration_failed", true, true, 5000),
	method[McpTestParams, []mcp.Server]("mcp.test", "integration", "integration_failed", true, true, 35000),
	method[MediaResolveParams, ResolvedMedia]("media.resolve", "generation", "generation_failed", false, false, 5000),
	method[NoParams, bool]("network.cancel", "network", "network_failed", true, true, 35000),
	method[NetworkConnectParams, network.Server]("network.connect", "network", "network_failed", true, true, 35000),
	method[NetworkRemoveParams, bool]("network.remove", "network", "network_failed", true, true, 35000),
	method[NoParams, []network.Server]("network.scan", "network", "network_failed", true, true, 35000),
	method[NoParams, []network.Server]("network.state", "network", "network_failed", true, true, 35000),
	method[PermissionsConfigureParams, settings.Values]("permissions.configure", "tool", "tool_request_failed", true, false, 5000),
	method[NoParams, Pending]("permissions.pending", "tool", "tool_request_failed", true, false, 5000),
	method[PermissionsRespondParams, struct{}]("permissions.respond", "tool", "tool_request_failed", true, false, 5000),
	method[SaveViewParams, struct{}]("save_view", "state", "request_failed", true, false, 5000),
	method[SaveViewsParams, struct{}]("save_views", "state", "request_failed", false, false, 5000),
	method[NoParams, Accepted]("shutdown", "shutdown", "request_failed", false, false, 5000),
	method[SkillsDetailParams, skills.Detail]("skills.detail", "integration", "integration_failed", true, true, 5000),
	method[SkillsReadParams, string]("skills.read", "integration", "integration_failed", true, true, 5000),
	method[SkillsDetailParams, []skills.Skill]("skills.remove", "integration", "integration_failed", true, true, 5000),
	method[skills.Edit, []skills.Skill]("skills.save", "integration", "integration_failed", true, true, 5000),
	method[NoParams, []skills.Skill]("skills.state", "integration", "integration_failed", true, true, 5000),
	method[SkillsValidateParams, SkillValidation]("skills.validate", "integration", "integration_failed", true, true, 5000),
	method[NoParams, app.Snapshot]("snapshot", "state", "request_failed", true, false, 5000),
	method[NoParams, Status]("status", "status", "request_failed", true, false, 5000),
	method[ToolsCancelParams, Accepted]("tools.cancel", "tool", "tool_request_failed", true, false, 5000),
	method[NoParams, []tools.Definition]("tools.definitions", "tool", "tool_request_failed", true, false, 5000),
	method[ToolsRunParams, tools.Result]("tools.run", "tool", "tool_request_failed", true, true, 0),
	method[WebImageParams, string]("web.icon", "tool", "tool_request_failed", true, true, 10000),
	method[WebImageParams, string]("web.image", "tool", "tool_request_failed", true, true, 20000),
}

func Lookup(name string) (Method, bool) {
	for _, m := range Methods {
		if m.Name == name {
			return m, true
		}
	}
	return Method{}, false
}
