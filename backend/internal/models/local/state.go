package local

import (
	"arx/internal/models/local/hardware"
	"arx/internal/models/local/library"
	"encoding/json"
)

type Runtime struct {
	State    string `json:"state"`
	Model    string `json:"model,omitempty"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
	Error    string `json:"error,omitempty"`
}
type State struct {
	Revision uint64          `json:"revision"`
	Hardware hardware.Info   `json:"hardware"`
	Models   []library.Entry `json:"models"`
	Runtime  Runtime         `json:"runtime"`
	Error    string          `json:"error,omitempty"`
}

func (m *Manager) snapshotLocked() State {
	data, _ := json.Marshal(m.state)
	var copy State
	json.Unmarshal(data, &copy)
	return copy
}
func (m *Manager) Snapshot() State { m.mutex.Lock(); defer m.mutex.Unlock(); return m.snapshotLocked() }
func (m *Manager) Subscribe(notify func(State)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notify = notify
}
func (m *Manager) publishLocked(save bool) {
	if save {
		if err := library.Save(m.directory, m.state.Models); err != nil {
			m.state.Error = "Could not save local models: " + err.Error()
		}
	}
	m.state.Revision++
	if m.notify != nil {
		m.notify(m.snapshotLocked())
	}
}
