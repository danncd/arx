package inference

import (
	attachment "arx/internal/media/attachments"
	session "arx/internal/sessions"
	tool "arx/internal/tools"
	context "context"
	json "encoding/json"
	errors "errors"
	"net/http"
	"time"
)

var ErrOutputLimit = errors.New("Reply reached the model’s output limit")

var ErrContextLength = errors.New("The conversation exceeds the model context limit")

type Message struct {
	Images    []attachment.Image `json:"images,omitempty"`
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	Reasoning string             `json:"reasoning_content,omitempty"`
	Calls     []Call             `json:"tool_calls,omitempty"`
	CallID    string             `json:"tool_call_id,omitempty"`
}

type Call struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Request struct {
	MaxOutputTokens int
	Model           string
	Effort          string
	Messages        []Message
	Tools           []tool.Definition
}

type ToolDelta struct {
	Index int
	Call  Call
}

type Delta struct {
	Tool      *ToolDelta
	Text      string
	Reasoning string
}

type Response struct {
	Message Message
	Usage   *session.ChunkUsage
}

type AuthenticatedCompleter interface {
	Complete(context.Context, string, Request, func(Delta) error) (Response, error)
}

func ToolMessage(id, content string) Message {
	message := Message{Role: "tool", CallID: id, Content: content}
	var result struct {
		Images []attachment.Image `json:"images"`
		Failed bool               `json:"failed"`
	}
	if json.Unmarshal([]byte(content), &result) == nil && !result.Failed {
		message.Images = result.Images
	}
	return message
}

func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
