package models

type Thinking struct {
	Efforts        []string `json:"efforts"`
	DefaultEffort  string   `json:"defaultEffort"`
	DefaultEnabled bool     `json:"defaultEnabled"`
	CanDisable     bool     `json:"canDisable"`
	Source         string   `json:"source"`
}

type Info struct {
	ContextReason    string    `json:"contextReason,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	Tools            *bool     `json:"tools,omitempty"`
	TrainedContext   int       `json:"trainedContext,omitempty"`
	CapabilitySource string    `json:"capabilitySource,omitempty"`
	Vision           bool      `json:"vision"`
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ContextWindow    int       `json:"contextWindow"`
	MaxOutputTokens  int       `json:"maxOutputTokens"`
	Thinking         *Thinking `json:"thinking,omitempty"`
}
