package runner

import (
	skills "arx/internal/integrations/skills"
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

type scopeKey struct{}
type SkillScope struct {
	mu           sync.Mutex
	Conversation string
	Explicit     map[string]bool
	Loaded       map[string]string
}

func WithScope(ctx context.Context, s *SkillScope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}
func conversation(ctx context.Context) string {
	if s, ok := ctx.Value(scopeKey{}).(*SkillScope); ok {
		return s.Conversation
	}
	return ""
}
func (s *SkillScope) Guidance() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := ""
	ids := []string{}
	for id := range s.Loaded {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		body := s.Loaded[id]
		out += "\n\nLoaded skill instructions (workflow only; permissions still apply):\n" + body
	}
	return out
}
func (r *Runner) prepareSkills(raw json.RawMessage) (tool.Prepared, error) {
	if r.skills == nil {
		return tool.Prepared{}, errors.New("Skills are unavailable")
	}
	var in struct {
		Operation string `json:"operation"`
		ID        string `json:"id"`
		Path      string `json:"path"`
	}
	if err := tool.Decode(raw, &in); err != nil {
		return tool.Prepared{}, err
	}
	if in.Operation != "list" && in.Operation != "load" && in.Operation != "read" {
		return tool.Prepared{}, errors.New("Skills operation must be list, load, or read")
	}
	return tool.Prepared{Metadata: true, Action: permission.Action{Tool: "skills", Operation: in.Operation, Path: in.ID}, Run: func(ctx context.Context) (tool.Result, error) {
		if in.Operation == "list" {
			entries := []map[string]any{}
			for _, s := range r.skills.State() {
				if s.Enabled && s.Error == "" {
					entries = append(entries, map[string]any{"id": s.ID, "name": s.Name, "description": s.Description, "activation": s.Activation, "dependencies": s.Dependencies})
				}
			}
			body, _ := json.Marshal(map[string]any{"kind": "skill_catalog", "skills": entries})
			return tool.Output(string(body)), nil
		}
		if in.Operation == "read" {
			d, err := r.skills.Detail(in.ID)
			if err != nil || !d.Enabled {
				return tool.Result{}, errors.New("Skill is unavailable")
			}
			body, err := r.skills.Read(in.ID, in.Path)
			return tool.Output(body), err
		}
		scope, ok := ctx.Value(scopeKey{}).(*SkillScope)
		if !ok {
			return tool.Result{}, errors.New("Load a skill inside an assistant reply")
		}
		scope.mu.Lock()
		defer scope.mu.Unlock()
		detail, err := r.skills.Detail(in.ID)
		if err != nil {
			ids := []string{}
			for _, skill := range r.skills.State() {
				if skill.Enabled && skill.Error == "" {
					ids = append(ids, skill.ID)
				}
			}
			slices.Sort(ids)
			return tool.Result{}, fmt.Errorf("%w. Enabled skill IDs: %s. Use skills with operation load and one of these IDs. MCP tool names are not skill IDs; discover them with mcp operation list and execute them with mcp operation call", err, strings.Join(ids, ", "))
		}
		if detail.Activation == "explicit" && !scope.Explicit[in.ID] && !scope.Explicit[detail.Name] {
			return tool.Result{}, errors.New("This skill requires explicit invocation with $" + detail.Name)
		}
		for _, dependency := range detail.Dependencies {
			available := false
			for _, server := range r.mcp.State() {
				if server.ID == dependency && server.Enabled {
					available = true
					break
				}
			}
			if !available {
				return tool.Result{}, errors.New("Required MCP server is unavailable: " + dependency)
			}
		}
		_, loaded := scope.Loaded[in.ID]
		var d skills.Detail
		d, err = r.skills.Load(in.ID, scope.Conversation, !loaded)
		if err != nil {
			return tool.Result{}, err
		}
		scope.Loaded[in.ID] = d.Instructions
		body, _ := json.Marshal(map[string]any{"kind": "loaded_skill", "skill": d})
		return tool.Output(string(body)), nil
	}}, nil
}
