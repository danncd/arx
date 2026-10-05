package app

import (
	"arx/internal/agent"
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"testing"
)

func TestContextUsesPersistedProviderUsageAfterRestart(t *testing.T) {
	directory := t.TempDir()
	app, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	user := session.TranscriptChunk{Conversation: "session", ID: "user", Role: "user", Text: "Hello", Status: "done"}
	if err := app.Record(user); err != nil {
		t.Fatal(err)
	}
	history, _ := agent.History([]session.TranscriptChunk{user})
	request := agent.Request(app.preferences.Values(), history, model.Info{ID: "test"})
	agent.AddGenerationGuidance(&request, app.generationGuidance())
	agent.AddGenerationGuidance(&request, app.integrationGuidance())
	request.Model = "test"
	reply := session.TranscriptChunk{Conversation: "session", ID: "reply", Role: "assistant", Text: "Hello back", Status: "done", Context: contextwindow.Measure(request, 730), Usage: &session.ChunkUsage{Input: 730, Cached: 500}}
	if err := app.Record(reply); err != nil {
		t.Fatal(err)
	}
	app.Close()
	app, err = Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	app.deepseek = catalog.NewConnection(chatCredentials{}, compactingProvider{})
	app.deepseek.Status(context.Background(), false)
	report, err := app.Context("session", "test")
	expected := 730 + contextwindow.Messages([]provider.Message{{Role: "assistant", Content: "Hello back"}})
	if err != nil || report.Used != expected {
		t.Fatal("actual input lost on restart", report, expected, err)
	}
	if report.Parts.System+report.Parts.Tools+report.Parts.Messages != expected {
		t.Fatal("parts do not match usage", report)
	}
}
