package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type streamChunk struct {
	// Error is the mid-stream failure frame (rate limit, upstream
	// error) providers emit on an already-200 SSE response.
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
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

// Stream is Chat with live output: content fragments stream to onToken
// as they arrive (thinking=true marks the model's reasoning channel,
// which is rendered but never stored — replaying it is both rejected by
// DeepSeek and a waste of context). The fully assembled message, tool
// calls included, is returned at the end exactly as Chat would.
func Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, onToken func(s string, thinking bool)) (Message, error) {
	resp, err := postChat(ctx, prof, msgs, tools, true)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	msg := Message{Role: "assistant"}
	var content strings.Builder
	finish := ""
	// fail flushes the accumulated content so a partial message rides
	// along with its error instead of contradicting what already
	// streamed to the user.
	fail := func(err error) (Message, error) {
		msg.Content = content.String()
		return msg, err
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		// The space after "data:" is optional in the SSE grammar.
		payload, ok := strings.CutPrefix(sc.Text(), "data:")
		if !ok {
			continue // blank separators and ": keepalive" comments
		}
		payload = strings.TrimSpace(payload)
		if payload == "" {
			continue // "data:" with no payload is a legal SSE heartbeat
		}
		if payload == "[DONE]" {
			break
		}
		var ch streamChunk
		if err := json.Unmarshal([]byte(payload), &ch); err != nil {
			// Some gateways send the error as a bare string rather than
			// the usual object; surface the provider's words either way.
			var alt struct {
				Error string `json:"error"`
			}
			if json.Unmarshal([]byte(payload), &alt) == nil && alt.Error != "" {
				return fail(fmt.Errorf("%s stream error: %s", prof.Provider.Name, alt.Error))
			}
			// A data: payload that fails to parse is corruption, not
			// noise: skipping it could drop text or argument fragments
			// and then report the call as successful.
			return fail(fmt.Errorf("malformed stream data: %w", err))
		}
		if ch.Error != nil {
			return fail(fmt.Errorf("%s stream error: %s", prof.Provider.Name, ch.Error.Message))
		}
		if len(ch.Choices) == 0 {
			continue // usage-only chunks carry no choices
		}
		if fr := ch.Choices[0].FinishReason; fr != "" {
			finish = fr
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
			if tc.Index < 0 || tc.Index > 63 {
				return fail(fmt.Errorf("malformed tool_call index %d in stream", tc.Index))
			}
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
	msg.Content = content.String()
	if err := sc.Err(); err != nil {
		return msg, fmt.Errorf("stream read: %w", err)
	}
	normalizeReply(&msg)
	return msg, completionErr(msg, finish)
}
