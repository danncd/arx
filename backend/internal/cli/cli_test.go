package cli

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"arx/internal/agent"
	"arx/internal/tool"
)

func TestBashIsRegistered(t *testing.T) {
	registerTools()
	if _, ok := tool.Get(tool.Bash.Name); !ok {
		t.Fatal("bash was not registered")
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

func openCard(t *testing.T) (model, approvalMsg) {
	t.Helper()
	m := newModel(nil, "test")
	m.tall = 30
	ask := approvalMsg{
		req:  agent.Request{Tool: "bash", Args: `{"command":"make"}`},
		resp: make(chan agent.Outcome, 1),
	}
	updated, _ := m.Update(ask)
	m = updated.(model)
	if m.ask.pending == nil {
		t.Fatal("card did not open")
	}
	m.ask.opened = time.Time{}
	return m, ask
}

func TestApprovalCardShortcutResolves(t *testing.T) {
	m, ask := openCard(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(model)
	if m.ask.pending != nil {
		t.Fatal("card did not close")
	}
	select {
	case out := <-ask.resp:
		if out != agent.AlwaysSession {
			t.Fatalf("outcome = %v, want AlwaysSession", out)
		}
	default:
		t.Fatal("no outcome delivered")
	}
}

func TestApprovalCardArrowsAndEnter(t *testing.T) {
	m, ask := openCard(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.ask.sel != 2 {
		t.Fatalf("selection did not move: %d", m.ask.sel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(model)
	if m.ask.sel != 2 {
		t.Fatalf("selection did not clamp: %d", m.ask.sel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if m.ask.pending != nil || m.waiting {
		t.Fatal("enter did not resolve the card")
	}
	if out := <-ask.resp; out != agent.Reject {
		t.Fatalf("outcome = %v, want Reject", out)
	}
}

func TestRunningRowShowsThenClears(t *testing.T) {
	m := newModel(nil, "test")
	m.vp.Width = 80
	updated, cmd := m.Update(toolStartMsg{name: "bash", args: `{"command":"go test ./..."}`})
	m = updated.(model)
	if m.run.running != "bash" || m.run.hint != "go test ./..." || cmd == nil {
		t.Fatalf("running row not set: running=%q hint=%q cmd=%v", m.run.running, m.run.hint, cmd)
	}
	if !strings.Contains(m.transcript(), "bash") || !strings.Contains(m.transcript(), "go test ./...") {
		t.Fatalf("running row not rendered: %q", m.transcript())
	}
	updated, _ = m.Update(toolMsg{name: "bash", out: "ok", took: time.Second})
	m = updated.(model)
	if m.run.running != "" {
		t.Fatal("running row not cleared on result")
	}
	updated, cmd = m.Update(spinner.TickMsg{})
	m = updated.(model)
	if m.run.spinning || cmd != nil {
		t.Fatalf("spinner did not stop: spinning=%v cmd=%v", m.run.spinning, cmd)
	}
}

func TestApprovalSettleIgnoresEarlyApprove(t *testing.T) {
	m := newModel(nil, "test")
	m.tall = 30
	ask := approvalMsg{req: agent.Request{Tool: "bash", Args: `{"command":"make"}`}, resp: make(chan agent.Outcome, 1)}
	updated, _ := m.Update(ask)
	m = updated.(model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(model)
	if m.ask.pending == nil {
		t.Fatal("early keystroke approved during the settle window")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(model)
	if m.ask.pending != nil {
		t.Fatal("deny did not resolve")
	}
	if out := <-ask.resp; out != agent.Reject {
		t.Fatalf("deny outcome = %v", out)
	}
}

func TestStaleCardClearsOnTurnEnd(t *testing.T) {
	m := newModel(nil, "test")
	ask := approvalMsg{req: agent.Request{Tool: "bash"}, resp: make(chan agent.Outcome, 1)}
	updated, _ := m.Update(ask)
	m = updated.(model)
	updated, _ = m.Update(doneMsg{err: context.Canceled})
	if m = updated.(model); m.ask.pending != nil {
		t.Fatal("dead turn left its card up")
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
