package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"testing"
	"time"
)

func TestGenerationTimingExcludesInitialWaitAndUsesProviderTokens(t *testing.T) {
	started := time.Unix(100, 0)
	timing := generationTiming{}
	timing.observe(provider.Delta{}, started)
	timing.observe(provider.Delta{Reasoning: "thinking"}, started.Add(5*time.Second))
	timing.observe(provider.Delta{Text: "answer"}, started.Add(6*time.Second))
	usage := &session.ChunkUsage{Output: 101}
	timing.apply(usage, started.Add(7*time.Second))
	if usage.Generation == nil || usage.Generation.Tokens != 100 || usage.Generation.Seconds != 2 {
		t.Fatalf("%+v", usage.Generation)
	}
}

func TestGenerationTimingOmitsUnmeasurableResponses(t *testing.T) {
	started := time.Now()
	for _, output := range []int{0, 1, 100} {
		timing := generationTiming{}
		timing.observe(provider.Delta{Text: "buffered"}, started)
		usage := &session.ChunkUsage{Output: output}
		timing.apply(usage, started.Add(time.Second))
		if usage.Generation != nil {
			t.Fatal("Invented timing for a single buffered event")
		}
	}
}
