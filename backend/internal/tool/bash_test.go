//go:build unix

package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestBashCapturesBothStreams(t *testing.T) {
	out, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"echo one; echo two 1>&2"}`))
	if err != nil || out != "one\ntwo\n" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestBashExitCarriesOutput(t *testing.T) {
	_, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"echo boom; exit 3"}`))
	if err == nil || !strings.Contains(err.Error(), "exit 3") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("want exit 3 with output, got %v", err)
	}
}

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

func TestBashTruncates(t *testing.T) {
	out, err := Bash.Run(context.Background(), json.RawMessage(`{"command":"head -c 100000 /dev/zero | tr '\\0' 'x'"}`))
	if err != nil {
		t.Fatalf("Bash: %v", err)
	}
	if !strings.HasSuffix(out, "[truncated at 64KB]") {
		t.Fatal("missing truncation marker")
	}
}

func TestBashSurvivesDetachedPipeHolder(t *testing.T) {
	if _, err := exec.LookPath("perl"); err != nil {
		t.Skip("perl not available")
	}
	old := bashDrainTimeout
	bashDrainTimeout = 200 * time.Millisecond
	t.Cleanup(func() { bashDrainTimeout = old })

	done := make(chan error, 1)
	go func() {
		_, err := Bash.Run(context.Background(), json.RawMessage(
			`{"command":"perl -MPOSIX -e 'POSIX::setsid() >= 0 or die; sleep 3' & sleep 0.5; echo started"}`))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Bash: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Bash.Run hung on a detached pipe holder")
	}
}

func TestBashKillsBackgroundChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	args, _ := json.Marshal(map[string]string{
		"command": fmt.Sprintf("sleep 30 >/dev/null 2>&1 & echo $! > %q", pidFile),
	})
	if _, err := Bash.Run(context.Background(), args); err != nil {
		t.Fatalf("Bash: %v", err)
	}
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatal("background child survived Bash.Run")
}
