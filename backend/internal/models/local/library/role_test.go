package library

import "testing"

func TestRejectHelperFiles(t *testing.T) {
	for _, name := range []string{
		"MTP/mtp-Qwen3.8-Flash-Next-Q4_K_M.gguf",
		"MTP/Q4_K_M.gguf",
		"mtp-model.gguf",
		"draft.gguf",
		"DRAFT/model.gguf",
		"dspark-model.gguf",
		"mmproj-F16.gguf",
		"DeepSeek-V4-Flash-Vision-Encoder.gguf",
		"imatrix-qwen3.8-27b.gguf",
	} {
		if err := ValidateChatFile(name); err == nil {
			t.Errorf("Accepted helper file %q", name)
		}
	}
	for _, name := range []string{"Qwen3.8-Flash-Next-Q4_K_M.gguf", "Q4_K_M/model-00001-of-00002.gguf", "PromptPilot-Q4_K_M.gguf", "Qwen-imatrix-Q4_K_M.gguf", "imatrix-Qwen-Q4_K_M.gguf"} {
		if err := ValidateChatFile(name); err != nil {
			t.Errorf("Rejected model %q: %v", name, err)
		}
	}
}
