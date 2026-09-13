package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

type OpenAIDialect struct {
	NewTokenParam bool
	ThinkingParam bool
}

type thinkingParam struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model               string         `json:"model"`
	Messages            []Message      `json:"messages"`
	MaxTokens           int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	Tools               []ToolSpec     `json:"tools,omitempty"`
	Thinking            *thinkingParam `json:"thinking,omitempty"`
	Stream              bool           `json:"stream,omitempty"`
}

type chatResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

var llmClient = &http.Client{}

const maxChatBytes int64 = 8 << 20

var idleTimeout = 120 * time.Second

type idleReader struct {
	inner io.Reader
	timer *time.Timer
	fired atomic.Bool
}

func newIdleReader(inner io.Reader, cancel context.CancelFunc) *idleReader {
	r := &idleReader{inner: inner}
	r.timer = time.AfterFunc(idleTimeout, func() {
		r.fired.Store(true)
		cancel()
	})
	return r
}

func (r *idleReader) Read(p []byte) (int, error) {
	r.timer.Reset(idleTimeout)
	return r.inner.Read(p)
}

func (r *idleReader) stop()          { r.timer.Stop() }
func (r *idleReader) timedOut() bool { return r.fired.Load() }

func (d OpenAIDialect) buildRequest(prof Profile, msgs []Message, tools []ToolSpec, opts Options, stream bool) chatRequest {
	r := chatRequest{Model: prof.Model, Messages: msgs, Tools: tools, Stream: stream}
	maxTokens := prof.MaxTokens
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	if d.NewTokenParam {
		r.MaxCompletionTokens = maxTokens
	} else {
		r.MaxTokens = maxTokens
	}
	if d.ThinkingParam && opts.Thinking != nil {
		mode := "disabled"
		if *opts.Thinking {
			mode = "enabled"
		}
		r.Thinking = &thinkingParam{Type: mode}
	}
	return r
}

func postChat(ctx context.Context, prof Profile, body chatRequest) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.JoinPath(prof.Provider.BaseURL, "chat/completions")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
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

func (d OpenAIDialect) Chat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options) (Message, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resp, err := postChat(ctx, prof, d.buildRequest(prof, msgs, tools, opts, false))
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	ir := newIdleReader(resp.Body, cancel)
	defer ir.stop()
	limited := &io.LimitedReader{R: ir, N: maxChatBytes + 1}
	raw, err := io.ReadAll(limited)
	if err != nil {
		if ir.timedOut() {
			return Message{}, fmt.Errorf("chat idle timeout after %s", idleTimeout)
		}
		return Message{}, fmt.Errorf("read chat response: %w", err)
	}
	if limited.N == 0 {
		return Message{}, fmt.Errorf("chat response exceeded %d bytes", maxChatBytes)
	}
	if !utf8.Valid(raw) {
		return Message{}, fmt.Errorf("decode chat response: invalid UTF-8")
	}
	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		var alt struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &alt) == nil && alt.Error != "" {
			return Message{}, fmt.Errorf("%s chat error: %s", prof.Provider.Name, alt.Error)
		}
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

func completionErr(msg Message, finish string) error {
	if finish == "" {
		return fmt.Errorf("response ended without a finish reason (connection lost mid-response?)")
	}
	if msg.Role != "assistant" {
		return fmt.Errorf("reply has role %q, want assistant", msg.Role)
	}
	if finish == "length" && len(msg.ToolCalls) > 0 {
		return fmt.Errorf("hit the token limit mid tool-call; arguments are truncated")
	}
	if len(msg.ToolCalls) > 0 && finish != "tool_calls" {
		return fmt.Errorf("reply carries tool calls with finish reason %q", finish)
	}
	if finish == "tool_calls" && len(msg.ToolCalls) == 0 {
		return fmt.Errorf("reply ended for tool calls without carrying any")
	}
	ids := make(map[string]struct{}, len(msg.ToolCalls))
	for _, c := range msg.ToolCalls {
		if c.ID == "" || c.Function.Name == "" {
			return fmt.Errorf("reply carries a tool call missing id or name")
		}
		if c.Type != "function" {
			return fmt.Errorf("reply carries unsupported tool call type %q", c.Type)
		}
		if _, exists := ids[c.ID]; exists {
			return fmt.Errorf("reply carries duplicate tool call id %q", c.ID)
		}
		ids[c.ID] = struct{}{}
		if !json.Valid([]byte(c.Function.Arguments)) {
			return fmt.Errorf("reply carries tool call %q with invalid JSON arguments", c.ID)
		}
	}
	if msg.Content == "" && len(msg.ToolCalls) == 0 {
		return fmt.Errorf("model produced no output (finish_reason %q)", finish)
	}
	return nil
}
