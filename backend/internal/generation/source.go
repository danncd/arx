package generation

import (
	attachments "arx/internal/media/attachments"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) attachmentSource(reference, stage, name string) (string, error) {
	store := attachments.Store{Directory: s.Directory}
	data, err := store.Read(strings.TrimPrefix(reference, "arx-image:"))
	if err != nil {
		return "", err
	}
	path := filepath.Join(stage, name)
	return path, os.WriteFile(path, data, 0600)
}

func (r Request) validateSources() error {
	if strings.Contains(r.Source, ",") {
		return errors.New(`source accepts one reference, not comma-separated IDs. For multiple images use "sources": ["arx-media:<id1>", "arx-media:<id2>"] and omit source. FLUX.2 Klein supports up to four; Z-Image Turbo and SD models support one`)
	}
	if r.Sources != nil {
		if r.Operation != "image" || r.Source != "" || len(r.Sources) == 0 || len(r.Sources) > 4 {
			return errors.New("Use one to four sources for image generation, without source")
		}
		for _, reference := range r.Sources {
			if strings.Contains(reference, ",") {
				return errors.New("Each sources array entry must contain exactly one image reference")
			}
			if strings.TrimSpace(reference) == "" {
				return errors.New("Image sources must not be empty")
			}
		}
	}
	if r.Operation == "speech" && r.Source != "" {
		return errors.New("Speech does not accept image sources")
	}
	return nil
}

func (s *Service) imageSources(request Request, stage string) ([]string, error) {
	references := request.Sources
	if request.Source != "" {
		references = []string{request.Source}
	}
	paths := make([]string, 0, len(references))
	for index, reference := range references {
		var path string
		var err error
		if strings.HasPrefix(reference, "arx-image:") {
			path, err = s.attachmentSource(reference, stage, fmt.Sprintf("source-%d.png", index))
		} else {
			artifact, resolved, resolveErr := s.Artifacts.Resolve(strings.TrimPrefix(reference, "arx-media:"))
			path, err = resolved, resolveErr
			if err == nil && artifact.MIME != "image/png" {
				err = errors.New("Image sources must point to images")
			}
		}
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
