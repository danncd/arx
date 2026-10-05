package codex_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type rpcMessage struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type testServer struct {
	command  *exec.Cmd
	input    io.WriteCloser
	messages chan rpcMessage
	stopped  chan error
	cancel   context.CancelFunc
	id       int
}

func startServer(t *testing.T, home string) *testServer {
	t.Helper()
	binary := os.Getenv("ARX_CODEX_BINARY")
	if binary == "" {
		t.Skip("Set ARX_CODEX_BINARY to run Codex compatibility checks")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	command := exec.CommandContext(ctx, binary, "app-server")
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CODEX_HOME=") {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, "CODEX_HOME="+home)
	input, err := command.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	server := &testServer{command: command, input: input, messages: make(chan rpcMessage, 256), stopped: make(chan error, 1), cancel: cancel}
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	go func() {
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, 4096), 8<<20)
		for scanner.Scan() {
			var message rpcMessage
			if json.Unmarshal(scanner.Bytes(), &message) != nil {
				continue
			}
			select {
			case server.messages <- message:
			case <-ctx.Done():
				return
			}
		}
		close(server.messages)
	}()
	go func() { server.stopped <- command.Wait() }()
	t.Cleanup(func() { server.close(t) })
	server.call(t, "initialize", map[string]any{
		"clientInfo":   map[string]string{"name": "arx_compatibility", "version": "0.1.0"},
		"capabilities": map[string]bool{"experimentalApi": true},
	}, nil)
	fmt.Fprintln(input, `{"method":"initialized"}`)
	return server
}

func (s *testServer) close(t *testing.T) {
	t.Helper()
	if s.cancel == nil {
		return
	}
	s.input.Close()
	select {
	case <-s.stopped:
	case <-time.After(3 * time.Second):
		s.cancel()
		<-s.stopped
	}
	s.cancel()
	s.cancel = nil
}

func (s *testServer) next(t *testing.T) rpcMessage {
	t.Helper()
	select {
	case message, ok := <-s.messages:
		if !ok {
			t.Fatal("Codex disconnected")
		}
		if message.ID != 0 && message.Method != "" {
			t.Fatalf("Unexpected server request: %s", message.Method)
		}
		return message
	case <-time.After(15 * time.Second):
		t.Fatal("Codex response timed out")
		return rpcMessage{}
	}
}

func (s *testServer) call(t *testing.T, method string, params any, result any) {
	t.Helper()
	s.id++
	if err := json.NewEncoder(s.input).Encode(map[string]any{"id": s.id, "method": method, "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		message := s.next(t)
		if message.ID != s.id {
			continue
		}
		if message.Error != nil {
			t.Fatalf("%s: %s", method, message.Error.Message)
		}
		if result != nil {
			if err := json.Unmarshal(message.Result, result); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
}

func (s *testServer) turn(t *testing.T, thread, text string) {
	t.Helper()
	s.call(t, "turn/start", map[string]any{"threadId": thread, "input": []any{map[string]string{"type": "text", "text": text}}}, nil)
	for {
		message := s.next(t)
		if message.Method != "turn/completed" {
			continue
		}
		var result struct {
			Turn struct {
				Status string `json:"status"`
			} `json:"turn"`
		}
		if err := json.Unmarshal(message.Params, &result); err != nil {
			t.Fatal(err)
		}
		if result.Turn.Status != "completed" {
			t.Fatalf("Turn status: %s", result.Turn.Status)
		}
		return
	}
}
