package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

const MaxBytes = 2 << 20
const MaxPerMessage = 4

type Image struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Store struct{ Directory string }

func (s Store) Read(id string) ([]byte, error) {
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != id {
		return nil, errors.New("Invalid image attachment")
	}
	file, err := os.Open(filepath.Join(s.Directory, "attachments", id+".png"))
	if err != nil {
		return nil, errors.New("Image attachment is missing. Attach it again")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxBytes+1))
	if err != nil || len(data) > MaxBytes {
		return nil, errors.New("Image attachment is too large")
	}
	digest := sha256.Sum256(data)
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width > 2048 || config.Height > 2048 || hex.EncodeToString(digest[:]) != id {
		return nil, errors.New("Image attachment is damaged. Attach it again")
	}
	return data, nil
}
