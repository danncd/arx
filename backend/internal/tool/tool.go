package tool

import (
	"context"
	"encoding/json"
	"sort"

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

// Specs renders the registry in the shape the chat request wants,
// sorted by name: map iteration is randomized per range, and a
// reshuffled tool list changes the request prefix every turn, breaking
// provider prompt caching and permuting the list mid-conversation.
func Specs() []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, t := range registry {
		schema := t.Schema
		if schema == nil {
			// A nil RawMessage marshals to JSON null, which providers
			// reject; "no arguments" must be an empty object schema.
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
