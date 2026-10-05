package deepseek

import (
	provider "arx/internal/inference"
	"bytes"
	"encoding/base64"
	"errors"
	"golang.org/x/image/draw"
	"image"
	"image/png"
)

const imagePayloadBytes = 24 << 20

func (c *Client) imagePayloads(input provider.Request) (map[string]string, error) {
	count := 0
	for _, message := range input.Messages {
		count += len(message.Images)
	}
	if count == 0 {
		return nil, nil
	}
	if !SupportsVision(input.Model) {
		return nil, errors.New("Choose DeepSeek Flash to use images")
	}
	if c.ReadImage == nil {
		return nil, errors.New("Image input is unavailable")
	}
	limit := (imagePayloadBytes/count - 64) / 4 * 3
	if limit < 1024 {
		return nil, provider.ErrContextLength
	}
	data := map[string][]byte{}
	total := 0
	for _, message := range input.Messages {
		for _, ref := range message.Images {
			if _, exists := data[ref.ID]; !exists {
				picture, err := c.ReadImage(ref.ID)
				if err != nil {
					return nil, err
				}
				data[ref.ID] = picture
			}
			total += base64.StdEncoding.EncodedLen(len(data[ref.ID])) + 64
		}
	}
	urls := make(map[string]string, len(data))
	for id, picture := range data {
		if total > imagePayloadBytes {
			var err error
			picture, err = imagePreview(picture, limit)
			if err != nil {
				return nil, err
			}
		}
		urls[id] = "data:image/png;base64," + base64.StdEncoding.EncodeToString(picture)
	}
	return urls, nil
}

func imagePreview(data []byte, limit int) ([]byte, error) {
	if len(data) <= limit {
		return data, nil
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("Saved image could not be decoded")
	}
	for len(data) > limit {
		width, height := picture.Bounds().Dx(), picture.Bounds().Dy()
		if width == 1 && height == 1 {
			return nil, provider.ErrContextLength
		}
		resized := image.NewNRGBA(image.Rect(0, 0, max(1, width*4/5), max(1, height*4/5)))
		draw.CatmullRom.Scale(resized, resized.Bounds(), picture, picture.Bounds(), draw.Src, nil)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, resized); err != nil {
			return nil, err
		}
		data, picture = encoded.Bytes(), resized
	}
	return data, nil
}
