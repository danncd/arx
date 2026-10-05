package context

import (
	provider "arx/internal/inference"
	"encoding/json"
)

func Text(text string) int {
	ascii, other := 0, 0
	for _, char := range text {
		if char < 128 {
			ascii++
		} else {
			other += 2
		}
	}
	return (ascii+2)/3 + other
}

func Messages(messages []provider.Message) int {
	total := 0
	for _, message := range messages {
		total += 1024 * len(message.Images)
		total += 16 + Text(message.Role) + Text(message.Content) + Text(message.Reasoning) + Text(message.CallID)
		if len(message.Calls) > 0 {
			calls, _ := json.Marshal(message.Calls)
			total += Text(string(calls))
		}
	}
	return total
}

func Estimate(request provider.Request) int {
	total := 256 + Messages(request.Messages)
	if len(request.Tools) > 0 {
		tools, err := json.Marshal(request.Tools)
		if err != nil {
			return int(^uint(0) >> 1)
		}
		total += Text(string(tools)) + 16*len(request.Tools)
	}
	return total
}
