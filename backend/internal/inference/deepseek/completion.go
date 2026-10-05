package deepseek

import (
	provider "arx/internal/inference"
	chatcompletions "arx/internal/inference/chatcompletions"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *Client) Complete(ctx context.Context, key string, input provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	messages, err := c.messages(input)
	if err != nil {
		return provider.Response{}, err
	}
	tools := make([]any, 0, len(input.Tools))
	for _, definition := range input.Tools {
		tools = append(tools, map[string]any{"type": "function", "function": definition})
	}
	body := map[string]any{
		"model":          input.Model,
		"messages":       messages,
		"stream":         true,
		"stream_options": map[string]bool{"include_usage": true},
		"tools":          tools,
	}
	if input.MaxOutputTokens > 0 {
		body["max_tokens"] = input.MaxOutputTokens
	}
	if len(tools) == 0 {
		delete(body, "tools")
	}
	if input.Effort == "none" {
		body["thinking"] = map[string]string{"type": "disabled"}
	} else if input.Effort != "" {
		body["thinking"] = map[string]string{"type": "enabled"}
		body["reasoning_effort"] = input.Effort
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return provider.Response{}, errors.New("Could not prepare reply")
	}
	if len(encoded) > 48<<20 {
		return provider.Response{}, provider.ErrContextLength
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return provider.Response{}, err
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	client := *c.client
	client.Timeout = 10 * time.Minute
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return provider.Response{}, ctx.Err()
		}
		return provider.Response{}, errors.New("Couldn’t reach DeepSeek. Try again")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == 400 {
			var payload struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if json.NewDecoder(io.LimitReader(response.Body, 32<<10)).Decode(&payload) == nil {
				message := strings.ToLower(payload.Error.Message)
				if payload.Error.Code == "context_length_exceeded" || strings.Contains(message, "maximum context length") || strings.Contains(message, "context window exceeded") {
					return provider.Response{}, provider.ErrContextLength
				}
			}
			return provider.Response{}, errors.New("DeepSeek rejected this conversation or its settings")
		}
		return provider.Response{}, responseError(response.StatusCode)
	}
	result, err := chatcompletions.Read(response.Body, emit)
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}
