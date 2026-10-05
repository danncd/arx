package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeHandoffPreservesPNGIdentityAndRejectsEscapes(t *testing.T) {
	store := Store{Directory: t.TempDir()}
	directory := filepath.Join(store.Directory, "attachment-imports")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	name := "00112233445566778899aabbccddeeff.png"
	file := filepath.Join(directory, name)
	if err := os.WriteFile(file, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Import(file, "picked.png")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(encoded.Bytes())
	if saved.ID != hex.EncodeToString(digest[:]) {
		t.Fatal("native image was re-encoded")
	}
	data, err := store.Read(saved.ID)
	if err != nil || !bytes.Equal(data, encoded.Bytes()) {
		t.Fatal("image not stored intact", err)
	}
	outside := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(outside, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Import(outside, "outside.png"); err == nil {
		t.Fatal("outside handoff accepted")
	}
	os.Remove(file)
	if err := os.Symlink(outside, file); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Import(file, "escape.png"); err == nil {
		t.Fatal("symlink escaped handoff root")
	}
}
