package local

import (
	"arx/internal/models/local/catalog"
	"arx/internal/models/local/hardware"
	"arx/internal/models/local/library"
	engine "arx/internal/models/local/runtime"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	lockFile      *os.File
	ReadImage     func(string) ([]byte, error)
	mutex         sync.Mutex
	state         State
	directory     string
	catalog       *catalog.Client
	engine        *engine.Server
	notify        func(State)
	jobs          map[string]*downloadJob
	done          sync.WaitGroup
	runtimeCancel context.CancelFunc
	runtimeDone   chan struct{}
	lastUsed      time.Time
	idleTimeout   time.Duration
	active        int
	closed        bool
}

func Open(directory string) (*Manager, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(directory, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("The local model library is already open in another Arx instance")
	}
	engine.Recover(directory)
	models, err := library.Read(directory)
	if err != nil {
		lock.Close()
		return nil, err
	}
	m := &Manager{lockFile: lock, directory: directory, catalog: catalog.New(), engine: &engine.Server{}, jobs: map[string]*downloadJob{}, state: State{Models: models, Hardware: hardware.Detect(), Runtime: Runtime{State: "stopped"}}, idleTimeout: idleTimeout}
	for i := range m.state.Models {
		if m.state.Models[i].Status == "installed" {
			entry := m.state.Models[i]
			if err := m.inspect(&entry); err == nil {
				m.state.Models[i] = entry
			}
		}
	}
	return m, nil
}
func (m *Manager) Search(ctx context.Context, options catalog.SearchOptions) (catalog.SearchPage, error) {
	return m.catalog.SearchPage(ctx, options)
}
func (m *Manager) Repository(ctx context.Context, id string) (catalog.Repository, error) {
	result, err := m.catalog.Repository(ctx, id)
	for i := range result.Variants {
		result.Variants[i].Fit = m.state.Hardware.Fit(result.Variants[i].Size)
	}
	return result, err
}
func (m *Manager) index(id string) int {
	for i := range m.state.Models {
		if m.state.Models[i].ID == id {
			return i
		}
	}
	return -1
}
func (m *Manager) Close() {
	m.mutex.Lock()
	m.closed = true
	for _, job := range m.jobs {
		job.cancel()
	}
	if m.runtimeCancel != nil {
		m.runtimeCancel()
	}
	m.mutex.Unlock()
	m.done.Wait()
	m.engine.Stop()
	m.lockFile.Close()
}
func (m *Manager) Unload() error {
	m.mutex.Lock()
	if m.runtimeCancel != nil {
		m.runtimeCancel()
	}
	done := m.runtimeDone
	m.mutex.Unlock()
	if done != nil {
		<-done
	}
	m.engine.Stop()
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i := range m.state.Models {
		if m.state.Models[i].Status == "loaded" || m.state.Models[i].Status == "loading" {
			m.state.Models[i].Status = "installed"
		}
	}
	m.state.Runtime = Runtime{State: "stopped"}
	m.publishLocked(true)
	return nil
}
func (m *Manager) Remove(id string, deleteFiles bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	index := m.index(id)
	if index < 0 {
		return errors.New("Model was not found")
	}
	entry := m.state.Models[index]
	if m.jobs[id] != nil || entry.Status == "loading" || entry.Status == "loaded" {
		return errors.New("Stop the download or unload this model first")
	}
	if deleteFiles && entry.Imported {
		if err := library.DeleteImported(entry, m.state.Models); err != nil {
			return err
		}
	}
	if deleteFiles && !entry.Imported {
		directory, err := library.ManagedDirectory(m.directory, id)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(directory); err != nil {
			return err
		}
	}
	m.state.Models = append(m.state.Models[:index], m.state.Models[index+1:]...)
	m.publishLocked(true)
	return nil
}

type downloadJob struct {
	cancel context.CancelFunc
	done   chan struct{}
}
