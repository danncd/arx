package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

type streamChunk struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Delta struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int          `json:"index"`
				ID       string       `json:"id"`
				Type     string       `json:"type"`
				Function FunctionCall `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func splitSSELines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		switch b {
		case '\n':
			end := i
			if i > 0 && data[i-1] == '\r' {
				end--
			}
			return i + 1, data[:end], nil
		case '\r':
			if i+1 == len(data) && !atEOF {
				return 0, nil, nil
			}
			if i+1 < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

type streamAssembler struct {
	msg       Message
	content   strings.Builder
	reasoning strings.Builder
	arguments [][]string
	eventData strings.Builder
	finish    string
	finished  bool
	done      bool
	hasData   bool
	provider  string
	onToken   func(string, bool)
}

func (a *streamAssembler) assemble() {
	a.msg.Content = a.content.String()
	a.msg.ReasoningContent = a.reasoning.String()
	for i := range a.msg.ToolCalls {
		a.msg.ToolCalls[i].Function.Arguments = strings.Join(a.arguments[i], "")
	}
}

func (a *streamAssembler) flush() error {
	if !a.hasData {
		return nil
	}
	payload := strings.TrimSuffix(a.eventData.String(), "\n")
	a.eventData.Reset()
	a.hasData = false
	return a.dispatch(payload)
}

func (a *streamAssembler) dispatch(payload string) error {
	if payload == "" {
		return nil
	}
	if payload == "[DONE]" {
		a.done = true
		return nil
	}

	var ch streamChunk
	if err := json.Unmarshal([]byte(payload), &ch); err != nil {
		var alt struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(payload), &alt) == nil && alt.Error != "" {
			return fmt.Errorf("%s stream error: %s", a.provider, alt.Error)
		}
		return fmt.Errorf("malformed stream data: %w", err)
	}
	if ch.Error != nil {
		return fmt.Errorf("%s stream error: %s", a.provider, ch.Error.Message)
	}
	if len(ch.Choices) == 0 {
		return nil
	}
	if a.finished {
		return fmt.Errorf("stream carried choices after finish reason %q", a.finish)
	}

	delta := ch.Choices[0].Delta
	if delta.Role != "" {
		if delta.Role != "assistant" {
			return fmt.Errorf("reply has role %q, want assistant", delta.Role)
		}
		if a.msg.Role != "" && a.msg.Role != delta.Role {
			return fmt.Errorf("stream changed reply role from %q to %q", a.msg.Role, delta.Role)
		}
		a.msg.Role = delta.Role
	}
	if delta.ReasoningContent != "" {
		a.reasoning.WriteString(delta.ReasoningContent)
		a.onToken(delta.ReasoningContent, true)
	}
	if delta.Content != "" {
		a.content.WriteString(delta.Content)
		a.onToken(delta.Content, false)
	}
	for _, tc := range delta.ToolCalls {
		if tc.Index < 0 || tc.Index > 63 {
			return fmt.Errorf("malformed tool_call index %d in stream", tc.Index)
		}
		for tc.Index >= len(a.msg.ToolCalls) {
			a.msg.ToolCalls = append(a.msg.ToolCalls, ToolCall{})
			a.arguments = append(a.arguments, nil)
		}
		cur := &a.msg.ToolCalls[tc.Index]
		if tc.ID != "" {
			if cur.ID != "" && cur.ID != tc.ID {
				return fmt.Errorf("stream changed tool call id from %q to %q", cur.ID, tc.ID)
			}
			cur.ID = tc.ID
		}
		if tc.Type != "" {
			if cur.Type != "" && cur.Type != tc.Type {
				return fmt.Errorf("stream changed tool call type from %q to %q", cur.Type, tc.Type)
			}
			cur.Type = tc.Type
		}
		if tc.Function.Name != "" {
			if cur.Function.Name != "" && cur.Function.Name != tc.Function.Name {
				return fmt.Errorf("stream changed tool name from %q to %q", cur.Function.Name, tc.Function.Name)
			}
			cur.Function.Name = tc.Function.Name
		}
		a.arguments[tc.Index] = append(a.arguments[tc.Index], tc.Function.Arguments)
	}
	if fr := ch.Choices[0].FinishReason; fr != "" {
		a.finish = fr
		a.finished = true
	}
	return nil
}
