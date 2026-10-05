package engine

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRecoverStopsOnlyOwnedEngine(t *testing.T) {
	directory := t.TempDir()
	engineDir := filepath.Join(directory, "engine")
	if err := os.Mkdir(engineDir, 0700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(engineDir, "test-server")
	if err := os.WriteFile(binary, []byte("sleep 30\n"), 0600); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(directory, ".runtime-key-test")
	if err := os.WriteFile(key, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/sh", binary, key)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { command.Wait(); close(done) }()
	defer syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	record := processRecord{PID: command.Process.Pid, Binary: binary, KeyFile: key}
	data, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(directory, "runtime-process.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	Recover(directory)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Owned runtime was not stopped")
	}
	if _, err := os.Stat(key); !os.IsNotExist(err) {
		t.Fatal("Runtime key was not removed")
	}
}

func TestRecoverIgnoresReusedPID(t *testing.T) {
	directory := t.TempDir()
	command := exec.Command("/bin/sleep", "30")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { command.Process.Kill(); command.Wait() }()
	record := processRecord{PID: command.Process.Pid, Binary: filepath.Join(directory, "engine", "llama-server"), KeyFile: filepath.Join(directory, ".runtime-key-old")}
	data, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(directory, "runtime-process.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	Recover(directory)
	if err := command.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("Unrelated process was stopped")
	}
}
