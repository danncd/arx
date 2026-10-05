package chatcompletions

import (
	provider "arx/internal/inference"
	"encoding/base64"
	"errors"
)

func (c *Client) messages(input provider.Request) ([]any, error) {
	result := make([]any, 0, len(input.Messages))
	total := 0
	var pending []any
	flush := func() {
		if len(pending) > 0 {
			result = append(result, map[string]any{"role": "user", "content": pending})
			pending = nil
		}
	}
	for _, message := range input.Messages {
		if message.Role != "tool" {
			flush()
		}
		value := map[string]any{"role": message.Role, "content": message.Content}
		if message.CallID != "" {
			value["tool_call_id"] = message.CallID
		}
		if len(message.Calls) > 0 {
			value["tool_calls"] = message.Calls
		}
		if len(message.Images) > 0 {
			if !c.Info.Vision {
				return nil, errors.New("This conversation contains images. Select a vision-capable model to continue")
			}
			if c.ReadImage == nil {
				return nil, errors.New("Saved image is unavailable")
			}
			parts := []any{map[string]string{"type": "text", "text": message.Content}}
			for _, image := range message.Images {
				data, err := c.ReadImage(image.ID)
				if err != nil {
					return nil, err
				}
				total += len(data)
				if total > 18<<20 {
					return nil, provider.ErrContextLength
				}
				parts = append(parts, map[string]string{"type": "text", "text": "Saved image: arx-image:" + image.ID}, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)}})
			}
			if message.Role == "tool" {
				result = append(result, value)
				pending = append(pending, parts...)
				continue
			}
			value["content"] = parts
		}
		result = append(result, value)
	}
	flush()
	return result, nil
}
