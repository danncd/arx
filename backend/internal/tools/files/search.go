package files

import (
	tool "arx/internal/tools"
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

func search(ctx context.Context, path, pattern string, limit int) (tool.Result, error) {
	if pattern == "" || len(pattern) > 4096 {
		return tool.Result{}, errors.New("A search pattern is required (up to 4096 characters)")
	}
	expression, err := regexp.Compile(pattern)
	if err != nil {
		return tool.Result{}, errors.New("Invalid search expression")
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return tool.Result{}, errors.New("Search requires a directory")
	}
	var output strings.Builder
	visited, matches := 0, 0
	stopped := errors.New("search limit reached")
	err = filepath.WalkDir(path, func(name string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		visited++
		if visited > 10000 {
			return stopped
		}
		if entry.IsDir() {
			if name != path && (entry.Name() == ".git" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		file, err := os.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return err
		}
		body, err := readText(file)
		file.Close()
		if err != nil {
			return nil
		}
		scanner := bufio.NewScanner(strings.NewReader(string(body)))
		scanner.Buffer(make([]byte, 4096), maxFile+1)
		line := 0
		for scanner.Scan() {
			if err := ctx.Err(); err != nil {
				return err
			}
			line++
			if !expression.MatchString(scanner.Text()) {
				continue
			}
			relative, _ := filepath.Rel(path, name)
			fmt.Fprintf(&output, "%s:%d: %s\n", relative, line, scanner.Text())
			matches++
			if matches >= limit || output.Len() >= tool.MaxOutput {
				return stopped
			}
		}
		return scanner.Err()
	})
	if err != nil && err != stopped {
		return tool.Result{}, err
	}
	result := tool.Output(output.String())
	result.Truncated = result.Truncated || err == stopped
	return result, nil
}
