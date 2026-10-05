package engine

import (
	provider "arx/internal/inference"
	localprovider "arx/internal/inference/chatcompletions"
	attachment "arx/internal/media/attachments"
	model "arx/internal/models"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLiveVision(t *testing.T) {
	binary := os.Getenv("ARX_TEST_ENGINE")
	if binary == "" {
		t.Skip("No local vision test runtime")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	server := &Server{}
	defer server.Stop()
	info, err := server.Start(ctx, binary, os.Getenv("ARX_TEST_MODEL"), os.Getenv("ARX_TEST_PROJECTOR"), model.Info{ID: "vision-test", ContextWindow: 8192}, filepath.Join(t.TempDir(), "runtime.log"))
	if err != nil {
		t.Fatal(err)
	}
	props, _ := server.properties(ctx)
	t.Logf("Runtime vision=%v props=%+v", info.Vision, props.Modalities)
	if !info.Vision {
		t.Fatal("Runtime lost vision capability")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(48, 48, 208, 208), &image.Uniform{color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	var data bytes.Buffer
	png.Encode(&data, canvas)
	if destination := os.Getenv("ARX_TEST_IMAGE"); destination != "" {
		if err := os.WriteFile(destination, data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	address, key, _, _ := server.Connection()
	client := localprovider.Client{URL: address, Key: key, Info: info, ReadImage: func(string) ([]byte, error) { return data.Bytes(), nil }}
	response, err := client.Complete(ctx, provider.Request{Effort: "none", MaxOutputTokens: 128, Messages: []provider.Message{{Role: "user", Content: "What color and shape are shown? Answer briefly.", Images: []attachment.Image{{ID: "test"}}}}}, func(provider.Delta) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Log(response.Message.Content)
	if !strings.Contains(strings.ToLower(response.Message.Content), "red") {
		t.Fatal("Image content not identified")
	}
}
