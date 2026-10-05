package stdio

import (
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) networkRequest(ctx context.Context, method string, params json.RawMessage) (any, error) {
	manager, err := s.service.NetworkModels()
	if err != nil {
		return nil, err
	}
	switch method {
	case "network.state":
		return manager.State(ctx), nil
	case "network.scan":
		return manager.Scan(ctx)
	case "network.cancel":
		manager.CancelScan()
		return true, nil
	case "network.connect":
		var p protocol.NetworkConnectParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return manager.Connect(ctx, p.URL, p.Name, p.Token)
	case "network.remove":
		var p protocol.NetworkRemoveParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return true, manager.Remove(ctx, p.ID)
	}
	return nil, errors.New("Unknown network request")
}
