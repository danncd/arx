package agent

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"encoding/json"
	"errors"
)

func History(chunks []session.TranscriptChunk) ([]provider.Message, error) {
	var order []string
	messages := map[string]*session.TranscriptChunk{}
	for _, chunk := range chunks {
		if chunk.Role != "user" && chunk.Role != "assistant" {
			continue
		}
		held := messages[chunk.ID]
		if held == nil {
			held = &session.TranscriptChunk{ID: chunk.ID, Role: chunk.Role}
			messages[chunk.ID] = held
			order = append(order, chunk.ID)
		}
		if chunk.Offset != len(held.Text) || chunk.ReasoningAt != len(held.Reasoning) {
			return nil, errors.New("Session history is incomplete")
		}
		if chunk.Runtime != nil {
			held.Runtime = chunk.Runtime
		}
		if len(chunk.Images) > 0 {
			held.Images = chunk.Images
		}
		if chunk.Interruption != "" {
			held.Interruption = chunk.Interruption
		}
		held.Round = held.Round || chunk.Round
		held.Text += chunk.Text
		held.Reasoning += chunk.Reasoning
		for _, record := range chunk.Tools {
			if record.Status == "preparing" {
				continue
			}
			found := false
			for index, existing := range held.Tools {
				if existing.ID == record.ID {
					held.Tools[index] = record
					found = true
					break
				}
			}
			if !found {
				held.Tools = append(held.Tools, record)
			}
		}
	}
	out := []provider.Message{}
	for _, id := range order {
		held := messages[id]
		if held.Role == "user" {
			text := held.Text
			if held.Runtime != nil {
				runtime, _ := json.Marshal(held.Runtime)
				text = "Runtime context: " + string(runtime) + "\n\n" + text
			}
			out = append(out, provider.Message{Role: "user", Content: text, Images: held.Images})
			continue
		}
		if held.Round {
			out = append(out, roundHistory(held)...)
			continue
		}
		textAt, thoughtAt := 0, 0
		for index := 0; index < len(held.Tools); {
			record := held.Tools[index]
			end, thoughtEnd := record.Offset, record.ThoughtAt
			if end < textAt || end > len(held.Text) || thoughtEnd < thoughtAt || thoughtEnd > len(held.Reasoning) {
				return nil, errors.New("Saved tool positions are invalid")
			}
			message := provider.Message{Role: "assistant", Content: held.Text[textAt:end], Reasoning: held.Reasoning[thoughtAt:thoughtEnd]}
			results := []provider.Message{}
			for index < len(held.Tools) && held.Tools[index].Offset == end && held.Tools[index].ThoughtAt == thoughtEnd {
				record = held.Tools[index]
				index++
				if record.ID == "" || record.Name == "" || record.Arguments == "" {
					continue
				}
				message.Calls = append(message.Calls, provider.Call{ID: record.ID, Type: "function", Function: provider.Function{Name: record.Name, Arguments: record.Arguments}})
				result := historyResult(record)
				results = append(results, provider.ToolMessage(record.ID, result))
			}
			if len(message.Calls) > 0 || message.Content != "" || message.Reasoning != "" {
				out = append(out, message)
			}
			out = append(out, results...)
			textAt, thoughtAt = end, thoughtEnd
		}
		if textAt < len(held.Text) || thoughtAt < len(held.Reasoning) {
			out = append(out, provider.Message{Role: "assistant", Content: held.Text[textAt:], Reasoning: held.Reasoning[thoughtAt:]})
		}
		if held.Interruption != "" {
			out = append(out, provider.Message{Role: "assistant", Content: held.Interruption})
		}
	}
	return out, nil
}
