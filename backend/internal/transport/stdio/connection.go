package stdio

import (
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) connectionRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "deepseek.status":
		return s.service.DeepSeekStatus(ctx, false)
	case "deepseek.refresh":
		return s.service.DeepSeekStatus(ctx, true)
	case "deepseek.disconnect":
		return s.service.DisconnectDeepSeek(ctx)
	case "deepseek.connect":
		var p protocol.DeepseekConnectParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return s.service.ConnectDeepSeek(ctx, p.Key)
	}
	return nil, errors.New("Unknown connection request")
}
