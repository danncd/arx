package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerationSchemaExplainsReferenceArguments(t *testing.T) {
	var schema struct {
		Properties map[string]struct {
			Description string `json:"description"`
		} `json:"properties"`
	}
	definition := generationDefinitions()[0]
	if err := json.Unmarshal(definition.Parameters, &schema); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(schema.Properties["source"].Description, "Never comma-separate") || !strings.Contains(schema.Properties["sources"].Description, "FLUX.2 Klein") {
		t.Fatal("missing reference guidance")
	}
}
