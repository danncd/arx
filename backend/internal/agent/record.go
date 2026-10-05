package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"time"
)

type recorder struct {
	dirty          bool
	toolPositions  map[int]int
	message        session.TranscriptChunk
	savedTools     map[string]session.ToolRecord
	savedText      int
	savedReasoning int
	last           time.Time
	save           func(session.TranscriptChunk) error
	emit           func(session.TranscriptChunk)
}

func (r *recorder) delta(delta provider.Delta) error {
	r.message.Text += delta.Text
	r.message.Reasoning += delta.Reasoning
	r.dirty = true
	if delta.Tool != nil && r.toolDelta(*delta.Tool) {
		return r.flush()
	}
	if time.Since(r.last) < 60*time.Millisecond {
		return nil
	}
	return r.flush()
}

func (r *recorder) flush() error {
	chunk := r.message
	chunk.Tools = nil
	for _, record := range r.message.Tools {
		if record.Status == "preparing" && r.message.Status == "running" {
			continue
		}
		if r.savedTools[record.ID] != record {
			chunk.Tools = append(chunk.Tools, record)
		}
	}
	chunk.Offset, chunk.ReasoningAt = r.savedText, r.savedReasoning
	chunk.Text = chunk.Text[r.savedText:]
	chunk.Reasoning = chunk.Reasoning[r.savedReasoning:]
	chunk.At = time.Now().UTC().Format(time.RFC3339Nano)
	if err := r.save(chunk); err != nil {
		return err
	}
	if r.savedTools == nil {
		r.savedTools = map[string]session.ToolRecord{}
	}
	for _, record := range chunk.Tools {
		r.savedTools[record.ID] = record
	}
	r.savedText, r.savedReasoning, r.last = len(r.message.Text), len(r.message.Reasoning), time.Now()
	r.message.At = chunk.At
	r.dirty = false
	snapshot := r.message
	snapshot.Tools = append([]session.ToolRecord(nil), r.message.Tools...)
	r.emit(snapshot)
	return nil
}
