package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"arx/internal/llm"
	"arx/internal/tool"
)

// recSink records a turn's events for assertions.
type recSink struct {
	tokens []string
	tools  []string
}

func (r *recSink) Token(s string, _ bool)      { r.tokens = append(r.tokens, s) }
func (r *recSink) ToolResult(name, out string) { r.tools = append(r.tools, name+"→"+out) }

func init() {
	tool.Register(tool.Tool{
		Name:        "test_echo",
		Description: "echoes for tests",
		Run: func(_ context.Context, args json.RawMessage) (string, error) {
			return "echoed:" + string(args), nil
		},
	})
}

// A full reason-act round: the model calls a tool, the result goes
// back labeled with the call id, and the second round answers in text.
func TestRunTurnToolRound(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		switch requests {
		case 1:
			w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{\"s\":1}"}}]}}]}` + "\n\n" +
				`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
		default:
			// The tool result must have come back, tied to its call.
			if !strings.Contains(string(body), `"tool_call_id":"c1"`) ||
				!strings.Contains(string(body), `echoed:{\"s\":1}`) {
				t.Errorf("round 2 request missing the tool result: %s", body)
			}
			w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"}}]}` + "\n\n" +
				`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
				"data: [DONE]\n"))
		}
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	c := New(prof, "system prompt")
	sink := &recSink{}

	if err := c.RunTurn(context.Background(), "use the tool", sink); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if requests != 2 {
		t.Fatalf("want 2 rounds, got %d", requests)
	}
	if len(sink.tools) != 1 || !strings.HasPrefix(sink.tools[0], "test_echo→echoed:") {
		t.Fatalf("tool events: %v", sink.tools)
	}
	if strings.Join(sink.tokens, "") != "done" {
		t.Fatalf("tokens: %v", sink.tokens)
	}
	// system + user + assistant(calls) + tool + assistant(answer)
	if len(c.msgs) != 5 {
		t.Fatalf("transcript length = %d, want 5: %+v", len(c.msgs), c.msgs)
	}
}

// A model that never stops calling tools trips the backstop, and the
// caller can tell that apart from other failures.
func TestRunTurnStepLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	c := New(prof, "sys")
	c.maxSteps = 3

	err := c.RunTurn(context.Background(), "loop forever", &recSink{})
	if !errors.Is(err, ErrStepLimit) {
		t.Fatalf("want ErrStepLimit, got: %v", err)
	}
	// 3 rounds ran: system + user + 3×(assistant + tool result).
	if len(c.msgs) != 8 {
		t.Fatalf("transcript length = %d, want 8", len(c.msgs))
	}
}
