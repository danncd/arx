package generation

import (
	"errors"
	"os"
	"path/filepath"
)

func (l *Library) reconcile() error {
	seen := map[string]bool{}
	for i := range l.state.Models {
		entry := &l.state.Models[i]
		var trusted *Model
		for j := range l.catalog {
			if l.catalog[j].ID == entry.Model.ID {
				trusted = &l.catalog[j]
				break
			}
		}
		if trusted == nil || seen[entry.Model.ID] {
			return errors.New("Generation library contains an unknown model")
		}
		seen[entry.Model.ID] = true
		entry.Model = *trusted
		if entry.Status == "downloading" {
			entry.Status = "paused"
		}
		if entry.Status == "installed" && !l.complete(*trusted) {
			entry.Status, entry.Error = "failed", "Model files are missing. Download again to repair them"
		}
	}
	return nil
}

func (l *Library) complete(model Model) bool {
	for _, file := range model.Files {
		info, err := os.Lstat(filepath.Join(l.directory, model.ID, file.Name))
		if err != nil || !info.Mode().IsRegular() || info.Size() != file.Size {
			return false
		}
	}
	return true
}
