package context

import provider "arx/internal/inference"

type Parts struct {
	System   int `json:"system"`
	Tools    int `json:"tools"`
	Messages int `json:"messages"`
	Summary  int `json:"summary"`
}

func (p Parts) Scale(total int) Parts {
	estimated := p.System + p.Tools + p.Messages + p.Summary
	if estimated <= 0 {
		return p
	}
	p.System = p.System * total / estimated
	p.Tools = p.Tools * total / estimated
	p.Summary = p.Summary * total / estimated
	p.Messages = total - p.System - p.Tools - p.Summary
	return p
}

func Breakdown(request provider.Request, summarized bool) Parts {
	base := provider.Request{}
	if len(request.Messages) > 0 {
		base.Messages = request.Messages[:1]
	}
	parts := Parts{System: Estimate(base)}
	base.Tools = request.Tools
	parts.Tools = Estimate(base) - parts.System
	if summarized && len(request.Messages) > 1 {
		base.Messages = request.Messages[:2]
		parts.Summary = Estimate(base) - parts.System - parts.Tools
	}
	parts.Messages = Estimate(request) - parts.System - parts.Tools - parts.Summary
	return parts
}
