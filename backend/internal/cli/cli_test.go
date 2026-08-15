package cli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"arx/internal/llm"
	"arx/internal/tool"
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

func TestBashIsRegistered(t *testing.T) {
	registerTools()
	if _, ok := tool.Get(tool.Bash.Name); !ok {
		t.Fatal("bash was not registered")
	}
}

func TestDefaultSpecRequiresCatalogMatch(t *testing.T) {
	t.Setenv("ARX_MODEL", "")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	models := []llm.ModelInfo{{Spec: "openai/gpt-5-nano", Provider: "openai", Model: "gpt-5-nano"}}
	if got := defaultSpec(models); got != "openai/gpt-5-nano" {
		t.Fatalf("default selected an undiscovered model: %q", got)
	}
}

func TestIsTerminalRejectsNullDevice(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isTerminal(f) {
		t.Fatal("null device was treated as a terminal")
	}
}

func TestTerminalTextDropsControls(t *testing.T) {
	in := "a\x1b]52;c;secret\a\rb\x7fc\u0085d\n\t"
	if got, want := terminalText(in), "a]52;c;secretbcd\n\t"; got != want {
		t.Fatalf("terminalText = %q, want %q", got, want)
	}
}

func TestTerminalLineCannotBreakBanner(t *testing.T) {
	in := "model\x1b]52;c;secret\a\nnext\tpart"
	if got, want := terminalLine(in), "model]52;c;secret next part"; got != want {
		t.Fatalf("terminalLine = %q, want %q", got, want)
	}
}

func TestFixHeadingsRespectsFenceDelimiter(t *testing.T) {
	in := "````go\n##inside\n```\n##still-inside\n~~~~\n##also-inside\n````\n##outside"
	want := "````go\n##inside\n```\n##still-inside\n~~~~\n##also-inside\n````\n## outside"
	if got := fixHeadings(in); got != want {
		t.Fatalf("fixHeadings = %q, want %q", got, want)
	}
}

func TestFixHeadingsRejectsBacktickInInfo(t *testing.T) {
	in := "```go`bad\n##heading"
	if got, want := fixHeadings(in), "```go`bad\n## heading"; got != want {
		t.Fatalf("fixHeadings = %q, want %q", got, want)
	}
}

func TestToolLineUsesExplicitFailure(t *testing.T) {
	m := newModel(nil, "test")
	m.vp.Width = 80
	ok := m.toolLine("test", "error: valid output", time.Second, false)
	if !strings.Contains(ok, "✓") || !strings.Contains(ok, "error: valid output") || strings.Contains(ok, "✗") {
		t.Fatalf("successful output misclassified: %q", ok)
	}
	failed := m.toolLine("test", "error: boom", time.Second, true)
	if !strings.Contains(failed, "✗") || strings.Contains(failed, "error: boom") {
		t.Fatalf("failed output misclassified: %q", failed)
	}
	if line := m.toolLine("test", "one\ttwo", time.Second, false); strings.Contains(line, "\t") {
		t.Fatalf("tab escaped the tool row: %q", line)
	}
}

func TestPlainSinkKeepsOnlyFinalToolStep(t *testing.T) {
	sink := &terminalSink{}
	sink.Token("Checking", false)
	sink.Token("private", true)
	sink.ToolResult("test", "result", 0, false)
	sink.Token("Done", false)
	if got := sink.answer.String(); got != "Done" {
		t.Fatalf("buffered answer = %q, want Done", got)
	}
}

func TestTUIWaitsForCanceledTurn(t *testing.T) {
	m := newModel(nil, "test")
	m.waiting = true
	canceled := false
	m.sh.cancel = func() { canceled = true }
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(model)
	if !canceled || !m.quitting || cmd != nil {
		t.Fatalf("first quit did not wait: canceled=%v quitting=%v cmd=%v", canceled, m.quitting, cmd)
	}
	_, cmd = m.Update(doneMsg{err: context.Canceled})
	if cmd == nil {
		t.Fatal("completed canceled turn did not quit")
	}
}
