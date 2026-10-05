package runner

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestIntegrationCatalogsDistinguishSkillsFromServersAndCorrectWrongSkillID(t *testing.T) {
	r, _, _, _ := managers(t)
	ctx := WithScope(context.Background(), &SkillScope{Loaded: map[string]string{}, Explicit: map[string]bool{}})
	directory := t.TempDir()
	policy := func() (string, permission.Policy) { return directory, permission.Policy{Mode: permission.Ask} }
	for _, entry := range []struct{ name, kind, key string }{{"skills", "skill_catalog", "skills"}, {"mcp", "mcp_server_catalog", "servers"}} {
		result, err := r.Run(ctx, tool.Call{ID: entry.name, Name: entry.name, Arguments: json.RawMessage(`{"operation":"list"}`)}, policy)
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]json.RawMessage
		if err := json.Unmarshal([]byte(result.Text), &catalog); err != nil {
			t.Fatal(err)
		}
		var kind string
		json.Unmarshal(catalog["kind"], &kind)
		if kind != entry.kind || len(catalog[entry.key]) == 0 {
			t.Fatalf("ambiguous catalog: %s", result.Text)
		}
	}
	_, err := r.Run(ctx, tool.Call{ID: "wrong-skill", Name: "skills", Arguments: json.RawMessage(`{"operation":"load","id":"memo_search_notes"}`)}, policy)
	if err == nil || !strings.Contains(err.Error(), "Enabled skill IDs: memo.") || !strings.Contains(err.Error(), "MCP tool names are not skill IDs") {
		t.Fatalf("missing recovery guidance: %v", err)
	}
	_, err = r.Run(ctx, tool.Call{ID: "wrong-operation", Name: "mcp", Arguments: json.RawMessage(`{"operation":"memo_read_note","server":"memo"}`)}, policy)
	if err == nil || !strings.Contains(err.Error(), `"operation":"call"`) || !strings.Contains(err.Error(), "goes in name, not operation") {
		t.Fatalf("missing wrapper guidance: %v", err)
	}
}
