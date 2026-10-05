package app

import (
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/settings"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type compactingProvider struct{}

func (compactingProvider) Models(context.Context, string) ([]model.Info, error) {
	return []model.Info{{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000}}, nil
}

func (compactingProvider) Complete(ctx context.Context, _ string, request provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
	if len(request.Tools) > 0 {
		return provider.Response{}, errors.New("Compaction was skipped")
	}
	<-ctx.Done()
	return provider.Response{}, ctx.Err()
}

func TestStopDuringCompactionPreservesSavedHistory(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	app.deepseek = catalog.NewConnection(chatCredentials{}, compactingProvider{})
	if _, err := app.deepseek.Status(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Configure(settings.Run{Model: "test"}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 20; index++ {
		if err := app.Record(session.TranscriptChunk{Conversation: "session", ID: fmt.Sprint(index), Role: "user", Text: strings.Repeat("Saved history. ", 150), Status: "done"}); err != nil {
			t.Fatal(err)
		}
	}
	run, err := app.Send(context.Background(), Send{ID: "compact", Conversation: "session", Text: "Continue"})
	if err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return app.ChatState().Compacting })
	if !app.StopReply(run.ID) {
		t.Fatal("stop was not accepted")
	}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	event := app.ChatState()
	if event.Compacting || event.Error != "" || event.Message.Status != "cancelled" {
		t.Fatalf("incorrect final state: %+v", event)
	}
	summary, err := app.sessions.Compaction("session")
	if err != nil || summary.Through != 0 {
		t.Fatal("partial summary saved", err)
	}
	entries := app.sessions.Entries("session")
	if len(entries) < 21 || entries[0].Text != strings.Repeat("Saved history. ", 150) {
		t.Fatal("history lost")
	}
}
