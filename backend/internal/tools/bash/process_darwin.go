package bash

import (
	permission "arx/internal/permissions"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func invocation(policy permission.Policy, temporary, command string) (string, []string, error) {
	args := []string{"--noprofile", "--norc", "-c", command}
	if policy.Mode != permission.Folders {
		return "/bin/bash", args, nil
	}
	if _, err := os.Stat("/usr/bin/sandbox-exec"); err != nil {
		return "", nil, errors.New("Folder-restricted commands are unavailable on this system")
	}
	for _, root := range policy.Roots {
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil || resolved != root {
			return "", nil, errors.New("A selected folder changed or is unavailable; select it again")
		}
	}
	var profile strings.Builder
	profile.WriteString("(version 1)(deny default)(allow file-read*)(allow process-exec)(allow process-fork)(allow sysctl-read)(allow signal (target self))(allow file-write* (literal \"/dev/null\"))")
	for _, root := range append(append([]string{}, policy.Roots...), temporary) {
		profile.WriteString("(allow file-write* (subpath " + strconv.Quote(root) + "))")
	}
	return "/usr/bin/sandbox-exec", append([]string{"-p", profile.String(), "/bin/bash"}, args...), nil
}

func configureProcess(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
	command.WaitDelay = 250 * time.Millisecond
}
func stopProcess(command *exec.Cmd) {
	if command.Process != nil {
		syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
}
