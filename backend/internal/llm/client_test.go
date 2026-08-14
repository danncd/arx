package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
