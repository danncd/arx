package storage

import (
	session "arx/internal/sessions"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRestartPreservesTranscriptAndMessagePages(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5; index++ {
		entry := session.TranscriptChunk{Conversation: "session", ID: fmt.Sprint(index), Role: "user", Text: "Hello 世界", At: fmt.Sprintf("2026-09-26T12:00:0%dZ", index)}
		if err := store.Append(entry); err != nil {
			t.Fatal(err)
		}
	}
	tool := session.ToolRecord{ID: "tool", Name: "files", Status: "done", Result: "one.go"}
	first := session.TranscriptChunk{Conversation: "session", ID: "reply", Role: "assistant", Text: "Hi 🌱", Reasoning: "Think", Tools: []session.ToolRecord{tool}, At: "2026-09-26T12:01:00Z"}
	second := session.TranscriptChunk{Conversation: "session", ID: "reply", Role: "assistant", Offset: len(first.Text), ReasoningAt: len(first.Reasoning), Text: " Danny", Status: "done", Usage: &session.ChunkUsage{Input: 40, Output: 10}, At: "2026-09-26T12:01:01Z"}
	for _, entry := range []session.TranscriptChunk{first, second} {
		if err := store.Append(entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	chunks, before, more, err := store.History("session", "", 1)
	if err != nil || !reflect.DeepEqual(chunks, []session.TranscriptChunk{first, second}) || !more || before != "reply" {
		t.Fatalf("history: %+v %q %v %v", chunks, before, more, err)
	}
	chunks, before, more, err = store.History("session", before, 200)
	if err != nil || len(chunks) != 5 || more || before != "" {
		t.Fatalf("previous: %d %q %v %v", len(chunks), before, more, err)
	}
	conversations := store.Conversations()
	if len(conversations) != 1 || conversations[0].Title != "Hello 世界" || conversations[0].Updated != second.At {
		t.Fatalf("summaries: %+v", conversations)
	}
	if _, _, _, err := store.History("session", "missing", 1); err == nil {
		t.Fatal("unknown cursor accepted")
	}
	info, err := os.Stat(filepath.Join(directory, "transcript.jsonl"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v %v", info, err)
	}
}

func TestRecoveryKeepsOriginalBytesAndSeparatesNextAppend(t *testing.T) {
	for _, tail := range []string{`{"conversation":"s","id":"incomplete"`, `{"conversation":"s","id":"valid","role":"user","text":"without newline"}`} {
		t.Run(tail, func(t *testing.T) {
			directory := t.TempDir()
			original := []byte(`{"conversation":"s","id":"first","role":"user","text":"original"}` + "\n" + tail)
			path := filepath.Join(directory, "transcript.jsonl")
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			store, err := Open(directory)
			if err != nil {
				t.Fatal(err)
			}
			if store.Recovered() != !json.Valid([]byte(tail)) {
				t.Fatal("incorrect recovery status")
			}
			if err := store.Append(session.TranscriptChunk{Conversation: "s", ID: "next", Role: "user", Text: "new"}); err != nil {
				t.Fatal(err)
			}
			store.Close()
			store, err = Open(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			chunks, _, _, err := store.History("s", "", 60)
			expected := 2
			if json.Valid([]byte(tail)) {
				expected++
			}
			if err != nil || len(chunks) != expected || chunks[len(chunks)-1].Text != "new" {
				t.Fatalf("recovered: %+v %v", chunks, err)
			}
			body, _ := os.ReadFile(path)
			if !strings.HasPrefix(string(body), string(original)+"\n") {
				t.Fatal("existing transcript changed")
			}
		})
	}
}

func TestAppendSnapshotsToolRecordsAndRejectsClosedStore(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tools := []session.ToolRecord{{ID: "tool", Result: "original"}}
	entry := session.TranscriptChunk{Conversation: "s", ID: "a", Role: "assistant", Tools: tools}
	if err := store.Append(entry); err != nil {
		t.Fatal(err)
	}
	tools[0].Result = "modified"
	chunks, _, _, _ := store.History("s", "", 1)
	if chunks[0].Tools[0].Result != "original" {
		t.Fatal("record mutated after saving")
	}
	store.Close()
	if err := store.Append(entry); err == nil {
		t.Fatal("append to closed store succeeded")
	}
}

func TestTitleDoesNotSplitUnicode(t *testing.T) {
	title := firstLine(strings.Repeat("🌱", 90))
	if !utf8.ValidString(title) || len([]rune(title)) != 80 {
		t.Fatal("invalid title")
	}
}
