package stdio

import (
	"arx/internal/models/local/catalog"
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) localRequest(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	if method == "local.configure" {
		var params protocol.LocalConfigureParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return s.service.ConfigureLocalIdleMinutes(params.IdleMinutes)
	}
	manager, err := s.service.LocalModels()
	if err != nil {
		return nil, err
	}
	if method == "local.state" {
		return manager.Snapshot(), nil
	}
	switch method {
	case "local.search":
		var params catalog.SearchOptions
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return manager.Search(ctx, params)
	case "local.repository":
		var params protocol.LocalRepositoryParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return manager.Repository(ctx, params.ID)
	case "local.download":
		var params protocol.LocalDownloadParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		id, err := manager.Download(ctx, params.Repository, params.Variant)
		return map[string]string{"id": id}, err
	case "local.import":
		var params protocol.LocalImportParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		id, err := manager.Import(params.Path, params.Projector)
		return map[string]string{"id": id}, err
	case "local.action":
		var params protocol.LocalActionParams
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		err := s.service.LocalAction(params.Action, params.ID, params.DeleteFiles)
		return manager.Snapshot(), err
	}
	return nil, errors.New("Unknown local model request")
}
