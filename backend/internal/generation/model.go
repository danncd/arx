package generation

import (
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	"errors"
	"slices"
	"strings"
)

func configureModel(request *Request, specification *engines.Request, model models.Model) error {
	specification.Runtime = model.Runtime
	policy := models.CapabilitiesFor(model)
	specification.Steps = policy.Steps
	if request.Operation == "video" && request.Frames == 0 {
		request.Frames = policy.DefaultFrames
	}
	specification.Frames = request.Frames
	if policy.MaxFrames > 0 && (request.Frames > policy.MaxFrames || request.Width*request.Height > policy.MaxPixels) {
		return errors.New(policy.BoundsError())
	}
	if policy.FixedSpeed && request.Speed != 1 {
		return errors.New("Qwen3 speech currently supports normal playback speed only")
	}
	if len(request.Sources) > 1 && len(request.Sources) > ImageReferenceLimit(model) {
		return errors.New("The selected model accepts only one reference image. Use one source or select FLUX.2 Klein for multiple references")
	}
	if request.Source != "" || len(request.Sources) > 0 {
		capability := "image-edit"
		if request.Operation == "video" {
			capability = "image-to-video"
		}
		if !slices.Contains(model.Operations, capability) {
			return errors.New("The selected model does not support an image input")
		}
	}
	if request.Operation == "speech" && request.Voice == "" && len(model.Voices) > 0 {
		request.Voice = model.Voices[0]
	}
	specification.Voice = request.Voice
	if request.Operation == "speech" && !slices.Contains(model.Voices, request.Voice) {
		return errors.New("Choose a voice supported by this model: " + strings.Join(model.Voices, ", "))
	}
	return nil
}

func ImageReferenceLimit(model models.Model) int { return models.CapabilitiesFor(model).MaxReferences }
