package generation

import (
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	"testing"
)

func TestModelSpecificDefaultsAndCapabilities(t *testing.T) {
	speech := Request{Operation: "speech", Prompt: "Hello"}
	if err := speech.Validate(); err != nil {
		t.Fatal(err)
	}
	spec := engines.Request{}
	kitten := models.Model{Runtime: "kitten", Voices: []string{"expr-voice-5-m"}}
	if err := configureModel(&speech, &spec, kitten); err != nil || spec.Voice != "expr-voice-5-m" {
		t.Fatalf("voice default: %+v %v", spec, err)
	}
	speech.Voice = "af_heart"
	if configureModel(&speech, &spec, kitten) == nil {
		t.Fatal("unsupported voice accepted")
	}
	video := Request{Operation: "video", Prompt: "Apple"}
	if err := video.Validate(); err != nil {
		t.Fatal(err)
	}
	motion := models.Model{Runtime: "animatediff", Operations: []string{"video"}}
	if err := configureModel(&video, &spec, motion); err != nil || (spec.Steps != 4 || spec.Frames != 9) {
		t.Fatalf("animation default: %+v %v", spec, err)
	}
	video.Source = "arx-image:source"
	if configureModel(&video, &spec, motion) == nil {
		t.Fatal("unsupported image input accepted")
	}
	video.Source, video.Frames = "", 97
	if configureModel(&video, &spec, motion) == nil {
		t.Fatal("excess frames accepted")
	}
	image := Request{Operation: "image", Prompt: "Apple"}
	if err := configureModel(&image, &spec, models.Model{Runtime: "sd"}); err != nil || spec.Steps != 25 {
		t.Fatalf("image quality steps: %+v %v", spec, err)
	}
}

func TestModernGenerationDefaults(t *testing.T) {
	image := Request{Operation: "image", Prompt: "Apple"}
	spec := engines.Request{Steps: 4}
	if err := configureModel(&image, &spec, models.Model{Runtime: "z-image"}); err != nil || spec.Steps != 9 {
		t.Fatalf("z-image steps: %+v %v", spec, err)
	}
	speech := Request{Operation: "speech", Prompt: "Hello", Speed: 1}
	model := models.Model{Runtime: "qwen3-tts", Voices: []string{"Ryan", "Vivian"}}
	if err := configureModel(&speech, &spec, model); err != nil || spec.Voice != "Ryan" {
		t.Fatalf("qwen voice: %+v %v", spec, err)
	}
	speech.Speed = 1.5
	if configureModel(&speech, &spec, model) == nil {
		t.Fatal("unsupported speed accepted")
	}
}

func TestFastMetalLimits(t *testing.T) {
	model := models.Model{Runtime: "fastmetal", Operations: []string{"video"}}
	request := Request{Operation: "video", Prompt: "An apple on a table"}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	spec := engines.Request{}
	if err := configureModel(&request, &spec, model); err != nil || spec.Steps != 3 || spec.Frames != 25 {
		t.Fatalf("fastmetal defaults: %+v %v", spec, err)
	}
	request.Frames, request.Width, request.Height = 25, 384, 256
	if err := configureModel(&request, &spec, model); err != nil {
		t.Fatal(err)
	}
	request.Frames = 33
	if configureModel(&request, &spec, model) == nil {
		t.Fatal("excess frames accepted")
	}
	request.Frames, request.Width, request.Height = 25, 768, 512
	if configureModel(&request, &spec, model) == nil {
		t.Fatal("excess resolution accepted")
	}
	request.Width, request.Height, request.Source = 384, 256, "arx-image:source"
	if configureModel(&request, &spec, model) == nil {
		t.Fatal("unsupported image input accepted")
	}
}
