package app

import (
	session "arx/internal/sessions"
	"fmt"
	"testing"
)

func TestSessionUsageSurvivesReloadAndHistoryPagination(t *testing.T) {
	directory := t.TempDir()
	app, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 75; index++ {
		entry := session.TranscriptChunk{Conversation: "session", ID: fmt.Sprint(index), Role: "assistant", Status: "done", Usage: &session.ChunkUsage{Input: 100, Output: 25, Cached: 80, Reasoning: 10}}
		for copy := 0; copy < 3; copy++ {
			if err := app.Record(entry); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := app.Record(session.TranscriptChunk{Conversation: "session", ID: "summary", Role: "usage", Usage: &session.ChunkUsage{Input: 500, Output: 100, Cached: 200}}); err != nil {
		t.Fatal(err)
	}
	if err := app.Record(session.TranscriptChunk{Conversation: "other", ID: "reply", Role: "assistant", Usage: &session.ChunkUsage{Input: 10000}}); err != nil {
		t.Fatal(err)
	}
	app.Close()
	app, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	report, err := app.Context("session", "unavailable-model")
	if err != nil || report.Usage.Input != 8000 || report.Usage.Output != 1975 || report.Usage.Cached != 6200 || report.Usage.Reasoning != 750 {
		t.Fatal(report, err)
	}
	page, err := app.History("session", "", 60)
	if err != nil || !page.More || len(page.Chunks) != 180 {
		t.Fatal("usage records affected pagination", page, err)
	}
	for _, chunk := range page.Chunks {
		if chunk.Role == "usage" {
			t.Fatal("usage entry leaked into messages")
		}
	}
	empty, err := app.Context("", "unavailable-model")
	if err != nil || empty.Usage != (session.ChunkUsage{}) {
		t.Fatal("new session inherited usage", empty, err)
	}
}
