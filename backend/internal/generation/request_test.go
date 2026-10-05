package generation

import "testing"

func TestRequestDefaultsAndVideoBounds(t *testing.T) {
	request := Request{Operation: "video", Prompt: "A moving cloud"}
	if err := request.Validate(); err != nil || request.Width != 384 || request.Height != 256 || request.Frames != 0 {
		t.Fatalf("invalid defaults: %+v %v", request, err)
	}
	request.Width, request.Height, request.Frames = 1024, 1024, 97
	if request.Validate() == nil {
		t.Fatal("unbounded video memory")
	}
	if (&Request{Operation: "combine", Source: "video"}).Validate() == nil {
		t.Fatal("missing audio accepted")
	}
}
