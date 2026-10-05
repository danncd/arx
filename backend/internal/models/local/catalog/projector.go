package catalog

import (
	download "arx/internal/platform/download"
	"strings"
)

func chooseProjector(files []download.File) *download.File {
	if len(files) == 1 {
		return &files[0]
	}
	var selected *download.File
	family := ""
	for i := range files {
		name := strings.ToLower(files[i].Name)
		normalized := name
		for _, suffix := range []string{"-f16.gguf", "-bf16.gguf", "-f32.gguf"} {
			normalized = strings.TrimSuffix(normalized, suffix)
		}
		if normalized == name || (family != "" && family != normalized) {
			return nil
		}
		family = normalized
		if strings.HasSuffix(name, "-f16.gguf") {
			selected = &files[i]
		}
	}
	return selected
}
