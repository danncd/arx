package cli

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"arx/internal/agent"
)

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	}

	if m.ask.pending != nil {
		settled := time.Since(m.ask.opened) >= approvalSettle
		switch msg.String() {
		case "left", "h":
			if m.ask.sel > 0 {
				m.ask.sel--
			}
		case "right", "l":
			if m.ask.sel < 2 {
				m.ask.sel++
			}
		case "enter":
			if settled {
				m.resolveApproval(selOutcome(m.ask.sel))
			}
		case "a":
			if settled {
				m.resolveApproval(agent.Once)
			}
		case "s":
			if settled {
				m.resolveApproval(agent.AlwaysSession)
			}
		case "d", "esc":
			m.resolveApproval(agent.Reject)
		}
		return m, nil
	}

	if msg.Type == tea.KeyEnter {
		return m.submitComposer()
	}

	var tiCmd, vpCmd tea.Cmd
	m.ti, tiCmd = m.ti.Update(msg)
	m.vp, vpCmd = m.vp.Update(msg)
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) submitComposer() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.ti.Value())
	if text == "" {
		return m, nil
	}
	if m.waiting {
		if m.ctrl.Steer(text) {
			m.ti.Reset()
			m.push(userMark.Render("» ") + userStyle.Render(terminalText(text)) + "\n\n")
			m.vp.GotoBottom()
		}
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
