package local

import (
	"errors"
	"sync"
	"time"
)

const idleTimeout = 5 * time.Minute

func (m *Manager) SetIdleMinutes(minutes int) {
	m.mutex.Lock()
	m.idleTimeout = time.Duration(minutes) * time.Minute
	m.mutex.Unlock()
}

func (m *Manager) Hold(id string) (func(), error) {
	m.mutex.Lock()
	if m.closed || m.state.Runtime.State != "ready" || m.state.Runtime.Model != id {
		m.mutex.Unlock()
		return nil, errors.New("The selected local model is not loaded")
	}
	m.active++
	m.lastUsed = time.Now()
	m.mutex.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			m.mutex.Lock()
			m.active--
			m.lastUsed = time.Now()
			m.mutex.Unlock()
		})
	}, nil
}

func (m *Manager) idleExpired(now time.Time) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	timeout := m.idleTimeout
	if timeout == 0 {
		timeout = idleTimeout
	}
	if m.state.Runtime.State != "ready" || m.active != 0 || m.lastUsed.IsZero() || now.Sub(m.lastUsed) < timeout {
		return false
	}
	m.state.Runtime.State = "stopping"
	return true
}
