package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"arx/internal/provider"
	"arx/internal/tool"
)

var ErrStepLimit = errors.New("step limit reached before a final answer")

type Sink interface {
	Token(s string, thinking bool)
	ToolStart(name, args string)
	ToolResult(name, out string, took time.Duration, failed bool)
}

type Controller struct {
	prof     provider.Profile
	maxSteps int
	msgs     []provider.Message
	gate     *Gate
	intent   string
	initErr  error

	steerMu     sync.Mutex
	steers      []string
	steerActive bool
}

func New(prof provider.Profile, system string) *Controller {
	c := &Controller{
		prof:     prof,
		maxSteps: 8,
		msgs:     []provider.Message{{Role: "system", Content: system}},
	}
	if !utf8.ValidString(system) {
		c.initErr = fmt.Errorf("system prompt contains invalid UTF-8")
	}
	return c
}

func (c *Controller) SetGate(g *Gate) { c.gate = g }

func (c *Controller) Steer(text string) bool {
	if !utf8.ValidString(text) {
		return false
	}
	c.steerMu.Lock()
	defer c.steerMu.Unlock()
	if !c.steerActive {
		return false
	}
	c.steers = append(c.steers, text)
	return true
}

func (c *Controller) takeSteers() []string {
	c.steerMu.Lock()
	defer c.steerMu.Unlock()
	s := c.steers
	c.steers = nil
	return s
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
	c.msgs = append(c.msgs, provider.Message{Role: "user", Content: text})
	c.intent = clipIntent(text)

	c.steerMu.Lock()
	c.steerActive = true
	c.steers = nil
	c.steerMu.Unlock()
	defer func() {
		c.steerMu.Lock()
		c.steerActive = false
		c.steers = nil
		c.steerMu.Unlock()
	}()

	toolsEnabled := !c.prof.ToolsKnown || c.prof.Tools
	for step := 0; step < c.maxSteps; step++ {
		for _, s := range c.takeSteers() {
			c.msgs = append(c.msgs, provider.Message{Role: "user", Content: s})
			c.intent = clipIntent(s)
		}
		var specs []provider.ToolSpec
		if toolsEnabled {
			specs = tool.Specs()
		}
		reply, err := provider.Stream(ctx, c.prof, c.msgs, specs, provider.Options{}, sink.Token)
		if err != nil {
			c.keepPartial(reply)
			return err
		}
		if !toolsEnabled && len(reply.ToolCalls) > 0 {
			c.keepPartial(reply)
			return fmt.Errorf("model returned tool calls while tools are disabled")
		}
		c.msgs = append(c.msgs, reply)

		if len(reply.ToolCalls) == 0 {
			return nil
		}
		decisions, err := c.authorizeAll(ctx, reply.ToolCalls)
		if err != nil {
			c.skipTools(reply.ToolCalls, err)
			return err
		}
		for i, tc := range reply.ToolCalls {
			if err := ctx.Err(); err != nil {
				c.skipTools(reply.ToolCalls[i:], err)
				return err
			}
			if !decisions[i].Allow {
				out := "error: denied: " + decisions[i].Reason
				sink.ToolResult(tc.Function.Name, out, 0, true)
				c.msgs = append(c.msgs, provider.Message{
					Role: "tool", ToolCallID: tc.ID, Content: out,
				})
				continue
			}
			sink.ToolStart(tc.Function.Name, tc.Function.Arguments)
			start := time.Now()
			out, failed := runTool(ctx, tc)
			if !utf8.ValidString(out) {
				err := fmt.Errorf("tool %q returned invalid UTF-8", tc.Function.Name)
				c.skipTools(reply.ToolCalls[i:], err)
				return err
			}
			sink.ToolResult(tc.Function.Name, out, time.Since(start), failed)
			c.msgs = append(c.msgs, provider.Message{
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

func (c *Controller) keepPartial(reply provider.Message) {
	if reply.Role != "" && reply.Role != "assistant" {
		return
	}
	if reply.Content == "" {
		return
	}
	if !utf8.ValidString(reply.Content) || !utf8.ValidString(reply.ReasoningContent) {
		return
	}
	c.msgs = append(c.msgs, provider.Message{
		Role:             "assistant",
		Content:          reply.Content,
		ReasoningContent: reply.ReasoningContent,
	})
}

func (c *Controller) skipTools(calls []provider.ToolCall, cause error) {
	for _, tc := range calls {
		c.msgs = append(c.msgs, provider.Message{
			Role: "tool", ToolCallID: tc.ID, Content: "error: " + cause.Error(),
		})
	}
}

func (c *Controller) authorizeAll(ctx context.Context, calls []provider.ToolCall) ([]Decision, error) {
	reqs := make([]Request, len(calls))
	mutating := make([]bool, len(calls))
	for i, tc := range calls {
		reqs[i] = Request{Tool: tc.Function.Name, Args: tc.Function.Arguments, Intent: c.intent}
		mutating[i] = tool.IsMutating(tc.Function.Name)
	}
	if c.gate == nil {
		out := make([]Decision, len(calls))
		for i := range calls {
			if mutating[i] {
				out[i] = Decision{Reason: "no permission gate configured"}
				continue
			}
			out[i] = Decision{Allow: true}
		}
		return out, nil
	}
	return c.gate.AuthorizeAll(ctx, reqs, mutating)
}

func clipIntent(text string) string {
	const limit = 2000
	if len(text) <= limit {
		return text
	}
	cut := text[:limit]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}

func runTool(ctx context.Context, tc provider.ToolCall) (string, bool) {
	t, ok := tool.Get(tc.Function.Name)
	if !ok {
		return "error: unknown tool " + tc.Function.Name, true
	}
	out, err := t.Run(ctx, json.RawMessage(tc.Function.Arguments))
	if err != nil {
		return "error: " + err.Error(), true
	}
	if out == "" {
		return "(no output)", false
	}
	return out, false
}
