package network

import (
	"context"
	"time"
)

const metadataLifetime = 30 * time.Second

type metadataEntry struct {
	server  Server
	err     error
	fetched time.Time
	pending chan struct{}
}

func (m *Manager) clearMetadata(id string) {
	m.cacheMutex.Lock()
	delete(m.cache, id)
	m.cacheMutex.Unlock()
}

func (m *Manager) metadata(ctx context.Context, entry savedServer, force bool) (Server, error) {
	m.cacheMutex.Lock()
	cached := m.cache[entry.ID]
	if cached != nil && cached.pending != nil {
		pending := cached.pending
		m.cacheMutex.Unlock()
		select {
		case <-pending:
			return cached.server, cached.err
		case <-ctx.Done():
			return Server{}, ctx.Err()
		}
	}
	if cached != nil && !force && time.Since(cached.fetched) < metadataLifetime {
		m.cacheMutex.Unlock()
		return cached.server, cached.err
	}
	current := &metadataEntry{pending: make(chan struct{})}
	m.cache[entry.ID] = current
	m.cacheMutex.Unlock()
	server := Server{ID: entry.ID, Name: entry.Name, URL: entry.URL, Models: []Model{}}
	token, err := m.token(ctx, entry)
	if err == nil {
		server, err = discover(ctx, entry.URL, token)
	}
	server.Name = entry.Name
	m.cacheMutex.Lock()
	current.server, current.err, current.fetched = server, err, time.Now()
	close(current.pending)
	current.pending = nil
	m.cacheMutex.Unlock()
	return server, err
}
