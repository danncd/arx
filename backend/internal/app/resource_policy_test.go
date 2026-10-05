package app

import (
	"arx/internal/inference"
	permission "arx/internal/permissions"
	session "arx/internal/sessions"
	"arx/internal/settings"
	"context"
	"testing"
)

func TestRendererResourcesRequireKnownPermissiveConversation(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	for _, id := range []string{"", "unknown"} {
		if _, err := app.WebImage(context.Background(), id, "https://example.com/image.png"); err == nil {
			t.Fatal("unknown conversation permitted")
		}
	}
	if err := app.Record(session.TranscriptChunk{ID: "user", Conversation: "chat", Role: "user", Text: "saved", Status: "done"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.preferences.SaveChatPermissions("chat", permission.Policy{Mode: permission.Ask}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.WebImage(context.Background(), "chat", "https://example.com/image.png"); err == nil {
		t.Fatal("Ask bypassed")
	}
	if _, err := app.WebIcon(context.Background(), "chat", "https://example.com"); err == nil {
		t.Fatal("Ask icon bypassed")
	}
	for _, mode := range []permission.Mode{permission.Folders, permission.Full} {
		app.preferences.SaveChatPermissions("chat", permission.Policy{Mode: mode, Roots: []string{app.directory}})
		if err := app.Actions.RemoteResourcePolicy("chat"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProviderRoutingKeepsLegacyIDsAndRejectsConflicts(t *testing.T) {
	for id, want := range map[string]string{"deepseek-chat": "deepseek", "local:test": "local", "network:server:model": "network"} {
		got, err := inference.ProviderFor(settings.Run{Model: id})
		if err != nil || got != want {
			t.Fatal(got, err)
		}
		if _, err := inference.ProviderFor(settings.Run{Model: id, Provider: "wrong"}); err == nil {
			t.Fatal("conflict accepted")
		}
	}
}
