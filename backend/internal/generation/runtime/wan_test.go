package engines

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestWanRuntimeExtraction(t *testing.T) {
	for _, complete := range []bool{false, true} {
		target := t.TempDir()
		archive := filepath.Join(target, "runtime.zip")
		file, err := os.Create(archive)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		for name := range wanRuntimeFiles {
			if !complete && name == "sd-cli" {
				continue
			}
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			entry.Write([]byte("fixture"))
		}
		entry, _ := writer.Create("../outside")
		entry.Write([]byte("ignored"))
		writer.Close()
		file.Close()
		err = unpackWan(archive, target)
		if (err == nil) != complete || wanRuntimeReady(target) != complete {
			t.Fatalf("complete=%v: %v", complete, err)
		}
		if complete {
			info, err := os.Stat(filepath.Join(target, "native", "sd-cli"))
			if err != nil || info.Mode()&0100 == 0 {
				t.Fatal("runtime is not executable", err)
			}
		}
		if _, err := os.Stat(filepath.Join(target, "outside")); !os.IsNotExist(err) {
			t.Fatal("unexpected archive entry extracted")
		}
	}
}
