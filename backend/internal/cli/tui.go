/*
	Bubble Tea frontend: a fixed Arx header, the transcript in a
	viewport that re-flows on resize, and the input bar pinned at the
	bottom. The framework owns the screen buffer, which is exactly the
	part hand-rolling could not do: resize repaints everything from
	state. The dependency stays inside this package; the agent only
	ever sees the Sink interface.
*/

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"arx/internal/agent"
	"arx/internal/llm"
)

/* True when stdout is a real terminal, not a pipe or file. */

func isTerminal() bool {
	st, err := os.Stdout.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

var (
	headerBase = lipgloss.NewStyle().
			Background(lipgloss.Color("#2ea77a")). // Arx jade
			Foreground(lipgloss.Color("#121212"))
	headerBrand = headerBase.Bold(true).
			Foreground(lipgloss.Color("#f0f0f0"))
	userStyle  = lipgloss.NewStyle().Bold(true)
	thinkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	failStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	sepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a"))
)

/* Turn events, sent from the agent's goroutine into the update loop. */

type tokenMsg struct {
	text     string
	thinking bool
}
type toolMsg struct{ name, out string }
type doneMsg struct{ err error }

/* teaSink forwards turn output to the program as messages. */

type teaSink struct{ p *tea.Program }

func (s teaSink) Token(text string, thinking bool) { s.p.Send(tokenMsg{text, thinking}) }
func (s teaSink) ToolResult(name, out string)      { s.p.Send(toolMsg{name, out}) }

/* shared carries the program handle; the model is copied by value. */

type shared struct {
	p      *tea.Program
	cancel context.CancelFunc
}

type model struct {
	ctrl    *agent.Controller
	header  string
	raw     string // the transcript with styles, re-wrapped on resize
	waiting bool
	width   int
	vp      viewport.Model
	ti      textinput.Model
	sh      *shared
}

func (m model) Init() tea.Cmd { return textinput.Blink }

/*
	Appends to the transcript. Follow-mode: the view sticks to the tail
	only while it is already there, so scrolling up during generation
	holds your place; returning to the bottom re-engages the follow.
*/

func (m *model) push(s string) {
	follow := m.vp.AtBottom()
	m.raw += s
	m.vp.SetContent(lipgloss.NewStyle().Width(max(m.width, 8)).Render(m.raw))
	if follow {
		m.vp.GotoBottom()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.vp.Width = msg.Width
		m.vp.Height = max(msg.Height-5, 0) // padding, header, separators, input, padding
		m.ti.Width = max(msg.Width-5, 8)
		m.push("") // re-wrap the transcript for the new width

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			if m.sh.cancel != nil {
				m.sh.cancel()
			}
			return m, tea.Quit
		case tea.KeyEnter:
			text := strings.TrimSpace(m.ti.Value())
			if text == "" || m.waiting {
				return m, nil
			}
			m.ti.Reset()
			m.waiting = true
			m.push(userStyle.Render("› "+text) + "\n")
			m.vp.GotoBottom() // sending always jumps to the latest
			ctx, cancel := context.WithCancel(context.Background())
			m.sh.cancel = cancel
			ctrl, p := m.ctrl, m.sh.p
			go func() {
				err := ctrl.RunTurn(ctx, text, teaSink{p})
				p.Send(doneMsg{err})
			}()
			return m, nil
		}
		var tiCmd, vpCmd tea.Cmd
		m.ti, tiCmd = m.ti.Update(msg)
		m.vp, vpCmd = m.vp.Update(msg) // pgup/pgdn scroll the transcript
		return m, tea.Batch(tiCmd, vpCmd)

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg) // wheel scrolls the transcript
		return m, cmd

	case tokenMsg:
		if msg.thinking {
			m.push(thinkStyle.Render(msg.text))
		} else {
			m.push(msg.text)
		}

	case toolMsg:
		m.push("  [" + msg.name + "] → " + msg.out + "\n")

	case doneMsg:
		m.waiting = false
		m.sh.cancel = nil
		switch {
		case errors.Is(msg.err, agent.ErrStepLimit):
			m.push(failStyle.Render("arx: step limit reached before a final answer") + "\n")
		case msg.err != nil:
			m.push(failStyle.Render("arx: "+msg.err.Error()) + "\n")
		default:
			m.push("\n")
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m model) View() string {
	brand := headerBrand.Render(" Arx ")
	rest := headerBase.Render("· " + m.header + " ")
	pad := max(m.width-lipgloss.Width(brand+rest), 0)
	header := brand + rest + headerBase.Render(strings.Repeat(" ", pad))
	sep := sepStyle.Render(strings.Repeat("─", max(m.width, 8)))
	return "\n" +
		header + "\n" +
		sep + "\n" +
		m.vp.View() + "\n" +
		sep + "\n" +
		m.ti.View() + "\n"
}

/* Runs the terminal UI; returns when the user leaves. */

func runTUI(ctrl *agent.Controller, prof llm.Profile) error {
	ti := textinput.New()
	ti.Prompt = " > "
	ti.Focus()

	m := model{
		ctrl:   ctrl,
		header: prof.Provider.Name + "/" + prof.Model + " · ctrl-c to leave",
		vp:     viewport.New(80, 22),
		ti:     ti,
		sh:     &shared{},
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.sh.p = p

	final, err := p.Run()
	if err != nil {
		return err
	}
	// The alt screen vanishes on exit; leave the conversation behind
	// in the real terminal, like the inline version did.
	if fm, ok := final.(model); ok && fm.raw != "" {
		fmt.Print(fm.raw)
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
