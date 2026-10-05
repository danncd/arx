package deepseek

import (
	provider "arx/internal/inference"
	chatcompletions "arx/internal/inference/chatcompletions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompletionStreamsReasoningCallsAndUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("incorrect authenticated endpoint")
		}
		var body struct {
			Model     string             `json:"model"`
			MaxOutput int                `json:"max_tokens"`
			Effort    string             `json:"reasoning_effort"`
			Messages  []provider.Message `json:"messages"`
			Tools     []json.RawMessage  `json:"tools"`
			Thinking  struct {
				Type string `json:"type"`
			} `json:"thinking"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.MaxOutput != 2048 || body.Model != "live-model" || body.Effort != "high" || body.Thinking.Type != "enabled" || len(body.Tools) != len(tool.Definitions()) || body.Messages[0].Reasoning != "earlier thought" {
			t.Errorf("lost request properties: %+v", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": keepalive\n\ndata: {\"choices\":[{\"delta\":{\"reasoning_content\":\"Think 🌱\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Reading\",\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"type\":\"function\",\"function\":{\"name\":\"files\",\"arguments\":\"{\\\"operation\\\":\\\"read\\\",\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"path\\\":\\\"notes.txt\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":90,\"completion_tokens\":15,\"prompt_cache_hit_tokens\":30,\"completion_tokens_details\":{\"reasoning_tokens\":8}}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client := &Client{client: server.Client(), baseURL: server.URL}
	var text, thought string
	response, err := client.Complete(context.Background(), "test-key", provider.Request{MaxOutputTokens: 2048, Model: "live-model", Effort: "high", Tools: tool.Definitions(), Messages: []provider.Message{{Role: "assistant", Content: "earlier", Reasoning: "earlier thought"}}}, func(delta provider.Delta) error { text += delta.Text; thought += delta.Reasoning; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if text != "Reading" || thought != "Think 🌱" || response.Message.Reasoning != thought {
		t.Fatalf("stream lost text: %+v", response)
	}
	if len(response.Message.Calls) != 1 || response.Message.Calls[0].Function.Arguments != `{"operation":"read","path":"notes.txt"}` {
		t.Fatalf("fragmented tool call: %+v", response.Message.Calls)
	}
	if response.Usage.Input != 90 || response.Usage.Cached != 30 || response.Usage.Reasoning != 8 {
		t.Fatal("lost usage")
	}
}

func TestStreamRejectsTruncatedMalformedAndUnfinishedCalls(t *testing.T) {
	cases := []string{
		`data: {"choices":[{"delta":{"content":"partial"}}]}` + "\n\n",
		"data: not-json\n\n",
		`data: {"choices":[{"delta":{},"finish_reason":"length"}]}` + "\n\ndata: [DONE]\n\n",
		`data: {"choices":[{"delta":{"tool_calls":[{"index":2,"id":"x","function":{"name":"files","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}` + "\n\ndata: [DONE]\n\n",
		`data: {"error":{"message":"private-server-detail"}}` + "\n\n",
	}
	for _, data := range cases {
		if _, err := chatcompletions.Read(strings.NewReader(data), func(provider.Delta) error { return nil }); err == nil {
			t.Fatalf("accepted %s", data)
		} else if strings.Contains(err.Error(), "private-server-detail") {
			t.Fatal("raw server error exposed")
		}
	}
}

func TestCompletionCancellationAndDisabledThinking(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["thinking"].(map[string]any)["type"] != "disabled" || body["reasoning_effort"] != nil || body["tools"] != nil {
			t.Error("thinking was not disabled")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		_, err := (&Client{client: server.Client(), baseURL: server.URL}).Complete(ctx, "test-key", provider.Request{Effort: "none"}, func(provider.Delta) error { return nil })
		finished <- err
	}()
	<-started
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not cancel")
	}
}

func TestContextOverflowIsDistinctFromOtherBadRequests(t *testing.T) {
	for _, test := range []struct {
		body     string
		overflow bool
	}{
		{`{"error":{"code":"context_length_exceeded","message":"Too long"}}`, true},
		{`{"error":{"message":"This model's maximum context length is exceeded"}}`, true},
		{`{"error":{"message":"Invalid reasoning_effort"}}`, false},
		{`invalid JSON`, false},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400); w.Write([]byte(test.body)) }))
		client := &Client{client: server.Client(), baseURL: server.URL}
		_, err := client.Complete(context.Background(), "key", provider.Request{}, func(provider.Delta) error { return nil })
		server.Close()
		if errors.Is(err, provider.ErrContextLength) != test.overflow {
			t.Fatal(err, test.body)
		}
	}
}

func TestToolArgumentsStreamBeforeCompletion(t *testing.T) {
	reader, writer := io.Pipe()
	received := make(chan provider.ToolDelta, 2)
	done := make(chan error, 1)
	go func() {
		_, err := chatcompletions.Read(reader, func(delta provider.Delta) error {
			if delta.Tool != nil {
				received <- *delta.Tool
			}
			return nil
		})
		done <- err
	}()
	defer reader.Close()
	defer writer.Close()
	first := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"write","function":{"name":"files","arguments":"{\"operation\":\"write\","}}]}}]}` + "\n\n"
	if _, err := io.WriteString(writer, first); err != nil {
		t.Fatal(err)
	}
	select {
	case delta := <-received:
		if delta.Call.ID != "write" || delta.Call.Function.Arguments != `{"operation":"write",` {
			t.Fatal(delta)
		}
	case <-time.After(time.Second):
		t.Fatal("tool delta waited for completion")
	}
	final := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"path\":\"a.txt\"}"}}]},"finish_reason":"tool_calls"}]}` + "\n\ndata: [DONE]\n\n"
	if _, err := io.WriteString(writer, final); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
