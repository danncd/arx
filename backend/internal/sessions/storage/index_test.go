package storage

import (
	session "arx/internal/sessions"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexedHistoryPreservesPagesAfterRestart(t *testing.T) {
	directory := t.TempDir()
	var data strings.Builder
	for conversation := 0; conversation < 200; conversation++ {
		for message := 0; message < 10; message++ {
			entry := session.TranscriptChunk{Conversation: fmt.Sprint(conversation), ID: fmt.Sprint(message), Role: "user", Text: fmt.Sprintf("chat %d message %d", conversation, message), At: fmt.Sprint(message)}
			body, err := json.Marshal(entry)
			if err != nil {
				t.Fatal(err)
			}
			data.Write(body)
			data.WriteByte('\n')
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "transcript.jsonl"), []byte(data.String()), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	chunks, before, more, err := store.History("42", "", 3)
	if err != nil || len(chunks) != 3 || before != "7" || !more || chunks[0].Text != "chat 42 message 7" {
		t.Fatalf("Latest page: %v %q %v %v", chunks, before, more, err)
	}
	older, cursor, more, err := store.History("42", before, 3)
	if err != nil || cursor != "4" || !more || older[0].ID != "4" {
		t.Fatalf("Older page: %v %q %v %v", older, cursor, more, err)
	}
	if len(store.Entries("42")) != 10 || len(store.Conversations()) != 200 {
		t.Fatal("Lost indexed history")
	}
	if err := store.Append(session.TranscriptChunk{Conversation: "42", ID: "10", Role: "user", Text: "new"}); err != nil {
		t.Fatal(err)
	}
	latest, _, _, err := store.History("42", "", 1)
	if err != nil || len(latest) != 1 || latest[0].ID != "10" {
		t.Fatal("New entry not indexed")
	}
}

func BenchmarkIndexedHistory(b *testing.B) {
	for _, count := range []int{10, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			store := &Store{conversations: map[string]*conversationIndex{}}
			for conversation := 0; conversation < count; conversation++ {
				for message := 0; message < 30; message++ {
					store.index(session.TranscriptChunk{Conversation: fmt.Sprint(conversation), ID: fmt.Sprint(message), Role: "user", Text: "message"})
				}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, _, err := store.History("0", "", 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
