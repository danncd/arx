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
	"time"

	"arx/internal/llm"
	"arx/internal/tool"
)

/* Records turn events. */

type recSink struct {
	tokens []string
	tools  []string
}

func (r *recSink) Token(s string, _ bool) { r.tokens = append(r.tokens, s) }
func (r *recSink) ToolResult(name, out string, _ time.Duration) {
	r.tools = append(r.tools, name+"→"+out)
}

type cancelSink struct {
	recSink
	cancel context.CancelFunc
}

func (s *cancelSink) ToolResult(name, out string, _ time.Duration) {
	s.recSink.ToolResult(name, out, 0)
	if len(s.tools) == 1 {
		s.cancel()
	}
}

func init() {
	tool.Register(tool.Tool{
		Name:        "test_echo",
		Description: "echoes for tests",
		Run: func(_ context.Context, args json.RawMessage) (string, error) {
			return "echoed:" + string(args), nil
		},
	})
}

/* Runs a full tool round. */

func TestRunTurnToolRound(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		switch requests {
		case 1:
			if !strings.Contains(string(body), `"tools"`) {
				t.Errorf("supported tools missing from request: %s", body)
			}
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

	prof := llm.Profile{
		Provider: llm.Provider{Name: "fake", BaseURL: srv.URL},
		Model:    "m", MaxTokens: 100, Tools: true, ToolsKnown: true,
	}
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

/* Stops a tool loop at the step limit. */

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

func TestRunTurnStopsToolBatchAfterCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}},{"index":1,"id":"c2","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	ctx, cancel := context.WithCancel(context.Background())
	sink := &cancelSink{cancel: cancel}

	err := c.RunTurn(ctx, "run both", sink)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context cancellation, got: %v", err)
	}
	if len(sink.tools) != 1 {
		t.Fatalf("tools executed after cancellation: %v", sink.tools)
	}
	if len(c.msgs) != 5 || c.msgs[4].ToolCallID != "c2" || !strings.Contains(c.msgs[4].Content, "context canceled") {
		t.Fatalf("canceled tool exchange is incomplete: %+v", c.msgs)
	}
}

func TestRunTurnDoesNotOfferUnsupportedTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), `"tools"`) {
			t.Errorf("request offered unsupported tools: %s", body)
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{
		Provider: llm.Provider{Name: "fake", BaseURL: srv.URL},
		Model:    "m", ToolsKnown: true,
	}
	if err := New(prof, "sys").RunTurn(context.Background(), "hello", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
}

func TestRunTurnOffersToolsWhenCapabilityIsUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"tools"`) {
			t.Errorf("unknown capability disabled tools: %s", body)
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}
	if err := New(prof, "sys").RunTurn(context.Background(), "hello", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
}

func TestRunTurnRejectsDisabledToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{
		Provider: llm.Provider{Name: "fake", BaseURL: srv.URL},
		Model:    "m", ToolsKnown: true,
	}
	c := New(prof, "sys")
	sink := &recSink{}
	err := c.RunTurn(context.Background(), "hello", sink)
	if err == nil || !strings.Contains(err.Error(), "tools are disabled") {
		t.Fatalf("disabled tool-call error = %v", err)
	}
	if len(sink.tools) != 0 || len(c.msgs) != 2 {
		t.Fatalf("disabled tool call entered the transcript: sink=%v msgs=%+v", sink.tools, c.msgs)
	}
}

func TestRunTurnRejectsCanceledEntryWithoutMutation(t *testing.T) {
	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: "http://unused.invalid"}, Model: "m"}
	c := New(prof, "sys")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.RunTurn(ctx, "must not stick", &recSink{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context cancellation, got: %v", err)
	}
	if len(c.msgs) != 1 {
		t.Fatalf("canceled user message entered history: %+v", c.msgs)
	}
}

func TestRunTurnRejectsInvalidUTF8(t *testing.T) {
	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: "http://unused.invalid"}, Model: "m"}
	bad := string([]byte{'a', 0xff, 'b'})

	if err := New(prof, bad).RunTurn(context.Background(), "hello", &recSink{}); err == nil || !strings.Contains(err.Error(), "system prompt") {
		t.Fatalf("invalid system prompt error = %v", err)
	}
	c := New(prof, "sys")
	if err := c.RunTurn(context.Background(), bad, &recSink{}); err == nil || !strings.Contains(err.Error(), "user message") {
		t.Fatalf("invalid user message error = %v", err)
	}
	if len(c.msgs) != 1 {
		t.Fatalf("invalid user message entered history: %+v", c.msgs)
	}
}

func TestRunTurnRejectsInvalidToolOutput(t *testing.T) {
	tool.Register(tool.Tool{
		Name: "test_invalid_utf8",
		Run: func(context.Context, json.RawMessage) (string, error) {
			return string([]byte{0xff}), nil
		},
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_invalid_utf8","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := llm.Profile{Provider: llm.Provider{Name: "fake", BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	err := c.RunTurn(context.Background(), "run", &recSink{})
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("invalid tool output error = %v", err)
	}
	if len(c.msgs) != 4 || c.msgs[3].ToolCallID != "c1" || !strings.Contains(c.msgs[3].Content, "invalid UTF-8") {
		t.Fatalf("invalid tool output left a broken exchange: %+v", c.msgs)
	}
}
