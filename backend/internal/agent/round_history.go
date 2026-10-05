package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
)

func roundHistory(saved *session.TranscriptChunk) []provider.Message {
	message := provider.Message{Role: "assistant", Content: saved.Text, Reasoning: saved.Reasoning}
	results := []provider.Message{}
	for _, record := range saved.Tools {
		message.Calls = append(message.Calls, provider.Call{ID: record.ID, Type: "function", Function: provider.Function{Name: record.Name, Arguments: record.Arguments}})
		results = append(results, provider.ToolMessage(record.ID, historyResult(record)))
	}
	return append([]provider.Message{message}, results...)
}

func historyResult(record session.ToolRecord) string {
	if record.Status == "pending" {
		return "Tool was not run because the reply was stopped."
	}
	if record.Status != "done" {
		return "Tool execution was interrupted. Its outcome is unknown; check before retrying."
	}
	return record.Result
}
