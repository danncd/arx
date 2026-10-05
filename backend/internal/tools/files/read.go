package files

import (
	tool "arx/internal/tools"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxFile = 2 << 20

func readText(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("Only regular files can be read")
	}
	if info.Size() > maxFile {
		return nil, errors.New("File exceeds the 2 MiB text limit")
	}
	body, err := io.ReadAll(io.LimitReader(file, maxFile+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFile {
		return nil, errors.New("File exceeds the 2 MiB text limit")
	}
	if !utf8.Valid(body) || strings.ContainsRune(string(body), 0) {
		return nil, errors.New("File is not UTF-8 text")
	}
	return body, nil
}

func read(ctx context.Context, path string, offset, limit int) (tool.Result, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return tool.Result{}, err
	}
	defer file.Close()
	body, err := readText(file)
	if err != nil {
		return tool.Result{}, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	scanner.Buffer(make([]byte, 4096), maxFile+1)
	var output strings.Builder
	line, count := 0, 0
	truncated := false
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return tool.Result{}, err
		}
		line++
		if line < offset {
			continue
		}
		if count >= limit || output.Len() >= tool.MaxOutput {
			truncated = true
			break
		}
		fmt.Fprintf(&output, "%d: %s\n", line, scanner.Text())
		count++
	}
	if err := scanner.Err(); err != nil {
		return tool.Result{}, err
	}
	result := tool.Output(output.String())
	result.Truncated = result.Truncated || truncated
	return result, nil
}

func list(ctx context.Context, path string, offset, limit int) (tool.Result, error) {
	directory, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return tool.Result{}, err
	}
	defer directory.Close()
	var output strings.Builder
	seen, count := 0, 0
	for {
		if err := ctx.Err(); err != nil {
			return tool.Result{}, err
		}
		entries, err := directory.ReadDir(128)
		for _, entry := range entries {
			seen++
			if seen < offset {
				continue
			}
			if count >= limit || output.Len() >= tool.MaxOutput {
				result := tool.Output(output.String())
				result.Truncated = true
				return result, nil
			}
			suffix := ""
			if entry.IsDir() {
				suffix = "/"
			} else if entry.Type()&os.ModeSymlink != 0 {
				suffix = "@"
			}
			fmt.Fprintf(&output, "%s%s\n", entry.Name(), suffix)
			count++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return tool.Result{}, err
		}
	}
	return tool.Output(output.String()), nil
}
