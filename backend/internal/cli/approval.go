package cli

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"arx/internal/agent"
	"arx/internal/tool"
)

type approvalMsg struct {
	req  agent.Request
	resp chan agent.Outcome
}

type tuiApprover struct{ sh *shared }

func (a tuiApprover) Ask(ctx context.Context, req agent.Request) (agent.Outcome, error) {
	msg := approvalMsg{req: req, resp: make(chan agent.Outcome, 1)}
	a.sh.p.Send(msg)
	select {
	case out := <-msg.resp:
		return out, nil
	case <-ctx.Done():
		return agent.Reject, ctx.Err()
	}
}

func (m *model) resolveApproval(out agent.Outcome) {
	if m.ask.pending == nil {
		return
	}
	m.ask.pending.resp <- out
	m.ask.pending = nil
	m.layout()
	m.setView()
}

func approvalText(req agent.Request) string {
	if req.Tool == "bash" {
		if command := tool.Command(req.Args); command != "" {
			return command
		}
	}
	return req.Args
}

const approvalSettleDefault = 250 * time.Millisecond

var approvalSettle = approvalSettleDefault

func selOutcome(sel int) agent.Outcome {
	switch sel {
	case 1:
		return agent.AlwaysSession
	case 2:
		return agent.Reject
	}
	return agent.Once
}

func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (m *model) optionRow() string {
	labels := []string{"allow once", "always this session", "deny"}
	parts := make([]string, len(labels))
	for i, label := range labels {
		style := askOpt
		if i == m.ask.sel {
			style = askOptSel
		}
		parts[i] = style.Render(" " + label + " ")
	}
	return " " + strings.Join(parts, "  ")
}

func (m *model) approvalCard() string {
	req := m.ask.pending.req
	head := toolBracket.Render(" 〔 ") + askMark.Render("?") + toolBracket.Render(" 〕") +
		toolName.Render(terminalLine(req.Tool))
	note := ""
	if req.Note != "" {
		note = thinkStyle.Render("  " + clipCell(terminalLine(req.Note), max(m.width/2, 8)))
	}
	lines := []string{head + note}
	if cmd := collapseSpaces(terminalLine(approvalText(req))); cmd != "" {
		first := max(m.width-lipgloss.Width(head+note)-3, 8)
		chunks := wrapPlain(cmd, first, max(m.width-3, 8), 4)
		lines[0] += toolBracket.Render(" → ") + chunks[0]
		for _, c := range chunks[1:] {
			lines = append(lines, "   "+c)
		}
	}
	return strings.Join(append(lines, m.optionRow()), "\n")
}
