package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"
)

const discoveryTimeout = 15 * time.Second
const callTimeout = 60 * time.Second
const messageLimit = 12 << 20

func (m *Manager) connect(parent context.Context, id string) (*connection, error) {
	if m.closed {
		return nil, errors.New("MCP manager is closed")
	}
	i := m.index(id)
	if i < 0 {
		return nil, errors.New("MCP server was not found")
	}
	s := &m.servers[i]
	if !s.Enabled {
		return nil, errors.New("MCP server is disabled")
	}
	ctx, cancel := context.WithTimeout(parent, discoveryTimeout)
	defer cancel()
	c := m.connections[id]
	if c == nil {
		var err error
		c, err = start(ctx, s.Configuration)
		if err != nil {
			s.Status = "Connection failed"
			s.Error = err.Error()
			return nil, err
		}
		m.connections[id] = c
	}
	tools := []Tool{}
	cursor := ""
	seen := map[string]bool{}
	names := map[string]bool{}
	for page := 0; page < 100; page++ {
		result, err := c.session.ListTools(ctx, &sdk.ListToolsParams{Cursor: cursor})
		if err != nil {
			m.disconnect(id)
			s.Status = "Connection failed"
			s.Error = "Could not discover MCP tools"
			return nil, errors.New(s.Error)
		}
		for _, t := range result.Tools {
			if t.Name == "" || len(t.Name) > 200 || names[t.Name] || len(tools) >= 1000 {
				m.disconnect(id)
				return nil, errors.New("Invalid or excessive MCP tool catalog")
			}
			names[t.Name] = true
			schema, err := json.Marshal(t.InputSchema)
			if err != nil || len(schema) > 256<<10 {
				m.disconnect(id)
				return nil, errors.New("MCP tool schema is too large")
			}
			hint := t.Annotations != nil && t.Annotations.ReadOnlyHint
			calls := 0
			for _, held := range s.Tools {
				if held.Name == t.Name {
					calls = held.Calls
					break
				}
			}
			tools = append(tools, Tool{Name: t.Name, Description: t.Description, InputSchema: schema, ReadOnlyHint: hint, ReadOnly: slices.Contains(s.ReadOnlyTools, t.Name), Calls: calls})
		}
		if result.NextCursor == "" {
			s.Tools = tools
			s.Status = "Connected"
			s.Error = ""
			return c, nil
		}
		if seen[result.NextCursor] {
			break
		}
		seen[result.NextCursor] = true
		cursor = result.NextCursor
	}
	m.disconnect(id)
	return nil, errors.New("MCP discovery pagination exceeded its limit")
}
func start(ctx context.Context, config Configuration) (*connection, error) {
	lifetime, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(lifetime, config.Command, config.Arguments...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = time.Second

	cmd.Stderr = io.Discard
	cmd.Env = []string{}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if _, overridden := config.Environment[key]; !overridden {
			cmd.Env = append(cmd.Env, v)
		}
	}
	for k, v := range config.Environment {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		cancel()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		in.Close()
		out.Close()
		cancel()
		return nil, fmt.Errorf("Could not start MCP executable: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	stop := func() {
		cancel()
		in.Close()
		out.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "Arx", Version: "0.1.0"}, &sdk.ClientOptions{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	session, err := client.Connect(ctx, &sdk.IOTransport{Reader: out, Writer: in, MaxLineLength: messageLimit}, &sdk.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		stop()
		return nil, errors.New("MCP initialization failed; check the executable and arguments")
	}
	return &connection{session: session, stop: func() { stop(); _ = session.Close() }}, nil
}
