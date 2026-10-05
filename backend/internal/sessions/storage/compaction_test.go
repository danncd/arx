package storage

import (
	session "arx/internal/sessions"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompactionPersistsSeparatelyWithoutChangingTranscript(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(session.TranscriptChunk{Conversation: "session", ID: "user", Role: "user", Text: "Keep this forever"}); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(filepath.Join(directory, "transcript.jsonl"))
	state := session.Compaction{Version: 2, User: 1, Through: 1, Digest: strings.Repeat("a", 64), Summary: "Saved summary 🌱"}
	if err := store.SaveCompaction("session", state); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(directory, "transcript.jsonl"))
	if !bytes.Equal(original, after) {
		t.Fatal("transcript changed")
	}
	path := store.compactionPath("session")
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("summary permissions")
	}
	store.Close()
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	actual, err := reopened.Compaction("session")
	if err != nil || actual != state {
		t.Fatal("summary did not survive restart", err)
	}
	other, err := reopened.Compaction("different")
	if err != nil || other.Through != 0 {
		t.Fatal("summary leaked across sessions")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := reopened.Compaction("session")
	if err != nil || recovered.Through != 0 || len(reopened.Entries("session")) != 1 {
		t.Fatal("bad summary broke transcript recovery")
	}
}
