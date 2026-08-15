package tool

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

/* Captures both streams in order. */

func TestBash(t *testing.T) {
	out, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"echo one; echo two 1>&2"}`))
	if err != nil || out != "one\ntwo\n" {
		t.Fatalf("got %q, %v", out, err)
	}
}

/* Exit codes fail the call but carry the output for the model. */

func TestBashExit(t *testing.T) {
	_, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"echo boom; exit 3"}`))
	if err == nil || !strings.Contains(err.Error(), "exit 3") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want exit 3 with output, got %v", err)
	}
}

/* The deadline kills the whole process group. */

func TestBashTimeout(t *testing.T) {
	start := time.Now()
	_, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"sleep 30","timeout_secs":1}`))
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("want timeout, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("deadline did not cut the command short")
	}
}

/* Credentials never reach the child process. */

func TestBashEnvScrub(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-canary")
	t.Setenv("GITHUB_PAT", "gh-canary")
	t.Setenv("LC_API_KEY", "lc-canary")
	out, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"env; true"}`))
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}
	if strings.Contains(out, "sk-canary") || strings.Contains(out, "gh-canary") || strings.Contains(out, "lc-canary") {
		t.Fatal("credential leaked into the child env")
	}
}

func TestBashCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(100*time.Millisecond, cancel)
	start := time.Now()
	_, err := Bash.Run(ctx, json.RawMessage(`{"command":"sleep 30"}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context cancellation, got %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancellation did not cut the command short")
	}
}

func TestBashRejectsCanceledEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ran")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args, _ := json.Marshal(map[string]string{"command": "touch " + strconv.Quote(path)})
	_, err := Bash.Run(ctx, args)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context cancellation, got %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("canceled command ran: %v", err)
	}
}

func TestBashTimeoutDoesNotOverflow(t *testing.T) {
	args := `{"command":"true","timeout_secs":` + strconv.Itoa(int(^uint(0)>>1)) + `}`
	if _, err := Bash.Run(context.Background(), json.RawMessage(args)); err != nil {
		t.Fatalf("large timeout overflowed: %v", err)
	}
}

func TestBashPreservesParentDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := Bash.Run(ctx, json.RawMessage(`{"command":"sleep 30"}`))
	if !errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "after 30s") {
		t.Fatalf("parent deadline was misreported: %v", err)
	}
}

func TestReadFileConfinesPaths(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	out, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"note.txt"}`))
	if err != nil || out != "inside" {
		t.Fatalf("in-tree read = %q, %v", out, err)
	}
	if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"../outside.txt"}`)); err == nil {
		t.Fatal("parent traversal was accepted")
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.txt")); err == nil {
		if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"link.txt"}`)); err == nil {
			t.Fatal("symlink escape was accepted")
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("credential"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".git", filepath.Join(root, "metadata")); err == nil {
		if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"metadata/config"}`)); err == nil {
			t.Fatal("protected directory symlink was accepted")
		}
	}
}

func TestReadFileRejectsProtectedAndSpecialFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=canary"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":".env"}`)); err == nil {
		t.Fatal(".env was readable")
	}
	if err := os.Link(filepath.Join(root, ".env"), filepath.Join(root, "notes.txt")); err == nil {
		if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"notes.txt"}`)); err == nil {
			t.Fatal(".env hard-link alias was readable")
		}
	}
	if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"."}`)); err == nil {
		t.Fatal("directory was readable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadFile.Run(ctx, json.RawMessage(`{"path":".env"}`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
}

func TestFetchRejectsPrivateTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("private"))
	}))
	t.Cleanup(srv.Close)

	args, _ := json.Marshal(map[string]string{"url": srv.URL})
	if out, err := Fetch.Run(context.Background(), args); err == nil {
		t.Fatalf("private target returned %q", out)
	}
}

func TestFetchAcceptsNoContent(t *testing.T) {
	old := fetchClient
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	fetchClient = srv.Client()
	t.Cleanup(func() { fetchClient = old })

	args, _ := json.Marshal(map[string]string{"url": srv.URL})
	out, err := Fetch.Run(context.Background(), args)
	if err != nil || out != "" {
		t.Fatalf("204 response = %q, %v", out, err)
	}
}

func TestPublicIP(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":              true,
		"2606:4700:4700::1111": true,
		"127.0.0.1":            false,
		"10.0.0.1":             false,
		"169.254.1.1":          false,
		"100.64.0.1":           false,
		"198.18.0.1":           false,
		"192.0.2.1":            false,
		"240.0.0.1":            false,
		"::1":                  false,
		"fc00::1":              false,
		"fe80::1":              false,
		"fec0::1":              false,
		"2001:db8::1":          false,
	}
	for raw, want := range cases {
		if got := publicIP(net.ParseIP(raw)); got != want {
			t.Errorf("publicIP(%s) = %v, want %v", raw, got, want)
		}
	}
}

/* Oversized output comes back capped, cut on a rune boundary. */

func TestBashTruncates(t *testing.T) {
	out, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"head -c 100000 /dev/zero | tr '\\0' 'x'"}`))
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}
	if !strings.HasSuffix(out, "[truncated at 64KB]") {
		t.Fatal("missing truncation marker")
	}
}
