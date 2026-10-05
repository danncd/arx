package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/settings"
	tool "arx/internal/tools"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestAutoContinueToolLimitResumesWithoutReplayingTools(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			requests, executions, continuations := 0, 0, 0
			seen := map[string]bool{}
			var last session.TranscriptChunk
			loop := Loop{Save: func(c session.TranscriptChunk) error { last = c; return nil }, Emit: func(session.TranscriptChunk) {}, Recovering: func(kind string, attempt int) {
				if kind != "tool_limit" || attempt != 1 {
					t.Fatalf("unexpected recovery %s %d", kind, attempt)
				}
				continuations++
			}, Tool: func(_ context.Context, c tool.Call) tool.Result {
				if seen[c.ID] {
					t.Fatal("tool replayed")
				}
				seen[c.ID] = true
				executions++
				return tool.Output("done")
			}, Complete: func(_ context.Context, r provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
				requests++
				if requests <= 32 {
					return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: fmt.Sprint(requests), Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"file"}`}}}}}, nil
				}
				if !strings.Contains(r.Messages[0].Content, "Completed tool actions remain completed") {
					t.Fatal("missing continuation guidance")
				}
				emit(provider.Delta{Text: "Finished"})
				return provider.Response{Message: provider.Message{Role: "assistant", Content: "Finished"}}, nil
			}}
			err := loop.Run(context.Background(), Input{Settings: settings.Values{AutoContinue: enabled}, Model: model.Info{ID: "test", ContextWindow: 100000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Finish the task"}}})
			if err != nil || executions != 32 {
				t.Fatal(err, executions)
			}
			if enabled && (requests != 33 || continuations != 1 || last.Failed) {
				t.Fatal(requests, continuations, last)
			}
			if !enabled && (requests != 32 || continuations != 0 || !last.Failed) {
				t.Fatal(requests, continuations, last)
			}
		})
	}
}
func TestAutoContinueHasThreeContinuationLimit(t *testing.T) {
	calls, recoveries := 0, 0
	loop := Loop{Save: func(session.TranscriptChunk) error { return nil }, Emit: func(session.TranscriptChunk) {}, Recovering: func(kind string, attempt int) {
		if kind != "tool_limit" || attempt > 3 {
			t.Fatal(kind, attempt)
		}
		recoveries++
	}, Tool: func(context.Context, tool.Call) tool.Result { return tool.Output("done") }, Complete: func(_ context.Context, _ provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: fmt.Sprint(calls), Type: "function", Function: provider.Function{Name: "files", Arguments: `{}`}}}}}, nil
	}}
	if err := loop.Run(context.Background(), Input{Settings: settings.Values{AutoContinue: true}, Model: model.Info{ID: "test", ContextWindow: 1000000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Task"}}}); err != nil {
		t.Fatal(err)
	}
	if calls != 128 || recoveries != 3 {
		t.Fatal(calls, recoveries)
	}
}
func TestAutoRecoverOutputLimitKeepsPartialAndNeverExecutesUnfinishedCall(t *testing.T) {
	calls, tools := 0, 0
	loop := Loop{Save: func(session.TranscriptChunk) error { return nil }, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result { tools++; return tool.Output("done") }, Complete: func(_ context.Context, r provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		calls++
		if calls == 1 {
			emit(provider.Delta{Text: "Partial work"})
			emit(provider.Delta{Tool: &provider.ToolDelta{Index: 0, Call: provider.Call{ID: "unfinished", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"write","path":"file","content":"partial`}}}})
			return provider.Response{}, provider.ErrOutputLimit
		}
		found := false
		for _, m := range r.Messages {
			if len(m.Calls) > 0 {
				t.Fatal("unfinished tool replayed")
			}
			if strings.Contains(m.Content, "Partial work") {
				found = true
			}
		}
		if !found || r.Effort != "none" {
			t.Fatal("lost partial or thinking recovery", r.Effort)
		}
		emit(provider.Delta{Text: "Finished"})
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Finished"}}, nil
	}}
	err := loop.Run(context.Background(), Input{Settings: settings.Values{AutoContinue: true, Run: settings.Run{Effort: "high"}}, Model: model.Info{ID: "test", ContextWindow: 32000, MaxOutputTokens: 4000, Thinking: &model.Thinking{CanDisable: true}}, History: []provider.Message{{Role: "user", Content: "Continue my task"}}})
	if err != nil || calls != 2 || tools != 0 {
		t.Fatal(err, calls, tools)
	}
}
func TestAutoRecoveryExhaustionAndCancellationAreBounded(t *testing.T) {
	for _, failure := range []error{provider.ErrOutputLimit, context.Canceled, errors.New("credentials rejected")} {
		calls := 0
		loop := Loop{Save: func(session.TranscriptChunk) error { return nil }, Emit: func(session.TranscriptChunk) {}, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
			calls++
			return provider.Response{}, failure
		}}
		err := loop.Run(context.Background(), Input{Settings: settings.Values{AutoContinue: true}, Model: model.Info{ID: "test", ContextWindow: 32000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Task"}}})
		want := 1
		if errors.Is(failure, provider.ErrOutputLimit) {
			want = 4
		}
		if !errors.Is(err, failure) || calls != want {
			t.Fatal(err, calls, want)
		}
	}
}

func TestOutputLimitAtLastAutomaticRoundIsReported(t *testing.T) {
	calls := 0
	loop := Loop{Save: func(session.TranscriptChunk) error { return nil }, Emit: func(session.TranscriptChunk) {}, Tool: func(context.Context, tool.Call) tool.Result { return tool.Output("done") }, Complete: func(_ context.Context, _ provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if calls == 128 {
			return provider.Response{}, provider.ErrOutputLimit
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: fmt.Sprint(calls), Type: "function", Function: provider.Function{Name: "files", Arguments: `{}`}}}}}, nil
	}}
	err := loop.Run(context.Background(), Input{Settings: settings.Values{AutoContinue: true}, Model: model.Info{ID: "test", ContextWindow: 1000000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Task"}}})
	if !errors.Is(err, provider.ErrOutputLimit) || calls != 128 {
		t.Fatal(err, calls)
	}
}
