//go:build aix || android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package tool

import (
	"context"
	"os/exec"
	"syscall"
)

func runCommand(ctx context.Context, cmd *exec.Cmd) error {
	// Kill the pipeline with its shell.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	kill := func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	select {
	case err := <-done:
		kill()
		return err
	case <-ctx.Done():
		kill()
		return <-done
	}
}
