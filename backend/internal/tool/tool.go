package tool

import (
	"context"
	"encoding/json"
	"sort"
	"unicode/utf8"

	"arx/internal/provider"
)

const maxToolBytes = 64 * 1024

func Command(args string) string {
	var a struct {
		Command string `json:"command"`
	}
	if json.Unmarshal([]byte(args), &a) != nil {
		return ""
	}
	return a.Command
}

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Mutating    bool
	Run         func(ctx context.Context, args json.RawMessage) (string, error)
}

var registry = map[string]Tool{}

func Register(t Tool) { registry[t.Name] = t }

func Get(name string) (Tool, bool) {
	t, ok := registry[name]
	return t, ok
}

func IsMutating(name string) bool {
	t, ok := registry[name]
	return ok && t.Mutating
}

func Specs() []provider.ToolSpec {
	out := make([]provider.ToolSpec, 0, len(registry))
	for _, t := range registry {
		schema := t.Schema
		if schema == nil {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, provider.ToolSpec{
			Type: "function",
			Function: provider.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Function.Name < out[j].Function.Name })
	return out
}

func clipUTF8(data []byte, limit int) (string, bool) {
	if len(data) <= limit {
		return string(data), false
	}
	data = data[:limit]
	for len(data) > 0 {
		r, size := utf8.DecodeLastRune(data)
		if r != utf8.RuneError || size > 1 {
			break
		}
		data = data[:len(data)-1]
	}
	return string(data), true
}
