package local

import (
	provider "arx/internal/inference"
	localprovider "arx/internal/inference/chatcompletions"
	model "arx/internal/models"
	engine "arx/internal/models/local/runtime"
	"context"
	"errors"
	"path/filepath"
	"time"
)

func (m *Manager) Load(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return errors.New("Backend is stopping")
	}
	if !m.state.Hardware.Supported {
		return errors.New("Local models currently require an Apple Silicon Mac")
	}
	if m.runtimeCancel != nil {
		if m.state.Runtime.Model == id && (m.state.Runtime.State == "loading" || m.state.Runtime.State == "ready" || m.state.Runtime.State == "installing") {
			return nil
		}
		return errors.New("Unload the current model first")
	}
	index := m.index(id)
	if index < 0 {
		return errors.New("Model was not found")
	}
	entry := m.state.Models[index]
	if entry.Status != "installed" && entry.Status != "loaded" {
		return errors.New("Finish downloading this model first")
	}
	if err := m.inspect(&entry); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.runtimeCancel = cancel
	m.runtimeDone = done
	m.state.Models[index].Info = entry.Info
	m.state.Models[index].Status = "loading"
	m.state.Runtime = Runtime{State: "installing", Model: id}
	m.publishLocked(false)
	m.done.Add(1)
	go func() {
		defer m.done.Done()
		var failure error
		defer func() {
			m.engine.Stop()
			m.mutex.Lock()
			if i := m.index(id); i >= 0 {
				m.state.Models[i].Status = "installed"
			}
			m.runtimeCancel = nil
			m.state.Runtime = Runtime{State: "stopped"}
			if failure != nil && ctx.Err() == nil {
				m.state.Runtime = Runtime{State: "failed", Model: id, Error: failure.Error()}
			}
			m.publishLocked(true)
			close(done)
			m.mutex.Unlock()
		}()
		binary, err := engine.Install(ctx, filepath.Join(m.directory, "engine"), func(n, total int64) {
			m.mutex.Lock()
			m.state.Runtime.Received = n
			m.state.Runtime.Total = total
			m.publishLocked(false)
			m.mutex.Unlock()
		})
		if err != nil {
			failure = err
			return
		}
		m.mutex.Lock()
		m.state.Runtime = Runtime{State: "loading", Model: id}
		m.publishLocked(false)
		m.mutex.Unlock()
		info, err := m.engine.Start(ctx, binary, entry.Path, entry.Projector, entry.Info, filepath.Join(m.directory, "runtime.log"))
		if err != nil {
			failure = err
			return
		}
		m.mutex.Lock()
		if i := m.index(id); i >= 0 {
			m.state.Models[i].Info = info
			m.state.Models[i].Status = "loaded"
		}
		m.state.Runtime = Runtime{State: "ready", Model: id}
		m.lastUsed = time.Now()
		m.publishLocked(true)
		m.mutex.Unlock()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if m.idleExpired(time.Now()) {
					return
				}
				if _, _, _, err := m.engine.Connection(); err != nil {
					failure = err
					return
				}
			}
		}
	}()
	return nil
}

func (m *Manager) Ensure(ctx context.Context, id string) (model.Info, error) {
	state := m.Snapshot()
	if state.Runtime.Model != "" && state.Runtime.Model != id {
		if err := m.Unload(); err != nil {
			return model.Info{}, err
		}
	}
	if err := m.Load(id); err != nil {
		return model.Info{}, err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		state = m.Snapshot()
		if state.Runtime.State == "failed" {
			return model.Info{}, errors.New(state.Runtime.Error)
		}
		if state.Runtime.State == "ready" && state.Runtime.Model == id {
			return m.Model(id)
		}
		if state.Runtime.State == "stopped" {
			return model.Info{}, errors.New("Model loading was cancelled")
		}
		select {
		case <-ctx.Done():
			return model.Info{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) Complete(ctx context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	release, err := m.Hold(request.Model)
	if err != nil {
		return provider.Response{}, err
	}
	defer release()
	endpoint, key, info, err := m.engine.Connection()
	if err != nil {
		return provider.Response{}, err
	}
	if info.ID != request.Model {
		return provider.Response{}, errors.New("The selected model is not loaded")
	}
	client := localprovider.Client{URL: endpoint, Key: key, Info: info, ReadImage: m.ReadImage}
	return client.Complete(ctx, request, emit)
}
