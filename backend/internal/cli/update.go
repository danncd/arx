package cli

import (
	"errors"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"arx/internal/agent"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.tall = msg.Height
		m.vp.Width = max(msg.Width-2, 1)
		m.layout()
		m.ti.Width = max(msg.Width-5, 8)
		m.mdr = newRenderer(m.vp.Width)
		m.rebake()

	case approvalMsg:
		m.ask.pending = &msg
		m.ask.sel = 0
		m.ask.opened = time.Now()
		m.layout()
		m.setView()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tokenMsg:
		return m.handleToken(msg)

	case renderMsg:
		m.renderWait = false
		m.setView()
		return m, nil

	case toolStartMsg:
		return m.handleToolStart(msg)

	case spinner.TickMsg:
		return m.handleTick(msg)

	case toolMsg:
		return m.handleToolResult(msg)

	case doneMsg:
		return m.handleDone(msg)
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m model) handleToken(msg tokenMsg) (tea.Model, tea.Cmd) {
	msg.text = terminalText(msg.text)
	if msg.thinking {
		if !m.th.active {
			m.finalizeCur()
			m.closeToolRun()
			m.th.active = true
			m.th.start = time.Now()
			label := thinkStyle.Render(" Thinking…") + "\n"
			m.tr.spans = append(m.tr.spans, span{text: label})
			m.th.idx = len(m.tr.spans) - 1
			m.tr.baked += label
		}
		m.th.raw += msg.text
	} else {
		m.endThink()
		m.closeToolRun()
		m.tr.cur += msg.text
	}
	return m, m.queueRender()
}

func (m model) handleToolStart(msg toolStartMsg) (tea.Model, tea.Cmd) {
	m.endThink()
	hadAnswer := m.tr.cur != ""
	m.finalizeCur()
	if hadAnswer {
		m.push("\n")
	}
	m.run.running = terminalLine(msg.name)
	m.run.hint = collapseSpaces(terminalLine(runHint(msg.name, msg.args)))
	m.setView()
	if m.run.spinning {
		return m, nil
	}
	m.run.spinning = true
	return m, m.spin.Tick
}

func (m model) handleTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if m.run.running == "" {
		m.run.spinning = false
		return m, nil
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	m.setView()
	return m, cmd
}

func (m model) handleToolResult(msg toolMsg) (tea.Model, tea.Cmd) {
	m.run.running = ""
	m.endThink()
	hadAnswer := m.tr.cur != ""
	m.finalizeCur()
	if hadAnswer {
		m.push("\n")
	}
	m.push(m.toolLine(terminalLine(msg.name), terminalText(msg.out), msg.took, msg.failed) + "\n")
	m.tr.afterTool = true
	return m, nil
}

func (m model) handleDone(msg doneMsg) (tea.Model, tea.Cmd) {
	m.resolveApproval(agent.Reject)
	m.run.running = ""
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
		m.push("\n")
	}
	return m, nil
}
