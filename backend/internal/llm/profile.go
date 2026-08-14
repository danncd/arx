package llm

import (
	"fmt"
	"os"
	"strings"
)

type Dialect string

const (
	OpenAI Dialect = "openai"
)

type Provider struct {
	Name    string
	BaseURL string
	KeyEnv  string
	Dialect Dialect
	// NewTokenParam marks providers that want "max_completion_tokens"
	// (OpenAI's reasoning-era name) instead of the classic "max_tokens".
	NewTokenParam bool
}

var Providers = map[string]Provider{
	"deepseek": {Name: "deepseek", BaseURL: "https://api.deepseek.com", KeyEnv: "DEEPSEEK_API_KEY", Dialect: OpenAI},
	"openai":   {Name: "openai", BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY", Dialect: OpenAI, NewTokenParam: true},
}

type Profile struct {
	Provider  Provider
	Model     string
	MaxTokens int
}

func Parse(spec string) (Profile, error) {
	prov, model, ok := strings.Cut(spec, "/")

	if !ok {
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
