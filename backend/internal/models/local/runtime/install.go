package engine

import (
	"archive/tar"
	download "arx/internal/platform/download"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const Version = "b11222"

var artifact = download.File{Name: "llama-b11222-bin-macos-arm64.tar.gz", URL: "https://github.com/ggml-org/llama.cpp/releases/download/b11222/llama-b11222-bin-macos-arm64.tar.gz", Size: 11756831, SHA256: "869b73f760042ac660e9453ff5e5cbb157725f2f8e01ec473006ffcf387a8410"}

func Install(ctx context.Context, directory string, progress func(int64, int64)) (string, error) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return "", errors.New("Local models currently require an Apple Silicon Mac")
	}
	target := filepath.Join(directory, Version)
	executable := findServer(target)
	if executable != "" {
		return executable, nil
	}
	archive := filepath.Join(directory, artifact.Name)
	if err := download.Transfer(ctx, artifact, archive, func(n int64) { progress(n, artifact.Size) }); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(directory, ".install-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	input, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer input.Close()
	gz, err := gzip.NewReader(input)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	total := int64(0)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		clean := filepath.Clean(header.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
			return "", errors.New("Invalid runtime archive path")
		}
		destination := filepath.Join(stage, clean)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destination, 0700); err != nil {
				return "", err
			}
		case tar.TypeReg:
			total += header.Size
			if total > 512<<20 {
				return "", errors.New("Runtime archive is too large")
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return "", err
			}
			output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
			if err != nil {
				return "", err
			}
			_, err = io.Copy(output, reader)
			closeErr := output.Close()
			if err != nil {
				return "", err
			}
			if closeErr != nil {
				return "", closeErr
			}
		case tar.TypeSymlink:
			link := filepath.Clean(filepath.Join(filepath.Dir(clean), header.Linkname))
			if filepath.IsAbs(header.Linkname) || link == ".." || strings.HasPrefix(link, "../") {
				return "", errors.New("Invalid runtime archive link")
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
				return "", err
			}
			if err := os.Symlink(header.Linkname, destination); err != nil {
				return "", err
			}
		default:
			return "", errors.New("Unsupported runtime archive entry")
		}
	}
	if findServer(stage) == "" {
		return "", errors.New("Runtime archive has no server")
	}
	if err := os.Rename(stage, target); err != nil {
		return "", err
	}
	os.Remove(archive)
	return findServer(target), nil
}

func findServer(directory string) string {
	found := ""
	filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && entry.Name() == "llama-server" {
			if info, e := entry.Info(); e == nil && info.Mode().IsRegular() && info.Mode()&0111 != 0 {
				found = path
			}
		}
		return nil
	})
	return found
}
