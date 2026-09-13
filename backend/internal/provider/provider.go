package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type Options struct {
	MaxTokens int
	Thinking  *bool
}

type Dialect interface {
	Chat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options) (Message, error)
	Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options, onToken func(s string, thinking bool)) (Message, error)
}

type Provider struct {
	Name    string
	BaseURL string
	KeyEnv  string
	Dialect Dialect
}

var Providers = map[string]Provider{
	"deepseek": {
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com",
		KeyEnv:  "DEEPSEEK_API_KEY",
		Dialect: OpenAIDialect{ThinkingParam: true},
	},
	"openai": {
		Name:    "openai",
		BaseURL: "https://api.openai.com/v1",
		KeyEnv:  "OPENAI_API_KEY",
		Dialect: OpenAIDialect{NewTokenParam: true},
	},
}

type Profile struct {
	Provider   Provider
	Model      string
	MaxTokens  int
	Tools      bool
	ToolsKnown bool
}

func Parse(spec string) (Profile, error) {
	prov, model, ok := strings.Cut(strings.TrimSpace(spec), "/")
	model = strings.TrimSpace(model)
	if !ok || model == "" {
		return Profile{}, fmt.Errorf("model spec %q: want provider/model", spec)
	}
	p, err := GetProvider(prov)
	if err != nil {
		return Profile{}, err
	}
	return Profile{Provider: p, Model: model, MaxTokens: 8192}, nil
}

func GetProvider(name string) (Provider, error) {
	p, ok := Providers[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", name)
	}
	return p, nil
}

func (p Provider) Key() string {
	if p.KeyEnv == "" {
		return ""
	}
	return os.Getenv(p.KeyEnv)
}

func Chat(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options) (Message, error) {
	if prof.Provider.Dialect == nil {
		return Message{}, fmt.Errorf("provider %q has no dialect", prof.Provider.Name)
	}
	return prof.Provider.Dialect.Chat(ctx, prof, msgs, tools, opts)
}

func Stream(ctx context.Context, prof Profile, msgs []Message, tools []ToolSpec, opts Options, onToken func(s string, thinking bool)) (Message, error) {
	if prof.Provider.Dialect == nil {
		return Message{}, fmt.Errorf("provider %q has no dialect", prof.Provider.Name)
	}
	return prof.Provider.Dialect.Stream(ctx, prof, msgs, tools, opts, onToken)
}
