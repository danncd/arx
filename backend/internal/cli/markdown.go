package cli

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerBase  = lipgloss.NewStyle()
	headerBrand = headerBase.Bold(true).Foreground(lipgloss.Color("#34bf8c"))
	headerDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	userMark    = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	userStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	thinkStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	failStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	sepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a"))

	toolBracket = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	toolName    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6fb7e6"))
	toolOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#34bf8c"))
	toolFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	sbTrack = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("│")
	sbThumb = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ea77a")).Render("┃")

	askMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	askOpt    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	askOptSel = lipgloss.NewStyle().Foreground(lipgloss.Color("#0b2018")).Background(lipgloss.Color("#34bf8c")).Bold(true)
)

func newRenderer(width int) *glamour.TermRenderer {
	cfg := styles.DarkStyleConfig
	margin := uint(1)
	cfg.Document.Margin = &margin
	chroma := *cfg.CodeBlock.Chroma
	chroma.Background.BackgroundColor = nil
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

func fenceLine(line string) (byte, int, string, bool) {
	body := strings.TrimLeft(line, " ")
	if len(line)-len(body) > 3 || body == "" || (body[0] != '`' && body[0] != '~') {
		return 0, 0, "", false
	}
	mark := body[0]
	n := 0
	for n < len(body) && body[n] == mark {
		n++
	}
	if n < 3 {
		return 0, 0, "", false
	}
	rest := body[n:]
	if mark == '`' && strings.Contains(rest, "`") {
		return 0, 0, "", false
	}
	return mark, n, rest, true
}

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
		body := strings.TrimLeft(ln, " ")
		indent := len(ln) - len(body)
		if indent > 3 {
			continue
		}
		hashes := 0
		for hashes < len(body) && body[hashes] == '#' {
			hashes++
		}
		if hashes >= 2 && hashes <= 6 && hashes < len(body) && body[hashes] != ' ' {
			lines[i] = ln[:indent] + body[:hashes] + " " + body[hashes:]
		}
	}
	return strings.Join(lines, "\n")
}

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

func (m *model) renderThink(s string) string {
	wrapped := lipgloss.NewStyle().Width(max(m.vp.Width-3, 8)).Render(s)
	lines := strings.Split(wrapped, "\n")
	for i, ln := range lines {
		lines[i] = thinkStyle.Render(" │ " + ln)
	}
	return strings.Join(lines, "\n")
}
