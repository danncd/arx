package agent

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	"arx/internal/settings"
	tool "arx/internal/tools"
	"strings"
)

func Request(saved settings.Values, history []provider.Message, capabilities ...model.Info) provider.Request {
	inspection := visionPrompt
	if len(capabilities) > 0 && !capabilities[0].Vision {
		inspection = textOnlyPrompt
	}
	prompt := strings.Join([]string{basePrompt, generationPrompt, filePrompt, inspection, networkPrompt, integrationsPrompt}, " ")
	messages := append([]provider.Message{{Role: "system", Content: prompt}}, history...)
	return provider.Request{Model: saved.Run.Model, Effort: saved.Run.Effort, Messages: messages, Tools: tool.Definitions()}
}

func AddGenerationGuidance(request *provider.Request, guidance string) {
	if guidance != "" && len(request.Messages) > 0 && request.Messages[0].Role == "system" {
		request.Messages[0].Content += "\n\n" + guidance
	}
}
