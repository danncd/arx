package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
)

func (r *recorder) toolDelta(delta provider.ToolDelta) bool {
	if r.toolPositions == nil {
		r.toolPositions = map[int]int{}
	}
	position, exists := r.toolPositions[delta.Index]
	if !exists {
		position = len(r.message.Tools)
		r.toolPositions[delta.Index] = position
		r.message.Tools = append(r.message.Tools, session.ToolRecord{Offset: len(r.message.Text), ThoughtAt: len(r.message.Reasoning), Status: "preparing"})
	}
	record := &r.message.Tools[position]
	record.ID = delta.Call.ID
	record.Name = delta.Call.Function.Name
	record.Arguments = delta.Call.Function.Arguments
	return !exists
}

func (r *recorder) completeTools(calls []provider.Call) {
	records := make([]session.ToolRecord, 0, len(calls))
	for index, call := range calls {
		record := session.ToolRecord{Offset: len(r.message.Text), ThoughtAt: len(r.message.Reasoning)}
		if position, exists := r.toolPositions[index]; exists {
			record = r.message.Tools[position]
		}
		record.ID, record.Name, record.Arguments, record.Status = call.ID, call.Function.Name, call.Function.Arguments, "pending"
		records = append(records, record)
	}
	r.message.Tools = records
}
