package provider

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

func (d OpenAIDialect) Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options, onToken func(s string, thinking bool)) (Message, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resp, err := postChat(ctx, prof, d.buildRequest(prof, msgs, tools, opts, true))
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	isSSE := mediaType == "text/event-stream"

	assembler := &streamAssembler{provider: prof.Provider.Name, onToken: onToken}
	fail := func(err error) (Message, error) {
		assembler.assemble()
		return assembler.msg, err
	}

	ir := newIdleReader(resp.Body, cancel)
	defer ir.stop()
	limited := &io.LimitedReader{R: ir, N: maxStreamBytes + 1}
	sc := bufio.NewScanner(limited)
	sc.Buffer(make([]byte, 0, 64*1024), int(maxStreamBytes)+1)
	sc.Split(splitSSELines)

	var nonSSE strings.Builder
	sawData := false
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
			if err := assembler.flush(); err != nil {
				return fail(err)
			}
			if assembler.done {
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
			assembler.eventData.WriteString(value)
			assembler.eventData.WriteByte('\n')
			assembler.hasData = true
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
		if ir.timedOut() {
			return fail(fmt.Errorf("stream idle timeout after %s", idleTimeout))
		}
		return fail(fmt.Errorf("stream read: %w", err))
	}
	if !assembler.done {
		if err := assembler.flush(); err != nil {
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
	assembler.assemble()
	normalizeReply(&assembler.msg)
	return assembler.msg, completionErr(assembler.msg, assembler.finish)
}
