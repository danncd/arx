package storage

import (
	session "arx/internal/sessions"
	"testing"
)

func TestInterruptedTitleRecoversAndLegacyTitleRemains(t *testing.T) {
	directory := t.TempDir()
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	empty := ""
	for _, entry := range []session.TranscriptChunk{
		{Conversation: "pending", ID: "first", Role: "user", Text: "A saved message", Title: &empty},
		{Conversation: "legacy", ID: "old", Role: "user", Text: "Existing title"},
	} {
		if err := store.Append(entry); err != nil {
			t.Fatal(err)
		}
	}
	if store.Conversation("pending").Title != "" {
		t.Fatal("Pending title shown early")
	}
	store.Close()
	store, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if store.Conversation("pending").Title != "A saved message" {
		t.Fatal("Interrupted chat remained hidden")
	}
	if store.Conversation("legacy").Title != "Existing title" {
		t.Fatal("Legacy title changed")
	}
}
