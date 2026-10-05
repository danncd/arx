package generation

import (
	models "arx/internal/generation/models"
	"fmt"
	"strings"
)

func Describe(state models.State) string {
	lines := []string{"Generation models selected for this turn:"}
	for _, category := range []string{"image", "video", "speech"} {
		selected := state.Defaults[category]
		var entry *models.Entry
		for index := range state.Models {
			candidate := &state.Models[index]
			if candidate.Model.ID == selected && candidate.Status == "installed" {
				entry = candidate
				break
			}
		}
		if entry == nil {
			lines = append(lines, category+": not configured.")
			continue
		}
		detail := fmt.Sprintf("%s: %s. Supported operations: %s.", category, entry.Model.Name, strings.Join(entry.Model.Operations, ", "))
		if category == "image" {
			detail += fmt.Sprintf(" Maximum reference images: %d. One image uses source; multiple images use the sources JSON array, never comma-separated text. Do not silently discard requested references if they exceed this limit.", ImageReferenceLimit(entry.Model))
		}
		lines = append(lines, detail)
	}
	return strings.Join(lines, "\n")
}
