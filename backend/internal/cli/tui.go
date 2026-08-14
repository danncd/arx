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
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
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
	headerBase  = lipgloss.NewStyle()
	headerBrand = headerBase.Bold(true).
			Foreground(lipgloss.Color("#34bf8c"))
	headerDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	userMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	userStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	thinkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	failStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	sepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a"))
	// tool line: gray brackets around a light blue name
	toolBracket = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	toolName    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6fb7e6"))
	// scrollbar: dim track, jade thumb
	sbTrack = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("│")
	sbThumb = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a")).Render("┃")
)

/* Turn events, sent from the agent's goroutine into the update loop. */

type tokenMsg struct {
	text     string
	thinking bool
}
type toolMsg struct {
	name, out string
	took      time.Duration
}
type doneMsg struct{ err error }

/* teaSink forwards turn output to the program as messages. */

type teaSink struct{ p *tea.Program }

func (s teaSink) Token(text string, thinking bool) { s.p.Send(tokenMsg{text, thinking}) }
func (s teaSink) ToolResult(name, out string, took time.Duration) {
	s.p.Send(toolMsg{name, out, took})
}

/* shared carries the program handle; the model is copied by value. */

type shared struct {
	p      *tea.Program
	cancel context.CancelFunc
}

/*
	The transcript is a list of spans: chrome (pre-styled text — user
	lines, thinking, tool runs) passes through verbatim, while markdown
	spans are the model's answers, rendered by glamour. Completed spans
	are baked into a cached string; the answer currently streaming
	stays raw in cur and is re-rendered on every token, so formatting
	appears live. A resize rebuilds the renderer and re-bakes it all
	at the new width.
*/

type spanKind int

const (
	chromeSpan spanKind = iota // pre-styled, passes through verbatim
	mdSpan                     // markdown, rendered by glamour
	thinkSpan                  // raw reasoning, wrapped with the │ gutter
)

type span struct {
	kind spanKind
	text string
}

type model struct {
	ctrl       *agent.Controller
	header     string
	spans      []span
	baked      string // rendered cache of spans
	cur        string // the streaming answer, raw markdown
	curThink   string // the streaming reasoning, raw
	mdr        *glamour.TermRenderer
	waiting    bool
	inThink    bool // a reasoning block is open and needs closing
	thinkIdx   int  // span index of the block's label, -1 when none
	thinkStart time.Time
	width      int
	vp         viewport.Model
	ti         textinput.Model
	sh         *shared
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func newRenderer(width int) *glamour.TermRenderer {
	// WithStandardStyle-derived config, not WithAutoStyle: auto probes
	// the terminal through stdin and the probe races the keyboard,
	// eating keystrokes.
	cfg := styles.DarkStyleConfig
	margin := uint(1)
	cfg.Document.Margin = &margin
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(max(width-2, 8)),
	)
	if err != nil {
		return nil
	}
	return r
}

/* Markdown to ANSI; raw text when the renderer is unavailable. */

func (m *model) renderMD(s string) string {
	if m.mdr == nil {
		return s
	}
	out, err := m.mdr.Render(s)
	if err != nil {
		return s
	}
	return strings.Trim(out, "\n") + "\n"
}

/*
	Wraps raw reasoning to the gutter width and prefixes every wrapped
	line with │, so a long thought never escapes the rail.
*/

func (m *model) renderThink(s string) string {
	wrapped := lipgloss.NewStyle().Width(max(m.width-4, 8)).Render(s)
	lines := strings.Split(wrapped, "\n")
	for i, ln := range lines {
		lines[i] = thinkStyle.Render(" │ " + ln)
	}
	return strings.Join(lines, "\n")
}

/*
	Sets the viewport from the baked transcript plus the live tails.
	Follow-mode: the view sticks to the bottom only while it is already
	there, so scrolling up during generation holds your place.
*/

func (m *model) setView() {
	follow := m.vp.AtBottom()
	content := m.baked
	if m.curThink != "" {
		content += m.renderThink(m.curThink) + "\n"
	}
	if m.cur != "" {
		content += m.renderMD(m.cur)
	}
	m.vp.SetContent(lipgloss.NewStyle().Width(max(m.vp.Width, 8)).Render(content))
	if follow {
		m.vp.GotoBottom()
	}
}

/* Appends pre-styled chrome, coalescing with a previous chrome span. */

func (m *model) push(s string) {
	if n := len(m.spans); n > 0 && m.spans[n-1].kind == chromeSpan {
		m.spans[n-1].text += s
	} else {
		m.spans = append(m.spans, span{text: s})
	}
	m.baked += s
	m.setView()
}

/* Bakes the streamed answer into the transcript as a markdown span. */

func (m *model) finalizeCur() {
	if m.cur == "" {
		return
	}
	m.spans = append(m.spans, span{kind: mdSpan, text: m.cur})
	m.baked += m.renderMD(m.cur)
	m.cur = ""
	m.setView()
}

/* Re-renders every span; called when the width changes. */

