package stdio

import (
	protocol "arx/internal/transport/contract"
	"encoding/base64"
	"encoding/json"
	"errors"
)

func (s *Server) attachmentRequest(method string, params json.RawMessage) (any, error) {
	switch method {
	case "attachments.import":
		var input protocol.AttachmentsImportParams
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		return s.service.Service.Images.Import(input.Path, input.Name)
	case "attachments.read":
		var input protocol.AttachmentsReadParams
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		data, err := s.service.Service.Images.Read(input.ID)
		if err != nil {
			return nil, err
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data), nil
	}
	return nil, errors.New("Unknown image request")
}
