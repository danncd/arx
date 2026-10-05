package chat

import (
	"arx/internal/agent"
	compactor "arx/internal/agent/compaction"
	contextwindow "arx/internal/agent/context"
	session "arx/internal/sessions"
	"arx/internal/settings"
)

type ContextReport struct {
	Usage session.ChunkUsage  `json:"usage"`
	Parts contextwindow.Parts `json:"parts"`
	Known bool                `json:"known"`
	Used  int                 `json:"used"`
	Limit int                 `json:"limit"`
	Basis string              `json:"basis"`
}

func (s *Service) Context(conversation, model string) (ContextReport, error) {
	state, stateErr := s.Sessions.Compaction(conversation)
	entries := s.Sessions.Entries(conversation)
	usage := session.Usage(entries)
	info, err := s.Inference.Model(settings.Run{Model: model})
	if err != nil || info.ContextWindow <= 0 {
		return ContextReport{Usage: usage}, nil
	}
	if stateErr != nil {
		return ContextReport{Usage: usage}, stateErr
	}
	history, err := agent.History(entries)
	if err != nil {
		return ContextReport{}, err
	}
	request := agent.Request(s.Preferences.Values(), history, info)
	agent.AddGenerationGuidance(&request, s.GenerationGuidance())
	agent.AddGenerationGuidance(&request, s.IntegrationGuidance())
	request = compactor.Preview(request, state)
	request.Model = model
	used := contextwindow.Count(request, contextwindow.Latest(entries))
	parts := contextwindow.Breakdown(request, compactor.Valid(state, history))
	parts = parts.Scale(used)
	return ContextReport{Usage: usage, Known: true, Used: used, Limit: info.ContextWindow, Basis: "estimated", Parts: parts}, nil
}
