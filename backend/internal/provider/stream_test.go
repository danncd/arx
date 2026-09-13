package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func streamFrom(t *testing.T, body string) (Message, []string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	var tokens []string
	msg, err := (OpenAIDialect{}).Stream(context.Background(), prof, []Message{{Role: "user", Content: "hi"}}, nil, Options{},
		func(s string, thinking bool) { tokens = append(tokens, s) })
	return msg, tokens, err
}

func TestStreamAssemblesContentAndToolCalls(t *testing.T) {
	msg, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"role":"assistant","reasoning_content":"think "}}]}

data: {"choices":[{"delta":{"reasoning_content":"first"}}]}

data: {"choices":[{"delta":{"content":"Hel"}}]}

data:{"choices":[{"delta":{"content":"lo"}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"current_time","arguments":"{\"a\":"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"1}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if msg.Content != "Hello" {
		t.Fatalf("content = %q, want Hello", msg.Content)
	}
	if msg.ReasoningContent != "think first" {
		t.Fatalf("reasoning = %q, want think first", msg.ReasoningContent)
	}
	if len(tokens) != 4 {
		t.Fatalf("tokens = %v", tokens)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].ID != "c1" ||
		msg.ToolCalls[0].Function.Arguments != `{"a":1}` {
		t.Fatalf("tool call misassembled: %+v", msg.ToolCalls)
	}
}

func TestStreamSurfacesErrorFrame(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"par"}}]}

data: {"error":{"message":"rate limit exceeded"}}
`)
	if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("mid-stream error frame must surface, got: %v", err)
	}
}

func TestStreamRequiresFinishReason(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"Hel"}}]}
`)
	if err == nil || !strings.Contains(err.Error(), "finish reason") {
		t.Fatalf("EOF without finish_reason must error, got: %v", err)
	}
	_, _, err = streamFrom(t, `{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`)
	if err == nil {
		t.Fatal("non-SSE body must not be treated as an empty success")
	}
}

func TestStreamRejectsTruncatedToolCalls(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"t","arguments":"{\"city\":\"San Fr"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"length"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "token limit") {
		t.Fatalf("length-truncated tool call must error, got: %v", err)
	}
}

func TestStreamRejectsEmptyOutput(t *testing.T) {
	msg, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"reasoning_content":"hmm"}}]}

data: {"choices":[{"delta":{},"finish_reason":"length"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "no output") {
		t.Fatalf("empty output must error, got: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "hmm" {
		t.Fatalf("reasoning should still have streamed: %v", tokens)
	}
	if msg.ReasoningContent != "hmm" {
		t.Fatalf("reasoning was not preserved: %+v", msg)
	}
}

func TestStreamRejectsMalformedIndex(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":-1,"id":"x","function":{"name":"t","arguments":"{}"}}]}}]}
`)
	if err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatalf("negative index must be an error (not a panic), got: %v", err)
	}
}

func TestStreamRejectsHollowToolCalls(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"c1","function":{"name":"t","arguments":"{}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "missing id or name") {
		t.Fatalf("sparse index must not yield hollow calls as success, got: %v", err)
	}

	_, _, err = streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"t","arguments":"{}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "missing id or name") {
		t.Fatalf("id-less call must be refused, got: %v", err)
	}
}

func TestStreamSkipsEmptyDataHeartbeat(t *testing.T) {
	msg, _, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"hi"}}]}

data:

data:

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`)
	if err != nil {
		t.Fatalf("heartbeat aborted the stream: %v", err)
	}
	if msg.Content != "hi" {
		t.Fatalf("content = %q, want hi", msg.Content)
	}
}

func TestStreamSurfacesStringErrorFrame(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"error":"rate limited, slow down"}
`)
	if err == nil || !strings.Contains(err.Error(), "rate limited, slow down") {
		t.Fatalf("string-form error frame must surface verbatim, got: %v", err)
	}
}

func TestStreamPreservesPartialContentOnError(t *testing.T) {
	msg, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"partial answer"}}]}

data: {"error":{"message":"upstream died"}}
`)
	if err == nil {
		t.Fatal("error frame must surface")
	}
	if len(tokens) != 1 || msg.Content != "partial answer" {
		t.Fatalf("partial content lost: msg=%q tokens=%v", msg.Content, tokens)
	}
}

