package web

import (
	"context"
	"os"
	"testing"
)

func TestLiveKeyFreeSearch(t *testing.T) {
	if os.Getenv("ARX_WEB_LIVE") != "1" {
		t.Skip("Set ARX_WEB_LIVE=1 to check public search providers")
	}
	client := New()
	for _, query := range []string{"Go documentation", "danncd github", "Qwen3.5 4B vision model"} {
		result, err := client.search(context.Background(), query)
		if err != nil {
			t.Logf("%s: %v", query, err)
			continue
		}
		t.Logf("%s: %d sources; %s", query, len(result.Sources), result.Sources[0].URL)
		for _, source := range result.Sources {
			if !matchesQuery(query, source.Title+" "+source.Snippet+" "+source.URL) {
				t.Fatalf("unrelated source returned: %+v", source)
			}
		}
	}
}
