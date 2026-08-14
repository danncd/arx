package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// llmClient carries no overall timeout: a completion near the token
// cap legally runs for minutes. The caller's ctx is the deadline
// authority; dial timeouts still apply via the default transport.
var llmClient = &http.Client{}

type Message struct {
	Role string `json:"role"` // "system", "user", "assistant", "tool"
	// Content is deliberately NOT omitempty: a tool result or an
	// action-only assistant turn may carry "", and dropping the key
	// produces messages the APIs reject on replay.
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

// buildRequest fills the token cap under the spelling the provider
// accepts; omitempty drops the unused one from the wire.
func buildRequest(prof Profile, msgs []Message, tools []ToolSpec, stream bool) chatRequest {
	r := chatRequest{Model: prof.Model, Messages: msgs, Tools: tools, Stream: stream}
	if prof.Provider.NewTokenParam {
		r.MaxCompletionTokens = prof.MaxTokens
	} else {
		r.MaxTokens = prof.MaxTokens
	}
	return r
}

// postChat sends one chat-completions request and hands back the open
// 200 response; the caller owns resp.Body.Close. Non-200s are read,
// closed, and returned as errors carrying the provider's explanation.
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
	req.Header.Set("Authorization", "Bearer "+prof.Provider.Key())

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
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Chat is the non-streaming completion call. Interactive turns use
// Stream; this stays for headless work (tests, future subagents).
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
	if len(cr.Choices) == 0 {
		return Message{}, fmt.Errorf("%s chat: empty choices", prof.Provider.Name)
	}
	m := cr.Choices[0].Message
	return m, completionErr(m, cr.Choices[0].FinishReason)
}

// completionErr is the completion contract shared by Chat and Stream:
// a reply is only usable when it finished for a stated reason, its
// tool calls are structurally whole, and it carries SOMETHING — an
// unusable reply appended to the transcript is replayed on every later
// request and bricks the session.
func completionErr(msg Message, finish string) error {
	// A dropped connection surfaces as a clean EOF, so the only
	// reliable completion signal is a non-empty finish_reason (same
	// contract as arx-1): its absence is never a finished message.
	if finish == "" {
		return fmt.Errorf("response ended without a finish reason (connection lost mid-response?)")
	}
	if finish == "length" && len(msg.ToolCalls) > 0 {
		// Truncated tool-call arguments are unparseable and would
		// poison the transcript on replay.
		return fmt.Errorf("hit the token limit mid tool-call; arguments are truncated")
	}
	for _, c := range msg.ToolCalls {
		// A sparse or id-less delta sequence can assemble hollow calls;
		// replaying an empty id (or omitting tool_call_id on the
		// result) is rejected by the APIs.
		if c.ID == "" || c.Function.Name == "" {
			return fmt.Errorf("assembled tool call missing id or name (malformed stream)")
		}
	}
	if msg.Content == "" && len(msg.ToolCalls) == 0 {
		// Nothing usable arrived (e.g. the whole budget went to
		// reasoning).
		return fmt.Errorf("model produced no output (finish_reason %q)", finish)
	}
	return nil
}
