package compaction

import (
	provider "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSummaryIncludesImages(t *testing.T) {
	images := []attachment.Image{{ID: "first", Name: "first.png"}, {ID: "second", Name: "second.png"}}
	history := []provider.Message{{Role: "tool", CallID: "image-call", Content: "Compare these", Images: images, Reasoning: "private"}}
	request := summaryRequest(model.Info{ID: "deepseek-flash", MaxOutputTokens: 1000}, "", "", history, 4000)
	if !reflect.DeepEqual(request.Messages[1].Images, images) {
		t.Fatal("summary cannot see images")
	}
	if history[0].Reasoning != "private" {
		t.Fatal("summary mutated saved history")
	}
}

func TestImagePressureCompactsBelowTheTokenThreshold(t *testing.T) {
	history := []provider.Message{}
	for index := range 25 {
		id := fmt.Sprint(index)
		history = append(history,
			provider.Message{Role: "user", Content: "Inspect this", Images: []attachment.Image{{ID: id, Name: id + ".png"}}},
			provider.Message{Role: "assistant", Content: "Observed diagram " + id},
		)
	}
	history = append(history, provider.Message{Role: "user", Content: "Compare the recent diagrams"})
	original := append([]provider.Message(nil), history...)
	info := model.Info{ID: "deepseek-flash", ContextWindow: 1048576, MaxOutputTokens: 32768}
	calls := 0
	manager := Compactor{Model: info, Save: func(session.Compaction) error { return nil }, Complete: func(_ context.Context, input provider.Request, _ func(provider.Delta) error) (provider.Response, error) {
		calls++
		if imageCount(input.Messages) > maxImages {
			t.Fatal("summary batch exceeded its image budget")
		}
		if !strings.Contains(input.Messages[0].Content, "visual findings") || !strings.Contains(input.Messages[0].Content, "arx-image:") {
			t.Fatal("summary did not preserve visual findings and saved references")
		}
		return provider.Response{Message: provider.Message{Role: "assistant", Content: "Compared diagrams. Saved reference arx-image:0 shows the earlier layout."}}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 || imageCount(result.Messages) > recentImages {
		t.Fatal("image-heavy history was not compacted in batches")
	}
	if !reflect.DeepEqual(history, original) {
		t.Fatal("saved history changed")
	}
	if result.Messages[len(result.Messages)-1].Content != history[len(history)-1].Content {
		t.Fatal("latest user request lost")
	}
	if !strings.Contains(result.Messages[1].Content, "arx-image:0") {
		t.Fatal("visual memory lost")
	}
	replay := Compactor{Model: info, State: manager.State, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		t.Fatal("replayed a completed summary")
		return provider.Response{}, nil
	}}
	restored, err := replay.Prepare(context.Background(), request(history))
	if err != nil || !reflect.DeepEqual(result, restored) {
		t.Fatal("image compaction did not restore", err)
	}
}

func TestLatestImageExchangeIsKeptForProviderSizing(t *testing.T) {
	history := []provider.Message{{Role: "user", Content: "Inspect all of these"}, {Role: "assistant"}}
	for index := range 32 {
		id := fmt.Sprint(index)
		history[1].Calls = append(history[1].Calls, provider.Call{ID: id})
		history = append(history, provider.Message{Role: "tool", CallID: id, Content: "Image loaded", Images: []attachment.Image{{ID: id}}})
	}
	manager := Compactor{Model: model.Info{ID: "deepseek-flash", ContextWindow: 1048576, MaxOutputTokens: 32768}, Complete: func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error) {
		t.Fatal("latest tool exchange was split or compacted")
		return provider.Response{}, nil
	}}
	result, err := manager.Prepare(context.Background(), request(history))
	if err != nil {
		t.Fatal(err)
	}
	if imageCount(result.Messages) != 32 {
		t.Fatal("unseen images were discarded")
	}
}
