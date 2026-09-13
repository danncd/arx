package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"arx/internal/agent"
)

type terminalSink struct {
	color  bool
	answer strings.Builder
}

func (t *terminalSink) Token(s string, thinking bool) {
	s = terminalText(s)
	if !t.color {
		if !thinking {
			t.answer.WriteString(s)
		}
		return
	}
	if thinking {
		fmt.Print("\033[90m" + s + "\033[0m")
		return
	}
	fmt.Print(s)
}

func (t *terminalSink) ToolStart(string, string) {}

func (t *terminalSink) ToolResult(name, out string, _ time.Duration, _ bool) {
	if !t.color {
		t.answer.Reset()
		return
	}
	fmt.Println("  [" + terminalLine(name) + "] → " + terminalText(out))
}

func (t *terminalSink) finish(success bool) {
	if t.color {
		return
	}
	if success {
		fmt.Print(t.answer.String())
	}
	t.answer.Reset()
}

type denyApprover struct{}

func (denyApprover) Ask(context.Context, agent.Request) (agent.Outcome, error) {
	return agent.Reject, nil
}
