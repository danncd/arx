package app

import (
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"arx/internal/settings"
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type titleProvider struct {
	requested chan provider.Request
	release   chan struct{}
	fail      bool
	titles    atomic.Int32
}

func (*titleProvider) Models(context.Context, string) ([]model.Info, error) {
	return []model.Info{{ID: "test", ContextWindow: 32768, MaxOutputTokens: 4096}}, nil
}

func (p *titleProvider) Complete(ctx context.Context, _ string, input provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	if len(input.Tools) == 0 {
		p.titles.Add(1)
		p.requested <- input
		select {
		case <-p.release:
		case <-ctx.Done():
			return provider.Response{}, ctx.Err()
		}
		if p.fail {
			return provider.Response{}, errors.New("Offline")
		}
		return provider.Response{Message: provider.Message{Content: "\"Organize project files\""}, Usage: &session.ChunkUsage{Input: 50, Output: 4}}, nil
	}
	if err := emit(provider.Delta{Text: "Ready"}); err != nil {
		return provider.Response{}, err
	}
	return provider.Response{Message: provider.Message{Role: "assistant", Content: "Ready"}}, nil
}

func TestFirstMessageGeneratesAndPersistsTitle(t *testing.T) {
	for _, outcome := range []string{"success", "failed", "cancelled"} {
		t.Run(outcome, func(t *testing.T) {
			directory := t.TempDir()
			app, err := Open(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer app.Close()
			client := &titleProvider{requested: make(chan provider.Request, 2), release: make(chan struct{}), fail: outcome == "failed"}
			app.deepseek = catalog.NewConnection(chatCredentials{}, client)
			if _, err := app.deepseek.Status(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			if _, err := app.Configure(settings.Run{Model: "test"}, directory); err != nil {
				t.Fatal(err)
			}
			run, err := app.Send(context.Background(), Send{ID: "first", Text: "Please organize my project"})
			if err != nil {
				t.Fatal(err)
			}
			await(t, func() bool { return len(client.requested) > 0 })
			request := <-client.requested
			if request.Effort != "none" || request.MaxOutputTokens != 128 || len(request.Messages) != 2 {
				t.Fatal("Unexpected title request", request)
			}
			if app.ChatState().Conversation.Title != "" || app.Snapshot().Conversations[0].Title != "" {
				t.Fatal("Title was exposed before generation")
			}
			if outcome == "cancelled" {
				app.StopReply(run.ID)
			} else {
				close(client.release)
			}
			await(t, func() bool { return app.ChatState().Run.State == Idle })
			want := "Organize project files"
			if outcome != "success" {
				want = "Please organize my project"
			}
			if app.ChatState().Conversation.Title != want {
				t.Fatal(app.ChatState().Conversation)
			}
			if outcome == "success" {
				usage := session.Usage(app.sessions.Entries(run.Conversation))
				if usage.Input != 50 || usage.Output != 4 {
					t.Fatal("Title usage was not counted", usage)
				}
			}
			if outcome != "cancelled" {
				if _, err := app.Send(context.Background(), Send{ID: "next", Conversation: run.Conversation, Text: "Continue"}); err != nil {
					t.Fatal(err)
				}
				await(t, func() bool { return app.ChatState().Run.State == Idle })
				if client.titles.Load() != 1 {
					t.Fatal("Follow-up renamed the chat")
				}
			}
			app.Close()
			reopened, err := Open(directory)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			if reopened.Snapshot().Conversations[0].Title != want {
				t.Fatal("Title lost on reload")
			}
			page, err := reopened.History(run.Conversation, "", 60)
			if err != nil {
				t.Fatal(err)
			}
			for _, chunk := range page.Chunks {
				if chunk.Role == "usage" {
					t.Fatal("Title metadata leaked into chat history")
				}
			}
		})
	}
}
