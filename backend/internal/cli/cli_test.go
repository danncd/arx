package cli

import (
	"testing"

	"arx/internal/agent"
	"arx/internal/llm"
)

func testModels() []llm.ModelInfo {
	return []llm.ModelInfo{
		{Spec: "deepseek/deepseek-v4-flash", Provider: "deepseek", Model: "deepseek-v4-flash", Tools: true, ToolsKnown: true},
		{Spec: "openai/gpt-5-nano", Provider: "openai", Model: "gpt-5-nano"},
	}
}

/* Capability flags ride from the catalog onto the parsed profile. */

func TestResolveProfile(t *testing.T) {
	prof, err := resolveProfile("deepseek/deepseek-v4-flash", testModels())
	if err != nil {
		t.Fatalf("resolveProfile: %v", err)
	}
	if !prof.Tools || !prof.ToolsKnown {
		t.Fatalf("capability flags not copied: %+v", prof)
	}

	// A model absent from the catalog parses fine, capabilities unknown.
	prof, err = resolveProfile("deepseek/uncatalogued", testModels())
	if err != nil || prof.Tools || prof.ToolsKnown {
		t.Fatalf("catalog miss must stay unknown: %+v err=%v", prof, err)
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

/* Commands mutate session state and report quit correctly. */

func TestCommands(t *testing.T) {
	prof, _ := resolveProfile("deepseek/deepseek-v4-flash", testModels())
	c := &cli{models: testModels(), prof: prof, ctrl: agent.New(prof, systemPrompt)}

	if c.command("/exit") != true || c.command("/quit") != true {
		t.Fatal("exit commands must quit")
	}
	if c.command("/help") || c.command("/models") || c.command("/nonsense") {
		t.Fatal("non-exit commands must not quit")
	}

	old := c.ctrl
	if c.command("/clear") {
		t.Fatal("/clear must not quit")
	}
	if c.ctrl == old {
		t.Fatal("/clear must start a fresh conversation")
	}

	if c.command("/model openai/gpt-5-nano") {
		t.Fatal("/model must not quit")
	}
	if c.prof.Provider.Name != "openai" || c.prof.Model != "gpt-5-nano" {
		t.Fatalf("model switch did not stick: %+v", c.prof)
	}

	// A bad spec leaves the session untouched.
	before := c.prof
	c.command("/model garbage")
	if c.prof != before {
		t.Fatalf("failed switch mutated the profile: %+v", c.prof)
	}
}
