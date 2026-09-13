package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type fixedApprover struct {
	outcome Outcome
	asks    int
}

func (f *fixedApprover) Ask(context.Context, Request) (Outcome, error) {
	f.asks++
	return f.outcome, nil
}

func init() {
	tool.Register(tool.Tool{
		Name:     "test_mutate",
		Mutating: true,
		Run: func(context.Context, json.RawMessage) (string, error) {
			return "mutated", nil
		},
	})
}

func mutateOnceServer(t *testing.T, onSecond func(body string)) *httptest.Server {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_mutate","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
			return
		}
		if onSecond != nil {
			body, _ := io.ReadAll(r.Body)
			onSecond(string(body))
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestDeniedCallContinuesTurn(t *testing.T) {
	srv := mutateOnceServer(t, func(body string) {
		if !strings.Contains(body, `error: denied: judge: unsafe`) {
			t.Errorf("denial missing from the model view: %s", body)
		}
	})
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	c.SetGate(NewGate(&fixedApprover{outcome: Reject}, &fakeJudge{allow: false, reason: "unsafe"}))
	sink := &recSink{}

	if err := c.RunTurn(context.Background(), "try it", sink); err != nil {
		t.Fatalf("denied call ended the turn: %v", err)
	}
	if len(sink.failures) != 1 || !sink.failures[0] {
		t.Fatalf("denial was not surfaced as a failed tool: %+v", sink)
	}
	if strings.Join(sink.tokens, "") != "done" {
		t.Fatalf("turn did not continue past the denial: %v", sink.tokens)
	}
}

func TestApprovedCallRuns(t *testing.T) {
	srv := mutateOnceServer(t, nil)
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	ap := &fixedApprover{outcome: Once}
	c.SetGate(NewGate(ap, nil))
	sink := &recSink{}

	if err := c.RunTurn(context.Background(), "try it", sink); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if ap.asks != 1 || len(sink.tools) != 1 || !strings.Contains(sink.tools[0], "mutated") {
		t.Fatalf("approved call did not run: asks=%d tools=%v", ap.asks, sink.tools)
	}
}

func TestNilGateFailsClosedForMutations(t *testing.T) {
	srv := mutateOnceServer(t, nil)
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	sink := &recSink{}

	if err := c.RunTurn(context.Background(), "try it", sink); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if len(sink.tools) != 1 || !strings.Contains(sink.tools[0], "no permission gate") {
		t.Fatalf("nil gate did not deny: %v", sink.tools)
	}
}

func TestReadsBypassTheGate(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","function":{"name":"test_echo","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\n" +
				"data: [DONE]\n"))
			return
		}
		w.Write([]byte(`data: {"choices":[{"delta":{"content":"done"},"finish_reason":"stop"}]}` + "\n\n" +
			"data: [DONE]\n"))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	ap := &fixedApprover{outcome: Reject}
	c.SetGate(NewGate(ap, nil))
	sink := &recSink{}

	if err := c.RunTurn(context.Background(), "read", sink); err != nil {
		t.Fatalf("RunTurn: %v", err)
	}
	if ap.asks != 0 || len(sink.failures) != 1 || sink.failures[0] {
		t.Fatalf("read was gated: asks=%d %+v", ap.asks, sink)
	}
}

type cancelingApprover struct {
	cancel context.CancelFunc
}

func (a *cancelingApprover) Ask(ctx context.Context, _ Request) (Outcome, error) {
	a.cancel()
	<-ctx.Done()
	return Reject, ctx.Err()
}

