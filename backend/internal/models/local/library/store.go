package library

import (
	"arx/internal/platform/atomicfile"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func Read(directory string) ([]Entry, error) {
	data, err := os.ReadFile(filepath.Join(directory, "library.json"))
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, errors.New("Could not read the local model library")
	}
	seen := map[string]bool{}
	for i := range entries {
		if err := ValidateEntry(directory, entries[i]); err != nil {
			return nil, err
		}
		if seen[entries[i].ID] {
			return nil, errors.New("Duplicate saved model ID")
		}
		seen[entries[i].ID] = true
		switch entries[i].Status {
		case "downloading", "queued", "verifying":
			entries[i].Status = "paused"
		case "loaded", "loading":
			entries[i].Status = "installed"
		}
	}
	return entries, nil
}

func Save(directory string, entries []Entry) error {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "    ")
	if err != nil {
		return err
	}
	return atomicfile.Replace(filepath.Join(directory, "library.json"), data, false)
}
