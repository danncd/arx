package compaction

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestGrowingHistoryPreservesToolOutputPrefix(t *testing.T) {
	history := []provider.Message{
		{Role: "user", Content: "Read file"},
		{Role: "assistant", Calls: []provider.Call{{ID: "read", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"read"}`}}}},
		{Role: "tool", CallID: "read", Content: strings.Repeat("file content ", 3000)},
		{Role: "assistant", Content: strings.Repeat("recent text ", 4000)},
	}
	manager := Compactor{Model: model.Info{ID: "test", ContextWindow: 1048576, MaxOutputTokens: 32768}}
	before, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	history = append(history, provider.Message{Role: "user", Content: "Continue"})
	after, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Messages, after.Messages[:len(before.Messages)]) || before.Messages[3].Content != history[2].Content {
		t.Fatal("appending a message rewrote the cached prefix")
	}
}

func TestCompactedUserStaysPinnedAcrossLaterTurns(t *testing.T) {
	history := []provider.Message{{Role: "user", Content: "Original constraints"}, {Role: "assistant", Content: "Old work"}, {Role: "assistant", Content: "Recent work"}}
	for _, version := range []int{1, 2} {
		state := session.Compaction{Version: version, Through: 2, User: 1, Digest: digest(history[:2]), Summary: "Progress"}
		if version == 1 {
			state.User = 0
		}
		before := Preview(request(history), state)
		extended := append(append([]provider.Message{}, history...), provider.Message{Role: "user", Content: "Next task"})
		after := Preview(request(extended), state)
		if !Valid(state, extended) || !reflect.DeepEqual(before.Messages, after.Messages[:len(before.Messages)]) {
			t.Fatalf("version %d rewrote its retained prefix", version)
		}
		if before.Messages[2].Content != history[0].Content {
			t.Fatal("lost original constraint")
		}
	}
	state := session.Compaction{Version: 2, Through: 2, User: 2, Digest: digest(history[:2]), Summary: "Progress"}
	if Valid(state, history) {
		t.Fatal("non-user pin accepted")
	}
}
