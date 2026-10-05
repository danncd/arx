package tools

import (
	"arx/internal/generation"
	permission "arx/internal/permissions"
	"context"
	"errors"
	"fmt"
)

func PrepareGeneration(ctx context.Context, call Call, factory func() (*generation.Service, error)) (Prepared, error) {
	var input generation.Request
	if err := Decode(call.Arguments, &input); err != nil {
		return Prepared{}, err
	}
	if (call.Name == "media") != (input.Operation == "combine") {
		return Prepared{}, errors.New("Unsupported tool operation")
	}
	if err := input.Validate(); err != nil {
		return Prepared{}, err
	}
	if err := ctx.Err(); err != nil {
		return Prepared{}, err
	}
	manager, err := factory()
	if err != nil {
		return Prepared{}, err
	}
	return Prepared{Action: permission.Action{Tool: call.Name, Operation: input.Operation, After: input.Prompt}, Run: func(ctx context.Context) (Result, error) {
		output, err := manager.Run(ctx, call.Conversation, call.ID, input)
		result := Result{}
		if output.ID != "" {
			result.Artifacts = append(result.Artifacts, output)
			result.Text = fmt.Sprintf("Saved %s. Media reference: arx-media:%s", output.Name, output.ID)
			if output.MIME == "image/png" {
				result.Text += fmt.Sprintf(". Show it in your reply using ![Generated image](arx-media:%s). Keep this reference unchanged. For optional visual inspection, use files operation image with arx-media:%s", output.ID, output.ID)
			}
		}
		return result, err
	}}, nil
}
