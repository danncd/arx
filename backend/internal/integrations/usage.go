package integrations

import (
	"encoding/json"
	"regexp"
	"time"
)

var Identifier = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type Activity struct {
	Time         string `json:"time"`
	Operation    string `json:"operation"`
	Conversation string `json:"conversation,omitempty"`
	Status       string `json:"status"`
}
type Usage struct {
	Count    int        `json:"count"`
	LastUsed string     `json:"lastUsed,omitempty"`
	Activity []Activity `json:"activity"`
}

func (u *Usage) Record(operation, conversation, status string) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	u.Count++
	u.LastUsed = now
	u.Activity = append([]Activity{{now, operation, conversation, status}}, u.Activity...)
	if len(u.Activity) > 100 {
		u.Activity = u.Activity[:100]
	}
}
func Copy[T any](value T) T {
	data, _ := json.Marshal(value)
	var out T
	_ = json.Unmarshal(data, &out)
	return out
}
