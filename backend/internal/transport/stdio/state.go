package stdio

import (
	protocol "arx/internal/transport/contract"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func decodeParams(raw json.RawMessage, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return errors.New("Invalid request parameters")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("Invalid request parameters")
	}
	return nil
}

func (s *Server) stateRequest(method string, params json.RawMessage) (any, error) {
	switch method {
	case "snapshot":
		return s.service.Snapshot(), nil
	case "history":
		var p protocol.HistoryParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.Conversation == "" || len(p.Conversation) > 200 {
			return nil, errors.New("A conversation is required")
		}
		return s.service.History(p.Conversation, p.Before, p.Limit)
	case "configure":
		var p protocol.ConfigureParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return s.service.Configure(p.Run, p.Directory, p.AutoContinue)
	case "save_views":
		var p protocol.SaveViewsParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return struct{}{}, s.service.SaveViews(p.Views)
	case "save_view":
		var p protocol.SaveViewParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		return struct{}{}, s.service.SaveView(p.Key, p.Value)
	}
	return nil, errors.New("Unknown request method")
}
