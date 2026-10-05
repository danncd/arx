package bash

import (
	permission "arx/internal/permissions"
	environment "arx/internal/platform/paths"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type arguments struct {
	Command   string `json:"command"`
	Directory string `json:"directory"`
	Timeout   int    `json:"timeout_seconds"`
}

func Prepare(directory string, policy permission.Policy, raw json.RawMessage) (tool.Prepared, error) {
	var args arguments
	if err := tool.Decode(raw, &args); err != nil {
		return tool.Prepared{}, err
	}
	if strings.TrimSpace(args.Command) == "" || len(args.Command) > 64<<10 {
		return tool.Prepared{}, errors.New("A command is required (up to 64 KiB)")
	}
	if args.Timeout == 0 {
		args.Timeout = 30
	}
	if args.Timeout < 1 || args.Timeout > 120 {
		return tool.Prepared{}, errors.New("Command timeout must be between 1 and 120 seconds")
	}
	if args.Directory != "" {
		resolved, err := environment.Resolve(directory, args.Directory)
		if err != nil {
			return tool.Prepared{}, err
		}
		directory = resolved
	}
	directory, err := environment.Directory(directory)
	if err != nil {
		return tool.Prepared{}, err
	}
	return tool.Prepared{Action: permission.Action{Tool: "bash", Operation: "run", Command: args.Command, Directory: directory, Writes: true}, Run: func(ctx context.Context) (tool.Result, error) { return run(ctx, directory, policy, args) }}, nil
}

func run(ctx context.Context, directory string, policy permission.Policy, args arguments) (tool.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(args.Timeout)*time.Second)
	defer cancel()
	temporary, err := os.MkdirTemp("", "arx-command-")
	if err != nil {
		return tool.Result{}, err
	}
	defer os.RemoveAll(temporary)
	temporary, err = filepath.EvalSymlinks(temporary)
	if err != nil {
		return tool.Result{}, err
	}
	executable, argv, err := invocation(policy, temporary, args.Command)
	if err != nil {
		return tool.Result{}, err
	}
	command := exec.CommandContext(ctx, executable, argv...)
	command.Dir = directory
	home, _ := os.UserHomeDir()
	command.Env = []string{"PATH=" + os.Getenv("PATH") + ":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin", "HOME=" + home, "TMPDIR=" + temporary, "LANG=en_US.UTF-8", "TERM=dumb"}
	configureProcess(command)
	output := &boundedOutput{}
	command.Stdout, command.Stderr = output, output
	err = command.Run()
	stopProcess(command)
	result := output.Result()
	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return result, err
		}
		code := exit.ExitCode()
		result.ExitCode = &code
		result.Failed = true
		return result, nil
	}
	code := 0
	result.ExitCode = &code
	return result, nil
}
