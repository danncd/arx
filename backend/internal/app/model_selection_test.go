package app

import (
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	model "arx/internal/models"
	permission "arx/internal/permissions"
	"arx/internal/settings"
	"context"
	"testing"
	"time"
)

type selectionProvider struct {
	requests chan provider.Request
	replies  chan provider.Message
}

func (*selectionProvider) Models(context.Context, string) ([]model.Info, error) {
	return []model.Info{
		{ID: "first", ContextWindow: 32768, MaxOutputTokens: 4096},
		{ID: "next", ContextWindow: 32768, MaxOutputTokens: 4096},
	}, nil
}

func (p *selectionProvider) Complete(ctx context.Context, _ string, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	if len(request.Tools) == 0 {
		return provider.Response{Message: provider.Message{Content: "Test conversation"}}, nil
	}
	select {
	case p.requests <- request:
	case <-ctx.Done():
		return provider.Response{}, ctx.Err()
	}
	select {
	case reply := <-p.replies:
		if err := emit(provider.Delta{Text: reply.Content}); err != nil {
			return provider.Response{}, err
		}
		return provider.Response{Message: reply}, nil
	case <-ctx.Done():
		return provider.Response{}, ctx.Err()
	}
}

func TestModelSelectionAppliesToNextTurn(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	client := &selectionProvider{requests: make(chan provider.Request, 1), replies: make(chan provider.Message, 1)}
	app.deepseek = catalog.NewConnection(chatCredentials{}, client)
	if _, err := app.deepseek.Status(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if _, err := app.Configure(settings.Run{Model: "first"}, directory); err != nil {
		t.Fatal(err)
	}
	received := func(want string) {
		t.Helper()
		select {
		case request := <-client.requests:
			if request.Model != want {
				t.Fatalf("Model = %q; want %q", request.Model, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("No model request")
		}
	}
	first, err := app.Send(context.Background(), Send{ID: "first-turn", Text: "List files", Permissions: &permission.Policy{Mode: permission.Full}})
	if err != nil {
		t.Fatal(err)
	}
	received("first")
	if _, err := app.Configure(settings.Run{Model: "next"}, directory); err != nil {
		t.Fatal(err)
	}
	client.replies <- provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "list", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"list","path":"."}`}}}}
	received("first")
	client.replies <- provider.Message{Role: "assistant", Content: "Done"}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	if _, err := app.Send(context.Background(), Send{ID: "next-turn", Conversation: first.Conversation, Text: "Continue"}); err != nil {
		t.Fatal(err)
	}
	received("next")
	client.replies <- provider.Message{Role: "assistant", Content: "Next model"}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
}
