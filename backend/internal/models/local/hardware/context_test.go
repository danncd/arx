package hardware

import "testing"

func TestContextUsesSeventyFivePercentWhenMemoryAllows(t *testing.T) {
	hardware := Info{Memory: 32 << 30, Supported: true}
	context, reason := hardware.Context(131072, 4<<30, 32768)
	if context != 98304 || reason != "75% of supported context" {
		t.Fatal(context, reason)
	}
}
func TestContextReducesTargetForMemory(t *testing.T) {
	hardware := Info{Memory: 16 << 30, Supported: true}
	context, _ := hardware.Context(131072, 8<<30, 262144)
	if context != 8192 {
		t.Fatal(context)
	}
	context, _ = hardware.Context(32768, 1<<30, 0)
	if context != 4096 {
		t.Fatal(context)
	}
	context, _ = hardware.Context(4096, 1<<30, 0)
	if context != 3072 {
		t.Fatal(context)
	}
}
