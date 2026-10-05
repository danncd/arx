package engine

import (
	"errors"
	"io"
	"os"
	"strings"
)

func loadFailure(path string) error {
	fallback := errors.New("The local engine stopped while loading. Check the runtime log")
	file, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return fallback
	}
	if _, err := file.Seek(max(0, stat.Size()-(16<<10)), io.SeekStart); err != nil {
		return fallback
	}
	data, err := io.ReadAll(io.LimitReader(file, 16<<10))
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(data), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "out of memory") || strings.Contains(lower, "failed to allocate") {
			return errors.New("Not enough memory to load this model. Choose a smaller model or quantization")
		}
		if _, detail, ok := strings.Cut(line, "error loading model:"); ok {
			detail = strings.TrimSpace(detail)
			if detail != "" {
				if len(detail) > 240 {
					detail = detail[:240] + "…"
				}
				return errors.New("Cannot load this model: " + detail)
			}
		}
	}
	return fallback
}
