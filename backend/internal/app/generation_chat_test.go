package app

import (
	"arx/internal/generation"
	provider "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	permission "arx/internal/permissions"
	"arx/internal/settings"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

type generationChatProvider struct {
	chatProvider
	output string
}

func (p *generationChatProvider) Complete(_ context.Context, _ string, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
	if len(input.Tools) == 0 {
		return provider.Response{Message: provider.Message{Content: "Generate apple"}}, nil
	}
	if p.calls.Add(1) == 1 {
		return provider.Response{Message: provider.Message{Role: "assistant", Calls: []provider.Call{{ID: "generate-apple", Type: "function", Function: provider.Function{Name: "generate", Arguments: `{"operation":"image","prompt":"A red apple"}`}}}}}, nil
	}
	p.output = input.Messages[len(input.Messages)-1].Content
	return provider.Response{Message: provider.Message{Role: "assistant", Content: "Configure the image model."}}, nil
}

func TestChatDispatchesGenerationTool(t *testing.T) {
	app, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	assets, err := filepath.Abs("../../../runtimes/generation")
	if err != nil {
		t.Fatal(err)
	}
	app.generation, err = generation.Open(t.TempDir(), assets)
	if err != nil {
		t.Fatal(err)
	}
	client := &generationChatProvider{}
	app.deepseek = catalog.NewConnection(chatCredentials{}, client)
	if _, err := app.deepseek.Status(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Configure(settings.Run{Model: "test"}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Send(context.Background(), Send{ID: "apple", Text: "Generate an apple", Permissions: &permission.Policy{Mode: permission.Full}}); err != nil {
		t.Fatal(err)
	}
	await(t, func() bool { return app.ChatState().Run.State == Idle })
	if !strings.Contains(client.output, "No model is configured") || strings.Contains(client.output, "Unknown tool") {
		t.Fatalf("generation was not dispatched: %s", client.output)
	}
}
