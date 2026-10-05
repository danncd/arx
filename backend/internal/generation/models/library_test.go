package generation

import (
	download "arx/internal/platform/download"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadConfigureRecoverAndRemove(t *testing.T) {
	payload := []byte("model test fixture")
	digest := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) }))
	defer server.Close()
	assets, directory := t.TempDir(), t.TempDir()
	model := Model{ID: "sample", Name: "Sample", Category: "image", Files: []download.File{{Name: "weights/model.bin", Size: int64(len(payload)), URL: server.URL, SHA256: hex.EncodeToString(digest[:])}}}
	data, _ := json.Marshal([]Model{model})
	if err := os.WriteFile(filepath.Join(assets, "catalog.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	library, err := Open(directory, assets)
	if err != nil {
		t.Fatal(err)
	}
	if err := library.Configure("image", "sample"); err == nil {
		t.Fatal("uninstalled model configured")
	}
	if err := library.Download("sample"); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(3 * time.Second)
	for library.Snapshot().Models[0].Status == "downloading" {
		select {
		case <-deadline:
			t.Fatal("download did not finish")
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := library.Configure("image", "sample"); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(directory, assets)
	if err != nil {
		t.Fatal(err)
	}
	_, resolved, err := restored.Resolve("image")
	if err != nil || resolved != filepath.Join(directory, "sample") {
		t.Fatalf("restore: %s %v", resolved, err)
	}
	if err := os.Remove(filepath.Join(resolved, "weights/model.bin")); err != nil {
		t.Fatal(err)
	}
	repaired, err := Open(directory, assets)
	if err != nil || repaired.Snapshot().Models[0].Status != "failed" {
		t.Fatal("missing dependency was not detected", err)
	}
	if err := repaired.Remove("sample"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repaired.Resolve("image"); err == nil {
		t.Fatal("removed model still configured")
	}
}

func TestSavedManifestCannotReplaceTrustedPaths(t *testing.T) {
	assets, directory := t.TempDir(), t.TempDir()
	trusted := Model{ID: "sample", Category: "image", Files: []download.File{{Name: "weights.bin", Size: 1, URL: "https://example.com/weights"}}}
	data, _ := json.Marshal([]Model{trusted})
	os.WriteFile(filepath.Join(assets, "catalog.json"), data, 0600)
	changed := trusted
	changed.Files = []download.File{{Name: "../outside", Size: 1}}
	state, _ := json.Marshal(State{Models: []Entry{{Model: changed, Status: "paused"}}})
	os.WriteFile(filepath.Join(directory, "library.json"), state, 0600)
	library, err := Open(directory, assets)
	if err != nil || library.Snapshot().Models[0].Model.Files[0].Name != "weights.bin" {
		t.Fatal("untrusted paths restored", err)
	}
}
