//go:build unix

package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	bashTimeout    = 30 * time.Second
	bashMaxTimeout = 120 * time.Second
)

var bashDrainTimeout = 5 * time.Second

var Bash = Tool{
	Name: "bash",
	Description: "Run a bash command in the working directory. " +
		"Returns stdout and stderr together; non-zero exits report the code.",
	Schema: json.RawMessage(`{"type":"object","properties":{` +
		`"command":{"type":"string","description":"Command for bash -c"},` +
		`"timeout_secs":{"type":"integer","description":"Deadline in seconds, default 30, max 120"}},` +
		`"required":["command"]}`),
	Mutating: true,
	Run: func(ctx context.Context, args json.RawMessage) (string, error) {
		var a struct {
			Command     string `json:"command"`
			TimeoutSecs int    `json:"timeout_secs"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("bad arguments: %w", err)
		}
		if strings.TrimSpace(a.Command) == "" {
			return "", fmt.Errorf("command is required")
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		timeout := bashTimeout
		if a.TimeoutSecs > 0 {
			secs := min(a.TimeoutSecs, int(bashMaxTimeout/time.Second))
			timeout = time.Duration(secs) * time.Second
		}
		parent := ctx
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		buf := &capBuf{limit: maxToolBytes + 1}
		cmd := exec.CommandContext(ctx, "bash", "-c", a.Command)
		cmd.Stdout, cmd.Stderr = buf, buf
		cmd.Env = scrubbedEnv()

		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		cmd.WaitDelay = bashDrainTimeout

		runErr := cmd.Run()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		if errors.Is(runErr, exec.ErrWaitDelay) {
			runErr = nil
		}

		out, truncated := clipUTF8(buf.b, maxToolBytes)
		if !utf8.ValidString(out) {
			return "", fmt.Errorf("output is not UTF-8 text")
		}
		if truncated {
			out += "\n[truncated at 64KB]"
		}
		if err := parent.Err(); err != nil {
			return "", err
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("timed out after %s: %w\n%s", timeout, context.DeadlineExceeded, out)
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		var exit *exec.ExitError
		if errors.As(runErr, &exit) {
			return "", fmt.Errorf("exit %d\n%s", exit.ExitCode(), out)
		}
		if runErr != nil {
			return "", runErr
		}
		return out, nil
	},
}

type capBuf struct {
	b     []byte
	limit int
}

func (w *capBuf) Write(p []byte) (int, error) {
	if room := w.limit - len(w.b); room > 0 {
		w.b = append(w.b, p[:min(room, len(p))]...)
	}
	return len(p), nil
}

func scrubbedEnv() []string {
	allowed := map[string]bool{
		"HOME": true, "LANG": true, "LC_ADDRESS": true, "LC_ALL": true,
		"LC_COLLATE": true, "LC_CTYPE": true, "LC_IDENTIFICATION": true,
		"LC_MEASUREMENT": true, "LC_MESSAGES": true, "LC_MONETARY": true,
		"LC_NAME": true, "LC_NUMERIC": true, "LC_PAPER": true,
		"LC_TELEPHONE": true, "LC_TIME": true, "PATH": true, "SHELL": true,
		"TEMP": true, "TERM": true, "TMP": true, "TMPDIR": true,
	}
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		name = strings.ToUpper(name)
		if allowed[name] {
			env = append(env, kv)
		}
	}
	return env
}
