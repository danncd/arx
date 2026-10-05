package artifacts

import (
	"bytes"
	"errors"
	"io"
)

func (s Store) verify(reader io.Reader, mime string) error {
	header := make([]byte, 12)
	if _, err := io.ReadFull(reader, header); err != nil {
		return errors.New("Generated media is incomplete")
	}
	valid := false
	switch mime {
	case "image/png":
		valid = bytes.Equal(header[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10})
	case "audio/wav":
		valid = string(header[:4]) == "RIFF" && string(header[8:12]) == "WAVE"
	case "video/mp4":
		valid = string(header[4:8]) == "ftyp"
	}
	if !valid {
		return errors.New("Generated media does not match its format")
	}
	return nil
}
