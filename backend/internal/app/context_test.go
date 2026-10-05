package app

import (
	"arx/internal/agent"
	compactor "arx/internal/agent/compaction"
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	session "arx/internal/sessions"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestContextReportIncludesSavedHistoryAndPersistedCompaction(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	app.deepseek = catalog.NewConnection(chatCredentials{}, compactingProvider{})
	app.deepseek.Status(context.Background(), false)
	empty, err := app.Context("", "test")
	if err != nil || !empty.Known || empty.Used <= 0 || empty.Limit != 16000 || empty.Basis != "estimated" {
		t.Fatal("missing base context", empty, err)
	}
	for index := 0; index < 80; index++ {
		if err := app.Record(session.TranscriptChunk{Conversation: "session", ID: fmt.Sprint(index), Role: "user", Text: strings.Repeat("Saved context. ", 50), Status: "done"}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := app.Context("session", "test")
	if err != nil || before.Used < 20000 {
		t.Fatal("full saved history was not counted", before, err)
	}
	history, err := agent.History(app.sessions.Entries("session"))
	if err != nil {
		t.Fatal(err)
	}
	info, _ := app.deepseek.Model("test")
	manager := compactor.Compactor{Model: info, Save: func(state session.Compaction) error { return app.sessions.SaveCompaction("session", state) }, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		return provider.Response{Message: provider.Message{Content: "Earlier conversation summary"}}, nil
	}}
	if _, err := manager.Prepare(context.Background(), agent.Request(app.preferences.Values(), history)); err != nil {
		t.Fatal(err)
	}
	after, err := app.Context("session", "test")
	if err != nil || after.Used >= before.Used/2 || after.Used <= empty.Used {
		t.Fatal("saved summary was not reflected", after, err)
	}
	if after.Parts.Summary <= 0 || after.Parts.System <= 0 || after.Parts.Tools <= 0 || after.Parts.Messages <= 0 {
		t.Fatal("context components are missing", after.Parts)
	}
	if after.Parts.System+after.Parts.Tools+after.Parts.Messages+after.Parts.Summary != after.Used {
		t.Fatal("breakdown does not add up", after)
	}
	other, err := app.Context("different", "test")
	if err != nil || other.Used != empty.Used {
		t.Fatal("context leaked between sessions")
	}
	unknown, err := app.Context("session", "missing")
	if err != nil || unknown.Known {
		t.Fatal("unknown model reported as measured")
	}
	if len(app.sessions.Entries("session")) != 80 {
		t.Fatal("context request altered saved history")
	}
}
