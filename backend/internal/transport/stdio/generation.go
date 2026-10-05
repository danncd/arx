package stdio

import (
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"errors"
)

func (s *Server) generationRequest(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	manager, err := s.service.Generation()
	if err != nil {
		return nil, err
	}
	switch method {
	case "generation.state":
		return map[string]any{"library": manager.Models.Snapshot(), "jobs": manager.Jobs.Snapshot(), "catalog": manager.Models.Catalog()}, nil
	case "generation.configure":
		var input protocol.GenerationConfigureParams
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		if err := manager.Models.Configure(input.Category, input.ID); err != nil {
			return nil, err
		}
		return manager.Models.Snapshot(), nil
	case "generation.action":
		var input protocol.GenerationActionParams
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		switch input.Action {
		case "download", "resume":
			err = manager.Models.Download(input.ID)
		case "pause":
			manager.Models.Pause()
		case "remove":
			if s.service.ChatState().Run.State != "idle" {
				return nil, errors.New("Stop the current reply before removing a model")
			}
			for _, job := range manager.Jobs.Snapshot() {
				if job.State == "queued" || job.State == "running" {
					return nil, errors.New("Stop generation before removing a model")
				}
			}
			err = manager.Models.Remove(input.ID)
		case "cancel":
			return manager.Jobs.Cancel(input.ID), nil
		default:
			return nil, errors.New("Unknown generation action")
		}
		if err != nil {
			return nil, err
		}
		return manager.Models.Snapshot(), nil
	case "media.resolve":
		var input protocol.MediaResolveParams
		if err := decodeParams(raw, &input); err != nil {
			return nil, err
		}
		artifact, path, err := manager.Artifacts.Resolve(input.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifact": artifact, "path": path}, nil
	}
	return nil, errors.New("Unknown generation request")
}
