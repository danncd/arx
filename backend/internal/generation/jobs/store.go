package jobs

import (
	"arx/internal/platform/atomicfile"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var validID = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Store struct{ Directory string }

func (s Store) Save(job Job) error {
	if !validID.MatchString(job.ID) {
		return errors.New("Invalid generation job")
	}
	if err := os.MkdirAll(s.Directory, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return atomicfile.Replace(filepath.Join(s.Directory, job.ID+".json"), data, false)
}

func (s Store) Recover() ([]Job, error) {
	entries, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return []Job{}, nil
	}
	if err != nil {
		return nil, err
	}
	jobs := []Job{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var job Job
		if json.Unmarshal(data, &job) != nil || !validID.MatchString(job.ID) || entry.Name() != job.ID+".json" {
			return nil, errors.New("Saved generation job is invalid")
		}
		if job.State == "queued" || job.State == "running" {
			job.State, job.Error = "interrupted", "Generation was interrupted. Try again."
			job.Updated = time.Now().UTC()
			if err := s.Save(job); err != nil {
				return nil, err
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
