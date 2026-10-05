package chatcompletions

import (
	provider "arx/internal/inference"

	model "arx/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type WireProfile uint8

const (
	LocalWire WireProfile = iota
	NetworkWire
)

type Client struct {
	Profile     WireProfile
	RemoteModel string
	URL         string
	Key         string
	Info        model.Info
	ReadImage   func(string) ([]byte, error)
}

func (c *Client) Complete(ctx context.Context, input provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	messages, err := c.messages(input)
	if err != nil {
		return provider.Response{}, err
	}
	body := map[string]any{"model": "arx-local", "messages": messages, "stream": true, "stream_options": map[string]bool{"include_usage": true}, "max_tokens": input.MaxOutputTokens, "cache_prompt": true}
	if c.Profile == NetworkWire {
		body["model"] = c.RemoteModel
		delete(body, "cache_prompt")
		if c.Info.Thinking != nil && input.Effort != "" {
			effort := input.Effort
			if effort == "on" {
				effort = "medium"
			}
			body["reasoning_effort"] = effort
		}
	}
	if len(input.Tools) > 0 {
		if c.Info.Tools == nil || !*c.Info.Tools {
			return provider.Response{}, errors.New("This model’s chat template does not support Arx tools")
		}
		tools := make([]any, 0, len(input.Tools))
		for _, definition := range input.Tools {
			tools = append(tools, map[string]any{"type": "function", "function": definition})
		}
		body["tools"] = tools
	}
	if c.Profile == LocalWire && c.Info.Thinking != nil && c.Info.Thinking.CanDisable {
		body["chat_template_kwargs"] = map[string]any{"enable_thinking": input.Effort != "none"}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return provider.Response{}, err
	}
	if len(data) > 48<<20 {
		return provider.Response{}, provider.ErrContextLength
	}
	request, err := http.NewRequestWithContext(ctx, "POST", c.URL+"/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return provider.Response{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		request.Header.Set("Authorization", "Bearer "+c.Key)
	}
	response, err := provider.NewHTTPClient(15 * time.Minute).Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return provider.Response{}, ctx.Err()
		}
		if c.Profile == NetworkWire {
			return provider.Response{}, errors.New("Could not reach the network model. Check the server connection")
		}
		return provider.Response{}, errors.New("Could not reach the local model. Reload it in Settings")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 32<<10))
		lower := strings.ToLower(string(payload))
		if response.StatusCode == 400 && (strings.Contains(lower, "context") || strings.Contains(lower, "exceed_context")) {
			return provider.Response{}, provider.ErrContextLength
		}
		if c.Profile == NetworkWire {
			return provider.Response{}, errors.New("The network model rejected this request. Check its server configuration")
		}
		return provider.Response{}, errors.New("The local model rejected this request. Check its tool and image support")
	}
	result, err := ReadDetailed(response.Body, emit)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}
