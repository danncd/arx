package config

import (
	"os"
	"strings"
)

// LoadDotEnv reads KEY=VALUE lines from path and exports each into the
// process environment, unless the variable is already set — a real
// exported variable always beats the file. A missing file is not an
// error: running without one is normal (e.g. keys arriving via systemd).
func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" && os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
	return nil
}
