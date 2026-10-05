package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteImportedOnlyRemovesItsFiles(t *testing.T) {
	directory := t.TempDir()
	names := []string{"model-00001-of-00002.gguf", "model-00002-of-00002.gguf", "projector.gguf", "other.gguf", "notes.txt"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	entry := Entry{ID: "test", Path: filepath.Join(directory, names[0]), Projector: filepath.Join(directory, names[2]), Imported: true}
	if size, err := entry.FileSize(); err != nil || size != 21 {
		t.Fatal(size, err)
	}
	if err := DeleteImported(entry, []Entry{entry}); err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		_, err := os.Stat(filepath.Join(directory, name))
		if i < 3 && !os.IsNotExist(err) {
			t.Fatal("Model file remains", name)
		}
		if i >= 3 && err != nil {
			t.Fatal("Unrelated file removed", name)
		}
	}
}
func TestDeleteImportedRejectsSharedFilesAndSymlinks(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "model.gguf")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := Entry{ID: "one", Path: path, Imported: true}
	other := Entry{ID: "two", Path: path, Imported: true}
	if err := DeleteImported(entry, []Entry{entry, other}); err == nil {
		t.Fatal("Shared file removed")
	}
	link := filepath.Join(directory, "link.gguf")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	entry.Path = link
	if err := DeleteImported(entry, nil); err == nil {
		t.Fatal("Symlink accepted")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
