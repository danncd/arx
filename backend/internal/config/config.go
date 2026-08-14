package config

import (
	"os"
	"strings"
)

// LoadDotEnv reads KEY=VALUE lines from path and exports each into the
// process environment, unless the variable already exists — a real
// exported variable always beats the file, INCLUDING one exported
// empty (blanking a key on purpose must stay blank). A missing file is
// not an error: running without one is normal (e.g. keys arriving via
// systemd). Shell-sourceable "export KEY=v" lines are accepted.
func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := strings.TrimPrefix(string(data), "\ufeff") // editors love BOMs
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		// Strip exactly one BALANCED pair of quotes. A cutset Trim here
		// ate unmatched trailing quote characters out of real secrets.
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		if k == "" {
			continue
		}
		if _, exists := os.LookupEnv(k); !exists {
			os.Setenv(k, v)
		}
	}
	return nil
}
