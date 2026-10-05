package jobs

import (
	"arx/internal/media/artifacts"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type Execute func(context.Context, func(float64, string)) (artifacts.Artifact, error)

type Manager struct {
	mutex  sync.Mutex
	store  Store
	active string
	cancel context.CancelFunc
	done   chan struct{}
	jobs   []Job
	notify func(Job)
}

func Open(directory string) (*Manager, error) {
	store := Store{Directory: directory}
	saved, err := store.Recover()
	if err != nil {
		return nil, err
	}
	return &Manager{store: store, jobs: saved}, nil
}

func (m *Manager) Subscribe(notify func(Job)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notify = notify
}

func (m *Manager) Snapshot() []Job {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]Job{}, m.jobs...)
}

func (m *Manager) Run(ctx context.Context, job Job, execute Execute) (Job, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return job, err
	}
	job.ID = hex.EncodeToString(token[:])
	job.State, job.Updated = "queued", time.Now().UTC()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	m.mutex.Lock()
	if m.active != "" {
		m.mutex.Unlock()
		return job, errors.New("Another generation is running")
	}
	if err := m.store.Save(job); err != nil {
		m.mutex.Unlock()
		return job, err
	}
	index := len(m.jobs)
	m.jobs = append(m.jobs, job)
	m.active, m.cancel = job.ID, cancel
	done := make(chan struct{})
	m.done = done
	defer close(done)
	notify := m.notify
	m.mutex.Unlock()
	if notify != nil {
		notify(job)
	}
	var saveErr error
	lastUpdate := time.Time{}
	update := func(progress float64, detail string) {
		m.mutex.Lock()
		if job.Detail == detail && progress < 1 && time.Since(lastUpdate) < 100*time.Millisecond {
			m.mutex.Unlock()
			return
		}
		lastUpdate = time.Now()
		job.State, job.Progress, job.Detail, job.Updated = "running", max(0, min(1, progress)), detail, time.Now().UTC()
		m.jobs[index] = job
		if saveErr == nil {
			saveErr = m.store.Save(job)
		}
		snapshot, notify := job, m.notify
		m.mutex.Unlock()
		if notify != nil {
			notify(snapshot)
		}
	}
	output, err := execute(ctx, update)
	m.mutex.Lock()
	job.State, job.Updated = "done", time.Now().UTC()
	if output.ID != "" && ctx.Err() == nil {
		job.Output = &output
	}
	if err == nil && saveErr != nil {
		err = saveErr
	}
	if ctx.Err() != nil {
		err = ctx.Err()
		job.State, job.Error = "cancelled", "Generation stopped"
	} else if err != nil {
		job.State, job.Error = "failed", err.Error()
	} else {
		job.Output, job.Progress = &output, 1
	}
	if e := m.store.Save(job); e != nil {
		err = errors.Join(err, e)
	}
	m.jobs[index] = job
	m.active, m.cancel = "", nil
	notify = m.notify
	m.mutex.Unlock()
	if notify != nil {
		notify(job)
	}
	return job, err
}

func (m *Manager) Cancel(id string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.cancel == nil || (id != "" && id != m.active && id != m.jobs[len(m.jobs)-1].ToolCall) {
		return false
	}
	m.cancel()
	return true
}

func (m *Manager) Close() {
	m.mutex.Lock()
	cancel, done := m.cancel, m.done
	m.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}
