package engines

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const supervisorFlag = "--arx-generation-supervisor"

type workerRecord struct {
	PID        int    `json:"pid"`
	Owner      int    `json:"owner"`
	OwnerStart string `json:"ownerStart"`
	Start      string `json:"start"`
	Binary     string `json:"binary"`
	Target     string `json:"target"`
	Token      string `json:"token"`
}

func SupervisorMain() bool {
	if len(os.Args) < 2 || os.Args[1] != supervisorFlag {
		return false
	}
	if len(os.Args) < 5 {
		os.Exit(2)
	}
	alive := os.NewFile(3, "owner-liveness")
	barrier := os.NewFile(4, "start-barrier")
	go func() {
		io.Copy(io.Discard, alive)
		syscall.Kill(-os.Getpid(), syscall.SIGKILL)
		os.Exit(1)
	}()
	var ready [1]byte
	if _, err := io.ReadFull(barrier, ready[:]); err != nil || ready[0] != 1 {
		os.Exit(2)
	}
	barrier.Close()
	target := exec.Command(os.Args[3], os.Args[4:]...)
	target.Stdin, target.Stdout, target.Stderr = os.Stdin, os.Stdout, os.Stderr
	target.Env = os.Environ()
	if err := target.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
	return true
}
func processStart(pid int) (string, error) {
	data, err := exec.Command("/bin/ps", "-p", strconv.Itoa(pid), "-o", "lstart=").Output()
	return strings.TrimSpace(string(data)), err
}
func supervisedCommand(ctx context.Context, directory, target string, args ...string) (*generationProcess, func() error, error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, nil, err
	}
	token := hex.EncodeToString(nonce[:])
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	barrier, release, err := os.Pipe()
	if err != nil {
		reader.Close()
		writer.Close()
		return nil, nil, err
	}
	arguments := append([]string{supervisorFlag, token, target}, args...)
	command := exec.CommandContext(ctx, binary, arguments...)
	command.ExtraFiles = []*os.File{reader, barrier}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
	command.WaitDelay = 2 * time.Second
	path := filepath.Join(directory, ".worker-"+token+".json")
	cleanup := func() error {
		reader.Close()
		barrier.Close()
		release.Close()
		writer.Close()
		if command.Process != nil {
			syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
			if err := syscall.Kill(-command.Process.Pid, 0); err == nil {
				return errors.New("Generation process group is still stopping; recovery record retained")
			}
		}
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	establish := func() error {
		reader.Close()
		barrier.Close()
		ownerStart, err := processStart(os.Getpid())
		if err != nil {
			return err
		}
		start, err := processStart(command.Process.Pid)
		if err != nil {
			return err
		}
		record := workerRecord{command.Process.Pid, os.Getpid(), ownerStart, start, binary, target, token}
		if err := saveWorkerRecord(path, record); err != nil {
			return err
		}
		_, err = release.Write([]byte{1})
		release.Close()
		return err
	}

	return &generationProcess{Cmd: command, establish: establish}, cleanup, nil
}
