package generation

import (
	models "arx/internal/generation/models"
	"strings"
	"testing"
)

func TestGenerationGuidanceTracksSelectedModelAndLimit(t *testing.T) {
	state := models.State{Defaults: map[string]string{"image": "flux"}, Models: []models.Entry{
		{Status: "installed", Model: models.Model{ID: "flux", Name: "FLUX.2 Klein", Runtime: "flux2", Operations: []string{"image", "image-edit"}}},
		{Status: "installed", Model: models.Model{ID: "z", Name: "Z-Image Turbo", Runtime: "z-image", Operations: []string{"image", "image-edit"}}},
	}}
	guidance := Describe(state)
	if !strings.Contains(guidance, "image: FLUX.2 Klein") || !strings.Contains(guidance, "Maximum reference images: 4") || !strings.Contains(guidance, "video: not configured") {
		t.Fatal(guidance)
	}
	state.Defaults["image"] = "z"
	guidance = Describe(state)
	if !strings.Contains(guidance, "image: Z-Image Turbo") || !strings.Contains(guidance, "Maximum reference images: 1") || strings.Contains(guidance, "FLUX.2 Klein") {
		t.Fatal(guidance)
	}
	state.Models[1].Status = "downloading"
	if !strings.Contains(Describe(state), "image: not configured") {
		t.Fatal("unavailable model advertised")
	}
}
