package runner

import (
	mcp "arx/internal/integrations/mcp"
	skills "arx/internal/integrations/skills"
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func managers(t *testing.T) (*Runner, *mcp.Manager, *skills.Manager, *permission.Manager) {
	t.Helper()
	root := t.TempDir()
	m, err := mcp.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	s, err := skills.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p := &permission.Manager{}
	r := New(p, nil)
	r.Integrations(m, s)
	return r, m, s, p
}
func TestSkillLoadCountsOnceAndRespectsExplicitAndDisabled(t *testing.T) {
	r, _, s, _ := managers(t)
	scope := &SkillScope{Loaded: map[string]string{}, Explicit: map[string]bool{}, Conversation: "chat"}
	ctx := WithScope(context.Background(), scope)
	policy := func() (string, permission.Policy) { return t.TempDir(), permission.Policy{Mode: permission.Ask} }
	load := tool.Call{ID: "load", Name: "skills", Arguments: json.RawMessage(`{"operation":"load","id":"memo"}`)}
	for range 2 {
		if _, err := r.Run(ctx, load, policy); err != nil {
			t.Fatal(err)
		}
	}
	d, _ := s.Detail("memo")
	if d.Usage.Count != 1 || !strings.Contains(scope.Guidance(), "# Memo") {
		t.Fatal(d, scope.Guidance())
	}
	if _, err := s.Save(skills.Edit{ID: d.ID, Path: d.Path, Activation: "explicit", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	scope.Loaded = map[string]string{}
	if _, err := r.Run(ctx, load, policy); err == nil {
		t.Fatal("explicit skill loaded automatically")
	}
	scope.Explicit["memo"] = true
	if _, err := r.Run(ctx, load, policy); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(skills.Edit{ID: d.ID, Path: d.Path, Activation: "auto", Enabled: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(ctx, load, policy); err == nil {
		t.Fatal("disabled skill loaded")
	}
}
func TestMCPApprovalHasServerAndExactArguments(t *testing.T) {
	r, m, _, permissions := managers(t)
	path := filepath.Join(t.TempDir(), "server")
	script := `#!/usr/bin/python3
import json,sys
for line in sys.stdin:
 r=json.loads(line)
 if 'id' not in r: continue
 if r['method']=='initialize': result={'protocolVersion':'2025-11-25','capabilities':{'tools':{}},'serverInfo':{'name':'test','version':'1'}}
 elif r['method']=='tools/list': result={'tools':[{'name':'change','inputSchema':{'type':'object'},'annotations':{'readOnlyHint':True}}]}
 elif r['method']=='tools/call': result={'content':[{'type':'text','text':'done'}]}
 else: result={}
 print(json.dumps({'jsonrpc':'2.0','id':r['id'],'result':result}),flush=True)
`
	os.WriteFile(path, []byte(script), 0700)
	if _, err := m.Save(mcp.Configuration{ID: "test", Name: "Test", Command: path, Enabled: true, Policy: "changes"}); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"list", "describe"} {
		result, err := r.Run(context.Background(), tool.Call{ID: "catalog-" + operation, Name: "mcp", Arguments: json.RawMessage(`{"operation":"` + operation + `","server":"test","name":"change"}`)}, func() (string, permission.Policy) { return t.TempDir(), permission.Policy{Mode: permission.Ask} })
		if err != nil {
			t.Fatal(err)
		}
		var catalog struct {
			Kind   string `json:"kind"`
			Server string `json:"server"`
		}
		if err := json.Unmarshal([]byte(result.Text), &catalog); err != nil || catalog.Server != "test" || !strings.HasPrefix(catalog.Kind, "mcp_tool_") {
			t.Fatalf("lost tool provenance: %s, %v", result.Text, err)
		}
	}
	events := make(chan *permission.Request, 2)
	permissions.Subscribe(func(p *permission.Request) { events <- p })
	done := make(chan error, 1)
	call := tool.Call{ID: "mcp-call", Name: "mcp", Arguments: json.RawMessage(`{"operation":"call","server":"test","name":"change","arguments":{"noteId":"note-42"}}`)}
	cfg := func() (string, permission.Policy) { return t.TempDir(), permission.Policy{Mode: permission.Folders} }
	go func() { _, err := r.Run(context.Background(), call, cfg); done <- err }()
	select {
	case p := <-events:
		if p.Action.Server != "test" || string(p.Action.Arguments) != `{"noteId":"note-42"}` {
			t.Fatal(p)
		}
		if err := permissions.Respond(p.ID, false); err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no approval")
	}
	if err := <-done; err == nil {
		t.Fatal("denial ignored")
	}
	for _, s := range m.State() {
		if s.ID == "test" && (s.Usage.Count != 1 || s.Tools[0].Calls != 0) {
			t.Fatal(s)
		}
	}
	if _, err := r.Run(context.Background(), call, func() (string, permission.Policy) { return t.TempDir(), permission.Policy{Mode: permission.Full} }); err != nil {
		t.Fatal("full mode", err)
	}
}
