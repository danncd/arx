package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Message struct {
	Role       string     `json:"role"` // "system", "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`
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

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int          `json:"index"`
				ID       string       `json:"id"`
				Function FunctionCall `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func Chat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec) (Message, error) {
	body, err := json.Marshal(buildRequest(prof, msgs, tools, false))
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		prof.Provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prof.Provider.Key())

	resp, err := httpClient.Do(req)
	if err != nil {
		return Message{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Message{}, fmt.Errorf("%s chat: HTTP %d: %s", prof.Provider.Name, resp.StatusCode, b)
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Message{}, fmt.Errorf("decode chat response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return Message{}, fmt.Errorf("%s chat: empty choices", prof.Provider.Name)
	}
	return cr.Choices[0].Message, nil
}

// Stream is Chat with live output: content fragments stream to onToken
// as they arrive (thinking=true marks the model's reasoning channel,
// which is rendered but never stored — replaying it is both rejected by
// DeepSeek and a waste of context). The fully assembled message, tool
// calls included, is returned at the end exactly as Chat would.
func Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, onToken func(s string, thinking bool)) (Message, error) {
	body, err := json.Marshal(buildRequest(prof, msgs, tools, true))
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		prof.Provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+prof.Provider.Key())

	var streamClient = &http.Client{}

	resp, err := streamClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Message{}, fmt.Errorf("%s chat: HTTP %d: %s", prof.Provider.Name, resp.StatusCode, b)
	}

	msg := Message{Role: "assistant"}
	var content strings.Builder

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue // blank separators and ": keepalive" comments
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var ch streamChunk
		if err := json.Unmarshal([]byte(payload), &ch); err != nil {
			return Message{}, fmt.Errorf("bad stream chunk: %w", err)
		}
		if len(ch.Choices) == 0 {
			continue
		}
		d := ch.Choices[0].Delta

		if d.ReasoningContent != "" {
			onToken(d.ReasoningContent, true)
		}
		if d.Content != "" {
			content.WriteString(d.Content)
			onToken(d.Content, false)
		}
		for _, tc := range d.ToolCalls {
			for tc.Index >= len(msg.ToolCalls) { // first fragment of a new call
				msg.ToolCalls = append(msg.ToolCalls, ToolCall{Type: "function"})
			}
			cur := &msg.ToolCalls[tc.Index]
			if tc.ID != "" {
				cur.ID = tc.ID
			}
			if tc.Function.Name != "" {
				cur.Function.Name = tc.Function.Name
			}
			cur.Function.Arguments += tc.Function.Arguments // fragments concatenate
		}
	}
	if err := sc.Err(); err != nil {
		return Message{}, fmt.Errorf("stream read: %w", err)
	}
	msg.Content = content.String()
	return msg, nil
}
