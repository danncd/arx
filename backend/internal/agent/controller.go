// Package agent owns the conversation: the transcript, the model
// choice, and the reason-act loop that advances them. It knows nothing
// about terminals or servers; presentation lives behind Sink.
package agent

import (
	"context"
	"encoding/json"
	"errors"

	"arx/internal/llm"
	"arx/internal/tool"
)

// ErrStepLimit reports a turn that ran out of steps while the model
// was still calling tools. The frontend decides how to phrase it.
var ErrStepLimit = errors.New("step limit reached before a final answer")

// A Sink receives a turn's output as it happens.
type Sink interface {
	Token(s string, thinking bool)
	ToolResult(name, out string)
}

type Controller struct {
	prof     llm.Profile
	maxSteps int
	msgs     []llm.Message
}

func New(prof llm.Profile, system string) *Controller {
	return &Controller{
		prof:     prof,
		maxSteps: 8, // backstop, not a leash
		msgs:     []llm.Message{{Role: "system", Content: system}},
	}
}

// RunTurn advances the conversation by one user message: stream a
// reply, execute any tool calls, feed the results back, and repeat
// until the model answers in text or the step backstop trips.
func (c *Controller) RunTurn(ctx context.Context, text string, sink Sink) error {
	c.msgs = append(c.msgs, llm.Message{Role: "user", Content: text})
	for step := 0; step < c.maxSteps; step++ {
		reply, err := llm.Stream(ctx, c.prof, c.msgs, tool.Specs(), sink.Token)
		if err != nil {
			return err
		}
		c.msgs = append(c.msgs, reply)

		if len(reply.ToolCalls) == 0 {
			return nil // the model spoke: turn complete
		}
		for _, tc := range reply.ToolCalls {
			out := runTool(ctx, tc)
			sink.ToolResult(tc.Function.Name, out)
			c.msgs = append(c.msgs, llm.Message{
				Role: "tool", ToolCallID: tc.ID, Content: out,
			})
		}
	}
	return ErrStepLimit
}

// runTool executes one call, converting every failure into an
// observation string: the model must SEE errors, not crash the loop.
func runTool(ctx context.Context, tc llm.ToolCall) string {
	t, ok := tool.Get(tc.Function.Name)
	if !ok {
		return "error: unknown tool " + tc.Function.Name
	}
	out, err := t.Run(ctx, json.RawMessage(tc.Function.Arguments))
	if err != nil {
		return "error: " + err.Error()
	}
	if out == "" {
		// The model can't distinguish "ran, no output" from a dropped
		// result; say it explicitly.
		return "(no output)"
	}
	return out
}
