package stdio

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) toolRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "web.image", "web.icon":
		var input protocol.WebImageParams
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		if method == "web.image" {
			return s.service.WebImage(ctx, input.Conversation, input.URL)
		}
		return s.service.WebIcon(ctx, input.Conversation, input.URL)
	case "tools.definitions":
		return tool.Definitions(), nil
	case "tools.run":
		var input protocol.ToolsRunParams
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		return s.service.RunChatTool(ctx, input.Conversation, input.Call), nil
	case "tools.cancel":
		var p protocol.ToolsCancelParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return struct {
			Accepted bool `json:"accepted"`
		}{s.service.CancelTool(p.ID)}, nil
	case "permissions.pending":
		return struct {
			Pending *permission.Request `json:"pending"`
		}{s.service.PendingPermission()}, nil
	case "permissions.respond":
		var p protocol.PermissionsRespondParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return struct{}{}, s.service.RespondPermission(p.ID, p.Allow)
	case "permissions.configure":
		var input protocol.PermissionsConfigureParams
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		if input.Conversation != "" {
			return s.service.ConfigureChatPermissions(input.Conversation, input.Policy)
		}
		return s.service.ConfigurePermissions(input.Policy)
	}
	return nil, errors.New("Unknown tool request")
}
