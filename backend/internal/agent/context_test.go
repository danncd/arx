package agent

import (
	compactor "arx/internal/agent/compaction"
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/sessions/storage"
	tool "arx/internal/tools"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCompactionDuringToolsResumesFromStoredSummary(t *testing.T) {
	directory := t.TempDir()
	store, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 20; index++ {
		for _, role := range []string{"user", "assistant"} {
			if err := store.Append(session.TranscriptChunk{Conversation: "session", ID: fmt.Sprintf("%s-%d", role, index), Role: role, Text: strings.Repeat("Keep this history. ", 100), Status: "done"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	user := session.TranscriptChunk{Conversation: "session", ID: "current", Role: "user", Text: "Read the notes. Do not change files.", Status: "done"}
	if err := store.Append(user); err != nil {
		t.Fatal(err)
	}
	original := store.Entries("session")
	history, err := History(original)
	if err != nil {
		t.Fatal(err)
	}
	info := model.Info{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000, Thinking: &model.Thinking{CanDisable: true}}
	budget, _ := contextwindow.New(info)
	rounds, summaries, saved := 0, 0, 0
	firstSummaryCount := 0
	loop := Loop{Save: store.Append, Emit: func(session.TranscriptChunk) {}, SaveCompaction: func(state session.Compaction) error { saved++; return store.SaveCompaction("session", state) }, Tool: func(context.Context, tool.Call) tool.Result {
		return tool.Result{Text: strings.Repeat("Read output. ", 750)}
	}, Complete: func(_ context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		if !budget.Fits(request) || request.MaxOutputTokens <= 0 {
			t.Fatal("request exceeded its budget")
		}
		if len(request.Tools) == 0 {
			summaries++
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "The user asked to read notes without changes. Earlier files were checked."}, Usage: &session.ChunkUsage{Input: 100, Output: 20, Cached: 40}}, nil
		}
		rounds++
		if rounds == 1 {
			firstSummaryCount = summaries
		}
		if rounds <= 4 {
			id := fmt.Sprint(rounds)
			text := "Reading notes " + id
			if err := emit(provider.Delta{Text: text}); err != nil {
				return provider.Response{}, err
			}
			return provider.Response{Message: provider.Message{Role: "assistant", Content: text, Calls: []provider.Call{{ID: id, Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read","path":"notes"}`}}}}}, nil
		}
		if err := emit(provider.Delta{Text: "Finished reading."}); err != nil {
			return provider.Response{}, err
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Finished reading."}}, nil
	}}
	if err := loop.Run(context.Background(), Input{Conversation: "session", Model: info, History: history}); err != nil {
		t.Fatal(err)
	}
	if firstSummaryCount == 0 || summaries <= firstSummaryCount || saved != summaries {
		t.Fatalf("no mid-run compaction: initial=%d total=%d saved=%d", firstSummaryCount, summaries, saved)
	}
	usage := session.Usage(store.Entries("session"))
	if usage.Input != summaries*100 || usage.Output != summaries*20 || usage.Cached != summaries*40 {
		t.Fatal("compaction usage missing or counted twice", usage, summaries)
	}
	if !reflect.DeepEqual(store.Entries("session")[:len(original)], original) {
		t.Fatal("original transcript changed")
	}
	store.Close()
	reopened, err := storage.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	history, err = History(reopened.Entries("session"))
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := reopened.Compaction("session")
	if err != nil || !compactor.Valid(checkpoint, history) {
		t.Fatal("stored checkpoint does not match replay", err)
	}
	initialSummaries := summaries
	loop.Save = reopened.Append
	loop.SaveCompaction = func(state session.Compaction) error { return reopened.SaveCompaction("session", state) }
	if err := loop.Run(context.Background(), Input{Conversation: "session", Model: info, History: history, Compaction: checkpoint}); err != nil {
		t.Fatal(err)
	}
	if summaries != initialSummaries {
		t.Fatal("restart unnecessarily repeated compaction")
	}
}
