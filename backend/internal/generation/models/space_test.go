package generation

import (
	download "arx/internal/platform/download"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledModelNeedsNoFreeSpaceAndResumeUsesActualFiles(t *testing.T) {
	directory := t.TempDir()
	model := Model{ID: "test", Size: 100, Files: []download.File{{Name: "weights", Size: 100}}}
	library := &Library{directory: directory, catalog: []Model{model}, state: State{Models: []Entry{{Model: model, Status: "installed"}}}, freeBytes: func(string) (uint64, error) { return 0, nil }}
	path := filepath.Join(directory, "test", "weights")
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, make([]byte, 100), 0600)
	if err := library.Download("test"); err != nil {
		t.Fatal("installed no-op required space", err)
	}
	os.Remove(path)
	os.WriteFile(path+".part", make([]byte, 80), 0600)
	remaining, err := download.Remaining(context.Background(), model.Files[0], path)
	if err != nil || remaining != 20 {
		t.Fatal(remaining, err)
	}
	library.freeBytes = func(string) (uint64, error) { return generationDiskMargin + 20, nil }
	if err := library.checkSpace(remaining); err != nil {
		t.Fatal("usable partial not accounted for", err)
	}
	library.freeBytes = func(string) (uint64, error) { return generationDiskMargin + 19, nil }
	if err := library.checkSpace(remaining); err == nil {
		t.Fatal("insufficient space allowed")
	}
}
