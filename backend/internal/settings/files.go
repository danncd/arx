package settings

import (
	"arx/internal/platform/atomicfile"
	"encoding/json"
	"os"
)

func readJSON(path string, into any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, into)
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return err
	}
	return atomicfile.Replace(path, append(body, '\n'), true)
}
