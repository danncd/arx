package chatcompletions

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type streamChunk struct {
	Error   json.RawMessage `json:"error"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning_content"`
			Calls     []struct {
				Index    int               `json:"index"`
				ID       string            `json:"id"`
				Type     string            `json:"type"`
				Function provider.Function `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		Finish string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		Input         int  `json:"prompt_tokens"`
		Cached        *int `json:"prompt_cache_hit_tokens"`
		PromptDetails struct {
			Cached *int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		Output  int `json:"completion_tokens"`
		Details struct {
			Reasoning int `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	} `json:"usage"`
}

func Read(reader io.Reader, emit func(provider.Delta) error) (provider.Response, error) {
	return read(reader, emit, false)
}

func ReadDetailed(reader io.Reader, emit func(provider.Delta) error) (provider.Response, error) {
	return read(reader, emit, true)
}

func read(reader io.Reader, emit func(provider.Delta) error, detailedErrors bool) (provider.Response, error) {
	result := provider.Response{Message: provider.Message{Role: "assistant"}}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	calls := map[int]*provider.Call{}
	var text, reasoning strings.Builder
	finish, event := "", ""
	total := 0
	consume := func(data string) (bool, error) {
		if data == "[DONE]" {
			return true, nil
		}
		var chunk streamChunk
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return false, errors.New("Model returned an invalid reply")
		}
		if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
			if detailedErrors {
				return false, modelError(chunk.Error)
			}
			return false, errors.New("Model could not complete this reply")
		}
		if chunk.Usage != nil {
			usage := chunk.Usage
			cached := 0
			if usage.Cached != nil {
				cached = *usage.Cached
			}
			if usage.PromptDetails.Cached != nil {
				cached = max(cached, *usage.PromptDetails.Cached)
			}
			result.Usage = &session.ChunkUsage{Input: usage.Input, Cached: cached, CacheUnknown: usage.Cached == nil && usage.PromptDetails.Cached == nil, Output: usage.Output, Reasoning: usage.Details.Reasoning}
		}
		for _, choice := range chunk.Choices {
			if choice.Index != 0 {
				continue
			}
			delta := choice.Delta
			if finish != "" && (delta.Content != "" || delta.Reasoning != "" || len(delta.Calls) > 0) {
				return false, errors.New("Model returned data after the reply ended")
			}
			total += len(delta.Content) + len(delta.Reasoning)
			for _, part := range delta.Calls {
				total += len(part.Function.Arguments)
			}
			if total > 2<<20 {
				return false, errors.New("Reply exceeded the supported size")
			}
			text.WriteString(delta.Content)
			reasoning.WriteString(delta.Reasoning)
			if delta.Content != "" || delta.Reasoning != "" {
				if err := emit(provider.Delta{Text: delta.Content, Reasoning: delta.Reasoning}); err != nil {
					return false, err
				}
			}
			for _, part := range delta.Calls {
				if part.Index < 0 || part.Index >= 32 {
					return false, errors.New("Model returned too many tool calls")
				}
				call := calls[part.Index]
				if call == nil {
					call = &provider.Call{Type: "function"}
					calls[part.Index] = call
				}
				if part.Type != "" && part.Type != "function" {
					return false, errors.New("Unsupported tool call")
				}
				call.ID += part.ID
				call.Function.Name += part.Function.Name
				call.Function.Arguments += part.Function.Arguments
				if len(call.Function.Arguments) > 256<<10 || len(call.ID) > 128 || len(call.Function.Name) > 64 {
					return false, errors.New("Tool call is too large")
				}
				if call.ID != "" && call.Function.Name != "" {
					if err := emit(provider.Delta{Tool: &provider.ToolDelta{Index: part.Index, Call: *call}}); err != nil {
						return false, err
					}
				}
			}
			if choice.Finish != "" {
				finish = choice.Finish
			}
		}
		return false, nil
	}
	done := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if event == "" {
				continue
			}
			var err error
			done, err = consume(event)
			event = ""
			if err != nil {
				return result, err
			}
			if done {
				break
			}
		} else if strings.HasPrefix(line, "data:") {
			if event != "" {
				event += "\n"
			}
			event += strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
			if len(event) > 1<<20 {
				return result, errors.New("Model event is too large")
			}
		}
	}
	if !done && event != "" {
		var err error
		done, err = consume(event)
		if err != nil {
			return result, err
		}
	}
	if scanner.Err() != nil || !done || finish == "" {
		return result, errors.New("Model connection ended before the reply finished")
	}
	result.Message.Content, result.Message.Reasoning = text.String(), reasoning.String()
	seen := map[string]bool{}
	for index := 0; index < len(calls); index++ {
		call := calls[index]
		if call == nil || call.ID == "" || call.Function.Name == "" || seen[call.ID] {
			return result, errors.New("Model returned an incomplete tool call")
		}
		seen[call.ID] = true
		result.Message.Calls = append(result.Message.Calls, *call)
	}
	switch finish {
	case "stop":
		if len(calls) > 0 {
			return result, errors.New("Model did not finish its tool calls")
		}
	case "tool_calls":
		if len(calls) == 0 {
			return result, errors.New("Model returned no tool calls")
		}
	case "length":
		return result, provider.ErrOutputLimit
	case "content_filter":
		return result, errors.New("Model declined this reply")
	default:
		return result, errors.New("Model interrupted this reply")
	}
	return result, nil
}

func modelError(raw json.RawMessage) error {
	var detail struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	}
	if json.Unmarshal(raw, &detail) != nil || detail.Message == "" {
		json.Unmarshal(raw, &detail.Message)
	}
	lower := strings.ToLower(detail.Message + " " + detail.Type + " " + detail.Code)
	if strings.Contains(lower, "context length") || strings.Contains(lower, "context window") || strings.Contains(lower, "exceed_context") {
		return provider.ErrContextLength
	}
	if strings.Contains(lower, "max tokens") || strings.Contains(lower, "output limit") || strings.Contains(lower, "generation limit") {
		return provider.ErrOutputLimit
	}
	message := strings.TrimSpace(detail.Message)
	if message == "" {
		return errors.New("Model server stopped the reply without a reason")
	}
	if len(message) > 300 {
		message = message[:300]
	}
	return fmt.Errorf("Model server stopped the reply: %s", message)
}
