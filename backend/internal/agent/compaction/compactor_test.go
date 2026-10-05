package compaction

import (
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	tool "arx/internal/tools"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func testModel() model.Info {
	return model.Info{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000, Thinking: &model.Thinking{CanDisable: true}}
}

func longHistory() []provider.Message {
	history := []provider.Message{}
	for index := 0; index < 20; index++ {
		id := fmt.Sprint(index)
		history = append(history,
			provider.Message{Role: "user", Content: "Read file " + id},
			provider.Message{Role: "assistant", Content: "Reading", Reasoning: "Internal thought", Calls: []provider.Call{{ID: id, Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read"}`}}}},
			provider.Message{Role: "tool", CallID: id, Content: strings.Repeat("file contents 🌱 ", 250)},
			provider.Message{Role: "assistant", Content: "Checked file " + id},
		)
	}
	return append(history, provider.Message{Role: "user", Content: "Continue; do not delete files."})
}

func request(history []provider.Message) provider.Request {
	return provider.Request{Model: "test", Effort: "high", Messages: append([]provider.Message{{Role: "system", Content: "System instructions"}}, history...)}
}

func TestCompactionBatchesPreserveRecentHistoryAndReuseCheckpoint(t *testing.T) {
	history := longHistory()
	original := append([]provider.Message(nil), history...)
	budget, _ := contextwindow.New(testModel())
	calls, saves := 0, 0
	var activity []bool
	manager := Compactor{Model: testModel(), Active: func(active bool) { activity = append(activity, active) }, Save: func(state session.Compaction) error {
		saves++
		if !Valid(state, history) {
			t.Fatal("invalid checkpoint")
		}
		return nil
	}, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if !budget.Fits(input) || len(input.Tools) != 0 || input.Effort != "none" || input.MaxOutputTokens <= 0 {
			t.Fatalf("unsafe summary request: %+v", input)
		}
		if strings.Contains(input.Messages[1].Content, "Internal thought") {
			t.Fatal("unnecessary reasoning included")
		}
		if calls > 1 && !strings.Contains(input.Messages[1].Content, "previous_summary") {
			t.Fatal("lost earlier summary")
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Read files. Never delete files. Continue checking."}}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 || saves != calls || !reflect.DeepEqual(activity, []bool{true, false}) {
		t.Fatalf("batch lifecycle: %d %d %v", calls, saves, activity)
	}
	if !budget.Fits(result) || result.MaxOutputTokens != budget.Output {
		t.Fatal("unbounded request")
	}
	if !reflect.DeepEqual(history, original) {
		t.Fatal("history mutated")
	}
	if _, err := boundaries(result.Messages[1:]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Messages[len(result.Messages)-4:], history[len(history)-4:]) {
		t.Fatal("recent messages were not retained")
	}
	before := calls
	resumed, err := manager.Prepare(context.Background(), request(history))
	if err != nil || calls != before || !reflect.DeepEqual(result, resumed) {
		t.Fatal("saved summary was not reused", err)
	}
	manager.Model.ContextWindow = 8000
	smaller, err := manager.Prepare(context.Background(), request(history))
	smallBudget, _ := contextwindow.New(manager.Model)
	if err != nil || !smallBudget.Fits(smaller) {
		t.Fatal("model change was not re-budgeted", err)
	}
}

func TestSmallConversationAndStaleSummary(t *testing.T) {
	history := []provider.Message{{Role: "user", Content: "Hello"}}
	manager := Compactor{Model: testModel(), State: session.Compaction{Version: 1, Through: 1, Digest: "stale", Summary: "Wrong history"}, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		t.Fatal("small conversation compacted")
		return provider.Response{}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil || !reflect.DeepEqual(result.Messages, request(history).Messages) {
		t.Fatal("stale summary used", err)
	}
	if manager.State.Through != 0 {
		t.Fatal("stale summary retained")
	}
}

func TestCompactionFailuresDoNotAdvanceCoverage(t *testing.T) {
	cases := []struct {
		name                 string
		output               provider.Response
		completeErr, saveErr error
	}{
		{name: "network", completeErr: errors.New("offline")},
		{name: "empty"},
		{name: "too large", output: provider.Response{Message: provider.Message{Content: strings.Repeat("x", 90000)}}},
		{name: "tool call", output: provider.Response{Message: provider.Message{Content: "summary", Calls: []provider.Call{{ID: "unexpected"}}}}},
		{name: "storage", output: provider.Response{Message: provider.Message{Content: "summary"}}, saveErr: errors.New("disk full")},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			history := longHistory()
			old := session.Compaction{Version: 1, Through: 4, Digest: digest(history[:4]), Summary: "Previous summary"}
			manager := Compactor{Model: testModel(), State: old, Save: func(session.Compaction) error { return item.saveErr }, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
				return item.output, item.completeErr
			}}
			if _, err := manager.Prepare(context.Background(), request(history)); err == nil {
				t.Fatal("failure ignored")
			}
			if manager.State != old {
				t.Fatal("checkpoint advanced after failure")
			}
		})
	}
}

func TestCancelledCompactionKeepsPreviousCheckpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	saved, active := false, false
	manager := Compactor{Model: testModel(), Active: func(value bool) { active = value }, Save: func(session.Compaction) error { saved = true; return nil }, Complete: func(ctx context.Context, _ provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		if !active {
			t.Fatal("compaction status missing")
		}
		cancel()
		return provider.Response{}, emit(provider.Delta{Text: "partial"})
	}}
	if _, err := manager.Prepare(ctx, request(longHistory())); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if saved || active || manager.State.Through != 0 {
		t.Fatal("cancelled summary committed or status stuck")
	}
}

func TestLatestUserSurvivesCompactionWithinToolLoop(t *testing.T) {
	user := provider.Message{Role: "user", Content: "Read all files; never write."}
	history := []provider.Message{user}
	for index := 0; index < 20; index++ {
		id := fmt.Sprint(index)
		history = append(history, provider.Message{Role: "assistant", Calls: []provider.Call{{ID: id, Function: provider.Function{Name: "files", Arguments: "{}"}}}}, provider.Message{Role: "tool", CallID: id, Content: strings.Repeat("contents", 300)})
	}
	manager := Compactor{Model: testModel(), Save: func(session.Compaction) error { return nil }, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		return provider.Response{Message: provider.Message{Content: "Files were read."}}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Messages[2], user) {
		t.Fatal("current user request was not kept verbatim")
	}
	if _, err := boundaries(result.Messages[1:]); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsIncompleteOrOversizedAtomicHistory(t *testing.T) {
	for _, history := range [][]provider.Message{
		{{Role: "assistant", Calls: []provider.Call{{ID: "missing"}}}},
		{{Role: "tool", CallID: "orphan"}},
		{{Role: "user", Content: strings.Repeat("x", 90000)}},
	} {
		manager := Compactor{Model: testModel(), Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
			t.Fatal("unsafe model request sent")
			return provider.Response{}, nil
		}}
		if _, err := manager.Prepare(context.Background(), request(history)); err == nil {
			t.Fatal("invalid history accepted")
		}
	}
}

func TestLargeContextCompactsToSmallTargetAndUpgradesOldCheckpoint(t *testing.T) {
	info := model.Info{ID: "test", ContextWindow: 1048576, MaxOutputTokens: 393216}
	history := []provider.Message{}
	for index := 0; index < 32; index++ {
		history = append(history, provider.Message{Role: "user", Content: fmt.Sprint(index) + strings.Repeat("past context ", 7200)}, provider.Message{Role: "assistant", Content: "Checked."})
	}
	latest := provider.Message{Role: "user", Content: "Continue without modifying files."}
	history = append(history, latest)
	original := request(history)
	original.Tools = tool.Definitions()
	for _, state := range []session.Compaction{
		{},
		{Version: 1, Through: 36, Digest: digest(history[:36]), Summary: "Previous half-window summary"},
	} {
		summaries := 0
		manager := Compactor{Model: info, State: state, Save: func(session.Compaction) error { return nil }, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
			summaries++
			budget, _ := contextwindow.New(info)
			if !budget.Fits(input) {
				t.Fatal("oversized summary request")
			}
			return provider.Response{Message: provider.Message{Content: strings.Repeat("S", 3300)}}, nil
		}}
		result, err := manager.Prepare(context.Background(), original)
		if err != nil {
			t.Fatal(err)
		}
		if size := contextwindow.Estimate(result); size > recentTokens+summaryTokens+contextwindow.Estimate(provider.Request{Messages: original.Messages[:1], Tools: original.Tools}) {
			t.Fatalf("compacted context still too large: %d", size)
		}
		if summaries == 0 || manager.State.Pending || manager.State.Target != recentTokens {
			t.Fatal("compaction did not finish its target", manager.State)
		}
		if !reflect.DeepEqual(result.Messages[len(result.Messages)-1], latest) {
			t.Fatal("latest request changed")
		}
		before := summaries
		if _, err := manager.Prepare(context.Background(), original); err != nil || summaries != before {
			t.Fatal("finished compaction repeated", err)
		}
	}
}

func TestPendingCompactionResumesBelowTrigger(t *testing.T) {
	info := model.Info{ID: "test", ContextWindow: 1048576, MaxOutputTokens: 32768}
	history := longHistory()
	state := session.Compaction{Version: 1, Through: 4, Digest: digest(history[:4]), Summary: "Partial summary", Target: recentTokens, Pending: true}
	calls := 0
	manager := Compactor{Model: info, State: state, Save: func(session.Compaction) error { return nil }, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		calls++
		return provider.Response{Message: provider.Message{Content: "Completed summary"}}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil || calls == 0 || manager.State.Pending || contextwindow.Estimate(result) > recentTokens+summaryTokens+1000 {
		t.Fatal("partial compaction did not finish", err)
	}
}

func TestNonReducingSummaryDoesNotReplaceCheckpoint(t *testing.T) {
	history := []provider.Message{{Role: "user", Content: "Hello"}, {Role: "assistant", Content: "Hi"}, {Role: "user", Content: "Continue"}}
	saved := false
	manager := Compactor{Model: testModel(), Force: true, Save: func(session.Compaction) error { saved = true; return nil }, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		return provider.Response{Message: provider.Message{Content: strings.Repeat("Repeated summary. ", 50)}}, nil
	}}
	if _, err := manager.Prepare(context.Background(), request(history)); err == nil || saved || manager.State.Through != 0 {
		t.Fatal("expanding summary was committed")
	}
}

func TestSummaryOverflowRetriesWithSmallerBatch(t *testing.T) {
	calls := 0
	firstSize := 0
	manager := Compactor{Model: testModel(), Save: func(session.Compaction) error { return nil }, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if calls == 1 {
			firstSize = contextwindow.Estimate(input)
			return provider.Response{}, provider.ErrContextLength
		}
		if calls == 2 && contextwindow.Estimate(input) >= firstSize {
			t.Fatal("overflow retried the same batch")
		}
		return provider.Response{Message: provider.Message{Content: "Files checked; continue without deleting."}}, nil
	}}
	if _, err := manager.Prepare(context.Background(), request(longHistory())); err != nil || calls < 2 {
		t.Fatal(err, calls)
	}
}

