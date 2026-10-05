package sessions

import attachment "arx/internal/media/attachments"

type RuntimeContext struct {
	Directory string `json:"directory"`
	Date      string `json:"date"`
}

type TranscriptChunk struct {
	Interruption string             `json:"interruption,omitempty"`
	Title        *string            `json:"session_title,omitempty"`
	Provider     string             `json:"provider,omitempty"`
	Model        string             `json:"model,omitempty"`
	Images       []attachment.Image `json:"images,omitempty"`
	Runtime      *RuntimeContext    `json:"runtime,omitempty"`
	Round        bool               `json:"provider_round,omitempty"`
	Context      *ContextUsage      `json:"context,omitempty"`
	Conversation string             `json:"conversation"`
	ID           string             `json:"id"`
	Role         string             `json:"role"`
	Offset       int                `json:"offset"`
	Text         string             `json:"text"`
	Reasoning    string             `json:"reasoning"`
	ReasoningAt  int                `json:"reasoning_offset"`
	Tools        []ToolRecord       `json:"tools"`
	Usage        *ChunkUsage        `json:"usage"`
	Status       string             `json:"status"`
	Failed       bool               `json:"failed"`
	Reason       string             `json:"reason"`
	At           string             `json:"at"`
}

type ToolRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Offset    int    `json:"offset"`
	ThoughtAt int    `json:"reasoning_offset"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	Added     int    `json:"added"`
	Removed   int    `json:"removed"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
	Failed    bool   `json:"failed"`
	Truncated bool   `json:"truncated"`
}

type GenerationUsage struct {
	Tokens  int     `json:"tokens"`
	Seconds float64 `json:"seconds"`
}

type ChunkUsage struct {
	Generation   *GenerationUsage `json:"generation,omitempty"`
	CacheUnknown bool             `json:"cacheUnknown,omitempty"`
	Input        int              `json:"input"`
	Cached       int              `json:"cached"`
	Output       int              `json:"output"`
	Reasoning    int              `json:"reasoning"`
}

type Conversation struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Updated string `json:"updated"`
	Status  string `json:"status"`
}

type ContextUsage struct {
	Model    string `json:"model"`
	Effort   string `json:"effort"`
	Messages int    `json:"messages"`
	Digest   string `json:"digest"`
	Input    int    `json:"input"`
}

type Compaction struct {
	User    int    `json:"user,omitempty"`
	Target  int    `json:"target,omitempty"`
	Pending bool   `json:"pending,omitempty"`
	Version int    `json:"version"`
	Through int    `json:"through"`
	Digest  string `json:"digest"`
	Summary string `json:"summary"`
}
