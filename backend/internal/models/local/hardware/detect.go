package hardware

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Name      string `json:"name"`
	Memory    int64  `json:"memory"`
	Supported bool   `json:"supported"`
}
type Fit struct {
	Label  string `json:"label"`
	Reason string `json:"reason"`
	Tone   string `json:"tone"`
}

func Detect() Info {
	result := Info{Name: runtime.GOOS + " / " + runtime.GOARCH, Supported: runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"}
	if runtime.GOOS != "darwin" {
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if data, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		result.Name = strings.TrimSpace(string(data))
	}
	if data, err := exec.CommandContext(ctx, "/usr/sbin/sysctl", "-n", "hw.memsize").Output(); err == nil {
		result.Memory, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	return result
}
func (h Info) Fit(size int64) Fit {
	if !h.Supported || h.Memory == 0 {
		return Fit{"Compatibility unknown", "Local runtime currently supports Apple Silicon Macs.", "caution"}
	}
	if size <= 0 {
		return Fit{"Compatibility unknown", "Model size is unavailable.", "caution"}
	}
	if size+4<<30 > h.Memory {
		return Fit{"Not recommended", "Weights and runtime memory leave too little room for macOS.", "unsuitable"}
	}
	if size+6<<30 > h.Memory {
		return Fit{"Limited headroom", "Limited room for context and other apps.", "caution"}
	}
	return Fit{"Recommended", "Weights leave room for context and the runtime. Compatibility is checked when loaded.", ""}
}
