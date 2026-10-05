package generation

import (
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	"arx/internal/media/artifacts"
	attachments "arx/internal/media/attachments"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceValidationAndModelLimits(t *testing.T) {
	for _, request := range []Request{
		{Operation: "image", Source: "one", Sources: []string{"two"}},
		{Operation: "video", Sources: []string{"one"}},
		{Operation: "combine", Source: "video", Audio: "audio", Sources: []string{"one"}},
		{Operation: "image", Sources: []string{}},
		{Operation: "image", Sources: []string{" "}},
		{Operation: "image", Sources: []string{"1", "2", "3", "4", "5"}},
	} {
		request.Prompt = "Reference test"
		if request.Validate() == nil {
			t.Fatalf("accepted invalid request: %+v", request)
		}
	}
	request := Request{Operation: "image", Prompt: "Preserve the characters", Sources: []string{"one", "two"}}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, runtime := range []string{"flux2", "z-image", "sd", "sd-turbo"} {
		model := models.Model{Runtime: runtime, Operations: []string{"image", "image-edit"}}
		err := configureModel(&request, &engines.Request{}, model)
		if (err == nil) != (runtime == "flux2") {
			t.Fatalf("%s: %v", runtime, err)
		}
	}
	request.Sources = []string{"one"}
	if err := configureModel(&request, &engines.Request{}, models.Model{Runtime: "z-image", Operations: []string{"image-edit"}}); err != nil {
		t.Fatal(err)
	}
}

func TestReferencesKeepOrderAndSeparateAttachmentFiles(t *testing.T) {
	directory, stage := t.TempDir(), t.TempDir()
	service := &Service{Directory: directory, Artifacts: artifacts.Store{Directory: directory}}
	references := []string{}
	expected := [][]byte{}
	for _, red := range []uint8{70, 180} {
		picture := image.NewNRGBA(image.Rect(0, 0, 2, 2))
		picture.Set(0, 0, color.NRGBA{R: red, A: 255})
		var data bytes.Buffer
		if err := png.Encode(&data, picture); err != nil {
			t.Fatal(err)
		}
		saved, err := (attachments.Store{Directory: directory}).Save(data.Bytes(), "test.png")
		if err != nil {
			t.Fatal(err)
		}
		normalized, err := (attachments.Store{Directory: directory}).Read(saved.ID)
		if err != nil {
			t.Fatal(err)
		}
		references = append(references, "arx-image:"+saved.ID)
		expected = append(expected, normalized)
	}
	original := filepath.Join(stage, "generated.png")
	if err := os.WriteFile(original, expected[0], 0600); err != nil {
		t.Fatal(err)
	}
	saved, err := service.Artifacts.Save(original, artifacts.Artifact{Name: "generated.png", MIME: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	references = append(references, "arx-media:"+saved.ID)
	expected = append(expected, expected[0])
	paths, err := service.imageSources(Request{Sources: references}, stage)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 || paths[0] == paths[1] {
		t.Fatalf("paths: %v", paths)
	}
	for index, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(data, expected[index]) {
			t.Fatalf("reference %d changed: %v", index, err)
		}
	}
	if _, err := service.imageSources(Request{Sources: []string{"arx-media:../../etc/passwd"}}, stage); err == nil {
		t.Fatal("unsafe reference accepted")
	}
}

func TestCommaSeparatedSourceExplainsArraySyntax(t *testing.T) {
	request := Request{Operation: "image", Prompt: "Combine", Source: "arx-media:one,arx-media:two"}
	err := request.Validate()
	if err == nil || !strings.Contains(err.Error(), `"sources": [`) {
		t.Fatalf("unhelpful error: %v", err)
	}
}
