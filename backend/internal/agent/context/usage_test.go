package context

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	tool "arx/internal/tools"
	"strings"
	"testing"
)

func TestUsageAnchorsActualInputAndEstimatesOnlyNewMessages(t *testing.T) {
	request := provider.Request{Model: "deepseek", Effort: "high", Messages: []provider.Message{{Role: "system", Content: "Instructions"}, {Role: "user", Content: "Hello"}}, Tools: tool.Definitions()}
	usage := Measure(request, 700)
	if got := Count(request, usage); got != 700 {
		t.Fatal("actual input ignored", got)
	}
	next := provider.Message{Role: "assistant", Content: strings.Repeat("Result ", 30)}
	request.Messages = append(request.Messages, next)
	if got := Count(request, usage); got != 700+Messages([]provider.Message{next}) {
		t.Fatal("new content not counted", got)
	}
	for _, mutate := range []func(*provider.Request){
		func(r *provider.Request) { r.Model = "other" },
		func(r *provider.Request) { r.Effort = "none" },
		func(r *provider.Request) { r.Tools = nil },
		func(r *provider.Request) { r.Messages = []provider.Message{{Role: "system", Content: "Changed"}} },
	} {
		changed := request
		mutate(&changed)
		if got := Count(changed, usage); got != Estimate(changed) {
			t.Fatal("stale measurement reused", got)
		}
	}
	if Measure(request, 0) != nil {
		t.Fatal("invalid usage accepted")
	}
	if got := Latest([]session.TranscriptChunk{{Context: usage}, {}}); got != usage {
		t.Fatal("measurement lost across transcript chunks")
	}
}

func TestEstimationAndScaledBreakdown(t *testing.T) {
	if Text(strings.Repeat("a", 3000)) != 1000 || Text(strings.Repeat("界", 100)) != 200 {
		t.Fatal("unexpected token estimate")
	}
	request := provider.Request{Messages: []provider.Message{{Role: "system", Content: "Instructions"}, {Role: "assistant", Content: "Summary"}, {Role: "user", Content: "Continue"}}, Tools: tool.Definitions()}
	parts := Breakdown(request, true).Scale(765)
	if parts.System+parts.Tools+parts.Messages+parts.Summary != 765 || parts.System <= 0 || parts.Summary <= 0 {
		t.Fatal(parts)
	}
}
