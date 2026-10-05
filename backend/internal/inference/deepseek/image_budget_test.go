package deepseek

import (
	provider "arx/internal/inference"
	attachment "arx/internal/media/attachments"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLargeImageBatchFitsWithoutDiscardingImages(t *testing.T) {
	picture := image.NewNRGBA(image.Rect(0, 0, 1024, 512))
	random := rand.New(rand.NewPCG(1, 2))
	for index := 0; index < len(picture.Pix); index += 4 {
		value := random.Uint32()
		picture.Pix[index], picture.Pix[index+1], picture.Pix[index+2], picture.Pix[index+3] = byte(value), byte(value>>8), byte(value>>16), 255
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, picture); err != nil {
		t.Fatal(err)
	}
	original := sha256.Sum256(encoded.Bytes())
	if base64.StdEncoding.EncodedLen(encoded.Len())*32 < 48<<20 {
		t.Fatal("fixture did not exceed the old request limit")
	}
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if len(body) > imagePayloadBytes+16384 {
			t.Error("image request exceeded budget")
		}
		var request struct {
			Messages []struct {
				Content []struct {
					Type  string
					Image struct{ URL string } `json:"image_url"`
				}
			}
		}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Error(err)
			return
		}
		count := 0
		for _, part := range request.Messages[0].Content {
			if part.Type != "image_url" {
				continue
			}
			count++
			if count != 1 {
				continue
			}
			data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(part.Image.URL, "data:image/png;base64,"))
			if err != nil {
				t.Error(err)
				continue
			}
			config, err := png.DecodeConfig(bytes.NewReader(data))
			if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width >= 1024 || abs(config.Width-2*config.Height) > 1 {
				t.Error("preview resizing lost its aspect ratio", config, err)
			}
		}
		if count != 32 {
			t.Error("images were discarded", count)
		}
		received = true
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Seen\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	reads := 0
	client := &Client{client: server.Client(), baseURL: server.URL, ReadImage: func(string) ([]byte, error) { reads++; return encoded.Bytes(), nil }}
	message := provider.Message{Role: "user", Content: "Compare these images"}
	for range 32 {
		message.Images = append(message.Images, attachment.Image{ID: "saved", Name: "diagram.png"})
	}
	if _, err := client.Complete(context.Background(), "test", provider.Request{Model: "deepseek-flash", Messages: []provider.Message{message}}, func(provider.Delta) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if !received || reads != 1 {
		t.Fatal("request failed or image was repeatedly loaded", reads)
	}
	if sha256.Sum256(encoded.Bytes()) != original || len(message.Images) != 32 {
		t.Fatal("saved image or history mutated")
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
