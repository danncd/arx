package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateReplacementAndFailedPublicationCleanup(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "saved.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, []byte("new"), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new" {
		t.Fatal("replacement missing", err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("private permissions lost", err)
	}
	blocked := filepath.Join(directory, "blocked")
	os.Mkdir(blocked, 0700)
	if err := Replace(blocked, []byte("never published"), true); err == nil {
		t.Fatal("published over a directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatal("temporary file leaked", entries)
	}
	data, _ = os.ReadFile(path)
	if string(data) != "new" {
		t.Fatal("unrelated saved value changed")
	}
}
