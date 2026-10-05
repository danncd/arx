package local

import (
	"arx/internal/models/local/library"
	download "arx/internal/platform/download"
	"context"
	"errors"
	"path/filepath"
	"syscall"
	"time"
)

func (m *Manager) Download(ctx context.Context, repository, variant string) (string, error) {
	details, err := m.catalog.Repository(ctx, repository)
	if err != nil {
		return "", err
	}
	if details.Gated {
		return "", errors.New("This repository requires access on Hugging Face")
	}
	for _, option := range details.Variants {
		if option.ID != variant {
			continue
		}
		entry := library.Entry{ID: modelID(repository + "@" + details.Revision + "/" + variant), Name: option.Name, Repository: repository, Revision: details.Revision, Quantization: option.Quantization, Files: option.Files, Size: option.Size, Status: "paused"}
		base, err := library.ManagedDirectory(m.directory, entry.ID)
		if err != nil {
			return "", err
		}
		entry.Path = filepath.Join(base, option.Files[0].Name)
		if option.Projector != "" {
			entry.Projector = filepath.Join(base, option.Projector)
		}
		m.mutex.Lock()
		if m.closed {
			m.mutex.Unlock()
			return "", errors.New("Backend is stopping")
		}
		if index := m.index(entry.ID); index >= 0 && (m.state.Models[index].Status == "installed" || m.state.Models[index].Status == "loaded" || m.state.Models[index].Status == "loading") {
			if m.state.Models[index].Projector == "" && entry.Projector != "" {
				if m.state.Models[index].Status != "installed" {
					m.mutex.Unlock()
					return "", errors.New("Unload the model before adding its vision files")
				}
				entry.Received = m.state.Models[index].Received
				m.state.Models[index] = entry
				m.publishLocked(true)
				m.mutex.Unlock()
				return entry.ID, m.Resume(entry.ID)
			}
			m.mutex.Unlock()
			return entry.ID, nil
		}
		if m.index(entry.ID) < 0 {
			m.state.Models = append(m.state.Models, entry)
			m.publishLocked(true)
		}
		m.mutex.Unlock()
		return entry.ID, m.Resume(entry.ID)
	}
	return "", errors.New("Model variant is unavailable")
}
func (m *Manager) Resume(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return errors.New("Backend is stopping")
	}
	index := m.index(id)
	if index < 0 {
		return errors.New("Download was not found")
	}
	entry := m.state.Models[index]
	if m.jobs[id] != nil {
		return nil
	}
	if entry.Imported || (entry.Status != "paused" && entry.Status != "failed") {
		return errors.New("This model has no paused download")
	}
	if len(m.jobs) > 0 {
		return errors.New("Pause the current download before starting another")
	}
	var disk syscall.Statfs_t
	if err := syscall.Statfs(m.directory, &disk); err == nil && uint64(max(0, entry.Size-entry.Received)+(512<<20)) > disk.Bavail*uint64(disk.Bsize) {
		return errors.New("Not enough disk space for this model")
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &downloadJob{cancel: cancel, done: make(chan struct{})}
	m.jobs[id] = job
	m.state.Models[index].Status = "downloading"
	m.state.Models[index].Error = ""
	m.publishLocked(true)
	m.done.Add(1)
	go func() { defer m.done.Done(); defer close(job.done); m.transfer(ctx, entry) }()
	return nil
}
func (m *Manager) Pause(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if job := m.jobs[id]; job != nil {
		job.cancel()
		return nil
	}
	return errors.New("Download is not running")
}
func (m *Manager) transfer(ctx context.Context, entry library.Entry) {
	var received int64
	var failure error
	last := time.Time{}
	for _, file := range entry.Files {
		destination, err := library.ManagedPath(m.directory, entry.ID, file.Name)
		if err != nil {
			failure = err
			break
		}
		failure = download.Transfer(ctx, file, destination, func(n int64) {
			if time.Since(last) < 150*time.Millisecond && n != file.Size {
				return
			}
			last = time.Now()
			m.mutex.Lock()
			index := m.index(entry.ID)
			if index >= 0 {
				m.state.Models[index].Received = received + n
				m.publishLocked(false)
			}
			m.mutex.Unlock()
		})
		if failure != nil {
			break
		}
		received += file.Size
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.jobs, entry.ID)
	index := m.index(entry.ID)
	if index < 0 {
		return
	}
	held := &m.state.Models[index]
	if ctx.Err() != nil {
		held.Status = "paused"
	} else if failure != nil {
		held.Status = "failed"
		held.Error = failure.Error()
	} else {
		held.Status = "verifying"
		m.publishLocked(false)
		if err := m.inspect(held); err != nil {
			held.Status = "failed"
			held.Error = err.Error()
		} else {
			held.Status = "installed"
			held.Received = held.Size
		}
	}
	m.publishLocked(true)
}

func (m *Manager) Cancel(id string) error {
	m.mutex.Lock()
	job := m.jobs[id]
	if job != nil {
		job.cancel()
	}
	m.mutex.Unlock()
	if job != nil {
		<-job.done
	}
	return m.Remove(id, true)
}
