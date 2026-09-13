package discovery

import "testing"

func testModels() []ModelInfo {
	return []ModelInfo{
		{
			Spec: "deepseek/deepseek-v4-flash", Provider: "deepseek",
			Model:     "deepseek-v4-flash",
			MaxOutput: 4096, Tools: true, ToolsKnown: true,
		},
		{Spec: "openai/gpt-5-nano", Provider: "openai", Model: "gpt-5-nano"},
	}
}

func TestResolveProfile(t *testing.T) {
	prof, err := ResolveProfile("deepseek/deepseek-v4-flash", testModels())
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}
	if !prof.Tools || !prof.ToolsKnown {
		t.Fatalf("capability flags not copied: %+v", prof)
	}
	if prof.MaxTokens != 4096 {
		t.Fatalf("MaxTokens not clamped to the catalog ceiling: %+v", prof)
	}

	prof, err = ResolveProfile("deepseek/uncatalogued", testModels())
	if err != nil || prof.Tools || prof.ToolsKnown {
		t.Fatalf("catalog miss must stay unknown: %+v err=%v", prof, err)
	}
	if prof.MaxTokens != 8192 {
		t.Fatalf("catalog miss must keep the default cap: %+v", prof)
	}

	if _, err := ResolveProfile("nope/model", testModels()); err == nil {
		t.Fatal("unknown provider must fail")
	}
}

func TestDefaultSpecHonorsARXModel(t *testing.T) {
	t.Setenv("ARX_MODEL", "openai/gpt-5-nano")
	if got := DefaultSpec(testModels()); got != "openai/gpt-5-nano" {
		t.Fatalf("ARX_MODEL ignored: %q", got)
	}
}

func TestDefaultSpecRequiresCatalogMatch(t *testing.T) {
	t.Setenv("ARX_MODEL", "")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	models := []ModelInfo{{Spec: "openai/gpt-5-nano", Provider: "openai", Model: "gpt-5-nano"}}
	if got := DefaultSpec(models); got != "openai/gpt-5-nano" {
		t.Fatalf("default selected an undiscovered model: %q", got)
	}
}

func TestDefaultSpecEmptyCatalog(t *testing.T) {
	t.Setenv("ARX_MODEL", "")
	if got := DefaultSpec(nil); got != "" {
		t.Fatalf("empty catalog must yield an empty spec, got %q", got)
	}
}