func TestCancelDuringPromptUnwinds(t *testing.T) {
	srv := mutateOnceServer(t, nil)
	prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m"}
	c := New(prof, "sys")
	ctx, cancel := context.WithCancel(context.Background())
	c.SetGate(NewGate(&cancelingApprover{cancel: cancel}, nil))

	err := c.RunTurn(ctx, "try it", &recSink{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want cancellation, got %v", err)
	}
	last := c.msgs[len(c.msgs)-1]
	if last.Role != "tool" || last.ToolCallID != "c1" {
		t.Fatalf("canceled prompt left a dangling tool call: %+v", last)
	}
}

func TestJudgeVerdictParsing(t *testing.T) {
	replies := map[string]struct {
		allow  bool
		reason string
	}{
		"ALLOW":              {true, ""},
		"ALLOW\nextra prose": {true, ""},
		"ALLOW: it is safe":  {true, ""},
		"Allow":              {true, ""},
		"DENY: touches prod": {false, "touches prod"},
		"DENY":               {false, ""},
		"deny it edits prod": {false, "it edits prod"},
		"sure, sounds fine":  {false, "unparseable verdict"},
		"  DENY: spaced  ":   {false, "spaced"},
		"ALLOWED":            {false, "unparseable verdict"},
		"DENYING: x":         {false, "unparseable verdict"},
	}
	for content, want := range replies {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]any{"choices": []any{map[string]any{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			}}}
			json.NewEncoder(w).Encode(resp)
		}))
		prof := provider.Profile{Provider: provider.Provider{Name: "fake", Dialect: provider.OpenAIDialect{}, BaseURL: srv.URL}, Model: "m", MaxTokens: 100}
		j := NewJudge(prof)
		allow, reason, err := j.Judge(context.Background(), Request{Tool: "bash", Args: "{}", Intent: "g"})
		srv.Close()
		if err != nil {
			t.Fatalf("%q: %v", content, err)
		}
		if allow != want.allow || reason != want.reason {
			t.Fatalf("%q = %v %q, want %v %q", content, allow, reason, want.allow, want.reason)
		}
	}
}

type barrierJudge struct {
	mu   sync.Mutex
	need int
	seen int
	open chan struct{}
}

func newBarrierJudge(need int) *barrierJudge {
	return &barrierJudge{need: need, open: make(chan struct{})}
}

func (b *barrierJudge) Judge(context.Context, Request) (bool, string, error) {
	b.mu.Lock()
	b.seen++
	if b.seen >= b.need {
		select {
		case <-b.open:
		default:
			close(b.open)
		}
	}
	b.mu.Unlock()
	select {
	case <-b.open:
		return true, "", nil
	case <-time.After(3 * time.Second):
		return false, "barrier timed out", nil
	}
}

func (b *barrierJudge) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.seen
}

type keyedJudge struct {
	mu    sync.Mutex
	calls []string
}

func (k *keyedJudge) Judge(_ context.Context, req Request) (bool, string, error) {
	k.mu.Lock()
	k.calls = append(k.calls, req.Args)
	k.mu.Unlock()
	if strings.Contains(req.Args, "deny-me") {
		return false, "nope", nil
	}
	return true, "", nil
}

func (k *keyedJudge) count() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.calls)
}

func TestAuthorizeAllJudgesConcurrently(t *testing.T) {
	const n = 4
	judge := newBarrierJudge(n)
	g := NewGate(&fakeApprover{outcome: Reject}, judge)

	reqs := make([]Request, n)
	mutating := make([]bool, n)
	for i := range reqs {
		reqs[i] = bashReq(fmt.Sprintf("echo %d", i))
		mutating[i] = true
	}

	dec, err := g.AuthorizeAll(context.Background(), reqs, mutating)
	if err != nil {
		t.Fatalf("AuthorizeAll: %v", err)
	}
	for i, d := range dec {
		if !d.Allow {
			t.Fatalf("decision %d denied (%q): judging was not concurrent", i, d.Reason)
		}
	}
	if judge.count() != n {
		t.Fatalf("judge calls = %d, want %d", judge.count(), n)
	}
}

