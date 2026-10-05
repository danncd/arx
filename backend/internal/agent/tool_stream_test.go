package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	tool "arx/internal/tools"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestToolAppearsBeforeArgumentsFinishAndRunsOnlyAfterCompletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	visible := make(chan struct{}, 1)
	executed := 0
	var chunks []session.TranscriptChunk
	rounds := 0
	loop := Loop{
		Save: func(chunk session.TranscriptChunk) error {
			for _, call := range chunk.Tools {
				if call.Status == "preparing" && chunk.Status == "running" {
					t.Error("partial arguments written repeatedly to disk")
				}
			}
			chunks = append(chunks, chunk)
			return nil
		},
		Emit: func(chunk session.TranscriptChunk) {
			if len(chunk.Tools) > 0 && chunk.Tools[0].Status == "preparing" {
				if chunk.Tools[0].Offset != len("Before 🌱") {
					t.Error("lost tool position")
				}
				select {
				case visible <- struct{}{}:
				default:
				}
			}
		},
		Tool: func(_ context.Context, call tool.Call) tool.Result {
			executed++
			if string(call.Arguments) != `{"operation":"write","path":"notes.txt","content":"hello"}` {
				t.Error("partial input executed")
			}
			return tool.Result{Text: "Written"}
		},
		Complete: func(ctx context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
			rounds++
			if rounds > 1 {
				emit(provider.Delta{Text: "Done"})
				return provider.Response{Message: provider.Message{Role: "assistant", Content: "Done"}}, nil
			}
			emit(provider.Delta{Text: "Before 🌱"})
			call := provider.Call{ID: "write", Type: "function", Function: provider.Function{Name: "files", Arguments: `{"operation":"write","path":"notes.txt","content":"`}}
			if err := emit(provider.Delta{Tool: &provider.ToolDelta{Index: 0, Call: call}}); err != nil {
				return provider.Response{}, err
			}
			select {
			case <-visible:
			case <-ctx.Done():
				return provider.Response{}, errors.New("tool card was hidden until completion")
			}
			call.Function.Arguments += `hello"}`
			if err := emit(provider.Delta{Tool: &provider.ToolDelta{Index: 0, Call: call}}); err != nil {
				return provider.Response{}, err
			}
			return provider.Response{Message: provider.Message{Role: "assistant", Content: "Before 🌱", Calls: []provider.Call{call}}}, nil
		},
	}
	err := loop.Run(ctx, Input{Conversation: "test", Model: model.Info{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Write the file"}}})
	if err != nil || executed != 1 {
		t.Fatal(err, executed)
	}
	history, err := History(chunks)
	if err != nil || len(history) != 3 || len(history[0].Calls) != 1 {
		t.Fatal("streamed call duplicated in saved history", history, err)
	}
}

func TestStoppedPartialToolIsNeverReplayed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var chunks []session.TranscriptChunk
	loop := Loop{Save: func(chunk session.TranscriptChunk) error { chunks = append(chunks, chunk); return nil }, Emit: func(chunk session.TranscriptChunk) {
		if len(chunk.Tools) > 0 {
			cancel()
		}
	}, Tool: func(context.Context, tool.Call) tool.Result {
		t.Fatal("unfinished tool executed")
		return tool.Result{}
	}, Complete: func(ctx context.Context, _ provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
		emit(provider.Delta{Text: "Starting"})
		call := provider.Call{ID: "partial", Function: provider.Function{Name: "files", Arguments: `{"operation":"write","content":"` + strings.Repeat("x", 100)}}
		emit(provider.Delta{Tool: &provider.ToolDelta{Index: 0, Call: call}})
		<-ctx.Done()
		return provider.Response{}, ctx.Err()
	}}
	err := loop.Run(ctx, Input{Conversation: "test", Model: model.Info{ID: "test", ContextWindow: 16000, MaxOutputTokens: 4000}, History: []provider.Message{{Role: "user", Content: "Write"}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	history, err := History(chunks)
	if err != nil || len(history) != 1 || len(history[0].Calls) != 0 {
		t.Fatal("unfinished call replayed", history, err)
	}
}
