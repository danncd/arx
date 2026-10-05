package bash

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func execute(t *testing.T, ctx context.Context, directory string, policy permission.Policy, command string, timeout int) (tool.Result, error) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"command": command, "timeout_seconds": timeout})
	prepared, err := Prepare(directory, policy, raw)
	if err != nil {
		t.Fatal(err)
	}
	return prepared.Run(ctx)
}

func TestBashOutputExitCodeAndWorkingDirectory(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	result, err := execute(t, context.Background(), root, permission.Policy{Mode: permission.Full}, "pwd; printf error >&2; exit 7", 5)
	if err != nil || !result.Failed || *result.ExitCode != 7 || !strings.Contains(result.Text, root) || !strings.Contains(result.Text, "error") {
		t.Fatalf("result: %+v %v", result, err)
	}
}

func TestSelectedFolderSandboxBlocksWritesAndSymlinkEscapes(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	outside := t.TempDir()
	policy := permission.Policy{Mode: permission.Folders, Roots: []string{root}}
	result, err := execute(t, context.Background(), root, policy, "printf allowed > inside.txt", 5)
	if err != nil || result.Failed {
		t.Fatalf("allowed write: %+v %v", result, err)
	}
	if body, _ := os.ReadFile(filepath.Join(root, "inside.txt")); string(body) != "allowed" {
		t.Fatal("allowed file missing")
	}
	path := filepath.Join(outside, "blocked")
	os.Symlink(outside, filepath.Join(root, "escape"))
	for _, target := range []string{path, filepath.Join(root, "escape", "blocked")} {
		result, err = execute(t, context.Background(), root, policy, "printf denied > "+strconv.Quote(target), 5)
		if err != nil || !result.Failed {
			t.Fatalf("outside write: %+v %v", result, err)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("sandbox escape wrote a file")
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Write([]byte("reachable"))
	}))
	defer server.Close()
	result, err = execute(t, context.Background(), root, policy, "/usr/bin/curl --max-time 2 -sS "+server.URL, 5)
	if err != nil || !result.Failed {
		t.Fatalf("network: %+v %v", result, err)
	}
	if requests.Load() != 0 {
		t.Fatal("sandbox reached local HTTP server")
	}
}

func TestTimeoutKillsChildBeforeDelayedWrite(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "late")
	result, err := execute(t, context.Background(), root, permission.Policy{Mode: permission.Full}, "(sleep 2; printf late > late) & wait", 1)
	if err != context.DeadlineExceeded {
		t.Fatalf("timeout: %+v %v", result, err)
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("child outlived cancellation")
	}
}

func TestCommandOutputIsBounded(t *testing.T) {
	result, err := execute(t, context.Background(), t.TempDir(), permission.Policy{Mode: permission.Full}, "/usr/bin/yes output | /usr/bin/head -c 100000", 5)
	if err != nil || !result.Truncated || len(result.Text) > tool.MaxOutput {
		t.Fatalf("output: %d %v %v", len(result.Text), result.Truncated, err)
	}
}

func TestSandboxCannotCreateWritableHardlinkToOutsideFile(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	outside := filepath.Join(t.TempDir(), "file")
	os.WriteFile(outside, []byte("untouched"), 0600)
	policy := permission.Policy{Mode: permission.Folders, Roots: []string{root}}
	execute(t, context.Background(), root, policy, "ln "+strconv.Quote(outside)+" link && printf changed > link", 5)
	body, _ := os.ReadFile(outside)
	if string(body) != "untouched" {
		t.Fatal("hardlink escaped sandbox")
	}
}
