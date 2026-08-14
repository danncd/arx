package llm

import (
	"fmt"
	"os"
	"strings"
)

/*
	Dialect: Way to parse data from LLMs
*/

type Dialect string

const (
	OpenAI Dialect = "openai"
)

/*
	Provider data structure and hardcoded list of providers and their information
*/

type Provider struct {
	Name    string
	BaseURL string
	KeyEnv  string
	Dialect Dialect
	// OpenAI's reasoning quirk: "max_completion_tokens"
	NewTokenParam bool
}

var Providers = map[string]Provider{
	"deepseek": {Name: "deepseek", BaseURL: "https://api.deepseek.com", KeyEnv: "DEEPSEEK_API_KEY", Dialect: OpenAI},
	"openai":   {Name: "openai", BaseURL: "https://api.openai.com/v1", KeyEnv: "OPENAI_API_KEY", Dialect: OpenAI, NewTokenParam: true},
}

/*
	Profile data structure, contains the provider, model and max tokens
*/

type Profile struct {
	Provider  Provider
	Model     string
	MaxTokens int
}

/*
	Parse takes a string like "deepseek/deepseek-v4-flash", and returns the provider, model, and added max tokens
*/

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

/*
	Checks that provider exists
*/

func GetProvider(name string) (Provider, error) {
	p, ok := Providers[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q", name)
	}
	return p, nil
}

/*
	Returns the key of input provider
*/

func (p Provider) Key() string {
	if p.KeyEnv == "" {
		return ""
	}
	return os.Getenv(p.KeyEnv)
}
