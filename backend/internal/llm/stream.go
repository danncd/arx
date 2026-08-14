package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"strings"
	"unicode/utf8"
)

const maxStreamBytes int64 = 4 << 20

/*
	Stream chunk data structure
*/

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

/*
	Sends a request to postChat with stream=true
*/

func Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, onToken func(s string, thinking bool)) (Message, error) {
	// postChat call and error handling
	resp, err := postChat(ctx, prof, msgs, tools, true)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	isSSE := mediaType == "text/event-stream"

	// Build assisstant reply skeleton
	msg := Message{}
	var content strings.Builder
	var arguments [][]string
	finish := ""
	finished := false
	done := false
	assemble := func() {
		msg.Content = content.String()
		for i := range msg.ToolCalls {
			msg.ToolCalls[i].Function.Arguments = strings.Join(arguments[i], "")
		}
	}
	fail := func(err error) (Message, error) {
		assemble()
		return msg, err
	}

	// Start a scanner that reads the response from postChat
	limited := &io.LimitedReader{R: resp.Body, N: maxStreamBytes + 1}
	sc := bufio.NewScanner(limited)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	sc.Split(splitSSELines)

	var eventData strings.Builder
	var nonSSE strings.Builder
	hasData := false
	sawData := false
	dispatch := func(payload string) error {
		if payload == "" {
			return nil
		}
		if payload == "[DONE]" {
			done = true
			return nil
		}

		var ch streamChunk
		if err := json.Unmarshal([]byte(payload), &ch); err != nil {
			var alt struct {
				Error string `json:"error"`
			}
			if json.Unmarshal([]byte(payload), &alt) == nil && alt.Error != "" {
				return fmt.Errorf("%s stream error: %s", prof.Provider.Name, alt.Error)
			}
			return fmt.Errorf("malformed stream data: %w", err)
		}
		if ch.Error != nil {
			return fmt.Errorf("%s stream error: %s", prof.Provider.Name, ch.Error.Message)
		}
		if len(ch.Choices) == 0 {
			return nil
		}
		if finished {
			return fmt.Errorf("stream carried choices after finish reason %q", finish)
		}

		d := ch.Choices[0].Delta
		if d.Role != "" {
			if msg.Role != "" && msg.Role != d.Role {
				return fmt.Errorf("stream changed reply role from %q to %q", msg.Role, d.Role)
			}
			msg.Role = d.Role
		}
		if d.ReasoningContent != "" {
			onToken(d.ReasoningContent, true)
		}
		if d.Content != "" {
			content.WriteString(d.Content)
			onToken(d.Content, false)
		}
		for _, tc := range d.ToolCalls {
			if tc.Index < 0 || tc.Index > 63 {
				return fmt.Errorf("malformed tool_call index %d in stream", tc.Index)
			}
			for tc.Index >= len(msg.ToolCalls) {
				msg.ToolCalls = append(msg.ToolCalls, ToolCall{})
				arguments = append(arguments, nil)
			}
			cur := &msg.ToolCalls[tc.Index]
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
			arguments[tc.Index] = append(arguments[tc.Index], tc.Function.Arguments)
		}
		if fr := ch.Choices[0].FinishReason; fr != "" {
			finish = fr
			finished = true
		}
		return nil
	}
	flush := func() error {
		if !hasData {
			return nil
		}
		payload := strings.TrimSuffix(eventData.String(), "\n")
		eventData.Reset()
		hasData = false
		return dispatch(payload)
	}

	firstLine := true
	for sc.Scan() {
		if limited.N == 0 {
			return fail(fmt.Errorf("stream exceeded %d bytes", maxStreamBytes))
		}
		line := sc.Text()
		if firstLine {
			line = strings.TrimPrefix(line, "\ufeff")
			firstLine = false
		}
		if !utf8.ValidString(line) {
			return fail(fmt.Errorf("stream data contains invalid UTF-8"))
		}
		if line == "" {
			if err := flush(); err != nil {
				return fail(err)
			}
			if done {
				break
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, found := strings.Cut(line, ":")
		if !found {
			value = ""
		}
		if strings.HasPrefix(value, " ") {
			value = value[1:]
		}
		if field == "data" {
			eventData.WriteString(value)
			eventData.WriteByte('\n')
			hasData = true
			sawData = true
		} else if !sawData && field != "event" && field != "id" && field != "retry" && nonSSE.Len() < 2048 {
			remaining := 2048 - nonSSE.Len()
			candidate := line + "\n"
			if len(candidate) > remaining {
				candidate = candidate[:remaining]
			}
			nonSSE.WriteString(candidate)
		}
	}
	if limited.N == 0 {
		return fail(fmt.Errorf("stream exceeded %d bytes", maxStreamBytes))
	}
	if err := sc.Err(); err != nil {
		return fail(fmt.Errorf("stream read: %w", err))
	}
	if !done {
		if err := flush(); err != nil {
			return fail(err)
		}
	}
	if !sawData && nonSSE.Len() > 0 {
		raw := []byte(strings.TrimSpace(nonSSE.String()))
		var object struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &object) == nil && object.Error != nil {
			return fail(fmt.Errorf("%s stream error: %s", prof.Provider.Name, object.Error.Message))
		}
		var text struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(raw, &text) == nil && text.Error != "" {
			return fail(fmt.Errorf("%s stream error: %s", prof.Provider.Name, text.Error))
		}
		if !isSSE {
			return fail(fmt.Errorf("%s stream: response was not SSE", prof.Provider.Name))
		}
	}
	assemble()
	normalizeReply(&msg)
	return msg, completionErr(msg, finish)
}
