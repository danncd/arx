package cli

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"arx/internal/agent"
	"arx/internal/provider"
)

type tokenMsg struct {
	text     string
	thinking bool
}

type toolStartMsg struct{ name, args string }

type toolMsg struct {
	name, out string
	took      time.Duration
	failed    bool
}

type doneMsg struct{ err error }

type renderMsg struct{}

var toolSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 10,
}

type spanKind int

const (
	chromeSpan spanKind = iota
	mdSpan
	thinkSpan
)

type span struct {
	kind spanKind
	text string
}

type transcriptState struct {
	spans     []span
	baked     string
	cur       string
	afterTool bool
}

type thinkState struct {
	active bool
	raw    string
	idx    int
	start  time.Time
}

type toolState struct {
	running  string
	hint     string
	spinning bool
}

type approvalState struct {
	pending *approvalMsg
	sel     int
	opened  time.Time
}

type model struct {
	ctrl   *agent.Controller
	gate   *agent.Gate
	header string
	width  int
	tall   int
	vp     viewport.Model
	ti     textinput.Model
	mdr    *glamour.TermRenderer
	spin   spinner.Model
	sh     *shared

	tr  transcriptState
	th  thinkState
	run toolState
	ask approvalState

	renderWait bool
	waiting    bool
	quitting   bool
}

type teaSink struct{ p *tea.Program }

func (s teaSink) Token(text string, thinking bool) { s.p.Send(tokenMsg{text, thinking}) }
func (s teaSink) ToolStart(name, args string)      { s.p.Send(toolStartMsg{name, args}) }
func (s teaSink) ToolResult(name, out string, took time.Duration, failed bool) {
	s.p.Send(toolMsg{name, out, took, failed})
}

type shared struct {
	p      *tea.Program
	cancel context.CancelFunc
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) View() string {
	brand := headerBrand.Render("〔 Arx 〕")
	line := m.header + " · ctrl-c to leave"
	rest := ""
	if width := m.width - lipgloss.Width(brand); width > 0 {
		rest = headerDim.MaxWidth(width).Render("· " + line + " ")
	}
	sep := sepStyle.Render(strings.Repeat("─", max(m.width, 8)))
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.vp.View(), m.scrollbar())
	bottom := m.ti.View()
	if m.ask.pending != nil {
		bottom = m.approvalCard()
	}
	return brand + rest + "\n" +
		sep + "\n" +
		body + "\n" +
		sep + "\n" +
		bottom + "\n"
}

func newModel(ctrl *agent.Controller, header string) model {
	ti := textinput.New()
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34bf8c"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	ti.Focus()

	vp := viewport.New(80, 22)
	vp.KeyMap.Up.SetKeys("up")
	vp.KeyMap.Down.SetKeys("down")
	vp.KeyMap.PageUp.SetKeys("pgup")
	vp.KeyMap.PageDown.SetKeys("pgdown")
	vp.KeyMap.HalfPageUp.SetEnabled(false)
	vp.KeyMap.HalfPageDown.SetEnabled(false)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	return model{
		ctrl:   ctrl,
		header: terminalLine(header),
		vp:     vp,
		ti:     ti,
		spin:   spinner.New(spinner.WithSpinner(toolSpinner)),
		sh:     &shared{},
		tr:     transcriptState{spans: []span{{text: "\n"}}, baked: "\n"},
		th:     thinkState{idx: -1},
	}
}

func runTUI(ctrl *agent.Controller, prof provider.Profile, judge agent.Judge) error {
	m := newModel(ctrl, prof.Model)
	gate := agent.NewGate(tuiApprover{sh: m.sh}, judge)
	m.gate = gate
	ctrl.SetGate(gate)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.sh.p = p

	_, err := p.Run()
	return err
}
