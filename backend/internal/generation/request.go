package generation

import (
	"errors"
	"strings"
	"unicode/utf8"
)

type Request struct {
	Operation string   `json:"operation"`
	Prompt    string   `json:"prompt,omitempty"`
	Source    string   `json:"source,omitempty"`
	Sources   []string `json:"sources,omitempty"`
	Audio     string   `json:"audio,omitempty"`
	Width     int      `json:"width,omitempty"`
	Height    int      `json:"height,omitempty"`
	Seed      int64    `json:"seed,omitempty"`
	Voice     string   `json:"voice,omitempty"`
	Speed     float64  `json:"speed,omitempty"`
	Frames    int      `json:"frames,omitempty"`
}

func (r *Request) Validate() error {
	if r.Operation != "image" && r.Operation != "video" && r.Operation != "speech" && r.Operation != "combine" {
		return errors.New("Unsupported generation operation")
	}
	if err := r.validateSources(); err != nil {
		return err
	}
	if r.Operation == "combine" {
		if r.Source == "" || r.Audio == "" {
			return errors.New("Choose a saved video and speech output to combine")
		}
		return nil
	}
	if strings.TrimSpace(r.Prompt) == "" || utf8.RuneCountInString(r.Prompt) > 16000 {
		return errors.New("Provide a prompt up to 16000 characters")
	}
	if r.Width == 0 {
		r.Width = 512
		if r.Operation == "video" {
			r.Width = 384
		}
	}
	if r.Height == 0 {
		r.Height = 512
		if r.Operation == "video" {
			r.Height = 256
		}
	}
	if r.Width < 128 || r.Width > 1024 || r.Height < 128 || r.Height > 1024 || r.Width%32 != 0 || r.Height%32 != 0 {
		return errors.New("Choose dimensions from 128 to 1024 in multiples of 32")
	}
	if r.Speed == 0 {
		r.Speed = 1
	}
	if r.Speed < 0.5 || r.Speed > 2 {
		return errors.New("Speech speed must be between 0.5 and 2")
	}
	if r.Frames != 0 && (r.Frames < 9 || r.Frames > 97 || (r.Frames-1)%8 != 0) {
		return errors.New("Video frames must be 9 to 97, in steps of 8")
	}
	if r.Operation == "video" && (r.Width*r.Height > 768*512 || r.Width*r.Height*r.Frames > 25_000_000) {
		return errors.New("Reduce video dimensions or frames to fit the local runtime")
	}
	return nil
}
