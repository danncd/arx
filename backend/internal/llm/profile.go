package llm

import (
	"fmt"
	"os"
	"strings"
)

/* Provider wire format. */

type Dialect string

const (
	OpenAI Dialect = "openai"
)

/* Provider endpoints and credentials. */

type Provider struct {
	Name    string
	BaseURL string
	KeyEnv  string
	Dialect Dialect
	// Use max_completion_tokens.
	NewTokenParam bool
}

var Providers = map[string]Provider{
	"deepseek": {Name: "deepseek", BaseURL: "https://api.deepseek.com", KeyEnv: "DEEPSEEK_API_KEY", Dialect: OpenAI},
	"openai":   {Name: "openai", BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY", Dialect: OpenAI, NewTokenParam: true},
}

/* Selected provider and model. */

type Profile struct {
	Provider   Provider
	Model      string
	MaxTokens  int
	Tools      bool
	ToolsKnown bool
}

/* Parses provider/model. */

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

/* Finds a configured provider. */

func GetProvider(name string) (Provider, error) {
	p, ok := Providers[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", name)
	}
	return p, nil
}

/* Reads the provider key. */

func (p Provider) Key() string {
	if p.KeyEnv == "" {
		return ""
	}
	return os.Getenv(p.KeyEnv)
}
