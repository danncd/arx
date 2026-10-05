package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRangeFallbackChecksSpaceBeforeTruncatingPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("complete")) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "weights")
	os.WriteFile(path+".part", []byte("part"), 0600)
	calls := []int64{}
	err := TransferChecked(context.Background(), File{URL: server.URL, Size: 8}, path, func(int64) {}, func(needed int64) error {
		calls = append(calls, needed)
		if needed > 4 {
			return errors.New("not enough space for restart")
		}
		return nil
	})
	if err == nil || len(calls) != 2 || calls[0] != 4 || calls[1] != 8 {
		t.Fatal(calls, err)
	}
	data, _ := os.ReadFile(path + ".part")
	if string(data) != "part" {
		t.Fatal("fallback truncated partial before checking space")
	}
}

func TestRemainingUsesFilesAndRejectsInvalidPartials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "weights")
	file := File{Size: 100, SHA256: "invalid"}
	for _, size := range []int{20, 100, 101} {
		os.WriteFile(path+".part", make([]byte, size), 0600)
		remaining, err := Remaining(context.Background(), file, path)
		want := int64(100)
		if size == 20 {
			want = 80
		}
		if err != nil || remaining != want {
			t.Fatal(size, remaining, err)
		}
	}
	file.SHA256 = ""
	os.WriteFile(path, make([]byte, 100), 0600)
	if n, err := Remaining(context.Background(), file, path); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
