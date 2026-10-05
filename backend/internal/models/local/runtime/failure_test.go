package engine

import (
	model "arx/internal/models"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartupReportsModelFailure(t *testing.T) {
	directory := t.TempDir()
	binary := filepath.Join(directory, "engine")
	script := "#!/bin/sh\nprintf '%s\\n' \"error loading model: tensor 'output_hc_norm.weight' not found\" >&2\nexit 1\n"
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	server := &Server{}
	defer server.Stop()
	_, err := server.Start(context.Background(), binary, "model.gguf", "", model.Info{ContextWindow: 8192}, filepath.Join(directory, "runtime.log"))
	if err == nil || !strings.Contains(err.Error(), "output_hc_norm.weight") {
		t.Fatalf("Missing engine failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "runtime-process.json")); !os.IsNotExist(err) {
		t.Fatal("Failed process record remains")
	}
}

func TestStartupFailureReadsBoundedLogTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 32<<10)+"\nfailed to allocate buffer\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadFailure(path); !strings.Contains(err.Error(), "Not enough memory") {
		t.Fatal(err)
	}
}
