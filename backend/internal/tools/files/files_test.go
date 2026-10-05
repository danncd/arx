package files

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepare(t *testing.T, root string, policy permission.Policy, args map[string]any) tool.Prepared {
	t.Helper()
	raw, _ := json.Marshal(args)
	prepared, err := Prepare(root, policy, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Close != nil {
		t.Cleanup(prepared.Close)
	}
	return prepared
}

func TestChangesArePreparedThenPublishedAtomically(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "file.txt")
	prepared := prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "write", "path": "nested/file.txt", "content": "Hello 🌱\n"})
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file changed before authorization")
	}
	if _, err := prepared.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	os.Chmod(path, 0750)
	edited := prepare(t, root, permission.Policy{Mode: permission.Full}, map[string]any{"operation": "edit", "path": path, "old_text": "Hello", "new_text": "Hi"})
	if edited.Action.Before != "Hello 🌱\n" || edited.Action.After != "Hi 🌱\n" {
		t.Fatal("incorrect review")
	}
	if _, err := edited.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	if string(body) != "Hi 🌱\n" || info.Mode().Perm() != 0750 {
		t.Fatal("content or permissions lost")
	}
	read := prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "read", "path": path})
	result, err := read.Run(context.Background())
	if err != nil || result.Text != "1: Hi 🌱\n" {
		t.Fatalf("read: %+v %v", result, err)
	}
}

func TestFileChangedWhileAwaitingApprovalIsNotOverwritten(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file")
	os.WriteFile(path, []byte("original"), 0600)
	prepared := prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "write", "path": path, "content": "replacement"})
	os.WriteFile(path, []byte("user change"), 0600)
	if _, err := prepared.Run(context.Background()); err == nil {
		t.Fatal("concurrent edit overwritten")
	}
	body, _ := os.ReadFile(path)
	if string(body) != "user change" {
		t.Fatal("user edit lost")
	}
}

func TestSelectedFoldersRejectSymlinkEscapes(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	outside := t.TempDir()
	policy := permission.Policy{Mode: permission.Folders, Roots: []string{root}}
	os.Symlink(outside, filepath.Join(root, "link"))
	raw, _ := json.Marshal(map[string]any{"operation": "write", "path": "link/file", "content": "denied"})
	if _, err := Prepare(root, policy, raw, nil); err == nil {
		t.Fatal("symlink escape prepared")
	}
	prepared := prepare(t, root, policy, map[string]any{"operation": "write", "path": "new/file", "content": "denied"})
	os.Symlink(outside, filepath.Join(root, "new"))
	if _, err := prepared.Run(context.Background()); err == nil {
		t.Fatal("late symlink escape succeeded")
	}
	if _, err := os.Stat(filepath.Join(outside, "file")); !os.IsNotExist(err) {
		t.Fatal("outside file written")
	}
}

func TestAmbiguousEditsAndSpecialFilesAreRejected(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "file"), []byte("same same"), 0600)
	for _, args := range []map[string]any{
		{"operation": "edit", "path": "file", "old_text": "same", "new_text": "changed"},
		{"operation": "edit", "path": "file", "old_text": "missing", "new_text": "changed"},
		{"operation": "write", "path": "file"},
		{"operation": "write", "path": ".", "content": "bad"},
	} {
		raw, _ := json.Marshal(args)
		if prepared, err := Prepare(root, permission.Policy{Mode: permission.Full}, raw, nil); err == nil {
			if prepared.Close != nil {
				prepared.Close()
			}
			t.Fatal("invalid change accepted")
		}
	}
	os.WriteFile(filepath.Join(root, "binary"), []byte{0, 1, 2}, 0600)
	prepared := prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "read", "path": "binary"})
	if _, err := prepared.Run(context.Background()); err == nil {
		t.Fatal("binary file read as text")
	}
}

func TestSearchAndReadLimits(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "file.go"), []byte("one\nmatch\nmatch\n"), 0600)
	prepared := prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "search", "path": ".", "pattern": "match", "limit": 1})
	result, err := prepared.Run(context.Background())
	if err != nil || !strings.Contains(result.Text, "file.go:2: match") || !result.Truncated {
		t.Fatalf("search: %+v %v", result, err)
	}
	prepared = prepare(t, root, permission.Policy{Mode: permission.Ask}, map[string]any{"operation": "read", "path": "file.go", "offset": 2, "limit": 1})
	result, err = prepared.Run(context.Background())
	if err != nil || result.Text != "2: match\n" || !result.Truncated {
		t.Fatalf("read: %+v %v", result, err)
	}
}
