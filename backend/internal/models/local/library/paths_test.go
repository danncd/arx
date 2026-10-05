package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedPathsRejectTraversalAndSymlinks(t *testing.T) {
	root := t.TempDir()
	id := "local:" + strings.Repeat("a", 32)
	if _, err := ManagedDirectory(root, "local:../../outside"); err == nil {
		t.Fatal("Accepted traversal ID")
	}
	if _, err := ManagedPath(root, id, "../outside"); err == nil {
		t.Fatal("Accepted traversal file")
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "models")); err != nil {
		t.Fatal(err)
	}
	if _, err := ManagedDirectory(root, id); err == nil {
		t.Fatal("Accepted external symlink")
	}
}

func TestSavedLibraryRejectsExternalManagedFilesAndDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	id := "local:" + strings.Repeat("b", 32)
	entry := Entry{ID: id, Path: filepath.Join(t.TempDir(), "model.gguf"), Status: "installed"}
	if err := Save(root, []Entry{entry}); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root); err == nil {
		t.Fatal("Accepted external managed file")
	}
	entry.Imported = true
	if err := Save(root, []Entry{entry}); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root); err != nil {
		t.Fatal(err)
	}
	if err := Save(root, []Entry{entry, entry}); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(root); err == nil {
		t.Fatal("Accepted duplicate IDs")
	}
}
