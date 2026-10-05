package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransferResumesAndVerifies(t *testing.T) {
	data := strings.Repeat("model", 1000)
	sum := sha256.Sum256([]byte(data))
	rangeSeen := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeSeen = r.Header.Get("Range")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 17-%d/%d", len(data)-1, len(data)))
		w.WriteHeader(206)
		w.Write([]byte(data[17:]))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "model.gguf")
	os.WriteFile(path+".part", []byte(data[:17]), 0600)
	file := File{URL: server.URL, Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])}
	last := int64(0)
	if err := Transfer(context.Background(), file, path, func(n int64) { last = n }); err != nil {
		t.Fatal(err)
	}
	actual, _ := os.ReadFile(path)
	if string(actual) != data || last != int64(len(data)) || rangeSeen != "bytes=17-" {
		t.Fatal("Resume lost bytes or progress")
	}
	if _, err := os.Stat(path + ".part"); !os.IsNotExist(err) {
		t.Fatal("Partial file remains")
	}
}
func TestTransferRejectsWrongRangeAndChecksum(t *testing.T) {
	for _, mode := range []string{"range", "hash"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if mode == "range" {
					w.Header().Set("Content-Range", "bytes 0-3/4")
					w.WriteHeader(206)
				}
				w.Write([]byte("data"))
			}))
			defer server.Close()
			path := filepath.Join(t.TempDir(), "model.gguf")
			if mode == "range" {
				os.WriteFile(path+".part", []byte("d"), 0600)
			}
			err := Transfer(context.Background(), File{URL: server.URL, Size: 4, SHA256: strings.Repeat("0", 64)}, path, func(int64) {})
			if err == nil {
				t.Fatal("Invalid download accepted")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("Invalid model installed")
			}
		})
	}
}
