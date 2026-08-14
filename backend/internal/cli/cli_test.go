package cli

import (
	"testing"

	"arx/internal/llm"
)

func testModels() []llm.ModelInfo {
	return []llm.ModelInfo{
		{
			Spec: "deepseek/deepseek-v4-flash", Provider: "deepseek",
			Model: "deepseek-v4-flash", ContextWindow: 128000,
			MaxOutput: 4096, Reasoning: true, Tools: true, ToolsKnown: true,
		},
		{Spec: "openai/gpt-5-nano", Provider: "openai", Model: "gpt-5-nano"},
	}
}

/* Copies model capabilities and adopts the completion ceiling. */

func TestResolveProfile(t *testing.T) {
	prof, err := resolveProfile("deepseek/deepseek-v4-flash", testModels())
	if err != nil {
		t.Fatalf("resolveProfile: %v", err)
	}
	if !prof.Tools || !prof.ToolsKnown {
		t.Fatalf("capability flags not copied: %+v", prof)
	}
	if prof.MaxTokens != 4096 {
		t.Fatalf("MaxTokens not clamped to the catalog ceiling: %+v", prof)
	}

	// Catalog misses keep capabilities unknown and the default cap.
	prof, err = resolveProfile("deepseek/uncatalogued", testModels())
	if err != nil || prof.Tools || prof.ToolsKnown {
		t.Fatalf("catalog miss must stay unknown: %+v err=%v", prof, err)
	}
	if prof.MaxTokens != 8192 {
		t.Fatalf("catalog miss must keep the default cap: %+v", prof)
	}

	if _, err := resolveProfile("nope/model", testModels()); err == nil {
		t.Fatal("unknown provider must fail")
	}
}

func TestDefaultSpecHonorsARXModel(t *testing.T) {
	t.Setenv("ARX_MODEL", "openai/gpt-5-nano")
	if got := defaultSpec(testModels()); got != "openai/gpt-5-nano" {
		t.Fatalf("ARX_MODEL ignored: %q", got)
	}
}
