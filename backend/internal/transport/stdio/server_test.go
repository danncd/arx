package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	service "arx/internal/app"
	protocol "arx/internal/transport/contract"
)

func TestRequestsAndShutdown(t *testing.T) {
	input := "bad json\n" + `{"id":"1","method":"missing"}` + "\n" + `{"id":"2","method":"status"}` + "\n" + `{"id":"3","method":"shutdown"}` + "\n" + `{"id":"4","method":"status"}` + "\n"
	var output bytes.Buffer
	if err := NewServer(testService(t), &output).Serve(context.Background(), strings.NewReader(input)); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&output)
	var ready protocol.Event
	if err := decoder.Decode(&ready); err != nil || ready.Event != "ready" {
		t.Fatalf("ready: %v, %v", ready, err)
	}
	for index, code := range []string{"invalid_request", "unknown_method", "", ""} {
		var response protocol.Response
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if code != "" && (response.Error == nil || response.Error.Code != code) {
			t.Fatalf("response %d: %+v", index, response)
		}
		if code == "" && (response.Error != nil || response.Result == nil) {
			t.Fatalf("response %d: %+v", index, response)
		}
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		t.Fatalf("request processed after shutdown: %v", err)
	}
}

func TestOversizedInput(t *testing.T) {
	err := NewServer(testService(t), io.Discard).Serve(context.Background(), strings.NewReader(strings.Repeat("x", protocol.MaxMessageBytes+2)))
	if err == nil {
		t.Fatal("oversized input accepted")
	}
}

func TestEOF(t *testing.T) {
	if err := NewServer(testService(t), io.Discard).Serve(context.Background(), strings.NewReader("")); err != nil {
		t.Fatal(err)
	}
}

func testService(t *testing.T) *service.Service {
	t.Helper()
	app, err := service.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Close() })
	return app
}
