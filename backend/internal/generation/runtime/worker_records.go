package engines

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type generationProcess struct {
	*exec.Cmd
	establish func() error
}

func (command *generationProcess) Start() error {
	if err := command.Cmd.Start(); err != nil {
		return err
	}
	if err := command.establish(); err != nil {
		command.Cancel()
		command.Wait()
		return err
	}
	return nil
}
func saveWorkerRecord(path string, record workerRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".record-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	writeErr := json.NewEncoder(file).Encode(record)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func Recover(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(directory, func(path string, item os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() || !strings.HasPrefix(item.Name(), ".worker-") || !strings.HasSuffix(item.Name(), ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var record workerRecord
		if json.Unmarshal(data, &record) != nil || record.PID <= 1 || record.Owner <= 1 || len(record.Token) != 48 || item.Name() != ".worker-"+record.Token+".json" {
			return fmt.Errorf("Invalid generation recovery record: %s", path)
		}
		ownerStart, ownerErr := processStart(record.Owner)
		if ownerErr == nil && ownerStart == record.OwnerStart {
			return nil
		}
		if ownerErr != nil && syscall.Kill(record.Owner, 0) != syscall.ESRCH {
			return errors.New("Cannot confirm generation owner's exit")
		}
		start, startErr := processStart(record.PID)
		if startErr != nil {
			if syscall.Kill(-record.PID, 0) == syscall.ESRCH {
				return os.Remove(path)
			}
			return errors.New("Cannot confirm generation worker identity; record retained")
		}
		command, err := exec.Command("/bin/ps", "-p", strconv.Itoa(record.PID), "-o", "pgid=", "-o", "command=").Output()
		fields := strings.Fields(string(command))
		identity := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(command)), strconv.Itoa(record.PID)))
		expected := record.Binary + " " + supervisorFlag + " " + record.Token + " " + record.Target
		if err != nil || start != record.Start || len(fields) < 2 || fields[0] != strconv.Itoa(record.PID) || (identity != expected && !strings.HasPrefix(identity, expected+" ")) {
			return errors.New("Generation worker identity changed; record retained")
		}
		if err := syscall.Kill(-record.PID, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return err
		}

		if syscall.Kill(-record.PID, 0) == syscall.ESRCH {
			return os.Remove(path)
		}
		return nil
	})
}
