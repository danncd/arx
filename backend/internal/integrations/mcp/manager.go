package mcp

import (
	"arx/internal/integrations"
	"arx/internal/platform/atomicfile"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type connection struct {
	session *sdk.ClientSession
	stop    func()
}
type Manager struct {
	snapshotMu  sync.Mutex
	snapshot    []Server
	mu          sync.Mutex
	path        string
	servers     []Server
	connections map[string]*connection
	closed      bool
}

func Open(directory string) (*Manager, error) {
	m := &Manager{path: filepath.Join(directory, "mcps.json"), connections: map[string]*connection{}}
	if err := atomicfile.ReadJSON(m.path, &m.servers); err != nil {
		return nil, err
	}
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		path := os.Getenv("ARX_MEMO_MCP_PATH")
		if path == "" {
			path = "/Applications/Memo.app/Contents/Resources/bin/memo-mcp"
		}
		if _, err := os.Stat(path); err != nil {
			if home, e := os.UserHomeDir(); e == nil {
				for _, project := range []string{"Memo", "memo"} {
					p := filepath.Join(home, "Documents/Projects", project, "desktop/.build/release/mac-arm64/Memo.app/Contents/Resources/bin/memo-mcp")
					if _, e := os.Stat(p); e == nil {
						path = p
						break
					}
				}
			}
		}
		m.servers = []Server{{Configuration: Configuration{ID: "memo", Name: "Memo", Command: path, Enabled: true, Policy: "changes", Arguments: []string{}, Environment: map[string]string{}, ReadOnlyTools: []string{"memo_list_folders", "memo_list_notes", "memo_search_notes", "memo_read_note", "memo_read_notes", "memo_read_note_metadata", "memo_read_note_section", "memo_get_graph", "memo_get_graph_health", "memo_query_graph", "memo_get_note_links", "memo_get_neighbors", "memo_find_graph_path", "memo_get_activity_cursor", "memo_list_activity", "memo_list_revisions", "memo_compare_revision", "memo_list_trashed_notes", "memo_list_trashed_folders"}}, Tools: []Tool{}, Status: "Not connected"}}
		if err := atomicfile.WriteJSON(m.path, m.servers); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	for i := range m.servers {
		s := &m.servers[i]
		if err := Validate(s.Configuration); err != nil {
			return nil, err
		}
		if seen[s.ID] {
			return nil, errors.New("Duplicate MCP server ID")
		}
		seen[s.ID] = true
		s.Status = "Not connected"
		s.Error = ""
		if !s.Enabled {
			s.Status = "Disabled"
		}
		if s.Tools == nil {
			s.Tools = []Tool{}
		}
		if s.Usage.Activity == nil {
			s.Usage.Activity = []integrations.Activity{}
		}
	}
	m.publish()
	return m, nil
}
func (m *Manager) publish() {
	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()
	m.snapshot = integrations.Copy(m.servers)
}
func (m *Manager) State() []Server {
	m.snapshotMu.Lock()
	defer m.snapshotMu.Unlock()
	out := integrations.Copy(m.snapshot)
	if out == nil {
		return []Server{}
	}
	return out
}

func (m *Manager) index(id string) int {
	return slices.IndexFunc(m.servers, func(s Server) bool { return s.ID == id })
}
func (m *Manager) Save(c Configuration) ([]Server, error) {
	if err := Validate(c); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	if m.closed {
		return nil, errors.New("MCP manager is closed")
	}
	if c.Arguments == nil {
		c.Arguments = []string{}
	}
	if c.Environment == nil {
		c.Environment = map[string]string{}
	}
	if c.ReadOnlyTools == nil {
		c.ReadOnlyTools = []string{}
	}
	next := integrations.Copy(m.servers)
	i := m.index(c.ID)
	s := Server{Configuration: c, Tools: []Tool{}, Status: "Not connected"}
	if !c.Enabled {
		s.Status = "Disabled"
	}
	if i >= 0 {
		s.Usage = next[i].Usage
		if c.Command == next[i].Command && slices.Equal(c.Arguments, next[i].Arguments) {
			s.Tools = next[i].Tools
			for j := range s.Tools {
				s.Tools[j].ReadOnly = slices.Contains(c.ReadOnlyTools, s.Tools[j].Name)
			}
		}
		next[i] = s
	} else {
		if len(next) >= 32 {
			return nil, errors.New("Choose up to 32 MCP servers")
		}
		next = append(next, s)
	}
	if err := atomicfile.WriteJSON(m.path, next); err != nil {
		return nil, err
	}
	m.disconnect(c.ID)
	m.servers = next
	return integrations.Copy(next), nil
}
func (m *Manager) Remove(id string) ([]Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	i := m.index(id)
	if i < 0 {
		return nil, errors.New("MCP server was not found")
	}
	next := slices.Delete(integrations.Copy(m.servers), i, i+1)
	if err := atomicfile.WriteJSON(m.path, next); err != nil {
		return nil, err
	}
	m.disconnect(id)
	m.servers = next
	return integrations.Copy(next), nil
}
func (m *Manager) disconnect(id string) {
	if c := m.connections[id]; c != nil {
		c.stop()
		delete(m.connections, id)
	}
}
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	m.closed = true
	for id := range m.connections {
		m.disconnect(id)
	}
}
func (m *Manager) Test(ctx context.Context, id string, reconnect bool) ([]Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	if reconnect {
		m.disconnect(id)
	}
	_, err := m.connect(ctx, id)
	return integrations.Copy(m.servers), err
}
func (m *Manager) Inspect(ctx context.Context, id string) ([]Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	if _, err := m.connect(ctx, id); err != nil {
		return nil, err
	}
	return integrations.Copy(m.servers[m.index(id)].Tools), nil
}
func (m *Manager) Action(ctx context.Context, id, name string) (Configuration, Tool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	if _, err := m.connect(ctx, id); err != nil {
		return Configuration{}, Tool{}, err
	}
	s := m.servers[m.index(id)]
	for _, t := range s.Tools {
		if t.Name == name {
			return integrations.Copy(s.Configuration), t, nil
		}
	}
	return Configuration{}, Tool{}, errors.New("Choose an available MCP tool")
}
func (m *Manager) Call(ctx context.Context, id, name string, args json.RawMessage, conversation string) (*sdk.CallToolResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	c, err := m.connect(ctx, id)
	if err != nil {
		return nil, err
	}
	i := m.index(id)
	if !slices.ContainsFunc(m.servers[i].Tools, func(t Tool) bool { return t.Name == name }) {
		return nil, errors.New("MCP tool is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	result, err := c.session.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	status := "Succeeded"
	if err != nil {
		status = "Failed"
	} else if result.IsError {
		status = "Tool error"
	}
	next := integrations.Copy(m.servers)
	next[i].Usage.Record(name, conversation, status)
	for j := range next[i].Tools {
		if next[i].Tools[j].Name == name {
			next[i].Tools[j].Calls++
		}
	}
	if err != nil {
		next[i].Status = "Connection failed"
		next[i].Error = "MCP call failed"
		m.disconnect(id)
	}
	if saveErr := atomicfile.WriteJSON(m.path, next); saveErr != nil {
		return result, fmt.Errorf("Call completed with status %s, but activity could not be saved: %w", status, saveErr)
	}
	m.servers = next
	return result, err
}
func (m *Manager) Denied(id, name, conversation string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer m.publish()
	i := m.index(id)
	if i < 0 {
		return errors.New("MCP server was not found")
	}
	next := integrations.Copy(m.servers)
	next[i].Usage.Record(name, conversation, "Permission denied")
	if err := atomicfile.WriteJSON(m.path, next); err != nil {
		return err
	}
	m.servers = next
	return nil
}

func Probe(ctx context.Context, c Configuration) ([]Server, error) {
	if err := Validate(c); err != nil {
		return nil, err
	}
	c.Enabled = true
	m := &Manager{servers: []Server{{Configuration: c, Tools: []Tool{}}}, connections: map[string]*connection{}}
	defer m.Close()
	return m.Test(ctx, c.ID, false)
}
