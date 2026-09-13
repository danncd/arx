package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"arx/internal/tool"
)

func thinkDuration(d time.Duration) string {
	secs := int(d.Round(time.Second).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	return fmt.Sprintf("%dm %ds", secs/60, secs%60)
}

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

func (m *model) runningRow() string {
	head := toolBracket.Render(" 〔 ") + sepStyle.Render(m.spin.View()) + toolBracket.Render(" 〕") + toolName.Render(m.run.running)
	if m.run.hint != "" {
		room := max(m.vp.Width-lipgloss.Width(head)-4, 8)
		head += toolBracket.Render(" → ") + clipCell(m.run.hint, room)
	}
	return head
}

func runHint(name, args string) string {
	if name != "bash" {
		return ""
	}
	return tool.Command(args)
}

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

func clipCell(s string, width int) string {
	if width < 2 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

func wrapPlain(s string, first, width, maxLines int) []string {
	head := ansi.Truncate(s, first, "")
	rest := strings.TrimLeft(strings.TrimPrefix(s, head), " ")
	var lines []string
	if head != "" {
		lines = append(lines, head)
	}
	if rest != "" {
		lines = append(lines, strings.Split(ansi.Hardwrap(rest, width, false), "\n")...)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = ansi.Truncate(lines[maxLines-1], max(width-1, 1), "") + "…"
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
