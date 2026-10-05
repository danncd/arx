package engines

import (
	"archive/zip"
	download "arx/internal/platform/download"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var wanRuntime = download.File{
	Name:   "wan-runtime.zip",
	URL:    "https://github.com/leejet/stable-diffusion.cpp/releases/download/master-929-3f8527a/sd-master-3f8527a-bin-Darwin-macOS-26.6.2-arm64.zip",
	Size:   35049086,
	SHA256: "1c8ee6c8e413e3335b1223bbc657ea5d86dae1819f8426a98b84c265587eb192",
}

var wanRuntimeFiles = map[string]os.FileMode{
	"sd-cli":                    0700,
	"libstable-diffusion.dylib": 0600,
	"stable-diffusion.cpp.txt":  0600,
	"ggml.txt":                  0600,
}

func wanRuntimeReady(target string) bool {
	for name := range wanRuntimeFiles {
		info, err := os.Stat(filepath.Join(target, "native", name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}

func installWan(ctx context.Context, target string, progress func(float64, string)) error {
	archive := filepath.Join(target, wanRuntime.Name)
	if err := download.Transfer(ctx, wanRuntime, archive, func(n int64) {
		progress(float64(n)/float64(wanRuntime.Size), "Installing Wan runtime")
	}); err != nil {
		return err
	}
	if err := unpackWan(archive, target); err != nil {
		return err
	}
	return os.Remove(archive)
}

func unpackWan(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	stage, err := os.MkdirTemp(target, ".native-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	found := map[string]bool{}
	for _, entry := range reader.File {
		mode, wanted := wanRuntimeFiles[entry.Name]
		if !wanted {
			continue
		}
		if found[entry.Name] || !entry.Mode().IsRegular() || entry.UncompressedSize64 > 128<<20 {
			return errors.New("Invalid Wan runtime archive")
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(filepath.Join(stage, entry.Name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, io.LimitReader(input, 128<<20))
		err = errors.Join(copyErr, output.Close(), input.Close())
		if err != nil {
			return err
		}
		found[entry.Name] = true
	}
	if len(found) != len(wanRuntimeFiles) {
		return errors.New("Wan runtime archive is incomplete")
	}
	if err := os.RemoveAll(filepath.Join(target, "native")); err != nil {
		return err
	}
	return os.Rename(stage, filepath.Join(target, "native"))
}
