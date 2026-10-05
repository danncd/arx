package artifacts

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveResolveAndRejectDamagedOutput(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "input.png")
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	store := Store{Directory: root}
	saved, err := store.Save(source, Artifact{Name: "lake.png", MIME: "image/png", Width: 2, Height: 2})
	if err != nil {
		t.Fatal(err)
	}
	held, path, err := store.Resolve(saved.ID)
	if err != nil || held != saved {
		t.Fatalf("resolve: %#v %v", held, err)
	}
	if _, _, err := store.Resolve("../input"); err == nil {
		t.Fatal("accepted traversal")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Resolve(saved.ID); err == nil {
		t.Fatal("accepted damaged file")
	}
	if _, err := store.Save(source, Artifact{MIME: "audio/wav"}); err == nil {
		t.Fatal("accepted wrong format")
	}
}
