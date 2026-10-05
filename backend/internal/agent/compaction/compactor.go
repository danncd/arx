package compaction

import (
	contextwindow "arx/internal/agent/context"
	provider "arx/internal/inference"
	model "arx/internal/models"
	session "arx/internal/sessions"
	"context"
	"errors"
	"fmt"
)

type Compactor struct {
	compactions  int
	recoveries   int
	Compression  int
	Continuation int
	AutoRecover  bool
	Recovery     func(string, int)
	Usage        *session.ContextUsage
	Force        bool
	Model        model.Info
	State        session.Compaction
	Complete     func(context.Context, provider.Request, func(provider.Delta) error) (provider.Response, error)
	Save         func(session.Compaction) error
	Active       func(bool)
}

func (c *Compactor) Prepare(ctx context.Context, request provider.Request) (provider.Request, error) {
	for {
		prepared, err := c.prepare(ctx, request)
		if err == nil || ctx.Err() != nil || !errors.Is(err, provider.ErrContextLength) || !c.Recover("context") {
			return prepared, err
		}
	}
}

func (c *Compactor) prepare(ctx context.Context, request provider.Request) (provider.Request, error) {
	budget, err := contextwindow.New(c.Model)
	if err != nil {
		return request, err
	}
	if len(request.Messages) == 0 || request.Messages[0].Role != "system" {
		return request, errors.New("Conversation system message is missing")
	}
	system, history := request.Messages[0], request.Messages[1:]
	ends, err := boundaries(history)
	if err != nil {
		return request, err
	}
	if !Valid(c.State, history) {
		c.State = session.Compaction{}
	}
	request.MaxOutputTokens = budget.Output
	request.Messages = c.compress(append([]provider.Message{system}, retainedMessages(history, c.State)...))
	target := min(recentTokens, budget.Input/3)
	desired := retentionCut(history, ends, c.State.Through, target)
	remainingImages := imageCount(history[desired:])
	if user := userToKeep(history, desired); user > 0 {
		remainingImages += len(history[user-1].Images)
	}
	images := imageCount(request.Messages)
	imagePressure := images > maxImages && remainingImages < images && desired > c.State.Through
	upgrading := c.State.Through > 0 && c.State.Target == 0 && contextwindow.Estimate(request) > target
	if contextwindow.Count(request, c.Usage) <= budget.Input && !c.State.Pending && !upgrading && !c.Force && !imagePressure {
		return request, ctx.Err()
	}
	if c.compactions >= 3 && !imagePressure {
		return request, contextLimit("Context filled repeatedly. Continue with a smaller request")
	}
	if !imagePressure {
		c.compactions++
	}
	if c.Active != nil {
		c.Active(true)
		defer c.Active(false)
	}
	maxBytes := min(summaryTokens, budget.Input/6) * 8
	if c.Force && desired == c.State.Through {
		for _, end := range ends {
			if end > c.State.Through && end < len(history) {
				desired = end
				break
			}
		}
	}
	if desired <= c.State.Through && c.State.Pending && !c.Force && budget.Fits(request) {
		next := c.State
		next.Pending = false
		if c.Save == nil {
			return request, errors.New("Compaction storage is unavailable")
		}
		if err := c.Save(next); err != nil {
			return request, err
		}
		c.State = next
		return request, ctx.Err()
	}
	if desired <= c.State.Through {
		return request, contextLimit("The latest request or tool exchange is too large to compact")
	}
	for c.State.Through < desired {
		if err := ctx.Err(); err != nil {
			return request, err
		}
		cut, text, err := c.summarize(ctx, request.Effort, history, ends, budget, maxBytes, desired)
		if err != nil {
			return request, err
		}
		if err := ctx.Err(); err != nil {
			return request, err
		}
		next := session.Compaction{Version: 2, User: userToKeep(history, cut), Through: cut, Digest: digest(history[:cut]), Summary: text, Target: target, Pending: cut < desired}
		before := contextwindow.Messages(retainedMessages(history, c.State))
		after := contextwindow.Messages(retainedMessages(history, next))
		if after >= before && imageCount(retainedMessages(history, next)) >= imageCount(retainedMessages(history, c.State)) {
			return request, contextLimit("Compaction did not reduce the conversation. Send a message to retry")
		}
		if c.Save == nil {
			return request, errors.New("Compaction storage is unavailable")
		}
		if err := c.Save(next); err != nil {
			return request, fmt.Errorf("Could not save conversation summary: %w", err)
		}
		c.State = next
		c.Usage = nil
	}
	request.Messages = c.compress(append([]provider.Message{system}, retainedMessages(history, c.State)...))
	if !budget.Fits(request) {
		return request, contextLimit("The latest request or tool exchange exceeds the available context. Start a new session with a smaller request")
	}
	return request, nil
}
