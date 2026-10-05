package catalog

import (
	download "arx/internal/platform/download"
	"testing"
)

func TestProjectorQuantizationsAreNotDifferentModels(t *testing.T) {
	files := []download.File{{Name: "mmproj-BF16.gguf"}, {Name: "mmproj-F16.gguf"}, {Name: "mmproj-F32.gguf"}}
	if selected := chooseProjector(files); selected == nil || selected.Name != "mmproj-F16.gguf" {
		t.Fatal("missing standard precision projector")
	}
	files = append(files, download.File{Name: "mmproj-other-F16.gguf"})
	if chooseProjector(files) != nil {
		t.Fatal("ambiguous model families must not be guessed")
	}
}
