package library

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestGGUFReadsMetadataWithoutLoadingWeights(t *testing.T) {
	var data bytes.Buffer
	write := func(value any) { binary.Write(&data, binary.LittleEndian, value) }
	text := func(value string) { write(uint64(len(value))); data.WriteString(value) }
	data.WriteString("GGUF")
	write(uint32(3))
	write(uint64(0))
	write(uint64(3))
	text("general.architecture")
	write(uint32(8))
	text("qwen3")
	text("qwen3.context_length")
	write(uint32(4))
	write(uint32(32768))
	text("tokenizer.chat_template")
	write(uint32(8))
	text("enable_thinking")
	path := filepath.Join(t.TempDir(), "model.gguf")
	os.WriteFile(path, data.Bytes(), 0600)
	metadata, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Context != 32768 || metadata.Architecture != "qwen3" || metadata.Template != "enable_thinking" {
		t.Fatal(metadata)
	}
	os.WriteFile(path, data.Bytes()[:14], 0600)
	if _, err := Inspect(path); err == nil {
		t.Fatal("Truncated header accepted")
	}
}

func TestGGUFReadsAttentionArrays(t *testing.T) {
	var data bytes.Buffer
	write := func(value any) { binary.Write(&data, binary.LittleEndian, value) }
	writeText := func(value string) { write(uint64(len(value))); data.WriteString(value) }
	data.WriteString("GGUF")
	write(uint32(3))
	write(uint64(0))
	write(uint64(3))
	writeText("general.architecture")
	write(uint32(8))
	writeText("gemma4")
	for _, entry := range []struct {
		key    string
		kind   uint32
		values []uint64
	}{
		{"gemma4.attention.head_count_kv", 4, []uint64{8, 1}},
		{"gemma4.attention.sliding_window_pattern", 7, []uint64{1, 0}},
	} {
		writeText(entry.key)
		write(uint32(9))
		write(entry.kind)
		write(uint64(len(entry.values)))
		for _, value := range entry.values {
			if entry.kind == 7 {
				write(uint8(value))
			} else {
				write(uint32(value))
			}
		}
	}
	path := filepath.Join(t.TempDir(), "model.gguf")
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	metadata, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if heads := metadata.NumberArrays["gemma4.attention.head_count_kv"]; len(heads) != 2 || heads[0] != 8 || heads[1] != 1 {
		t.Fatal(heads)
	}
	if pattern := metadata.NumberArrays["gemma4.attention.sliding_window_pattern"]; len(pattern) != 2 || pattern[0] != 1 || pattern[1] != 0 {
		t.Fatal(pattern)
	}
}
