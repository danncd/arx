package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

/*
	HTTP client that waits forever
*/

var llmClient = &http.Client{}

/*
	Data structures for chats, messages, tools and functions
*/

type Message struct {
	Role string `json:"role"`
	// no omitempty: the APIs require the content key even when empty
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolSpec struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type chatRequest struct {
	Model               string     `json:"model"`
	Messages            []Message  `json:"messages"`
	MaxTokens           int        `json:"max_tokens,omitempty"`
	MaxCompletionTokens int        `json:"max_completion_tokens,omitempty"`
	Tools               []ToolSpec `json:"tools,omitempty"`
	Stream              bool       `json:"stream,omitempty"`
}

/*
	Takes the profile (model, provider, max tokens), message list, tools, stream
	returns r (request) to a chatRequest with the correct arguments
*/

func buildRequest(prof Profile, msgs []Message, tools []ToolSpec, stream bool) chatRequest {
	r := chatRequest{Model: prof.Model, Messages: msgs, Tools: tools, Stream: stream}
	if prof.Provider.NewTokenParam {
		r.MaxCompletionTokens = prof.MaxTokens
	} else {
		r.MaxTokens = prof.MaxTokens
	}
	return r
}

/*
	Takes context, profile, messages, tools, stream,
	builds a json body, sends request to provider api
	and returns an unread body.
*/

func postChat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, stream bool) (*http.Response, error) {
	body, err := json.Marshal(buildRequest(prof, msgs, tools, stream))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST",
		prof.Provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if k := prof.Provider.Key(); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}

	resp, err := llmClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("%s chat: HTTP %d: %s", prof.Provider.Name, resp.StatusCode, b)
	}
	return resp, nil
}

type chatResponse struct {
	// error envelopes can arrive on a 200, same as in streams
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

/*
	Takes the response by postChat
	and returns the full message.
*/

func Chat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec) (Message, error) {
	resp, err := postChat(ctx, prof, msgs, tools, false)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Message{}, fmt.Errorf("decode chat response: %w", err)
	}
	if cr.Error != nil {
		return Message{}, fmt.Errorf("%s chat error: %s", prof.Provider.Name, cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return Message{}, fmt.Errorf("%s chat: empty choices", prof.Provider.Name)
	}
	m := cr.Choices[0].Message
	normalizeReply(&m)
	return m, completionErr(m, cr.Choices[0].FinishReason)
}

/*
	Enforces what both entry points promise: assistant role and
	JSON-valid tool arguments, whatever the wire omitted
*/

func normalizeReply(m *Message) {
	if m.Role == "" {
		m.Role = "assistant"
	}
	for i := range m.ToolCalls {
		if m.ToolCalls[i].Type == "" {
			m.ToolCalls[i].Type = "function"
		}
		if m.ToolCalls[i].Function.Arguments == "" {
			m.ToolCalls[i].Function.Arguments = "{}"
		}
	}
}

/*
	Rejects unusable replies: no finish reason, truncated or
	hollow tool calls, empty output
*/

func completionErr(msg Message, finish string) error {
	if finish == "" {
		return fmt.Errorf("response ended without a finish reason (connection lost mid-response?)")
	}
	if finish == "length" && len(msg.ToolCalls) > 0 {
		return fmt.Errorf("hit the token limit mid tool-call; arguments are truncated")
	}
	for _, c := range msg.ToolCalls {
		if c.ID == "" || c.Function.Name == "" {
			return fmt.Errorf("assembled tool call missing id or name (malformed stream)")
		}
	}
	if msg.Content == "" && len(msg.ToolCalls) == 0 {
		return fmt.Errorf("model produced no output (finish_reason %q)", finish)
	}
	return nil
}