func TestSummaryOutputLimitRetriesWithSmallerBatch(t *testing.T) {
	calls := 0
	firstSize := 0
	manager := Compactor{Model: testModel(), Save: func(session.Compaction) error { return nil }, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if calls == 1 {
			firstSize = contextwindow.Estimate(input)
			return provider.Response{Usage: &session.ChunkUsage{Output: input.MaxOutputTokens}}, provider.ErrOutputLimit
		}
		if calls == 2 && contextwindow.Estimate(input) >= firstSize {
			t.Fatal("output limit retried the same batch")
		}
		return provider.Response{Message: provider.Message{Content: "Files checked; continue without deleting."}}, nil
	}}
	if _, err := manager.Prepare(context.Background(), request(longHistory())); err != nil || calls < 2 {
		t.Fatal(err, calls)
	}
}

func TestProviderSizedSummaryIsAccepted(t *testing.T) {
	history := longHistory()
	ends, err := boundaries(history)
	if err != nil {
		t.Fatal(err)
	}
	budget, err := contextwindow.New(testModel())
	if err != nil {
		t.Fatal(err)
	}
	summary := strings.Repeat("long summary text ", 500)
	manager := Compactor{Model: testModel(), Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		return provider.Response{Message: provider.Message{Content: summary}, Usage: &session.ChunkUsage{Output: 1970}}, nil
	}}
	cut, text, err := manager.summarize(context.Background(), "high", history, ends, budget, 16000, len(history))
	if err != nil || cut <= 0 || text != strings.TrimSpace(summary) {
		t.Fatal("provider-sized summary rejected", err)
	}
}

