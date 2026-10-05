package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const fixture = `import json,sys
for line in sys.stdin:
 r=json.loads(line)
 if 'id' not in r: continue
 method=r['method']
 if method=='initialize': result={'protocolVersion':'2025-11-25','capabilities':{'tools':{}},'serverInfo':{'name':'test','version':'1'}}
 elif method=='tools/list':
  if not r.get('params',{}).get('cursor'): result={'tools':[{'name':'get','description':'Read','inputSchema':{'type':'object'},'annotations':{'readOnlyHint':True}}],'nextCursor':'next'}
  else: result={'tools':[{'name':'change','description':'Write','inputSchema':{'type':'object'}}]}
 elif method=='tools/call': result={'content':[{'type':'text','text':'first'},{'type':'text','text':'second'}],'structuredContent':r['params']['arguments']}
 else: result={}
 print(json.dumps({'jsonrpc':'2.0','method':'notifications/message','params':{'level':'info','data':'log'}}),flush=True)
 print(json.dumps({'jsonrpc':'2.0','id':r['id'],'result':result}),flush=True)
`

func executable(t *testing.T, code string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "server")
	if err := os.WriteFile(p, []byte("#!/usr/bin/python3\n"+code), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestDiscoveryCallsAndDurableUsage(t *testing.T) {
	directory := t.TempDir()
	m, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	c := Configuration{ID: "test", Name: "Test", Command: executable(t, fixture), Enabled: true, Policy: "changes", ReadOnlyTools: []string{}}
	if _, err = m.Save(c); err != nil {
		t.Fatal(err)
	}
	tools, err := m.Inspect(context.Background(), c.ID)
	if err != nil || len(tools) != 2 {
		t.Fatal(tools, err)
	}
	if !tools[0].ReadOnlyHint || tools[0].ReadOnly {
		t.Fatal("Server hint authorized a read")
	}
	first := m.connections[c.ID]
	args := json.RawMessage(`{"id":"note-42"}`)
	result, err := m.Call(context.Background(), c.ID, "change", args, "chat")
	if err != nil || len(result.Content) != 2 {
		t.Fatal(result, err)
	}
	if m.connections[c.ID] != first {
		t.Fatal("session was not reused")
	}
	c.ReadOnlyTools = []string{"get"}
	if _, err = m.Save(c); err != nil {
		t.Fatal(err)
	}
	state := m.State()
	var saved Server
	for _, s := range state {
		if s.ID == c.ID {
			saved = s
		}
	}
	if len(saved.Tools) != 2 || !saved.Tools[0].ReadOnly || saved.Usage.Count != 1 {
		t.Fatal(saved)
	}
	reopened, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, s := range reopened.State() {
		if s.ID == c.ID && s.Usage.Count != 1 {
			t.Fatal("usage lost")
		}
	}
	if _, err = m.Save(Configuration{ID: "bad", Name: "Test", Command: "x", Policy: "invalid"}); err == nil {
		t.Fatal("invalid policy accepted")
	}
}
func TestProbeCancellationAndDisabledServers(t *testing.T) {
	c := Configuration{ID: "stall", Name: "Stall", Command: executable(t, "import time\nwhile True: time.sleep(1)\n"), Enabled: true, Policy: "ask"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := Probe(ctx, c); err == nil {
		t.Fatal("stall passed")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("shutdown unbounded")
	}
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	c.Enabled = false
	if _, err = m.Save(c); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Inspect(context.Background(), c.ID); err == nil {
		t.Fatal("disabled server launched")
	}
}
func TestStateIsDetachedAndRemovePersists(t *testing.T) {
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	s := m.State()
	s[0].Name = "changed"
	s[0].Environment["TOKEN"] = "secret"
	if m.State()[0].Name == "changed" || m.State()[0].Environment["TOKEN"] != "" {
		t.Fatal("mutable state escaped")
	}
	if _, err = m.Remove("memo"); err != nil {
		t.Fatal(err)
	}
	next, err := Open(filepath.Dir(m.path))
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if len(next.State()) != 0 {
		t.Fatal("removed default resurrected")
	}
}

func TestStateDoesNotWaitForExternalCall(t *testing.T) {
	m, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.mu.Lock()
	defer m.mu.Unlock()
	done := make(chan []Server, 1)
	go func() { done <- m.State() }()
	select {
	case result := <-done:
		if len(result) != 1 {
			t.Fatal(result)
		}
	case <-time.After(time.Second):
		t.Fatal("UI state waited behind an external call")
	}
}
