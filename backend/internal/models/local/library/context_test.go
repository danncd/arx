package library

import "testing"

func TestCacheAccountsForGroupedHeadsAndExplicitDimensions(t *testing.T) {
	m := Metadata{Architecture: "qwen3", Numbers: map[string]uint64{
		"qwen3.block_count": 28, "qwen3.embedding_length": 1024, "qwen3.attention.head_count": 16,
		"qwen3.attention.head_count_kv": 8, "qwen3.attention.key_length": 128, "qwen3.attention.value_length": 128,
	}}
	if got := m.CacheBytesPerToken(); got != 114688 {
		t.Fatal(got)
	}
	delete(m.Numbers, "qwen3.attention.key_length")
	delete(m.Numbers, "qwen3.attention.value_length")
	if got := m.CacheBytesPerToken(); got != 57344 {
		t.Fatal(got)
	}
	delete(m.Numbers, "qwen3.block_count")
	if got := m.CacheBytesPerToken(); got != 0 {
		t.Fatal(got)
	}
}

func TestCacheAccountsForGemmaPerLayerHeads(t *testing.T) {
	heads := make([]uint64, 48)
	pattern := make([]uint64, 48)
	for layer := range heads {
		if layer%6 == 5 {
			heads[layer] = 1
		} else {
			heads[layer] = 8
			pattern[layer] = 1
		}
	}
	m := Metadata{
		Architecture: "gemma4",
		Numbers: map[string]uint64{
			"gemma4.block_count":                48,
			"gemma4.embedding_length":           3840,
			"gemma4.attention.head_count":       16,
			"gemma4.attention.key_length":       512,
			"gemma4.attention.value_length":     512,
			"gemma4.attention.key_length_swa":   256,
			"gemma4.attention.value_length_swa": 256,
		},
		NumberArrays: map[string][]uint64{
			"gemma4.attention.head_count_kv":          heads,
			"gemma4.attention.sliding_window_pattern": pattern,
		},
	}
	if got := m.CacheBytesPerToken(); got != 344064 {
		t.Fatal(got)
	}
}
