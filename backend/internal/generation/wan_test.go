package generation

import (
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	"testing"
)

func TestWanLimits(t *testing.T) {
	model := models.Model{Runtime: "wan-cpp", Operations: []string{"video"}}
	request := Request{Operation: "video", Prompt: "An apple on a table"}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	spec := engines.Request{}
	if err := configureModel(&request, &spec, model); err != nil || spec.Steps != 30 || spec.Frames != 25 {
		t.Fatalf("wan defaults: %+v %v", spec, err)
	}
	for _, invalid := range []Request{
		{Operation: "video", Width: 384, Height: 256, Frames: 33},
		{Operation: "video", Width: 768, Height: 512, Frames: 25},
		{Operation: "video", Width: 384, Height: 256, Frames: 25, Source: "arx-image:source"},
	} {
		if configureModel(&invalid, &spec, model) == nil {
			t.Fatalf("unsupported request accepted: %+v", invalid)
		}
	}
}
