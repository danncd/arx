package local

import (
	provider "arx/internal/inference"
	tool "arx/internal/tools"
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveLocalModel(t *testing.T) {

	path := os.Getenv("ARX_TEST_MODEL")
	if path == "" {
		t.Skip("Set ARX_TEST_MODEL to a tool-capable GGUF for local inference checks")
	}
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	id, err := manager.Import(path, os.Getenv("ARX_TEST_PROJECTOR"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Loading local engine")
	info, err := manager.Ensure(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Loaded: context=%d vision=%v tools=%v thinking=%v", info.ContextWindow, info.Vision, info.Tools, info.Thinking)
	text := ""
	response, err := manager.Complete(ctx, provider.Request{Model: id, Effort: "none", MaxOutputTokens: 64, Messages: []provider.Message{{Role: "user", Content: "Reply with just the word ready."}}}, func(delta provider.Delta) error { text += delta.Text; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if text == "" || response.Message.Content != text || response.Usage == nil || response.Usage.Output <= 0 {
		t.Fatal("Invalid streamed response or usage")
	}
	t.Log("Streamed text and token usage verified")
	response, err = manager.Complete(ctx, provider.Request{Model: id, Effort: "none", MaxOutputTokens: 256, Tools: tool.Definitions(), Messages: []provider.Message{{Role: "system", Content: "You are a coding assistant. Use the files tool to list the directory when asked. Do not execute bash for this task."}, {Role: "user", Content: "List the files in the current directory using the files tool now."}}}, func(delta provider.Delta) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Message.Calls) == 0 {
		t.Fatal("Model did not call a tool")
	}
	t.Log("Streamed tool call verified", response.Message.Calls[0].Function.Name)
}
