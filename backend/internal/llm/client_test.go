package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// streamFrom serves body as the SSE response and runs Stream against
// it, collecting emitted tokens.
func streamFrom(t *testing.T, body string) (Message, []string, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	var tokens []string
	msg, err := Stream(context.Background(), prof, []Message{{Role: "user", Content: "hi"}}, nil,
		func(s string, thinking bool) { tokens = append(tokens, s) })
	return msg, tokens, err
}

func TestStreamAssemblesContentAndToolCalls(t *testing.T) {
	msg, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"role":"assistant","content":"Hel"}}]}

data:{"choices":[{"delta":{"content":"lo"}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"current_time","arguments":"{\"a\":"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"1}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	// The second delta used the spec-legal spaceless "data:" form and
	// must not be dropped.
	if msg.Content != "Hello" {
		t.Fatalf("content = %q, want Hello", msg.Content)
	}
	if len(tokens) != 2 {
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
	// A cut connection is a clean EOF; without a finish_reason the
	// message must not be reported as complete.
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"content":"Hel"}}]}
`)
	if err == nil || !strings.Contains(err.Error(), "finish reason") {
		t.Fatalf("EOF without finish_reason must error, got: %v", err)
	}
	// A plain JSON (non-SSE) 200 body has no data: lines at all.
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
	// Reasoning-only stream: the whole budget went to thinking. The
	// returned message would marshal to {"role":"assistant"} and brick
	// the session on replay — must be an error, not a success.
	_, tokens, err := streamFrom(t, `data: {"choices":[{"delta":{"reasoning_content":"hmm"}}]}

data: {"choices":[{"delta":{},"finish_reason":"length"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "no output") {
		t.Fatalf("empty output must error, got: %v", err)
	}
	if len(tokens) != 1 || tokens[0] != "hmm" {
		t.Fatalf("reasoning should still have streamed: %v", tokens)
	}
}

func TestStreamRejectsMalformedIndex(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":-1,"id":"x","function":{"name":"t","arguments":"{}"}}]}}]}
`)
	if err == nil || !strings.Contains(err.Error(), "index") {
		t.Fatalf("negative index must be an error (not a panic), got: %v", err)
	}
}

// A sparse index pads earlier slots with hollow calls, and some
// gateways omit ids entirely; either way an assembled call without an
// id or name would be rejected on replay — success must be refused.
func TestStreamRejectsHollowToolCalls(t *testing.T) {
	// First fragment arrives at index 1: slot 0 is a fabricated shell.
	_, _, err := streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"c1","function":{"name":"t","arguments":"{}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "missing id or name") {
		t.Fatalf("sparse index must not yield hollow calls as success, got: %v", err)
	}

	// Fragments that never carry an id.
	_, _, err = streamFrom(t, `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"name":"t","arguments":"{}"}}]}}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]
`)
	if err == nil || !strings.Contains(err.Error(), "missing id or name") {
		t.Fatalf("id-less call must be refused, got: %v", err)
	}
}

// Legal SSE heartbeats — bare "data:" with no payload — must be
// skipped, not treated as malformed data.
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

// Some gateways send the error frame as a bare string, not an object;
// the provider's words must surface either way.
func TestStreamSurfacesStringErrorFrame(t *testing.T) {
	_, _, err := streamFrom(t, `data: {"error":"rate limited, slow down"}
`)
	if err == nil || !strings.Contains(err.Error(), "rate limited, slow down") {
		t.Fatalf("string-form error frame must surface verbatim, got: %v", err)
	}
}

// Content that already streamed to the user must ride along with a
// mid-stream error, not vanish from the returned message.
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

// Providers may omit arguments for zero-arg tools; tools unmarshal
// their args, so "" must be normalized to valid JSON.
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

// Chat shares Stream's completion contract: truncated tool calls and
// empty replies must not be returned as success.
func TestChatGuards(t *testing.T) {
	serve := func(body string) Profile {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	}

	// Happy path still works.
	prof := serve(`{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`)
	msg, err := Chat(context.Background(), prof, []Message{{Role: "user", Content: "x"}}, nil)
	if err != nil || msg.Content != "hi" {
		t.Fatalf("happy path: msg=%+v err=%v", msg, err)
	}

	// Token limit mid tool-call: truncated arguments.
	prof = serve(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"t","arguments":"{\"city\":\"San Fr"}}]},"finish_reason":"length"}]}`)
	if _, err := Chat(context.Background(), prof, nil, nil); err == nil || !strings.Contains(err.Error(), "token limit") {
		t.Fatalf("truncated tool call must error, got: %v", err)
	}

	// Empty reply (budget spent on reasoning).
	prof = serve(`{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"length"}]}`)
	if _, err := Chat(context.Background(), prof, nil, nil); err == nil || !strings.Contains(err.Error(), "no output") {
		t.Fatalf("empty reply must error, got: %v", err)
	}
}

// The token cap must go out under the spelling each provider accepts:
// classic "max_tokens" for deepseek, "max_completion_tokens" for openai.
// Exactly one of the two may appear on the wire.
func TestBuildRequestTokenParam(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hi"}}

	classic, _ := GetProvider("deepseek")
	b, err := json.Marshal(buildRequest(Profile{Provider: classic, Model: "m", MaxTokens: 8192}, msgs, nil, false))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"max_tokens":8192`) || strings.Contains(string(b), "max_completion_tokens") {
		t.Fatalf("deepseek request has wrong token param: %s", b)
	}

	modern, _ := GetProvider("openai")
	b, err = json.Marshal(buildRequest(Profile{Provider: modern, Model: "m", MaxTokens: 8192}, msgs, nil, true))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"max_completion_tokens":8192`) {
		t.Fatalf("openai request missing max_completion_tokens: %s", b)
	}
	// "max_completion_tokens" contains "max_tokens" as a substring, so
	// check for the classic key's quoted form specifically.
	if strings.Contains(string(b), `"max_tokens"`) {
		t.Fatalf("openai request still carries classic max_tokens: %s", b)
	}
	if !strings.Contains(string(b), `"stream":true`) {
		t.Fatalf("stream flag lost its wire tag again: %s", b)
	}
}

// Wire names are load-bearing: an untagged field silently marshals
// under its Go name and the API ignores it. This has bitten three
// times (tools, tool_calls, stream) — pin the whole message shape.
func TestMessageMarshalsToWireNames(t *testing.T) {
	m := Message{
		Role: "assistant",
		ToolCalls: []ToolCall{{
			ID: "abc", Type: "function",
			Function: FunctionCall{Name: "current_time", Arguments: "{}"},
		}},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"tool_calls"`, `"id":"abc"`, `"function"`, `"name":"current_time"`, `"arguments"`, `"content":""`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("wire JSON missing %s: %s", want, b)
		}
	}

	tr := Message{Role: "tool", ToolCallID: "abc", Content: "9:41 PM"}
	b, err = json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"tool_call_id":"abc"`) {
		t.Fatalf("tool result missing tool_call_id: %s", b)
	}
}
