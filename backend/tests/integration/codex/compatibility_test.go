package codex_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDesktopCompatibility(t *testing.T) {
	for _, model := range []string{"gpt-5.5", "gpt-6-astra"} {
		t.Run(model, func(t *testing.T) { checkCompatibility(t, model) })
	}
}

func checkCompatibility(t *testing.T, modelName string) {
	if os.Getenv("ARX_CODEX_BINARY") == "" {
		t.Skip("Set ARX_CODEX_BINARY to run Codex compatibility checks")
	}
	requests := make(chan map[string]json.RawMessage, 8)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request map[string]json.RawMessage
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&request); err != nil {
			http.Error(w, "Invalid request", 400)
			return
		}
		requests <- request
		w.Header().Set("Content-Type", "text/event-stream")
		message := map[string]any{"id": "msg_test", "type": "message", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": "ARX_CHECK_OK", "annotations": []any{}}}}
		events := []any{
			map[string]any{"type": "response.created", "response": map[string]any{"id": "resp_test", "status": "in_progress", "output": []any{}}},
			map[string]any{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{"id": "msg_test", "type": "message", "role": "assistant", "status": "in_progress", "content": []any{}}},
			map[string]any{"type": "response.output_text.delta", "item_id": "msg_test", "output_index": 0, "content_index": 0, "delta": "ARX_CHECK_OK"},
			map[string]any{"type": "response.output_item.done", "output_index": 0, "item": message},
			map[string]any{"type": "response.completed", "response": map[string]any{"id": "resp_test", "status": "completed", "output": []any{message}, "usage": map[string]int{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}},
		}
		for _, event := range events {
			body, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", body)
		}
	}))
	defer model.Close()
	home := t.TempDir()
	config := fmt.Sprintf(`model = %q
model_provider = "local_test"
web_search = "disabled"
[model_inference.local_test]
name = "Local test"
base_url = %q
wire_api = "responses"
request_max_retries = 0
stream_max_retries = 0
[features]
multi_agent = false
multi_agent_v2 = false
apps = false
shell_tool = false
unified_exec = false
shell_snapshot = false
goals = false
sleep_tool = false
tool_suggest = false
`, modelName, model.URL+"/v1")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	server := startServer(t, home)
	var features struct {
		Data []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"data"`
	}
	server.call(t, "experimentalFeature/list", map[string]any{"limit": 200}, &features)
	for _, feature := range features.Data {
		if (feature.Name == "multi_agent" || feature.Name == "multi_agent_v2") && feature.Enabled {
			t.Fatalf("Delegation enabled: %s", feature.Name)
		}
	}
	tools := []any{}
	for _, name := range []string{"files", "web", "bash"} {
		tools = append(tools, map[string]any{"type": "function", "name": name, "description": "Compatibility check", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}})
	}
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Sandbox struct {
			Type string `json:"type"`
		} `json:"sandbox"`
		ApprovalPolicy string `json:"approvalPolicy"`
	}
	for _, mode := range []struct{ sandbox, approval, expected string }{
		{"read-only", "on-request", "readOnly"},
		{"workspace-write", "on-request", "workspaceWrite"},
		{"danger-full-access", "never", "dangerFullAccess"},
	} {
		server.call(t, "thread/start", map[string]any{"cwd": t.TempDir(), "sandbox": mode.sandbox, "approvalPolicy": mode.approval, "approvalsReviewer": "user", "environments": []any{}, "dynamicTools": tools}, &started)
		if started.Sandbox.Type != mode.expected || started.ApprovalPolicy != mode.approval {
			t.Fatalf("Unexpected permission mapping: %+v", started)
		}
	}
	server.turn(t, started.Thread.ID, "Remember ARX_SAVED_HISTORY. Reply ARX_CHECK_OK without using tools.")
	request := <-requests
	names := advertisedTools(t, request)
	expected := []string{"bash", "files", "request_user_input", "web"}
	if modelName == "gpt-6-astra" {
		expected = []string{"collaboration.followup_task", "collaboration.interrupt_agent", "collaboration.list_agents", "collaboration.send_message", "collaboration.spawn_agent", "collaboration.wait_agent", "functions.exec", "functions.request_user_input", "functions.request_user_input_async", "functions.wait"}
		t.Log("GPT-6 still advertises delegation with multi_agent and multi_agent_v2 disabled; it does not meet Arx's tool restriction")
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("Tool surface changed for %s: %v", modelName, names)
	}
	thread := started.Thread.ID
	server.close(t)
	server = startServer(t, home)
	var resumed struct {
		Thread struct {
			ID    string            `json:"id"`
			Turns []json.RawMessage `json:"turns"`
		} `json:"thread"`
	}
	server.call(t, "thread/resume", map[string]any{"threadId": thread, "environments": []any{}}, &resumed)
	if resumed.Thread.ID != thread || len(resumed.Thread.Turns) != 1 {
		t.Fatalf("Session was not restored: %+v", resumed)
	}
	server.turn(t, thread, "What was the test marker?")
	request = <-requests
	if !strings.Contains(string(request["input"]), "ARX_SAVED_HISTORY") {
		t.Fatal("Resumed request lost earlier history")
	}
}

func advertisedTools(t *testing.T, request map[string]json.RawMessage) []string {
	t.Helper()
	type tool struct {
		Type  string            `json:"type"`
		Name  string            `json:"name"`
		Tools []json.RawMessage `json:"tools"`
	}
	var tools []json.RawMessage
	if len(request["tools"]) > 0 {
		if err := json.Unmarshal(request["tools"], &tools); err != nil {
			t.Fatal(err)
		}
	}
	var input []struct {
		Type  string            `json:"type"`
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(request["input"], &input); err != nil {
		t.Fatal(err)
	}
	for _, item := range input {
		if item.Type == "additional_tools" {
			tools = append(tools, item.Tools...)
		}
	}
	names := []string{}
	var visit func(json.RawMessage, string)
	visit = func(raw json.RawMessage, prefix string) {
		var value tool
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		if value.Type == "namespace" {
			for _, child := range value.Tools {
				visit(child, prefix+value.Name+".")
			}
		} else {
			names = append(names, prefix+value.Name)
		}
	}
	for _, value := range tools {
		visit(value, "")
	}
	sort.Strings(names)
	return names
}
