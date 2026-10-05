package files

import (
	attachment "arx/internal/media/attachments"
	tool "arx/internal/tools"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func readImage(ctx context.Context, path string) (tool.Result, error) {
	if err := ctx.Err(); err != nil {
		return tool.Result{}, err
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return tool.Result{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return tool.Result{}, err
	}
	if !info.Mode().IsRegular() {
		return tool.Result{}, errors.New("Only regular image files can be read")
	}
	if info.Size() > attachment.MaxSourceBytes {
		return tool.Result{}, errors.New("Image exceeds 8 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, attachment.MaxSourceBytes+1))
	if err != nil {
		return tool.Result{}, err
	}
	if len(data) > attachment.MaxSourceBytes {
		return tool.Result{}, errors.New("Image exceeds 8 MiB")
	}
	if len(data) == 0 {
		return tool.Result{}, errors.New("Image file is empty")
	}
	return tool.Result{Text: "Image loaded for visual inspection: " + path, ImageData: data, ImageName: filepath.Base(path)}, ctx.Err()
}

func reopenImage(ctx context.Context, images *attachment.Store, id string) (tool.Result, error) {
	if err := ctx.Err(); err != nil {
		return tool.Result{}, err
	}
	if images == nil {
		return tool.Result{}, errors.New("Image storage is unavailable")
	}
	if _, err := images.Read(id); err != nil {
		return tool.Result{}, err
	}
	return tool.Result{Text: "Reopened saved image: arx-image:" + id, Images: []attachment.Image{{ID: id}}}, ctx.Err()
}
