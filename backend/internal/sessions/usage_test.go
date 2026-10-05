package sessions

import "testing"

func TestUsageShowsPreviouslySavedLocalCacheCounts(t *testing.T) {
	entries := []TranscriptChunk{
		{ID: "title", Role: "usage", Usage: &ChunkUsage{Input: 50, Cached: 0, CacheUnknown: true}},
		{ID: "first", Role: "assistant", Provider: "local", Usage: &ChunkUsage{Input: 100, Cached: 0, CacheUnknown: true}},
		{ID: "second", Role: "assistant", Provider: "local", Usage: &ChunkUsage{Input: 200, Cached: 90, CacheUnknown: true}},
	}
	usage := Usage(entries)
	if usage.CacheUnknown || usage.Input != 350 || usage.Cached != 90 {
		t.Fatalf("usage = %+v", usage)
	}
}

func TestUsageKeepsUnknownForMixedProviders(t *testing.T) {
	usage := Usage([]TranscriptChunk{
		{ID: "title", Role: "usage", Usage: &ChunkUsage{Input: 50, CacheUnknown: true}},
		{ID: "local", Role: "assistant", Provider: "local", Usage: &ChunkUsage{Input: 100, Cached: 80, CacheUnknown: true}},
		{ID: "other", Role: "assistant", Provider: "deepseek", Usage: &ChunkUsage{Input: 200, Cached: 100}},
	})
	if !usage.CacheUnknown {
		t.Fatalf("usage = %+v", usage)
	}
}
