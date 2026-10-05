package deepseek

import (
	provider "arx/internal/inference"
	"errors"
)

func SupportsVision(model string) bool {
	return model == "deepseek-flash" || model == "deepseek-v4-flash-vision-exp"
}

func (c *Client) messages(input provider.Request) ([]any, error) {
	images, err := c.imagePayloads(input)
	if err != nil {
		return nil, err
	}
	messages := make([]any, 0, len(input.Messages))
	for _, message := range input.Messages {
		if len(message.Images) == 0 {
			messages = append(messages, message)
			continue
		}
		if !SupportsVision(input.Model) {
			return nil, errors.New("Choose DeepSeek Flash to use images")
		}
		if (message.Role != "user" && message.Role != "tool") || c.ReadImage == nil {
			return nil, errors.New("Image input is unavailable")
		}
		parts := []any{}
		if message.Content != "" {
			parts = append(parts, map[string]any{"type": "text", "text": message.Content})
		}
		for _, image := range message.Images {
			parts = append(parts, map[string]any{"type": "text", "text": "Saved image: arx-image:" + image.ID + " (" + image.Name + ")"})
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": images[image.ID]}})
		}
		encoded := map[string]any{"role": message.Role, "content": parts}
		if message.Role == "tool" {
			encoded["tool_call_id"] = message.CallID
		}
		messages = append(messages, encoded)
	}
	return messages, nil
}
