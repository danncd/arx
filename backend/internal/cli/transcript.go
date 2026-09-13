package cli

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) transcript() string {
	out := m.tr.baked
	if m.th.raw != "" {
		out += m.renderThink(m.th.raw) + "\n"
	}
	if m.tr.cur != "" {
		out += m.renderMD(m.tr.cur)
	}
	if m.run.running != "" {
		out += m.runningRow() + "\n"
	}
	return out
}

func (m *model) setView() {
	follow := m.vp.AtBottom()
	m.vp.SetContent(lipgloss.NewStyle().Width(max(m.vp.Width, 8)).Render(m.transcript()))
	if follow {
		m.vp.GotoBottom()
	}
}

func (m *model) push(s string) {
	if n := len(m.tr.spans); n > 0 && m.tr.spans[n-1].kind == chromeSpan {
		m.tr.spans[n-1].text += s
	} else {
		m.tr.spans = append(m.tr.spans, span{text: s})
	}
	m.tr.baked += s
	m.setView()
}

func (m *model) finalizeCur() {
	if m.tr.cur == "" {
		return
	}
	m.tr.spans = append(m.tr.spans, span{kind: mdSpan, text: m.tr.cur})
	m.tr.baked += m.renderMD(m.tr.cur)
	m.tr.cur = ""
	m.setView()
}

func (m *model) rebake() {
	m.tr.baked = ""
	for _, sp := range m.tr.spans {
		switch sp.kind {
		case mdSpan:
			m.tr.baked += m.renderMD(sp.text)
		case thinkSpan:
			m.tr.baked += m.renderThink(sp.text)
		default:
			m.tr.baked += sp.text
		}
	}
	m.setView()
}

func (m *model) closeToolRun() {
	if m.tr.afterTool {
		m.tr.afterTool = false
		m.push("\n")
	}
}

func (m *model) endThink() {
	if !m.th.active {
		return
	}
	m.th.active = false
	m.tr.spans = append(m.tr.spans, span{kind: thinkSpan, text: m.th.raw})
	m.th.raw = ""
	m.tr.spans = append(m.tr.spans, span{text: "\n\n"})
	if m.th.idx >= 0 && m.th.idx < len(m.tr.spans) {
		m.tr.spans[m.th.idx].text = thinkStyle.Render(" Thought for "+thinkDuration(time.Since(m.th.start))) + "\n"
		m.th.idx = -1
	}
	m.rebake()
}

func (m *model) queueRender() tea.Cmd {
	if m.renderWait {
		return nil
	}
	m.renderWait = true
	return tea.Tick(time.Second/30, func(time.Time) tea.Msg { return renderMsg{} })
}

func (m *model) layout() {
	extra := 0
	if m.ask.pending != nil {
		extra = strings.Count(m.approvalCard(), "\n")
	}
	m.vp.Height = max(m.tall-5-extra, 0)
}
