package app

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPermissionPreferencesAndToolExecution(t *testing.T) {
	state := t.TempDir()
	directory, _ := filepath.EvalSymlinks(t.TempDir())
	app, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	initial := app.Snapshot().Settings
	if initial.Permissions.Mode != permission.Folders {
		t.Fatal("new profile must use Auto")
	}
	if _, err := app.Configure(initial.Run, directory); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ConfigurePermissions(permission.Policy{Mode: permission.Folders, Roots: []string{directory}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"operation": "write", "path": "file", "content": "saved"})
	result := app.RunTool(context.Background(), tool.Call{ID: "write", Name: "files", Arguments: raw})
	if result.Failed {
		t.Fatal(result.Text)
	}
	app.Close()
	app, err = Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	if app.Snapshot().Settings.Permissions.Mode != permission.Folders {
		t.Fatal("mode not restored")
	}
	if body, _ := os.ReadFile(filepath.Join(directory, "file")); string(body) != "saved" {
		t.Fatal("tool failed to save")
	}
}
