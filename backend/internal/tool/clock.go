package tool

import (
	"context"
	"encoding/json"
	"time"
)

var Clock = Tool{
	Name:        "current_time",
	Description: "Get the current local date and time.",
	Schema:      json.RawMessage(`{"type":"object","properties":{}}`),
	Run: func(_ context.Context, _ json.RawMessage) (string, error) {
		return time.Now().Format("Monday, January 2, 2006 15:04:05 MST"), nil
	},
}
