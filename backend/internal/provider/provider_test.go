package provider

import "testing"

func TestGetProvider(t *testing.T) {
	p, err := GetProvider("deepseek")
	if err != nil {
		t.Fatalf("known provider returned error: %v", err)
	}

	if p.BaseURL != "https://api.deepseek.com" {
		t.Fatalf("wrong provider: %+v", p)
	}

	if _, err := GetProvider("nope"); err == nil {
		t.Fatal("unknown provider must return an error")
	}
}

func TestParse(t *testing.T) {
	p, err := Parse("deepseek/deepseek-v4-flash")
	if err != nil {
		t.Fatalf("valid spec returned error: %v", err)
	}
	if p.Provider.Name != "deepseek" || p.Model != "deepseek-v4-flash" {
		t.Fatalf("wrong profile: %+v", p)
	}

	if _, err := Parse("deepseek-v4-flash"); err == nil {
		t.Fatal("spec without a slash must fail")
	}
	if _, err := Parse("nope/model"); err == nil {
		t.Fatal("unknown provider must fail")
	}

	p, err = Parse("deepseek/org/model")
	if err != nil {
		t.Fatalf("slashed model name: %v", err)
	}
	if p.Model != "org/model" {
		t.Fatalf("model = %q, want org/model", p.Model)
	}

	if _, err := Parse("deepseek/"); err == nil {
		t.Fatal("empty model half must fail")
	}
	if _, err := Parse("deepseek/   "); err == nil {
		t.Fatal("whitespace model half must fail")
	}
	p, err = Parse("  deepseek/deepseek-v4-flash  ")
	if err != nil || p.Model != "deepseek-v4-flash" {
		t.Fatalf("padded spec not trimmed: %+v err=%v", p, err)
	}
}

func TestKey(t *testing.T) {
	t.Setenv("DEEPSEEK_API_KEY", "sk-test-123")
	p, _ := GetProvider("deepseek")
	if got := p.Key(); got != "sk-test-123" {
		t.Fatalf("Key() = %q, want the env value", got)
	}

	t.Setenv("DEEPSEEK_API_KEY", "")
	if got := p.Key(); got != "" {
		t.Fatalf("Key() with empty env = %q, want empty", got)
	}
}
