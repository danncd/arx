package app

import (
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	model "arx/internal/models"
	permission "arx/internal/permissions"
	"arx/internal/settings"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type chatCredentials struct{}

func (chatCredentials) Load(context.Context) (string, error) { return "test", nil }
func (chatCredentials) Save(context.Context, string) error   { return nil }
func (chatCredentials) Delete(context.Context) error         { return nil }

type chatProvider struct{ calls atomic.Int32 }

func (*chatProvider) Models(context.Context, string) ([]model.Info, error) {
	return []model.Info{{ID: "test", ContextWindow: 1048576, MaxOutputTokens: 32768}}, nil
}
func (p *chatProvider) Complete(ctx context.Context, _ string, input provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	if len(input.Tools) == 0 {
		return provider.Response{Message: provider.Message{Content: "Test conversation"}}, nil
	}
	round := p.calls.Add(1)
	if round == 1 {
		if err := emit(provider.Delta{Text: "Writing"}); err != nil {
			return provider.Response{}, err
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Writing", Calls: []provider.Call{{ID: "write", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"write","path":"note.txt","content":"approved"}`}}}}}, nil
	}
	if round == 2 {
		if !strings.Contains(input.Messages[len(input.Messages)-1].Content, "Permission denied") {
			return provider.Response{}, context.Canceled
		}
		if err := emit(provider.Delta{Text: "Denied"}); err != nil {
			return provider.Response{}, err
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Denied"}}, nil
	}
	if err := emit(provider.Delta{Text: "Partial response"}); err != nil {
		return provider.Response{}, err
	}
	<-ctx.Done()
	return provider.Response{}, ctx.Err()
}

func TestChatPermissionsStopConcurrencyAndRestore(t *testing.T) {
	state, directory := t.TempDir(), t.TempDir()
	app, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	client := &chatProvider{}
	app.deepseek = catalog.NewConnection(chatCredentials{}, client)
	if _, err := app.deepseek.Status(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Configure(settings.Run{Model: "test"}, directory); err != nil {
		t.Fatal(err)
	}
	first, err := app.Send(context.Background(), Send{ID: "first", Text: "Write a note", Permissions: &permission.Policy{Mode: permission.Ask}})
	if err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return app.PendingPermission() != nil })
	if _, err := os.Stat(filepath.Join(directory, "note.txt")); !os.IsNotExist(err) {
		t.Fatal("unapproved file created")
	}
	if _, err := app.Send(context.Background(), Send{ID: "second", Text: "Another"}); err == nil {
		t.Fatal("concurrent run allowed")
	}
	repeated, err := app.Send(context.Background(), Send{ID: "first", Text: "Write a note", Permissions: &permission.Policy{Mode: permission.Ask}})
	if err != nil || repeated.Conversation != first.Conversation || client.calls.Load() != 1 {
		t.Fatal("duplicate send reran")
	}
	if err := app.RespondPermission(app.PendingPermission().ID, false); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	if _, err := os.Stat(filepath.Join(directory, "note.txt")); !os.IsNotExist(err) {
		t.Fatal("denied write executed")
	}
	second, err := app.Send(context.Background(), Send{ID: "second", Conversation: first.Conversation, Text: "Continue"})
	if err != nil {
		t.Fatal(err)
	}
	await(t, func() bool {
		event := app.ChatState()
		return event.Message != nil && event.Message.Text == "Partial response"
	})
	if app.StopReply("stale-id") {
		t.Fatal("stale stop cancelled another run")
	}
	if !app.StopReply(second.ID) {
		t.Fatal("stop was not accepted")
	}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	if app.ChatState().Message.Status != "cancelled" {
		t.Fatal("cancelled status missing")
	}
	app.Close()
	reopened, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	history, err := reopened.History(first.Conversation, "", 60)
	if err != nil {
		t.Fatal(err)
	}
	if history.Chunks[len(history.Chunks)-1].Status != "cancelled" {
		t.Fatal("stopped reply not persisted")
	}
}

func await(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestStoppingPendingApprovalPreventsTheWrite(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	app.deepseek = catalog.NewConnection(chatCredentials{}, &chatProvider{})
	app.deepseek.Status(context.Background(), false)
	directory := t.TempDir()
	if _, err := app.Configure(settings.Run{Model: "test"}, directory); err != nil {
		t.Fatal(err)
	}
	run, err := app.Send(context.Background(), Send{ID: "pending", Text: "Write a note", Permissions: &permission.Policy{Mode: permission.Ask}})
	if err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return app.PendingPermission() != nil })
	if !app.StopReply(run.ID) {
		t.Fatal("stop not accepted")
	}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	if app.PendingPermission() != nil {
		t.Fatal("approval survived stop")
	}
	if err := app.RespondPermission("write", true); err == nil {
		t.Fatal("expired approval accepted")
	}
	if _, err := os.Stat(filepath.Join(directory, "note.txt")); !os.IsNotExist(err) {
		t.Fatal("cancelled write executed")
	}
}

func TestNewChatsInheritDefaultPermissions(t *testing.T) {
	for _, mode := range []permission.Mode{permission.Ask, permission.Folders, permission.Full} {
		t.Run(string(mode), func(t *testing.T) {
			app, err := Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer app.Close()
			client := &chatProvider{}
			client.calls.Store(2)
			app.deepseek = catalog.NewConnection(chatCredentials{}, client)
			if _, err := app.deepseek.Status(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			directory := t.TempDir()
			if _, err := app.Configure(settings.Run{Model: "test"}, directory); err != nil {
				t.Fatal(err)
			}
			directory = app.Snapshot().Settings.Directory
			policy := permission.Policy{Mode: mode, Roots: []string{directory}}
			if _, err := app.ConfigurePermissions(policy); err != nil {
				t.Fatal(err)
			}
			for index, override := range []*permission.Policy{nil, {Mode: permission.Ask}} {
				run, err := app.Send(context.Background(), Send{ID: []string{"default", "override"}[index], Text: "Hello", Permissions: override})
				if err != nil {
					t.Fatal(err)
				}
				want := mode
				if override != nil {
					want = override.Mode
				}
				saved := app.Snapshot().Settings.ChatPermissions[run.Conversation]
				if saved.Mode != want {
					t.Fatalf("chat mode = %s, want %s", saved.Mode, want)
				}
				if want == permission.Folders && (len(saved.Roots) != 1 || saved.Roots[0] != directory) {
					t.Fatal("new chat lost default folders")
				}
				await(t, func() bool {
					event := app.ChatState()
					return event.Message != nil && event.Message.Text == "Partial response"
				})
				app.StopReply(run.ID)
				await(t, func() bool { return app.ChatState().Run.State == Idle })
				if _, err := app.ConfigurePermissions(permission.Policy{Mode: permission.Full}); err != nil {
					t.Fatal(err)
				}
				if app.Snapshot().Settings.ChatPermissions[run.Conversation].Mode != want {
					t.Fatal("default changed an existing chat")
				}
			}
		})
	}
}
