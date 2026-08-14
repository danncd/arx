package tool

import (
	"context"
	"encoding/json"

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
		out = append(out, llm.ToolSpec{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Schema,
			},
		})
	}
	return out
}
