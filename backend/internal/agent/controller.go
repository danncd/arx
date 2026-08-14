package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"arx/internal/llm"
	"arx/internal/tool"
)

var ErrStepLimit = errors.New("step limit reached before a final answer")

type Sink interface {
	Token(s string, thinking bool)
	ToolResult(name, out string, took time.Duration)
}

type Controller struct {
	prof     llm.Profile
	maxSteps int
	msgs     []llm.Message
	initErr  error
}

func New(prof llm.Profile, system string) *Controller {
	c := &Controller{
		prof:     prof,
		maxSteps: 8,
		msgs:     []llm.Message{{Role: "system", Content: system}},
	}
	if !utf8.ValidString(system) {
		c.initErr = fmt.Errorf("system prompt contains invalid UTF-8")
	}
	return c
}

func (c *Controller) RunTurn(ctx context.Context, text string, sink Sink) error {
	if c.initErr != nil {
		return c.initErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !utf8.ValidString(text) {
		return fmt.Errorf("user message contains invalid UTF-8")
	}
	c.msgs = append(c.msgs, llm.Message{Role: "user", Content: text})
	toolsEnabled := !c.prof.ToolsKnown || c.prof.Tools
	for step := 0; step < c.maxSteps; step++ {
		var specs []llm.ToolSpec
		if toolsEnabled {
			specs = tool.Specs()
		}
		reply, err := llm.Stream(ctx, c.prof, c.msgs, specs, sink.Token)
		if err != nil {
			return err
		}
		if !toolsEnabled && len(reply.ToolCalls) > 0 {
			return fmt.Errorf("model returned tool calls while tools are disabled")
		}
		c.msgs = append(c.msgs, reply)

		if len(reply.ToolCalls) == 0 {
			return nil
		}
		for i, tc := range reply.ToolCalls {
			if err := ctx.Err(); err != nil {
				c.skipTools(reply.ToolCalls[i:], err)
				return err
			}
			start := time.Now()
			out := runTool(ctx, tc)
			if !utf8.ValidString(out) {
				err := fmt.Errorf("tool %q returned invalid UTF-8", tc.Function.Name)
				c.skipTools(reply.ToolCalls[i:], err)
				return err
			}
			sink.ToolResult(tc.Function.Name, out, time.Since(start))
			c.msgs = append(c.msgs, llm.Message{
				Role: "tool", ToolCallID: tc.ID, Content: out,
			})
			if err := ctx.Err(); err != nil {
				c.skipTools(reply.ToolCalls[i+1:], err)
				return err
			}
		}
	}
	return ErrStepLimit
}

func (c *Controller) skipTools(calls []llm.ToolCall, cause error) {
	for _, tc := range calls {
		c.msgs = append(c.msgs, llm.Message{
			Role: "tool", ToolCallID: tc.ID, Content: "error: " + cause.Error(),
		})
	}
}

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
		return "(no output)"
	}
	return out
}
