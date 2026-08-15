//go:build aix || android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package tool

import (
	"context"
	"encoding/json"
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

func TestReadFileRejectsFIFO(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	if _, err := ReadFile.Run(context.Background(), json.RawMessage(`{"path":"pipe"}`)); err == nil {
		t.Fatal("FIFO was readable")
	}
}

/* A child in its own session survives the group kill and keeps the
   pipe open; the drain must give up instead of hanging the turn. */

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
