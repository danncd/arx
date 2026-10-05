package local

import (
	"arx/internal/models/local/library"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelperCannotBeImportedOrLoaded(t *testing.T) {
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	path := filepath.Join(t.TempDir(), "mtp-Qwen3.8-Flash-Next-Q4_K_M.gguf")
	if err := os.WriteFile(path, []byte("GGUF"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Import(path, ""); err == nil || !strings.Contains(err.Error(), "prediction helper") {
		t.Fatalf("Helper import was not rejected: %v", err)
	}
	manager.state.Hardware.Supported = true
	id := modelID(path)
	manager.state.Models = []library.Entry{{ID: id, Path: path, Status: "installed"}}
	if err := manager.Load(id); err == nil || !strings.Contains(err.Error(), "prediction helper") {
		t.Fatalf("Previously downloaded helper was not rejected: %v", err)
	}
	if manager.runtimeCancel != nil {
		t.Fatal("Helper started the runtime")
	}
}
