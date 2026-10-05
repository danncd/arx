package stdio

import (
	protocol "arx/internal/transport/contract"
	"context"
	"errors"
)

func (s *Server) dispatch(ctx context.Context, request protocol.Request) (protocol.Response, bool) {
	response := protocol.Response{ID: request.ID}
	var result any
	var err error
	descriptor, known := protocol.Lookup(request.Method)
	if !known {
		return protocol.Response{ID: request.ID, Error: &protocol.Error{Code: "unknown_method", Message: "Unknown request method"}}, false
	}
	code := descriptor.Error
	switch descriptor.Group {
	case "status":
		status := s.service.Status()
		result = protocol.Status{Version: status.Version, Execution: string(status.Execution)}
	case "state":
		result, err = s.stateRequest(request.Method, request.Params)
	case "connection":
		result, err = s.connectionRequest(ctx, request.Method, request.Params)
	case "tool":
		result, err = s.toolRequest(ctx, request.Method, request.Params)
	case "chat":
		result, err = s.chatRequest(ctx, request.Method, request.Params)
	case "local":
		result, err = s.localRequest(ctx, request.Method, request.Params)
	case "generation":
		result, err = s.generationRequest(ctx, request.Method, request.Params)
	case "network":
		result, err = s.networkRequest(ctx, request.Method, request.Params)
	case "integration":
		result, err = s.integrationRequest(ctx, request.Method, request.Params)
	case "attachment":
		result, err = s.attachmentRequest(request.Method, request.Params)
	case "shutdown":
		response.Result = struct {
			Accepted bool `json:"accepted"`
		}{true}
		return response, true
	default:
		err = errors.New("Unknown request method")
	}
	if err != nil {
		response.Error = &protocol.Error{Code: code, Message: err.Error()}
	} else {
		response.Result = result
	}
	return response, false
}
