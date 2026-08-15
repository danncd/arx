package cli

import (
	"context"
	"errors"
	"fmt"
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

/*
	Lipgloss styles for the full screen frontend
*/

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
	// Tool call.
	toolBracket = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	toolName    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6fb7e6"))
	toolOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	toolFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	// Scrollbar.
	sbTrack = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("│")
	sbThumb = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a")).Render("┃")
)

/*
	Agent events sent into the update loop
*/

type tokenMsg struct {
	text     string
	thinking bool
}
type toolMsg struct {
	name, out string
	took      time.Duration
	failed    bool
}
type doneMsg struct{ err error }
type renderMsg struct{}

/*
	Sends turn output to the update loop
*/

type teaSink struct{ p *tea.Program }

func (s teaSink) Token(text string, thinking bool) { s.p.Send(tokenMsg{text, thinking}) }
func (s teaSink) ToolResult(name, out string, took time.Duration, failed bool) {
	s.p.Send(toolMsg{name, out, took, failed})
}

/*
	State shared across model copies
*/

type shared struct {
	p      *tea.Program
	cancel context.CancelFunc
}

/*
	Transcript and input state; spans remember what was said
	so the transcript can rewrap on resize
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
	quitting   bool
	afterTool  bool // last baked row is a tool line
	inThink    bool
	thinkIdx   int
	thinkStart time.Time
	renderWait bool
	width      int
	vp         viewport.Model
	ti         textinput.Model
	sh         *shared
}

/*
	Builds the markdown renderer for a given width
*/

func newRenderer(width int) *glamour.TermRenderer {
	// Auto style probes stdin and can eat keystrokes.
	cfg := styles.DarkStyleConfig
	margin := uint(1)
	cfg.Document.Margin = &margin
	// Copy Chroma before clearing the fenced block background.
	chroma := *cfg.CodeBlock.Chroma
	chroma.Background.BackgroundColor = nil
	// Treat lexer errors as text before glamour caches the palette.
	chroma.Error = chroma.Text
	cfg.CodeBlock.Chroma = &chroma
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(max(width-2, 8)),
	)
	if err != nil {
		return nil
	}
	return r
}

/*
	Models often emit ##Heading without the space CommonMark
	requires, which renders as literal hashes. Give line-start
	runs of two or more their space; a single # stays untouched
	since #1 or #hashtag prose is too easy to mistake, and
	fenced code passes through as written
*/

func fixHeadings(s string) string {
	lines := strings.Split(s, "\n")
	var fence byte
	fenceLen := 0
	for i, ln := range lines {
		mark, run, rest, ok := fenceLine(ln)
		if fence != 0 {
			if ok && mark == fence && run >= fenceLen && strings.TrimSpace(rest) == "" {
				fence, fenceLen = 0, 0
			}
			continue
		}
		if ok {
			fence, fenceLen = mark, run
			continue
		}
		indent := 0
		for indent < len(ln) && indent < 3 && ln[indent] == ' ' {
			indent++
		}
		if indent < len(ln) && (ln[indent] == ' ' || ln[indent] == '\t') {
			continue
		}
		t := ln[indent:]
		hashes := 0
		for hashes < len(t) && t[hashes] == '#' {
			hashes++
		}
		if hashes >= 2 && hashes <= 6 && hashes < len(t) && t[hashes] != ' ' {
			lines[i] = ln[:indent] + t[:hashes] + " " + t[hashes:]
		}
	}
	return strings.Join(lines, "\n")
}

func fenceLine(line string) (byte, int, string, bool) {
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || line[i] != '`' && line[i] != '~' {
		return 0, 0, "", false
	}
	mark := line[i]
	j := i
	for j < len(line) && line[j] == mark {
		j++
	}
	if j-i < 3 {
		return 0, 0, "", false
	}
	rest := line[j:]
	if mark == '`' && strings.Contains(rest, "`") {
		return 0, 0, "", false
	}
	return mark, j - i, rest, true
}

/*
	Renders Markdown, with raw text as fallback
*/

func (m *model) renderMD(s string) string {
	if m.mdr == nil {
		return s
	}
	out, err := m.mdr.Render(fixHeadings(s))
	if err != nil {
		return s
	}
	return strings.Trim(out, "\n") + "\n"
}

/*
	Wraps reasoning inside its gutter
*/

func (m *model) renderThink(s string) string {
	// Leave three cells for the gutter.
	wrapped := lipgloss.NewStyle().Width(max(m.vp.Width-3, 8)).Render(s)
	lines := strings.Split(wrapped, "\n")
	for i, ln := range lines {
		lines[i] = thinkStyle.Render(" │ " + ln)
	}
	return strings.Join(lines, "\n")
}

