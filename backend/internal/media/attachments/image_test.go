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

func TestReadValidatesStoredImage(t *testing.T) {
	directory := t.TempDir()
	os.Mkdir(filepath.Join(directory, "attachments"), 0700)
	var encoded bytes.Buffer
	png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	digest := sha256.Sum256(encoded.Bytes())
	id := hex.EncodeToString(digest[:])
	path := filepath.Join(directory, "attachments", id+".png")
	os.WriteFile(path, encoded.Bytes(), 0600)
	store := Store{Directory: directory}
	if data, err := store.Read(id); err != nil || !bytes.Equal(data, encoded.Bytes()) {
		t.Fatal("stored image did not round trip", err)
	}
	for _, invalid := range []string{"../settings", "abc", string(bytes.Repeat([]byte("a"), 64))} {
		if _, err := store.Read(invalid); err == nil {
			t.Fatal("accepted invalid or missing image", invalid)
		}
	}
	os.WriteFile(path, []byte("changed"), 0600)
	if _, err := store.Read(id); err == nil {
		t.Fatal("accepted changed image")
	}
}
