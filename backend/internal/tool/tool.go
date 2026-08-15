package tool

import (
	"context"
	"encoding/json"
	"sort"
	"unicode/utf8"

	"arx/internal/llm"
)

type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Run         func(ctx context.Context, args json.RawMessage) (string, error)
}

var registry = map[string]Tool{}

func Register(t Tool) { registry[t.Name] = t }

func Get(name string) (Tool, bool) {
	t, ok := registry[name]
	return t, ok
}

func Specs() []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, t := range registry {
		schema := t.Schema
		if schema == nil {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out = append(out, llm.ToolSpec{
			Type: "function",
			Function: llm.ToolFunction{
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