/*
	Transcript with any in-flight reasoning or answer appended
*/

func (m *model) transcript() string {
	out := m.baked
	if m.curThink != "" {
		out += m.renderThink(m.curThink) + "\n"
	}
	if m.cur != "" {
		out += m.renderMD(m.cur)
	}
	return out
}

/*
	Refreshes the viewport without stealing a manual scroll
*/

func (m *model) setView() {
	follow := m.vp.AtBottom()
	m.vp.SetContent(lipgloss.NewStyle().Width(max(m.vp.Width, 8)).Render(m.transcript()))
	if follow {
		m.vp.GotoBottom()
	}
}

/*
	Appends styled terminal text
*/

func (m *model) push(s string) {
	if n := len(m.spans); n > 0 && m.spans[n-1].kind == chromeSpan {
		m.spans[n-1].text += s
	} else {
		m.spans = append(m.spans, span{text: s})
	}
	m.baked += s
	m.setView()
}

/*
	Bakes the current answer into the transcript
*/

func (m *model) finalizeCur() {
	if m.cur == "" {
		return
	}
	m.spans = append(m.spans, span{kind: mdSpan, text: m.cur})
	m.baked += m.renderMD(m.cur)
	m.cur = ""
	m.setView()
}

/*
	Re-renders the transcript at its current width
*/

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

/*
	Formats a compact duration
*/

func thinkDuration(d time.Duration) string {
	secs := int(d.Round(time.Second).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm %ds", secs/60, secs%60)
}

/*
	Formats a tool call as one row: status mark, name, the first
	line of output, then a rule pushing the duration to the edge
*/

func (m *model) toolLine(name, out string, took time.Duration, failed bool) string {
	mark := toolOK.Render("✓")
	if failed {
		mark = toolFail.Render("✗")
		out = strings.TrimPrefix(out, "error: ")
	}
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		out = out[:i]
	}
	out = strings.ReplaceAll(out, "\t", " ")
	head := toolBracket.Render(" 〔 ") + mark + toolBracket.Render(" 〕") + toolName.Render(name)
	dur := thinkDuration(took)
	if out = clipCell(strings.TrimSpace(out), m.vp.Width-lipgloss.Width(head)-lipgloss.Width(dur)-8); out != "" {
		head += toolBracket.Render(" → ") + out
	}
	fill := max(m.vp.Width-lipgloss.Width(head)-lipgloss.Width(dur)-2, 3)
	return head + " " + toolBracket.Render(strings.Repeat("─", fill)) + " " + dur
}

/*
	Cuts text to a cell budget, marking the cut with an ellipsis
*/

func clipCell(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width < 2 {
		return ""
	}
	cells := 0
	var b strings.Builder
	for _, r := range s {
		w := lipgloss.Width(string(r))
		if cells+w > width-1 {
			break
		}
		b.WriteRune(r)
		cells += w
	}
	return b.String() + "…"
}

/*
	Ends a run of stacked tool lines with its blank row
*/

func (m *model) closeToolRun() {
	if m.afterTool {
		m.afterTool = false
		m.push("\n")
	}
}

/*
	Closes the current reasoning block and stamps how long it took
*/

func (m *model) endThink() {
	if !m.inThink {
		return
	}
	m.inThink = false
	m.spans = append(m.spans, span{kind: thinkSpan, text: m.curThink})
	m.curThink = ""
	m.spans = append(m.spans, span{text: "\n\n"})
	if m.thinkIdx >= 0 && m.thinkIdx < len(m.spans) {
		m.spans[m.thinkIdx].text = thinkStyle.Render(" Thought for "+thinkDuration(time.Since(m.thinkStart))) + "\n"
		m.thinkIdx = -1
	}
	m.rebake()
}

/*
	Schedules a repaint, at most 30 a second
*/

func (m *model) queueRender() tea.Cmd {
	if m.renderWait {
		return nil
	}
	m.renderWait = true
	return tea.Tick(time.Second/30, func(time.Time) tea.Msg { return renderMsg{} })
}

func (m model) Init() tea.Cmd { return textinput.Blink }

