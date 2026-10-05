package generation

import (
	download "arx/internal/platform/download"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
)

type Model struct {
	Capabilities  *Capabilities   `json:"capabilities,omitempty"`
	Experimental  bool            `json:"experimental,omitempty"`
	Runtime       string          `json:"runtime"`
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Company       string          `json:"company"`
	Category      string          `json:"category"`
	Repository    string          `json:"repository"`
	Revision      string          `json:"revision"`
	Operations    []string        `json:"operations"`
	Voices        []string        `json:"voices"`
	MinimumMemory int64           `json:"minimumMemory"`
	Files         []download.File `json:"files"`
	Size          int64           `json:"size"`
}

func Catalog(assets string) ([]Model, error) {
	data, err := os.ReadFile(filepath.Join(assets, "catalog.json"))
	if err != nil {
		return nil, err
	}
	var models []Model
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for i := range models {
		model := &models[i]
		policy := CapabilitiesFor(*model)
		model.Capabilities = &policy
		if model.ID == "" || !filepath.IsLocal(model.ID) || filepath.Base(model.ID) != model.ID || model.ID == "." || seen[model.ID] || len(model.Files) == 0 {
			return nil, errors.New("Invalid generation model catalogue")
		}
		seen[model.ID] = true
		for _, file := range model.Files {
			if !filepath.IsLocal(file.Name) || file.Size <= 0 {
				return nil, errors.New("Invalid model dependency")
			}
			model.Size += file.Size
		}
	}
	return models, nil
}

type Capabilities struct {
	Steps         int  `json:"steps"`
	DefaultFrames int  `json:"defaultFrames"`
	MaxFrames     int  `json:"maxFrames"`
	MaxPixels     int  `json:"maxPixels"`
	MaxReferences int  `json:"maxReferences"`
	FixedSpeed    bool `json:"fixedSpeed"`
	boundsError   string
}

var runtimePolicy = map[string]Capabilities{
	"sd":          {Steps: 25},
	"z-image":     {Steps: 9},
	"flux2":       {Steps: 4, MaxReferences: 4},
	"wan-cpp":     {Steps: 30, DefaultFrames: 25, MaxFrames: 25, MaxPixels: 384 * 256, boundsError: "Wan supports up to 25 frames and 384 by 256 pixels in Arx"},
	"fastmetal":   {Steps: 3, DefaultFrames: 25, MaxFrames: 25, MaxPixels: 384 * 256, boundsError: "FastMetal supports up to 25 frames and 384 by 256 pixels in Arx"},
	"animatediff": {Steps: 4, DefaultFrames: 9, MaxFrames: 17, MaxPixels: 384 * 256, boundsError: "AnimateDiff supports up to 17 frames and 384 by 256 pixels in Arx"},
	"qwen3-tts":   {FixedSpeed: true},
}

func CapabilitiesFor(model Model) Capabilities {
	policy := runtimePolicy[model.Runtime]
	if policy.Steps == 0 {
		policy.Steps = 4
		if model.Category == "video" || slices.Contains(model.Operations, "video") {
			policy.Steps = 30
		}
	}
	if policy.DefaultFrames == 0 {
		policy.DefaultFrames = 25
	}
	if slices.Contains(model.Operations, "image-edit") {
		if policy.MaxReferences == 0 {
			policy.MaxReferences = 1
		}
	} else {
		policy.MaxReferences = 0
	}
	return policy
}
func (p Capabilities) BoundsError() string { return p.boundsError }
