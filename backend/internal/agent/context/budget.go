package context

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	"errors"
	"fmt"
)

type Budget struct {
	Input  int
	Output int
}

func New(info model.Info) (Budget, error) {
	if info.ContextWindow <= 0 || info.MaxOutputTokens <= 0 {
		return Budget{}, errors.New("Model context limits are unavailable. Refresh models in Settings")
	}
	if info.ContextWindow < 4096 {
		return Budget{}, fmt.Errorf("The active context is %d tokens; Arx requires at least 4096. Use a smaller model or increase its context if memory allows", info.ContextWindow)
	}
	output := min(32768, info.MaxOutputTokens, info.ContextWindow/3)
	input := min(info.ContextWindow*85/100, info.ContextWindow-output-max(512, info.ContextWindow/20))
	return Budget{Input: input, Output: output}, nil
}

func (b Budget) Fits(request provider.Request) bool {
	return Estimate(request) <= b.Input
}
