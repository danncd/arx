package attachments

import (
	"arx/internal/platform/atomicfile"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

const MaxSourceBytes = 8 << 20

func (s Store) Save(data []byte, name string) (Image, error) {
	if len(data) > MaxSourceBytes {
		return Image{}, errors.New("Image exceeds 8 MiB")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 8192 || config.Height > 8192 || int64(config.Width)*int64(config.Height) > 32<<20 {
		return Image{}, errors.New("Use a PNG, JPEG, GIF, or WebP image up to 8192 pixels per side and 32 megapixels")
	}
	picture, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Image{}, errors.New("Image could not be decoded")
	}
	width, height := config.Width, config.Height
	if max(width, height) > 2048 {
		scale := 2048.0 / float64(max(width, height))
		width, height = max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale))
	}
	var encoded bytes.Buffer
	for {
		if picture.Bounds().Dx() != width || picture.Bounds().Dy() != height {
			resized := image.NewNRGBA(image.Rect(0, 0, width, height))
			draw.CatmullRom.Scale(resized, resized.Bounds(), picture, picture.Bounds(), draw.Src, nil)
			picture = resized
		}
		encoded.Reset()
		if err := png.Encode(&encoded, picture); err != nil {
			return Image{}, err
		}
		if encoded.Len() <= MaxBytes {
			break
		}
		width, height = max(1, width*4/5), max(1, height*4/5)
	}
	return s.SavePNG(encoded.Bytes(), name)
}
func (s Store) SavePNG(data []byte, name string) (Image, error) {
	if len(data) > MaxBytes {
		return Image{}, errors.New("Image attachment is too large")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 2048 || config.Height > 2048 {
		return Image{}, errors.New("Invalid normalized image")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return Image{}, errors.New("Invalid normalized image")
	}
	digest := sha256.Sum256(data)
	id := hex.EncodeToString(digest[:])
	path := filepath.Join(s.Directory, "attachments", id+".png")
	if _, err := s.Read(id); err != nil {
		if err := atomicfile.Replace(path, data, false); err != nil {
			return Image{}, err
		}
	}
	if len([]rune(name)) > 80 {
		name = string([]rune(name)[:80])
	}
	return Image{ID: id, Name: name}, nil
}
func (s Store) Import(file, name string) (Image, error) {
	handoff := filepath.Join(s.Directory, "attachment-imports")
	if filepath.Dir(file) != handoff || filepath.Base(file) != fileName(filepath.Base(file)) {
		return Image{}, errors.New("Invalid image handoff")
	}
	root, err := os.OpenRoot(handoff)
	if err != nil {
		return Image{}, err
	}
	defer root.Close()
	input, err := root.Open(filepath.Base(file))
	if err != nil {
		return Image{}, err
	}
	defer input.Close()
	stat, err := input.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > MaxBytes {
		return Image{}, errors.New("Invalid image handoff")
	}
	data, err := io.ReadAll(io.LimitReader(input, MaxBytes+1))
	if err != nil {
		return Image{}, err
	}
	return s.SavePNG(data, name)
}
func fileName(name string) string {
	if len(name) != 36 || name[32:] != ".png" {
		return ""
	}
	decoded, err := hex.DecodeString(name[:32])
	if err != nil || len(decoded) != 16 || hex.EncodeToString(decoded) != name[:32] {
		return ""
	}
	return name
}
