package stdio

import (
	service "arx/internal/app"
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) chatRequest(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	switch method {
	case "chat.context":
		var input protocol.ChatContextParams
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		return s.service.Context(input.Conversation, input.Model)
	case "chat.state":
		return s.service.ChatState(), nil
	case "chat.send":
		var input service.Send
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		return s.service.Send(ctx, input)
	case "chat.stop":
		var input protocol.ChatStopParams
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		return struct {
			Accepted bool `json:"accepted"`
		}{s.service.StopReply(input.ID)}, nil
	}
	return nil, errors.New("Unknown chat request")
}