func TestRepeatedSummaryOutputLimitFallsBackAndSavesCheckpoint(t *testing.T) {
	history := []provider.Message{{Role: "user", Content: "Build a game and keep the existing files."}}
	for index := 0; index < 10; index++ {
		id := fmt.Sprint(index)
		history = append(history,
			provider.Message{Role: "assistant", Calls: []provider.Call{{ID: id, Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"game.html"}`}}}},
			provider.Message{Role: "tool", CallID: id, Content: strings.Repeat("large file contents ", 1000)},
		)
	}
	history = append(history, provider.Message{Role: "user", Content: "Continue the game."})
	info := model.Info{ID: "deepseek-flash", ContextWindow: 1048576, MaxOutputTokens: 32768, Thinking: &model.Thinking{CanDisable: true}}
	calls, saves := 0, 0
	manager := Compactor{Model: info, Force: true, Save: func(state session.Compaction) error {
		saves++
		if !Valid(state, history) || !strings.Contains(state.Summary, "game.html") {
			t.Fatal("invalid recovery checkpoint")
		}
		return nil
	}, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if input.MaxOutputTokens != 4096 {
			t.Fatalf("summary output budget = %d", input.MaxOutputTokens)
		}
		return provider.Response{}, provider.ErrOutputLimit
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil || saves == 0 || calls == 0 || !strings.Contains(result.Messages[1].Content, "Automatic recovery summary") {
		t.Fatalf("recovery failed: calls=%d saves=%d err=%v", calls, saves, err)
	}
}

func TestSummaryTrimsOldToolDataWithoutMutatingHistory(t *testing.T) {
	toolText := strings.Repeat("tool output ", 10000)
	history := []provider.Message{{Role: "tool", CallID: "read", Content: toolText}}
	output := summaryRequest(testModel(), "none", "", history, 16000)
	if len(output.Messages[1].Content) >= len(toolText) || history[0].Content != toolText {
		t.Fatal("historical tool output was not compacted safely")
	}
}
