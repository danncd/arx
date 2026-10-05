package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryAndRelativePathsDoNotCreateASandbox(t *testing.T) {
	directory := t.TempDir()
	root, err := Directory(directory)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(root, "../elsewhere/file.go")
	if err != nil || got != filepath.Join(root, "../elsewhere/file.go") {
		t.Fatalf("relative path: %s %v", got, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.go")
	if got, err := Resolve(root, outside); err != nil || got != outside {
		t.Fatalf("absolute path: %s %v", got, err)
	}
	link := filepath.Join(directory, "alias")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	if got, err := Directory(link); err != nil || got != root {
		t.Fatalf("symlink: %s %v", got, err)
	}
	for _, value := range []string{"", "relative", filepath.Join(root, "missing")} {
		if _, err := Directory(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
