package engines

import (
	download "arx/internal/platform/download"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var fastMetalSource = download.File{
	Name:   "fastmetal-source.tar.gz",
	URL:    "https://codeload.github.com/hao-ai-lab/FastVideo/tar.gz/dd35763ad6ba8db441fdccbe05f66fba797b8931",
	Size:   166229866,
	SHA256: "d75037a2b6fa44b54230f0193badac6aa78c11612a61b0a8ba01c0c48113c071",
}

func installFastMetal(ctx context.Context, target, assets, python string, log io.Writer, progress func(float64, string)) (failure error) {
	archive := filepath.Join(target, fastMetalSource.Name)
	progress(0, "Installing FastMetal runtime")
	if err := download.Transfer(ctx, fastMetalSource, archive, func(n int64) {
		progress(float64(n)/float64(fastMetalSource.Size), "Installing FastMetal runtime")
	}); err != nil {
		return err
	}
	command, cleanup, err := supervisedCommand(ctx, target, python, filepath.Join(assets, "video", "fastmetal", "setup.py"), archive, filepath.Join(target, "source"))
	if err != nil {
		return err
	}
	defer func() {
		if err := cleanup(); err != nil {
			failure = errors.Join(failure, fmt.Errorf("Generation installer cleanup: %w", err))
		}
	}()
	command.Stdout, command.Stderr = log, log
	if err := command.Start(); err != nil {
		return err
	}
	if err := command.Wait(); err != nil {
		return fmt.Errorf("Could not install FastMetal runtime: %w", err)
	}
	os.Remove(archive)
	return nil
}
