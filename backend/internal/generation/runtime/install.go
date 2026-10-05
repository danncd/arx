package engines

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
)

const Version = "generation-v1"

var uvArchive = download.File{
	Name:   "uv-aarch64-apple-darwin.tar.gz",
	URL:    "https://github.com/astral-sh/uv/releases/download/0.12.19/uv-aarch64-apple-darwin.tar.gz",
	Size:   16988553,
	SHA256: "a9a8df1eedeb192f2e47e40e2faabfb387db4b850209118786d42f89dde3e0ba",
}

func WorkerDirectory() (string, error) {
	binary, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(binary), "runtime", "generation"), nil
}

func Install(ctx context.Context, directory, assets, operation, modelRuntime string, progress func(float64, string)) (string, error) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return "", errors.New("Local generation currently requires Apple Silicon")
	}
	engine := "diffusion"
	if operation == "speech" {
		engine = "speech"
	}
	if modelRuntime == "flux2" || modelRuntime == "z-image" {
		engine = "mflux"
	}
	if modelRuntime == "fastmetal" {
		engine = "fastmetal"
	}
	if modelRuntime == "wan-cpp" {
		engine = "wan-cpp"
	}
	target := filepath.Join(directory, Version, engine)
	python := filepath.Join(target, "venv", "bin", "python")
	if _, err := os.Stat(filepath.Join(target, "ready")); err == nil {
		if _, err := os.Stat(python); err == nil && (engine != "wan-cpp" || wanRuntimeReady(target)) {
			return python, nil
		}
	}
	if err := os.MkdirAll(target, 0700); err != nil {
		return "", err
	}
	archive := filepath.Join(target, uvArchive.Name)
	uv := filepath.Join(target, "uv")
	if _, err := os.Stat(uv); err != nil {
		progress(0, "Downloading generation runtime")
		if err := download.Transfer(ctx, uvArchive, archive, func(n int64) {
			progress(float64(n)/float64(uvArchive.Size), "Downloading generation runtime")
		}); err != nil {
			return "", err
		}
		if err := unpackUV(archive, uv); err != nil {
			return "", err
		}
		os.Remove(archive)
	}
	log, err := os.OpenFile(filepath.Join(target, "install.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer log.Close()
	progress(0, "Installing isolated Python runtime")
	environment := append(os.Environ(), "UV_PYTHON_INSTALL_DIR="+filepath.Join(target, "python"), "UV_CACHE_DIR="+filepath.Join(target, "cache"))
	run := func(args ...string) (failure error) {
		command, cleanup, err := supervisedCommand(ctx, target, uv, args...)
		if err != nil {
			return err
		}
		defer func() {
			if err := cleanup(); err != nil {
				failure = errors.Join(failure, errors.New("Generation installer cleanup: "+err.Error()))
			}
		}()
		command.Env, command.Stdout, command.Stderr = environment, log, log
		if err := command.Start(); err != nil {
			return err
		}
		if err := command.Wait(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("Generation runtime installation failed. Check the install log")
		}
		return nil
	}
	if err := run("venv", "--python", "3.12.14", filepath.Join(target, "venv")); err != nil {
		return "", err
	}
	progress(0, "Installing generation libraries")
	if err := run("pip", "sync", "--python", python, filepath.Join(assets, "requirements", engine+".txt")); err != nil {
		return "", err
	}
	if err := run("pip", "check", "--python", python); err != nil {
		return "", err
	}
	if engine == "fastmetal" {
		if err := installFastMetal(ctx, target, assets, python, log, progress); err != nil {
			return "", err
		}
	}
	if engine == "wan-cpp" {
		if err := installWan(ctx, target, progress); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(filepath.Join(target, "ready"), []byte(Version), 0600); err != nil {
		return "", err
	}
	return python, nil
}

func unpackUV(archive, destination string) error {
	input, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer input.Close()
	compressed, err := gzip.NewReader(input)
	if err != nil {
		return err
	}
	defer compressed.Close()
	reader := tar.NewReader(compressed)
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			return errors.New("Runtime archive is missing its installer")
		}
		if err != nil {
			return err
		}
		if entry.Name != "uv-aarch64-apple-darwin/uv" {
			continue
		}
		if entry.Typeflag != tar.TypeReg || entry.Size > 100<<20 {
			return errors.New("Invalid installer archive")
		}
		file, err := os.OpenFile(destination+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		_, copyErr := io.Copy(file, reader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		return os.Rename(file.Name(), destination)
	}
}
