package files

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func destination(path string) (string, string, error) {
	candidate := path
	suffix := []string{}
	for {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err == nil {
			info, err := os.Stat(resolved)
			if err != nil {
				return "", "", err
			}
			if len(suffix) == 0 && !info.IsDir() {
				return resolved, filepath.Dir(resolved), nil
			}
			if !info.IsDir() {
				return "", "", errors.New("Parent path is not a directory")
			}
			full := resolved
			for index := len(suffix) - 1; index >= 0; index-- {
				full = filepath.Join(full, suffix[index])
			}
			return full, resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", "", err
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", "", err
		}
		if info, lstatErr := os.Lstat(candidate); lstatErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", "", errors.New("Path contains a broken symbolic link")
		}
		suffix = append(suffix, filepath.Base(candidate))
		candidate = parent
	}
}

func prepareChange(path string, policy permission.Policy, args arguments) (tool.Prepared, error) {
	target, scope, err := destination(path)
	if err != nil {
		return tool.Prepared{}, err
	}
	if policy.Mode == permission.Folders {
		scope, err = policy.Root(target)
		if err != nil {
			return tool.Prepared{}, err
		}
	}
	root, err := os.OpenRoot(scope)
	if err != nil {
		return tool.Prepared{}, err
	}
	name, err := filepath.Rel(scope, target)
	if err != nil {
		root.Close()
		return tool.Prepared{}, err
	}
	action := permission.Action{Tool: "files", Operation: args.Operation, Path: target, Writes: true}
	fail := func(err error) (tool.Prepared, error) {
		root.Close()
		return tool.Prepared{}, err
	}
	if args.Operation == "mkdir" {
		return tool.Prepared{Action: action, Close: func() { root.Close() }, Run: func(ctx context.Context) (tool.Result, error) {
			if err := ctx.Err(); err != nil {
				return tool.Result{}, err
			}
			if err := root.MkdirAll(name, 0755); err != nil {
				return tool.Result{}, err
			}
			return tool.Result{Text: "Created directory: " + target}, nil
		}}, nil
	}
	before, mode, exists, err := existing(root, name)
	if err != nil {
		return fail(err)
	}
	var after string
	if args.Operation == "write" {
		if args.Content == nil {
			return fail(errors.New("Write requires content"))
		}
		after = *args.Content
	} else {
		if !exists {
			return fail(errors.New("Cannot edit a missing file"))
		}
		if args.OldText == nil || *args.OldText == "" || args.NewText == nil {
			return fail(errors.New("Edit requires old_text and new_text"))
		}
		count := strings.Count(string(before), *args.OldText)
		if count == 0 {
			return fail(errors.New("The old text was not found"))
		}
		if count > 1 && !args.ReplaceAll {
			return fail(errors.New("The old text matches more than once; provide more context or use replace_all"))
		}
		after = strings.ReplaceAll(string(before), *args.OldText, *args.NewText)
	}
	if len(after) > 256<<10 {
		return fail(errors.New("File changes are limited to 256 KiB"))
	}
	action.Before, action.After = string(before), after
	return tool.Prepared{Action: action, Close: func() { root.Close() }, Run: func(ctx context.Context) (tool.Result, error) {
		if err := ctx.Err(); err != nil {
			return tool.Result{}, err
		}
		current, _, present, err := existing(root, name)
		if err != nil {
			return tool.Result{}, err
		}
		if present != exists || !bytes.Equal(current, before) {
			return tool.Result{}, errors.New("File changed while waiting; read it again before editing")
		}
		if err := publish(root, name, []byte(after), mode); err != nil {
			return tool.Result{}, err
		}
		return tool.Result{Text: "Saved: " + target}, nil
	}}, nil
}

func existing(root *os.Root, name string) ([]byte, os.FileMode, bool, error) {
	info, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return nil, 0644, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, false, errors.New("Target is not a regular file")
	}
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, 0, false, err
	}
	defer file.Close()
	body, err := readText(file)
	return body, info.Mode().Perm(), true, err
}

func publish(root *os.Root, name string, body []byte, mode os.FileMode) error {
	parent := filepath.Dir(name)
	if err := root.MkdirAll(parent, 0755); err != nil {
		return err
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	temporary := filepath.Join(parent, ".arx-"+hex.EncodeToString(random))
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	defer file.Close()
	if _, err := file.Write(body); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return root.Rename(temporary, name)
}
