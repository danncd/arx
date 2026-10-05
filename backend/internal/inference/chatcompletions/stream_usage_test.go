package chatcompletions

import (
	provider "arx/internal/inference"
	"errors"
	"strings"
	"testing"
)

func TestCacheUsageDistinguishesReportedZeroFromMissing(t *testing.T) {
	for _, test := range []struct {
		name    string
		usage   string
		cached  int
		unknown bool
	}{
		{"reused", `,"prompt_tokens_details":{"cached_tokens":80}`, 80, false},
		{"zero", `,"prompt_tokens_details":{"cached_tokens":0}`, 0, false},
		{"missing", "", 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := `data: {"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":2` + test.usage + `}}` + "\n\ndata: [DONE]\n\n"
			response, err := Read(strings.NewReader(stream), func(provider.Delta) error { return nil })
			if err != nil || response.Usage == nil || response.Usage.Cached != test.cached || response.Usage.CacheUnknown != test.unknown {
				t.Fatalf("usage = %+v, error = %v", response.Usage, err)
			}
		})
	}
}

func TestStreamErrorReportsServerReason(t *testing.T) {
	tests := []struct {
		payload string
		want    string
		kind    error
	}{
		{`{"error":{"message":"The model ran out of memory"}}`, "ran out of memory", nil},
		{`{"error":{"message":"Context length exceeded"}}`, "", provider.ErrContextLength},
		{`{"error":{"message":"Maximum output limit reached"}}`, "", provider.ErrOutputLimit},
	}
	for _, test := range tests {
		stream := "data: " + test.payload + "\n\ndata: [DONE]\n\n"
		_, err := ReadDetailed(strings.NewReader(stream), func(provider.Delta) error { return nil })
		if test.kind != nil && !errors.Is(err, test.kind) {
			t.Fatalf("expected %v, got %v", test.kind, err)
		}
		if test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)) {
			t.Fatalf("server detail missing: %v", err)
		}
	}
}
