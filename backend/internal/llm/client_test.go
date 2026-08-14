package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Chat shares Stream's completion contract: truncated tool calls and
// empty replies must not be returned as success.
func TestChatGuards(t *testing.T) {
	// The fake inspects the request: path, headers, and the absence of
	// the stream flag — a request-blind fake once let all of those
	// mutate freely with the suite green.
	t.Setenv("FAKE_KEY", "sk-test")
	serve := func(body string) Profile {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/chat/completions" {
				t.Errorf("path = %q, want /chat/completions", r.URL.Path)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("content-type = %q", got)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
				t.Errorf("auth = %q, want Bearer sk-test", got)
			}
			reqBody, _ := io.ReadAll(r.Body)
			if strings.Contains(string(reqBody), `"stream"`) {
				t.Errorf("Chat must not request streaming: %s", reqBody)
			}
			w.Write([]byte(body))
		}))
		t.Cleanup(srv.Close)
		return Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL, KeyEnv: "FAKE_KEY"}, Model: "m", MaxTokens: 100}
	}

	// Happy path still works.
	prof := serve(`{"choices":[{"message":{"role":"assistant","content":"hi"},"finish_reason":"stop"}]}`)
	msg, err := Chat(context.Background(), prof, []Message{{Role: "user", Content: "x"}}, nil)
	if err != nil || msg.Content != "hi" {
		t.Fatalf("happy path: msg=%+v err=%v", msg, err)
	}

	// Wire omissions are normalized exactly as Stream does: missing
	// role becomes assistant, missing arguments become {}.
	prof = serve(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"current_time"}}]},"finish_reason":"tool_calls"}]}`)
	msg, err = Chat(context.Background(), prof, nil, nil)
	if err != nil {
		t.Fatalf("zero-arg tool call: %v", err)
	}
	if msg.Role != "assistant" || msg.ToolCalls[0].Function.Arguments != "{}" {
		t.Fatalf("reply not normalized: %+v", msg)
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

// The request and tool-spec wire tags, pinned the same way: a renamed
// tag ships a key the API silently ignores.
func TestRequestMarshalsToWireNames(t *testing.T) {
	classic, _ := GetProvider("deepseek")
	req := buildRequest(
		Profile{Provider: classic, Model: "m", MaxTokens: 100},
		[]Message{{Role: "user", Content: "hi"}},
		[]ToolSpec{{Type: "function", Function: ToolFunction{
			Name:        "current_time",
			Description: "d",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		}}},
		false,
	)
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"model":"m"`, `"messages":[`, `"tools":[`,
		`"type":"function"`, `"function":{`, `"name":"current_time"`,
		`"description":"d"`, `"parameters":{"type":"object"}`,
	} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("request wire JSON missing %s: %s", want, b)
		}
	}
}
