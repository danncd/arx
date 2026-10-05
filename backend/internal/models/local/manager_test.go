package local

import (
	"arx/internal/models/local/library"
	download "arx/internal/platform/download"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCancelStopsTransferAndRemovesEntry(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	manager, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	id := modelID("fixture")
	manager.state.Models = []library.Entry{{ID: id, Status: "paused", Size: 1000, Files: []download.File{{Name: "test.gguf", URL: server.URL, Size: 1000}}}}
	if err := manager.Resume(id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("Download did not start")
	}
	done := make(chan error, 1)
	go func() { done <- manager.Cancel(id) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Cancel blocked")
	}
	if len(manager.Snapshot().Models) != 0 {
		t.Fatal("Cancelled download remains")
	}
}
func TestLibraryIsExclusiveAndEntryRemovalPreservesSource(t *testing.T) {
	directory := t.TempDir()
	manager, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := Open(directory); err == nil {
		other.Close()
		t.Fatal("Second library owner accepted")
	}
	manager.Close()
	other, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	source := filepath.Join(t.TempDir(), "original.gguf")
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	active, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer active.Close()
	id := modelID(source)
	active.state.Models = []library.Entry{{ID: id, Path: source, Imported: true, Status: "installed"}}
	if err := active.Remove(id, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("Imported file was deleted", err)
	}
}
