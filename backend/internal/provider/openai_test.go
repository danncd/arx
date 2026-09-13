package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatGuards(t *testing.T) {
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
		return Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL + "/", KeyEnv: "FAKE_KEY"}, Model: "m", MaxTokens: 100}
	}

	prof := serve(`{"choices":[{"message":{"role":"assistant","content":"hi","reasoning_content":"thought"},"finish_reason":"stop"}]}`)
	msg, err := (OpenAIDialect{}).Chat(context.Background(), prof, []Message{{Role: "user", Content: "x"}}, nil, Options{})
	if err != nil || msg.Content != "hi" || msg.ReasoningContent != "thought" {
		t.Fatalf("happy path: msg=%+v err=%v", msg, err)
	}

	prof = serve(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"c1","function":{"name":"current_time"}}]},"finish_reason":"tool_calls"}]}`)
	msg, err = (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{})
	if err != nil {
		t.Fatalf("zero-arg tool call: %v", err)
	}
	if msg.Role != "assistant" || msg.ToolCalls[0].Type != "function" || msg.ToolCalls[0].Function.Arguments != "{}" {
		t.Fatalf("reply not normalized: %+v", msg)
	}

	prof = serve(`{"error":{"message":"Insufficient Balance"}}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "Insufficient Balance") {
		t.Fatalf("error envelope lost: %v", err)
	}

	prof = serve(`{"choices":[]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "empty choices") {
		t.Fatalf("empty choices: %v", err)
	}

	prof = serve(`{"error":"rate limited, retry later"}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "rate limited, retry later") {
		t.Fatalf("string error envelope lost: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"t","arguments":"{\"city\":\"San Fr"}}]},"finish_reason":"length"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "token limit") {
		t.Fatalf("truncated tool call must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"length"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "no output") {
		t.Fatalf("empty reply must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"tool_calls":[{"id":"c1","function":{"name":"current_time","arguments":"{"}}]},"finish_reason":"tool_calls"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("invalid tool arguments must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"tool_calls":[{"id":"c1","type":"custom","function":{"name":"current_time","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "unsupported tool call type") {
		t.Fatalf("unsupported tool call type must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"tool_calls":[{"id":"c1","type":"function","function":{"name":"current_time","arguments":"{}"}}]},"finish_reason":"stop"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "finish reason") {
		t.Fatalf("mismatched tool call finish must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"content":"hi"},"finish_reason":"tool_calls"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "without carrying any") {
		t.Fatalf("missing tool calls must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"tool_calls":[
		{"id":"c1","type":"function","function":{"name":"first","arguments":"{}"}},
		{"id":"c1","type":"function","function":{"name":"second","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "duplicate tool call id") {
		t.Fatalf("duplicate tool call ids must error, got: %v", err)
	}

	badUTF8 := `{"choices":[{"message":{"tool_calls":[{"id":"c1","function":{"name":"current_time","arguments":"{\"key\":\"` + string([]byte{0xff}) + `\"}"}}]},"finish_reason":"tool_calls"}]}`
	prof = serve(badUTF8)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("invalid UTF-8 must error, got: %v", err)
	}

	prof = serve(`{"choices":[{"message":{"role":"system","content":"promoted"},"finish_reason":"stop"}]}`)
	if _, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{}); err == nil || !strings.Contains(err.Error(), "want assistant") {
		t.Fatalf("wrong reply role must error, got: %v", err)
	}
}

func TestChatSurfacesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	t.Cleanup(srv.Close)
	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}

	_, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("want status and body in the error, got: %v", err)
	}
}

func TestChatLimitsResponseSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(strings.Repeat("x", int(maxChatBytes)+1)))
	}))
	t.Cleanup(srv.Close)
	prof := Profile{Provider: Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}

	_, err := (OpenAIDialect{}).Chat(context.Background(), prof, nil, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("oversized response error = %v", err)
	}
}

func TestBuildRequestTokenParam(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hi"}}

	classic, _ := GetProvider("deepseek")
	b, err := json.Marshal(classic.Dialect.(OpenAIDialect).buildRequest(Profile{Provider: classic, Model: "m", MaxTokens: 8192}, msgs, nil, Options{}, false))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"max_tokens":8192`) || strings.Contains(string(b), "max_completion_tokens") {
		t.Fatalf("deepseek request has wrong token param: %s", b)
	}

	modern, _ := GetProvider("openai")
	b, err = json.Marshal(modern.Dialect.(OpenAIDialect).buildRequest(Profile{Provider: modern, Model: "m", MaxTokens: 8192}, msgs, nil, Options{}, true))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"max_completion_tokens":8192`) {
		t.Fatalf("openai request missing max_completion_tokens: %s", b)
	}
	if strings.Contains(string(b), `"max_tokens"`) {
		t.Fatalf("openai request still carries classic max_tokens: %s", b)
	}
	if !strings.Contains(string(b), `"stream":true`) {
		t.Fatalf("stream flag lost its wire tag again: %s", b)
	}
}

func TestMessageMarshalsToWireNames(t *testing.T) {
	m := Message{
		Role:             "assistant",
		ReasoningContent: "thought",
		ToolCalls: []ToolCall{{
			ID: "abc", Type: "function",
			Function: FunctionCall{Name: "current_time", Arguments: "{}"},
		}},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"reasoning_content":"thought"`, `"tool_calls"`, `"id":"abc"`, `"function"`, `"name":"current_time"`, `"arguments"`, `"content":""`} {
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

func TestRequestMarshalsToWireNames(t *testing.T) {
	classic, _ := GetProvider("deepseek")
	req := classic.Dialect.(OpenAIDialect).buildRequest(
		Profile{Provider: classic, Model: "m", MaxTokens: 100},
		[]Message{{Role: "user", Content: "hi"}},
		[]ToolSpec{{Type: "function", Function: ToolFunction{
			Name:        "current_time",
			Description: "d",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		}}},
		Options{},
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
