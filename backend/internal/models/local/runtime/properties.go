package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

func (s *Server) properties(ctx context.Context) (properties, error) {
	endpoint, key, _, err := s.Connection()
	if err != nil {
		return properties{}, err
	}
	request, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/props", nil)
	if err != nil {
		return properties{}, err
	}
	request.Header.Set("Authorization", "Bearer "+key)
	response, err := (&http.Client{Timeout: 2 * time.Second}).Do(request)
	if err != nil {
		return properties{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return properties{}, errors.New("Model is still loading")
	}
	var p properties
	err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&p)
	return p, err
}
