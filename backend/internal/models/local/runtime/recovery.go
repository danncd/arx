package engine

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type processRecord struct {
	PID     int    `json:"pid"`
	Binary  string `json:"binary"`
	KeyFile string `json:"keyFile"`
}

func Recover(directory string) {
	path := filepath.Join(directory, "runtime-process.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var record processRecord
	if json.Unmarshal(data, &record) != nil || record.PID <= 1 {
		return
	}
	relative, err := filepath.Rel(filepath.Join(directory, "engine"), record.Binary)
	if err != nil || relative == ".." || strings.HasPrefix(relative, "../") || filepath.Dir(record.KeyFile) != directory || !strings.HasPrefix(filepath.Base(record.KeyFile), ".runtime-key-") {
		return
	}
	command, err := exec.Command("/bin/ps", "-p", strconv.Itoa(record.PID), "-o", "command=").Output()
	if err == nil && strings.Contains(string(command), record.Binary) && strings.Contains(string(command), record.KeyFile) {
		syscall.Kill(-record.PID, syscall.SIGTERM)
		for attempt := 0; attempt < 20; attempt++ {
			if syscall.Kill(record.PID, 0) != nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		command, err = exec.Command("/bin/ps", "-p", strconv.Itoa(record.PID), "-o", "command=").Output()
		if err == nil && strings.Contains(string(command), record.KeyFile) {
			syscall.Kill(-record.PID, syscall.SIGKILL)
		}
	}
	os.Remove(record.KeyFile)
	os.Remove(path)
}
