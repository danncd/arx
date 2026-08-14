package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

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
	for _, want := range []string{`"tool_calls"`, `"id":"abc"`, `"function"`, `"name":"current_time"`, `"arguments"`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("wire JSON missing %s: %s", want, b)
		}
	}
	if strings.Contains(string(b), `"content"`) {
		t.Fatalf("empty content should be omitted on action messages: %s", b)
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
