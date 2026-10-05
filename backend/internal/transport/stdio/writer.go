package stdio

import (
	"encoding/json"
	"io"
	"sync"
)

type writer struct {
	mutex   sync.Mutex
	encoder *json.Encoder
}

func (w *writer) send(value any) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.encoder.Encode(value)
}

func newWriter(output io.Writer) *writer {
	return &writer{encoder: json.NewEncoder(output)}
}
