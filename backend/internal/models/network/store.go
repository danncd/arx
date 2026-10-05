package network

import (
	credentials "arx/internal/platform/keychain"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Manager struct {
	cacheMutex sync.Mutex
	cache      map[string]*metadataEntry
	mutex      sync.Mutex
	scanMutex  sync.Mutex
	cancelScan context.CancelFunc
	directory  string
	servers    []savedServer
	credential func(string) (credentials.Store, error)
}

func Open(directory string) (*Manager, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	m := &Manager{directory: directory, servers: []savedServer{}, cache: map[string]*metadataEntry{}}
	m.credential = func(id string) (credentials.Store, error) { return credentials.New(filepath.Join(directory, id)) }
	data, err := os.ReadFile(filepath.Join(directory, "servers.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(data, &m.servers); err != nil {
			return nil, err
		}
	}
	return m, nil
}
func (m *Manager) save(servers []savedServer) error {
	data, err := json.MarshalIndent(servers, "", "    ")
	if err != nil {
		return err
	}
	path := filepath.Join(m.directory, "servers.json")
	if err = os.WriteFile(path+".tmp", data, 0600); err != nil {
		return err
	}
	if err = os.Rename(path+".tmp", path); err != nil {
		return err
	}
	m.servers = servers
	return nil
}
func (m *Manager) Connect(ctx context.Context, address, name, token string) (Server, error) {
	if len(token) > 4096 || strings.ContainsAny(token, "\r\n") {
		return Server{}, errors.New("Invalid API token")
	}
	server, err := discover(ctx, address, token)
	if err != nil {
		return server, err
	}
	if !server.Connected {
		return server, errors.New("Enter a valid API token")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "LM Studio"
	}
	if len(name) > 80 {
		return server, errors.New("Server name is too long")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	store, err := m.credential(server.ID)
	if err != nil {
		return server, err
	}
	if token != "" {
		if err = store.Save(ctx, token); err != nil {
			return server, err
		}
	}
	entry := savedServer{ID: server.ID, Name: name, URL: server.URL, HasToken: token != ""}
	next := append([]savedServer{}, m.servers...)
	replaced := false
	for i, existing := range next {
		if existing.ID == entry.ID {
			next[i] = entry
			replaced = true
		}
	}
	if !replaced {
		next = append(next, entry)
	}
	if err = m.save(next); err != nil {
		return server, err
	}
	server.Name = name
	m.clearMetadata(server.ID)
	return server, nil
}
func (m *Manager) Remove(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	next := []savedServer{}
	for _, entry := range m.servers {
		if entry.ID != id {
			next = append(next, entry)
		} else if entry.HasToken {
			store, err := m.credential(id)
			if err != nil {
				return err
			}
			if err = store.Delete(ctx); err != nil {
				return err
			}
		}
	}
	if err := m.save(next); err != nil {
		return err
	}
	m.clearMetadata(id)
	return nil
}
func (m *Manager) entries() []savedServer {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]savedServer{}, m.servers...)
}
func (m *Manager) token(ctx context.Context, entry savedServer) (string, error) {
	if !entry.HasToken {
		return "", nil
	}
	store, err := m.credential(entry.ID)
	if err != nil {
		return "", err
	}
	return store.Load(ctx)
}
func (m *Manager) State(ctx context.Context) []Server {
	entries := m.entries()
	result := make([]Server, len(entries))
	var pending sync.WaitGroup
	for i, entry := range entries {
		pending.Go(func() {
			server, err := m.metadata(ctx, entry, true)
			if err != nil {
				server.Error = err.Error()
			}
			result[i] = server
		})
	}
	pending.Wait()
	return result
}
