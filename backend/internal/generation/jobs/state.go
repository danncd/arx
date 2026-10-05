package jobs

import (
	"arx/internal/media/artifacts"
	"time"
)

type Job struct {
	ID           string              `json:"id"`
	Conversation string              `json:"conversation"`
	ToolCall     string              `json:"toolCall"`
	Operation    string              `json:"operation"`
	Model        string              `json:"model"`
	State        string              `json:"state"`
	Progress     float64             `json:"progress"`
	Detail       string              `json:"detail"`
	Error        string              `json:"error,omitempty"`
	Output       *artifacts.Artifact `json:"output,omitempty"`
	Updated      time.Time           `json:"updated"`
}
