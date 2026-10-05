package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"time"
)

type generationTiming struct {
	first  time.Time
	events int
}

func (t *generationTiming) observe(delta provider.Delta, now time.Time) {
	if delta.Text == "" && delta.Reasoning == "" && delta.Tool == nil {
		return
	}
	if t.first.IsZero() {
		t.first = now
	}
	t.events++
}

func (t generationTiming) apply(usage *session.ChunkUsage, finished time.Time) {
	if usage == nil || usage.Output <= 1 || t.events < 2 || t.first.IsZero() {
		return
	}
	seconds := finished.Sub(t.first).Seconds()
	if seconds < 0.05 {
		return
	}
	usage.Generation = &session.GenerationUsage{Tokens: usage.Output - 1, Seconds: seconds}
}
