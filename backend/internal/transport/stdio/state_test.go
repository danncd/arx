package stdio

import (
	service "arx/internal/app"
	session "arx/internal/sessions"
	protocol "arx/internal/transport/contract"
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestPersistentStateRequests(t *testing.T) {
	directory := t.TempDir()
	app, err := service.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if err := app.Record(session.TranscriptChunk{Conversation: "one", ID: "message", Role: "user", Text: "Persist me"}); err != nil {
		t.Fatal(err)
	}
	server := NewServer(app, io.Discard)
	for _, request := range []struct{ method, params string }{
		{"save_view", `{"key":"startup-mode","value":"new"}`},
		{"configure", `{"run":{"model":"","effort":""},"directory":` + quote(directory) + `}`},
		{"snapshot", `{}`},
		{"history", `{"conversation":"one","limit":10}`},
	} {
		response, _ := server.dispatch(context.Background(), protocol.Request{ID: "1", Method: request.method, Params: json.RawMessage(request.params)})
		if response.Error != nil {
			t.Fatalf("%s: %+v", request.method, response.Error)
		}
	}
	app.Close()
	app, err = service.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	snapshot := app.Snapshot()
	if len(snapshot.Conversations) != 1 || snapshot.Views["startup-mode"] != "new" || snapshot.Settings.Directory == "" {
		t.Fatalf("snapshot: %+v", snapshot)
	}
	server = NewServer(app, io.Discard)
	for _, params := range []string{`{}`, `{"key":"x","value":false}`, `{"key":"x","value":"v","extra":true}`} {
		response, _ := server.dispatch(context.Background(), protocol.Request{ID: "1", Method: "save_view", Params: json.RawMessage(params)})
		if response.Error == nil {
			t.Fatalf("invalid view accepted: %s", params)
		}
	}
}

func quote(value string) string { body, _ := json.Marshal(value); return string(body) }
