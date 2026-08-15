package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxToolBytes = 64 * 1024

var ReadFile = Tool{
	Name:        "read_file",
	Description: "Read a UTF-8 text file under the current working directory.",
	Schema: json.RawMessage(`{"type":"object","properties":{` +
		`"path":{"type":"string","description":"File path, relative to the working directory"}},` +
		`"required":["path"]}`),
	Run: func(ctx context.Context, args json.RawMessage) (string, error) {
		var a struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("bad arguments: %w", err)
		}
		if a.Path == "" {
			return "", fmt.Errorf("path is required")
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if filepath.IsAbs(a.Path) || filepath.VolumeName(a.Path) != "" {
			return "", fmt.Errorf("path must be relative to the working directory")
		}
		clean := filepath.Clean(a.Path)
		for _, part := range strings.Split(clean, string(filepath.Separator)) {
			part = strings.ToLower(part)
			if part == ".git" || part == ".env" || strings.HasPrefix(part, ".env.") {
				return "", fmt.Errorf("path %q is protected", a.Path)
			}
		}

		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root, err := os.OpenRoot(cwd)
		if err != nil {
			return "", err
		}
		defer root.Close()
		parts := strings.Split(clean, string(filepath.Separator))
		current := ""
		for i, part := range parts {
			if part == "." {
				continue
			}
			current = filepath.Join(current, part)
			info, err := root.Lstat(current)
			if err != nil {
				return "", err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("path %q contains a symbolic link", a.Path)
			}
			if i < len(parts)-1 && !info.IsDir() {
				return "", fmt.Errorf("%s is not a directory", current)
			}
			if i == len(parts)-1 && !info.Mode().IsRegular() {
				return "", fmt.Errorf("%s is not a regular file", a.Path)
			}
		}

		f, err := root.Open(clean)
		if err != nil {
			return "", err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s is not a regular file", a.Path)
		}
		protected, err := sameAsProtected(root, info)
		if err != nil {
			return "", err
		}
		if protected {
			return "", fmt.Errorf("path %q aliases a protected file", a.Path)
		}
		data, err := io.ReadAll(io.LimitReader(f, maxToolBytes+1))
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		out, truncated := clipUTF8(data, maxToolBytes)
		if !utf8.ValidString(out) {
			return "", fmt.Errorf("%s is not UTF-8 text", a.Path)
		}
		if truncated {
			out += "\n[truncated at 64KB]"
		}
		return out, nil
	},
}

func sameAsProtected(root *os.Root, target fs.FileInfo) (bool, error) {
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if name != ".env" && !strings.HasPrefix(name, ".env.") {
			continue
		}
		info, err := root.Stat(entry.Name())
		if err != nil {
			return false, err
		}
		if os.SameFile(target, info) {
			return true, nil
		}
	}
	return false, nil
}
