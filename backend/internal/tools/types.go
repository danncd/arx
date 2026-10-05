package tools

import (
	"arx/internal/media/artifacts"
	attachment "arx/internal/media/attachments"
	permission "arx/internal/permissions"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
)

const MaxOutput = 64 << 10

type Call struct {
	Conversation string          `json:"-"`
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Arguments    json.RawMessage `json:"arguments"`
}

type Source struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Snippet string `json:"snippet,omitempty"`
}

type Result struct {
	Artifacts []artifacts.Artifact `json:"artifacts,omitempty"`
	Images    []attachment.Image   `json:"images,omitempty"`
	ImageData []byte               `json:"-"`
	ImageName string               `json:"-"`
	Sources   []Source             `json:"sources,omitempty"`
	Text      string               `json:"text"`
	Failed    bool                 `json:"failed"`
	Truncated bool                 `json:"truncated"`
	ExitCode  *int                 `json:"exitCode,omitempty"`
}

type Prepared struct {
	Metadata        bool
	RequireApproval bool
	Denied          func(context.Context) error
	Action          permission.Action
	Run             func(context.Context) (Result, error)
	Close           func()
}

func Decode(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New("Invalid tool arguments")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("Invalid tool arguments")
	}
	return nil
}

func Output(text string) Result {
	if len(text) <= MaxOutput {
		return Result{Text: text}
	}
	cut := MaxOutput
	for cut > 0 && text[cut]&0xc0 == 0x80 {
		cut--
	}
	return Result{Text: text[:cut], Truncated: true}
}
