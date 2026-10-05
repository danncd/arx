package bash

import (
	tool "arx/internal/tools"
	"bytes"
	"strings"
	"sync"
)

type boundedOutput struct {
	mutex     sync.Mutex
	data      bytes.Buffer
	truncated bool
}

func (b *boundedOutput) Write(body []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	available := tool.MaxOutput - b.data.Len()
	count := min(available, len(body))
	b.data.Write(body[:count])
	if count < len(body) {
		b.truncated = true
	}
	return len(body), nil
}
func (b *boundedOutput) Result() tool.Result {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return tool.Result{Text: strings.ToValidUTF8(b.data.String(), "�"), Truncated: b.truncated}
}
