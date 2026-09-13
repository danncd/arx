package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"arx/internal/provider"
	"arx/internal/tool"
)

type recSink struct {
	tokens   []string
	tools    []string
	failures []bool
}

func (r *recSink) Token(s string, _ bool)   { r.tokens = append(r.tokens, s) }
func (r *recSink) ToolStart(string, string) {}
func (r *recSink) ToolResult(name, out string, _ time.Duration, failed bool) {
	r.tools = append(r.tools, name+"→"+out)
	r.failures = append(r.failures, failed)
}

type cancelSink struct {
	recSink
	cancel context.CancelFunc
}

func (s *cancelSink) ToolResult(name, out string, _ time.Duration, failed bool) {
	s.recSink.ToolResult(name, out, 0, failed)
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

	prof := provider.Profile{
		Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL},
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
	if len(c.msgs) != 5 {
		t.Fatalf("transcript length = %d, want 5: %+v", len(c.msgs), c.msgs)
	}
}

func TestRunTurnStepLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]}}]}` + "\n\n" +
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
	c := New(prof, "sys")
	c.maxSteps = 3

	err := c.RunTurn(context.Background(), "loop forever", &recSink{})
	if !errors.Is(err, ErrStepLimit) {
		t.Fatalf("want ErrStepLimit, got: %v", err)
	}
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

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
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

	prof := provider.Profile{
		Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL},
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

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	if err := New(prof, "sys").RunTurn(context.Background(), "hello", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
}

func TestRunTurnRejectsDisabledToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"visible","tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{
		Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL},
		Model:    "m", ToolsKnown: true,
	}
	c := New(prof, "sys")
	sink := &recSink{}
	err := c.RunTurn(context.Background(), "hello", sink)
	if err == nil || !strings.Contains(err.Error(), "tools are disabled") {
		t.Fatalf("disabled tool-call error = %v", err)
	}
	if len(sink.tools) != 0 || len(c.msgs) != 3 || c.msgs[2].Role != "assistant" || c.msgs[2].Content != "visible" || len(c.msgs[2].ToolCalls) != 0 {
		t.Fatalf("disabled tool call entered the transcript: sink=%v msgs=%+v", sink.tools, c.msgs)
	}
}

func TestRunTurnRejectsCanceledEntryWithoutMutation(t *testing.T) {
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: "http://unused.invalid"}, Model: "m"}
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
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: "http://unused.invalid"}, Model: "m"}
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

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	err := c.RunTurn(context.Background(), "run", &recSink{})
	if err == nil || !strings.Contains(err.Error(), "invalid UTF-8") {
		t.Fatalf("invalid tool output error = %v", err)
	}
	if len(c.msgs) != 4 || c.msgs[3].ToolCallID != "c1" || !strings.Contains(c.msgs[3].Content, "invalid UTF-8") {
		t.Fatalf("invalid tool output left a broken exchange: %+v", c.msgs)
	}
}

func TestRunTurnReplaysReasoningWithToolCall(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		if requests == 2 && !strings.Contains(string(body), `"reasoning_content":"chain"`) {
			t.Errorf("reasoning missing from tool replay: %s", body)
		}
		if requests == 1 {
			w.Write([]byte(`data: {"choices":[{"delta":{"reasoning_content":"chain","tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
			return
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	if err := New(prof, "sys").RunTurn(context.Background(), "run", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
}

func TestRunTurnKeepsDisplayedPartialAnswer(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		if requests == 1 {
			w.Write([]byte(`data: {"choices":[{"delta":{"content":"partial","tool_calls":[{"index":0,"id":"c-broken","function":{"name":"test_echo","arguments":"{"}}]}}]}` + "\n\n" +
				`data: {"error":{"message":"upstream died"}}` + "\n\n"))
			return
		}
		raw := string(body)
		first := strings.Index(raw, `"content":"first"`)
		partial := strings.Index(raw, `"role":"assistant","content":"partial"`)
		second := strings.Index(raw, `"content":"second"`)
		if first < 0 || partial < first || second < partial {
			t.Errorf("partial answer missing or out of order: %s", body)
		}
		if strings.Contains(raw, "c-broken") {
			t.Errorf("incomplete tool call entered history: %s", body)
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	if err := c.RunTurn(context.Background(), "first", &recSink{}); err == nil {
		t.Fatal("broken first stream must fail")
	}
	if err := c.RunTurn(context.Background(), "second", &recSink{}); err != nil {
		t.Fatalf("second RunTurn: %v", err)
	}
}

func TestRunTurnReportsToolFailureExplicitly(t *testing.T) {
	tool.Register(tool.Tool{
		Name: "test_error_text",
		Run: func(context.Context, json.RawMessage) (string, error) {
			return "error: this is valid output", nil
		},
	})
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_error_text","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
			return
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	sink := &recSink{}
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	if err := New(prof, "sys").RunTurn(context.Background(), "run", sink); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if len(sink.failures) != 1 || sink.failures[0] {
		t.Fatalf("successful output was marked failed: %+v", sink)
	}
}

func TestSteerFoldsIntoTurn(t *testing.T) {
	var c *Controller
	var once sync.Once
	steered := make(chan struct{})

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		body, _ := io.ReadAll(r.Body)
		switch requests {
		case 1:
			once.Do(func() {
				if !c.Steer("also check the tests") {
					t.Error("steer was rejected during an active turn")
				}
				close(steered)
			})
			w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
		default:
			if !strings.Contains(string(body), "also check the tests") {
				t.Errorf("steer not folded into the next request: %s", body)
			}
			w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
				"data: [DONE]\n"))
		}
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c = New(prof, "sys")
	if err := c.RunTurn(context.Background(), "start", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	<-steered
	if requests != 2 {
		t.Fatalf("want 2 rounds, got %d", requests)
	}
}

func TestSteerRejectedOutsideTurn(t *testing.T) {
	c := New(provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}}, Model: "m"}, "sys")
	if c.Steer("nobody home") {
		t.Fatal("steer accepted with no turn running")
	}
}

func TestSteerDroppedAfterTurn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	if err := c.RunTurn(context.Background(), "hi", &recSink{}); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if c.Steer("late") {
		t.Fatal("steer accepted after the turn ended")
	}
}