func TestAuthorizeAllMixedDecisions(t *testing.T) {
	ap := &fakeApprover{outcome: Reject}
	judge := &keyedJudge{}
	g := NewGate(ap, judge)

	reqs := []Request{
		{Tool: "bash", Args: `{"command":"ls"}`},
		bashReq("rm -rf /"),
		bashReq("echo deny-me"),
		bashReq("go test ./..."),
	}
	mutating := []bool{false, true, true, true}

	dec, err := g.AuthorizeAll(context.Background(), reqs, mutating)
	if err != nil {
		t.Fatalf("AuthorizeAll: %v", err)
	}
	if len(dec) != len(reqs) {
		t.Fatalf("decisions = %d, want %d", len(dec), len(reqs))
	}
	if !dec[0].Allow {
		t.Fatalf("non-mutating call was gated: %q", dec[0].Reason)
	}
	if dec[1].Allow || !strings.HasPrefix(dec[1].Reason, "blocked: ") {
		t.Fatalf("hard-deny hit was not floored: allow=%v reason=%q", dec[1].Allow, dec[1].Reason)
	}
	if dec[2].Allow || dec[2].Reason != "judge: nope" {
		t.Fatalf("judge denial was not carried: allow=%v reason=%q", dec[2].Allow, dec[2].Reason)
	}
	if !dec[3].Allow {
		t.Fatalf("judge allow was denied: %q", dec[3].Reason)
	}
	if judge.count() != 2 {
		t.Fatalf("judge calls = %d, want 2 (only the judgeable calls)", judge.count())
	}
}

func TestAuthorizeAllSessionGrantSkipsJudge(t *testing.T) {
	ap := &fakeApprover{outcome: AlwaysSession}
	judge := &keyedJudge{}
	g := NewGate(ap, judge)
	reqs := []Request{bashReq("echo deny-me")}
	mutating := []bool{true}

	first, err := g.AuthorizeAll(context.Background(), reqs, mutating)
	if err != nil {
		t.Fatalf("AuthorizeAll: %v", err)
	}
	if !first[0].Allow {
		t.Fatalf("granted call denied: %q", first[0].Reason)
	}
	if ap.asks != 1 {
		t.Fatalf("approver asks = %d, want 1", ap.asks)
	}
	before := judge.count()

	second, err := g.AuthorizeAll(context.Background(), reqs, mutating)
	if err != nil {
		t.Fatalf("AuthorizeAll: %v", err)
	}
	if !second[0].Allow {
		t.Fatalf("granted call denied on replay: %q", second[0].Reason)
	}
	if judge.count() != before {
		t.Fatalf("judge re-ran after a session grant: %d -> %d", before, judge.count())
	}
	if ap.asks != 1 {
		t.Fatalf("approver re-asked after a session grant: %d", ap.asks)
	}
}

func TestJudgeDisablesThinkingForDialectsThatSupportIt(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ALLOW"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{
		Provider: provider.Provider{
			Name:    "fake",
			BaseURL: srv.URL,
			Dialect: provider.OpenAIDialect{ThinkingParam: true},
		},
		Model:     "m",
		MaxTokens: 100,
	}
	allow, _, err := NewJudge(prof).Judge(context.Background(), Request{
		Tool: "bash", Args: `{"command":"ls"}`, Intent: "survey",
	})
	if err != nil || !allow {
		t.Fatalf("judge: allow=%v err=%v", allow, err)
	}
	if !strings.Contains(body, `"thinking":{"type":"disabled"}`) {
		t.Fatalf("judge request did not disable thinking: %s", body)
	}
}

func TestJudgeOmitsThinkingForDialectsWithoutIt(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ALLOW"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(srv.Close)

	prof := provider.Profile{
		Provider:  provider.Provider{Name: "fake", BaseURL: srv.URL, Dialect: provider.OpenAIDialect{}},
		Model:     "m",
		MaxTokens: 100,
	}
	if _, _, err := NewJudge(prof).Judge(context.Background(), Request{
		Tool: "bash", Args: `{"command":"ls"}`, Intent: "survey",
	}); err != nil {
		t.Fatalf("judge: %v", err)
	}
	if strings.Contains(body, "thinking") {
		t.Fatalf("thinking leaked into a dialect that has no such field: %s", body)
	}
}
