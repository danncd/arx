package stdio

import (
	"arx/internal/integrations/mcp"
	"arx/internal/integrations/skills"
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) integrationRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "mcp.state":
		return s.service.MCPState(), nil
	case "skills.state":
		return s.service.SkillsState(), nil
	case "mcp.save":
		var in mcp.Configuration
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		return s.service.SaveMCP(in)
	case "mcp.test":
		var in protocol.McpTestParams
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		return s.service.TestMCP(ctx, in.ID, in.Configuration)
	case "mcp.remove":
		var in protocol.McpRemoveParams
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		return s.service.RemoveMCP(in.ID)
	case "skills.save":
		var in skills.Edit
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		return s.service.SaveSkill(in)
	case "skills.detail", "skills.remove":
		var in protocol.SkillsDetailParams
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		if method == "skills.detail" {
			return s.service.SkillDetail(in.ID)
		}
		return s.service.RemoveSkill(in.ID)
	case "skills.read":
		var in protocol.SkillsReadParams
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		return s.service.SkillRead(in.ID, in.Path)
	case "skills.validate":
		var in protocol.SkillsValidateParams
		if err := decodeParams(params, &in); err != nil {
			return nil, err
		}
		name, description, err := skills.Validate(in.Instructions)
		return map[string]string{"name": name, "description": description}, err
	}
	return nil, errors.New("Unknown integration request")
}