/*
	Update handles one message: keys, resizes, agent events
*/

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.vp.Width = max(msg.Width-2, 1)   // Gap and scrollbar.
		m.vp.Height = max(msg.Height-5, 0) // Static rows around the transcript.
		m.ti.Width = max(msg.Width-5, 8)
		m.mdr = newRenderer(m.vp.Width)
		m.rebake()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			if m.quitting {
				return m, tea.Quit
			}
			if m.sh.cancel != nil {
				m.quitting = true
				m.sh.cancel()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyEnter:
			text := strings.TrimSpace(m.ti.Value())
			if text == "" || m.waiting {
				return m, nil
			}
			m.ti.Reset()
			m.waiting = true
			m.push(userMark.Render("» ") + userStyle.Render(terminalText(text)) + "\n\n")
			m.vp.GotoBottom()
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
		m.vp, vpCmd = m.vp.Update(msg)
		return m, tea.Batch(tiCmd, vpCmd)

	case tokenMsg:
		msg.text = terminalText(msg.text)
		if msg.thinking {
			if !m.inThink {
				m.finalizeCur()
				m.closeToolRun()
				m.inThink = true
				m.thinkStart = time.Now()
				label := thinkStyle.Render(" Thinking…") + "\n"
				m.spans = append(m.spans, span{text: label})
				m.thinkIdx = len(m.spans) - 1
				m.baked += label
			}
			m.curThink += msg.text
		} else {
			m.endThink()
			m.closeToolRun()
			m.cur += msg.text
		}
		return m, m.queueRender()

	case renderMsg:
		m.renderWait = false
		m.setView()
		return m, nil

	case toolMsg:
		m.endThink()
		hadAnswer := m.cur != ""
		m.finalizeCur()
		if hadAnswer {
			m.push("\n") // blank row between the answer text and the tool run
		}
		m.push(m.toolLine(terminalLine(msg.name), terminalText(msg.out), msg.took, msg.failed) + "\n")
		m.afterTool = true

	case doneMsg:
		m.waiting = false
		m.sh.cancel = nil
		if m.quitting {
			return m, tea.Quit
		}
		m.endThink()
		m.finalizeCur()
		m.closeToolRun()
		switch {
		case errors.Is(msg.err, agent.ErrStepLimit):
			m.push(failStyle.Render("arx: step limit reached before a final answer") + "\n\n")
		case msg.err != nil:
			m.push(failStyle.Render("arx: "+terminalText(msg.err.Error())) + "\n\n")
		default:
			m.push("\n") // rendered markdown ends its own line; add the blank row
		}
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

/*
	Renders the transcript scrollbar
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
			b.WriteString("  ")
		}
		return b.String()
	}
	thumb := max(h*h/total, 1)
	pos := int(m.vp.ScrollPercent()*float64(h-thumb) + 0.5)
	for i := 0; i < h; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteByte(' ')
		if i >= pos && i < pos+thumb {
			b.WriteString(sbThumb)
		} else {
			b.WriteString(sbTrack)
		}
	}
	return b.String()
}

/*
	View draws the header, the transcript with its scrollbar,
	and the input line
*/

func (m model) View() string {
	brand := headerBrand.Render("〔 Arx 〕")
	rest := ""
	if width := m.width - lipgloss.Width(brand); width > 0 {
		rest = headerDim.MaxWidth(width).Render("· " + m.header + " ")
	}
	header := brand + rest
	sep := sepStyle.Render(strings.Repeat("─", max(m.width, 8)))
	body := lipgloss.JoinHorizontal(lipgloss.Top, m.vp.View(), m.scrollbar())
	return header + "\n" +
		sep + "\n" +
		body + "\n" +
		sep + "\n" +
		m.ti.View() + "\n"
}

/*
	Builds the initial model; the program handle lands in sh afterwards
*/

func newModel(ctrl *agent.Controller, header string) model {
	ti := textinput.New()
	ti.Prompt = "» "
	ti.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#34bf8c"))
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	ti.Focus()

	vp := viewport.New(80, 22)
	// Keep scrolling on arrow and page keys only.
	vp.KeyMap.Up.SetKeys("up")
	vp.KeyMap.Down.SetKeys("down")
	vp.KeyMap.PageUp.SetKeys("pgup")
	vp.KeyMap.PageDown.SetKeys("pgdown")
	vp.KeyMap.HalfPageUp.SetEnabled(false)
	vp.KeyMap.HalfPageDown.SetEnabled(false)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	return model{
		ctrl:     ctrl,
		header:   terminalLine(header),
		spans:    []span{{text: "\n"}},
		baked:    "\n",
		thinkIdx: -1,
		vp:       vp,
		ti:       ti,
		sh:       &shared{},
	}
}

/*
	Runs the full screen frontend; returns when the user leaves
*/

func runTUI(ctrl *agent.Controller, prof llm.Profile) error {
	m := newModel(ctrl, prof.Model+" · ctrl-c to leave")
	// Leave mouse selection to the terminal.
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.sh.p = p

	_, err := p.Run()
	return err
}
