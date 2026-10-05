package compaction

import (
	provider "arx/internal/inference"
	"encoding/json"
	"fmt"
	"strings"
)

type contextLimit string

func (e contextLimit) Error() string        { return string(e) }
func (e contextLimit) Is(target error) bool { return target == provider.ErrContextLength }

func (c *Compactor) Recover(kind string) bool {
	if !c.AutoRecover || c.recoveries >= 3 {
		return false
	}
	c.recoveries++
	c.compactions = 0
	c.Force = false
	if kind == "context" {
		c.Compression++
	}
	if c.Recovery != nil {
		c.Recovery(kind, c.recoveries)
	}
	return true
}
func (c *Compactor) Progress() { c.compactions = 0 }

func (c *Compactor) Continue() {
	c.Continuation++
	c.compactions = 0
}

func (c *Compactor) compress(messages []provider.Message) []provider.Message {
	if c.Compression == 0 {
		return messages
	}
	out := append([]provider.Message(nil), messages...)
	limit := 16000
	switch c.Compression {
	case 2:
		limit = 6000
	case 3:
		limit = 1500
	}
	for i := range out {
		if out[i].Role == "assistant" {
			out[i].Reasoning = ""
		}
		if out[i].Role != "tool" {
			continue
		}
		var references strings.Builder
		keep := len(out[i].Images)
		if c.Compression >= 2 {
			keep = min(keep, max(0, 4-c.Compression*2))
		}
		for _, image := range out[i].Images[keep:] {
			fmt.Fprintf(&references, "\nSaved image available for a targeted read: arx-image:%s %s", image.ID, image.Name)
		}
		out[i].Images = out[i].Images[:keep]
		if len(out[i].Content) <= limit && references.Len() == 0 {
			continue
		}
		text := out[i].Content
		failed := false
		var result struct {
			Text   string `json:"text"`
			Failed bool   `json:"failed"`
		}
		if json.Unmarshal([]byte(text), &result) == nil {
			failed = result.Failed
			if result.Text != "" {
				text = result.Text
			}
		}
		body, _ := json.Marshal(map[string]any{
			"text": compactText(text, limit) + references.String(), "failed": failed, "truncated": true,
			"context_recovery": "This context copy omits large saved tool output. The full result remains in chat history. Use targeted file/section/paginated reads when omitted details are required; completed actions must not be repeated.",
		})
		out[i].Content = string(body)
	}
	return out
}