func (m *model) rebake() {
	m.baked = ""
	for _, sp := range m.spans {
		switch sp.kind {
		case mdSpan:
			m.baked += m.renderMD(sp.text)
		case thinkSpan:
			m.baked += m.renderThink(sp.text)
		default:
			m.baked += sp.text
		}
	}
	m.setView()
}

/* Compact duration: 0s, 45s, 1m 20s. */

func thinkDuration(d time.Duration) string {
	secs := int(d.Round(time.Second).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm %ds", secs/60, secs%60)
}

/* Closes an open reasoning block so the answer starts on its own line. */

func (m *model) endThink() {
	if !m.inThink {
		return
	}
	m.inThink = false
	m.spans = append(m.spans, span{kind: thinkSpan, text: m.curThink})
	m.baked += m.renderThink(m.curThink)
	m.curThink = ""
	m.push("\n\n") // close the block, then a blank row before what follows
	if m.thinkIdx >= 0 && m.thinkIdx < len(m.spans) {
		m.spans[m.thinkIdx].text = thinkStyle.Render(" Thought for "+thinkDuration(time.Since(m.thinkStart))) + "\n"
		m.thinkIdx = -1
	}
	m.rebake()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.vp.Width = max(msg.Width-1, 1)   // the right column belongs to the scrollbar
		m.vp.Height = max(msg.Height-5, 0) // padding, header, separators, input, padding
		m.ti.Width = max(msg.Width-5, 8)
		m.mdr = newRenderer(m.vp.Width)
		m.rebake() // re-render the transcript for the new width

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
			m.push(userMark.Render("» ") + userStyle.Render(text) + "\n\n")
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
			if !m.inThink {
				// Open the block: a label span of its own (so endThink
				// can rewrite it); the body accumulates raw in curThink
				// and is wrapped live by renderThink.
				m.inThink = true
				m.thinkStart = time.Now()
				label := thinkStyle.Render(" Thinking…") + "\n"
				m.spans = append(m.spans, span{text: label})
				m.thinkIdx = len(m.spans) - 1
				m.baked += label
			}
			m.curThink += msg.text
			m.setView()
		} else {
			m.endThink()
			m.cur += msg.text // raw markdown, rendered live by setView
			m.setView()
		}

	case toolMsg:
		m.endThink()
		m.finalizeCur() // any answer text before the call bakes first
		m.push(toolBracket.Render(" 〔 ") + toolName.Render(msg.name) +
			toolBracket.Render(" 〕") + thinkDuration(msg.took) + "\n\n")

	case doneMsg:
		m.waiting = false
		m.sh.cancel = nil
		m.endThink()
		m.finalizeCur()
		switch {
		case errors.Is(msg.err, agent.ErrStepLimit):
			m.push(failStyle.Render("arx: step limit reached before a final answer") + "\n\n")
		case msg.err != nil:
			m.push(failStyle.Render("arx: "+msg.err.Error()) + "\n\n")
		default:
			m.push("\n") // rendered markdown ends its own line; add the blank row
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

/*
	A vertical scrollbar for the viewport: dim track, jade thumb sized
	and placed from the scroll state. A blank column when everything
	already fits, so the layout never shifts.
*/

func (m model) scrollbar() string {
	h := m.vp.Height
	total := m.vp.TotalLineCount()
	var b strings.Builder
	if h <= 0 || total <= h {
		for i := 0; i < h; i++ {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteByte(' ')
		}
		return b.String()
	}
	thumb := max(h*h/total, 1)
	pos := int(m.vp.ScrollPercent()*float64(h-thumb) + 0.5)
	for i := 0; i < h; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i >= pos && i < pos+thumb {
			b.WriteString(sbThumb)
		} else {
			b.WriteString(sbTrack)
		}
	}
	return b.String()
}

func (m model) View() string {
	brand := headerBrand.Render("〔 Arx 〕")
	rest := headerDim.Render("· " + m.header + " ")
	header := brand + rest
	sep := sepStyle.Render(strings.Repeat("─", max(m.width, 8)))
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.vp.View(), m.scrollbar())
	return "\n" +
		header + "\n" +
		sep + "\n" +
		body + "\n" +
		sep + "\n" +
		m.ti.View() + "\n"
}

/* Runs the terminal UI; returns when the user leaves. */

func runTUI(ctrl *agent.Controller, prof llm.Profile) error {
	ti := textinput.New()
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34bf8c"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	ti.Focus()

	m := model{
		ctrl:     ctrl,
		header:   prof.Model + " · ctrl-c to leave",
		spans:    []span{{text: "\n"}}, // breathing room above the first message
		baked:    "\n",
		thinkIdx: -1,
		vp:       viewport.New(80, 22),
		ti:       ti,
		sh:       &shared{},
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.sh.p = p

	final, err := p.Run()
	if err != nil {
		return err
	}
	// The alt screen vanishes on exit; leave the conversation behind
	// in the real terminal, like the inline version did.
	if fm, ok := final.(model); ok {
		out := fm.baked
		if fm.cur != "" {
			out += fm.renderMD(fm.cur)
		}
		if out != "" {
			fmt.Print(out)
		}
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
