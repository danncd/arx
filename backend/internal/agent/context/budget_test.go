package context

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	tool "arx/internal/tools"
	"strings"
	"testing"
)

func TestBudgetIncludesToolsUnicodeAndOutputReserve(t *testing.T) {
	info := model.Info{ContextWindow: 16000, MaxOutputTokens: 1000}
	budget, err := New(info)
	if err != nil || budget.Output != 1000 || budget.Input != 13600 || budget.Input+budget.Output >= info.ContextWindow {
		t.Fatal("invalid reserve", budget, err)
	}
	request := provider.Request{Messages: []provider.Message{{Role: "user", Content: "Hello 世界 🌱"}}}
	small := Estimate(request)
	request.Tools = tool.Definitions()
	if Estimate(request) <= small {
		t.Fatal("tool schemas not counted")
	}
	request.Messages[0].Content = strings.Repeat("🌱", 8000)
	if budget.Fits(request) {
		t.Fatal("unicode text not budgeted")
	}
	if _, err := New(model.Info{}); err == nil {
		t.Fatal("missing model limits accepted")
	}
}

func TestCompactionThresholdPreservesOutputSpace(t *testing.T) {
	for _, info := range []model.Info{
		{ContextWindow: 1048576, MaxOutputTokens: 393216},
		{ContextWindow: 16000, MaxOutputTokens: 4000},
	} {
		budget, err := New(info)
		if err != nil {
			t.Fatal(err)
		}
		if budget.Input > info.ContextWindow*85/100 || budget.Input+budget.Output+max(512, info.ContextWindow/20) > info.ContextWindow {
			t.Fatal("threshold exceeds available context", budget)
		}
		if info.ContextWindow == 1048576 && budget.Input != 891289 {
			t.Fatal("compaction threshold is not 85 percent", budget)
		}
	}
}

func TestNetworkContextCanReserveMoreThanFourThousandOutputTokens(t *testing.T) {
	budget, err := New(model.Info{ContextWindow: 20156, MaxOutputTokens: 6718})
	if err != nil || budget.Output != 6718 || budget.Input+budget.Output+1007 > 20156 {
		t.Fatal("network output reserve is wrong", budget, err)
	}
}

func TestSmallKnownContextIsNotReportedAsMissing(t *testing.T) {
	_, err := New(model.Info{ContextWindow: 2048, MaxOutputTokens: 1024})
	if err == nil || !strings.Contains(err.Error(), "2048 tokens") || strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("misleading context error: %v", err)
	}
	if _, err := New(model.Info{ContextWindow: 4096, MaxOutputTokens: 1024}); err != nil {
		t.Fatal(err)
	}
}
