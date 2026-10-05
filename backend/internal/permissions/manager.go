package permissions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
)

type Action struct {
	Server    string          `json:"server,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Query     string          `json:"query,omitempty"`
	URL       string          `json:"url,omitempty"`
	Tool      string          `json:"tool"`
	Operation string          `json:"operation"`
	Path      string          `json:"path,omitempty"`
	Command   string          `json:"command,omitempty"`
	Directory string          `json:"directory,omitempty"`
	Before    string          `json:"before,omitempty"`
	After     string          `json:"after,omitempty"`
	Writes    bool            `json:"-"`
}

type Request struct {
	ID     string `json:"id"`
	Action Action `json:"action"`
}

type pending struct {
	request Request
	answer  chan bool
}
type Manager struct {
	mutex   sync.Mutex
	pending *pending
	notify  func(*Request)
}

func (m *Manager) Subscribe(notify func(*Request)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notify = notify
}

func (m *Manager) Pending() *Request {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.pending == nil {
		return nil
	}
	request := m.pending.request
	return &request
}

func (m *Manager) Authorize(ctx context.Context, policy Policy, action Action) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch policy.Mode {
	case Full:
		return nil
	case Folders:
		if !action.Writes || action.Tool == "bash" {
			return nil
		}
		_, err := policy.Root(action.Path)
		return err
	case Ask:
	default:
		return errors.New("Unknown permission mode")
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	held := &pending{request: Request{ID: hex.EncodeToString(nonce[:]), Action: action}, answer: make(chan bool, 1)}
	m.mutex.Lock()
	if m.pending != nil {
		m.mutex.Unlock()
		return errors.New("Another action is waiting for approval")
	}
	m.pending = held
	if m.notify != nil {
		m.notify(&held.request)
	}
	m.mutex.Unlock()
	defer func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if m.pending == held {
			m.pending = nil
			if m.notify != nil {
				m.notify(nil)
			}
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case allowed := <-held.answer:
		if !allowed {
			return errors.New("Permission denied")
		}
		return ctx.Err()
	}
}

func (m *Manager) Respond(id string, allow bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.pending == nil || m.pending.request.ID != id {
		return errors.New("This approval request has expired")
	}
	select {
	case m.pending.answer <- allow:
		return nil
	default:
		return errors.New("This approval request was already answered")
	}
}