func TestStreamNormalizesEmptyArguments(t *testing.T) {
	msg, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"current_time"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if got := msg.ToolCalls[0].Function.Arguments; got != "{}" {
		t.Fatalf("empty arguments = %q, want {}", got)
	}
}

func TestStreamRejectsExplicitWrongRole(t *testing.T) {
	_, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"role":"user","content":"promoted"}}]}

data: {"choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "want assistant") {
		t.Fatalf("wrong streamed role must error, got: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("wrong-role content reached the callback: %v", tokens)
	}
}

func TestStreamRejectsExplicitToolType(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"custom","function":{"name":"current_time","arguments":"{}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "unsupported tool call type") {
		t.Fatalf("unsupported streamed tool type must error, got: %v", err)
	}
}

func TestStreamRejectsChoicesAfterFinish(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"No tool needed"},"finish_reason":"stop"}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"current_time","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "after finish reason") {
		t.Fatalf("choices after finish must error, got: %v", err)
	}
}

func TestStreamLimitsTotalBytes(t *testing.T) {
	line := ": " + strings.Repeat("x", 64*1024) + "\n"
	body := strings.Repeat(line, int(maxStreamBytes/int64(len(line)))+2)

	_, _, err := streamFrom(t, body)
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("oversized stream must error, got: %v", err)
	}
}

func TestStreamAcceptsLargeLineWithinLimit(t *testing.T) {
	want := strings.Repeat("x", 1024*1024+1)
	body := `data: {"choices":[{"delta":{"content":"` + want + `"},"finish_reason":"stop"}]}` + "\n\n"

	msg, _, err := streamFrom(t, body)
	if err != nil || msg.Content != want {
		t.Fatalf("large stream line: len=%d err=%v", len(msg.Content), err)
	}
}

func TestStreamRejectsInvalidUTF8(t *testing.T) {
	body := `data: {"choices":[{"delta":{"content":"` + string([]byte{0xff}) + `"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n"

	_, _, err := streamFrom(t, body)
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("invalid UTF-8 must error, got: %v", err)
	}
}

func TestStreamSurfacesPlainJSONError(t *testing.T) {
	cases := []string{
		`{"error":{"message":"insufficient credits"}}`,
		`{"error":"rate limited"}`,
	}

	for _, body := range cases {
		_, _, err := streamFrom(t, body)
		if err == nil || (!strings.Contains(err.Error(), "insufficient credits") && !strings.Contains(err.Error(), "rate limited")) {
			t.Fatalf("plain JSON error lost: %v", err)
		}
	}
}

func TestStreamDoesNotMislabelLegalSSE(t *testing.T) {
	_, _, err := streamFrom(t, ": keepalive\nx-proxy-heartbeat: 1\n\n")
	if err == nil || !strings.Contains(err.Error(), "finish reason") || strings.Contains(err.Error(), "not SSE") {
		t.Fatalf("legal SSE got the wrong error: %v", err)
	}
}

func TestStreamAcceptsSSEFraming(t *testing.T) {
	cases := map[string]string{
		"bom": "\ufeff" + `data: {"choices":[{"delta":{"content":"hi"}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n\n",
		"cr": `data: {"choices":[{"delta":{"content":"hi"}}]}` + "\r\r" +
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\r\r" +
			"data: [DONE]\r\r",
		"multiline": `data: {"choices":[{"delta":` + "\n" +
			`data: {"content":"hi"}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n\n",
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			msg, _, err := streamFrom(t, body)
			if err != nil {
				t.Fatalf("Stream: %v", err)
			}
			if msg.Content != "hi" {
				t.Fatalf("content = %q, want hi", msg.Content)
			}
		})
	}
}
