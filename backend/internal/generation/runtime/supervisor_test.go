package engines

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	SupervisorMain()
	if len(os.Args) == 3 && os.Args[1] == "--test-generation-owner" {
		command, cleanup, err := supervisedCommand(context.Background(), os.Args[2], "/bin/sh", "-c", "echo $$ > \"$1/child.pid\"; while :; do sleep 1; done", "worker", os.Args[2])
		if err != nil {
			panic(err)
		}
		defer cleanup()
		if err := command.Start(); err != nil {
			panic(err)
		}
		command.Wait()
		return
	}
	os.Exit(m.Run())
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("subprocess did not reach expected state")
}

func TestGenerationOwnerDeathStopsWorkerAndRecoveryKeepsLiveOwner(t *testing.T) {
	directory := t.TempDir()
	binary, _ := os.Executable()
	owner := exec.Command(binary, "--test-generation-owner", directory)
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { owner.Process.Kill(); owner.Wait() })
	waitFor(t, func() bool { _, err := os.Stat(filepath.Join(directory, "child.pid")); return err == nil })
	data, _ := os.ReadFile(filepath.Join(directory, "child.pid"))
	child, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	if err := Recover(directory); err != nil {
		t.Fatal(err)
	}
	if syscall.Kill(child, 0) != nil {
		t.Fatal("recovery killed a live owner's worker")
	}
	owner.Process.Kill()
	owner.Wait()
	waitFor(t, func() bool {
		state, err := exec.Command("/bin/ps", "-p", strconv.Itoa(child), "-o", "stat=").Output()
		return err != nil || strings.HasPrefix(strings.TrimSpace(string(state)), "Z")
	})
	if err := Recover(directory); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryDoesNotSignalAnUnrelatedProcess(t *testing.T) {
	directory := t.TempDir()
	command := exec.Command("/bin/sleep", "30")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { command.Process.Kill(); command.Wait() }()
	start, _ := processStart(command.Process.Pid)
	token := strings.Repeat("a", 48)
	record := workerRecord{PID: command.Process.Pid, Owner: 99999999, Start: start, Token: token, Binary: "wrong", Target: "wrong"}
	path := filepath.Join(directory, ".worker-"+token+".json")
	if err := saveWorkerRecord(path, record); err != nil {
		t.Fatal(err)
	}
	if err := Recover(directory); err == nil {
		t.Fatal("identity mismatch silently accepted")
	}
	if err := syscall.Kill(command.Process.Pid, 0); err != nil {
		t.Fatal("unrelated process killed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("unconfirmed record removed")
	}
}

func TestCancellationStopsInstallerGroupAndStartBarrierPrecedesWork(t *testing.T) {
	directory := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command, cleanup, err := supervisedCommand(ctx, directory, "/bin/sh", "-c", "echo $$ > \"$1/child.pid\"; sleep 30", "installer", directory)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { _, err := os.Stat(filepath.Join(directory, "child.pid")); return err == nil })
	records, _ := filepath.Glob(filepath.Join(directory, ".worker-*.json"))
	if len(records) != 1 {
		t.Fatal("work began without a record")
	}
	data, _ := os.ReadFile(records[0])
	var record workerRecord
	if json.Unmarshal(data, &record) != nil || record.PID != command.Process.Pid {
		t.Fatal("wrong recorded identity")
	}
	cancel()
	if err := command.Wait(); err == nil {
		t.Fatal("cancelled installer succeeded")
	}
}
