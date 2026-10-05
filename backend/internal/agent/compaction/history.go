package compaction

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

func digest(messages []provider.Message) string {
	body, _ := json.Marshal(messages)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func boundaries(messages []provider.Message) ([]int, error) {
	var ends []int
	for index := 0; index < len(messages); {
		message := messages[index]
		if message.Role == "tool" {
			return nil, errors.New("Tool history is incomplete")
		}
		index++
		pending := map[string]bool{}
		for _, call := range message.Calls {
			if call.ID == "" || pending[call.ID] {
				return nil, errors.New("Tool history has invalid call IDs")
			}
			pending[call.ID] = true
		}
		for len(pending) > 0 {
			if index >= len(messages) || messages[index].Role != "tool" || !pending[messages[index].CallID] {
				return nil, errors.New("Tool history is incomplete")
			}
			delete(pending, messages[index].CallID)
			index++
		}
		ends = append(ends, index)
	}
	return ends, nil
}

func Valid(state session.Compaction, history []provider.Message) bool {
	if (state.Version != 1 && state.Version != 2) || state.Through <= 0 || state.Through > len(history) || state.Summary == "" || len(state.Summary) > 64<<10 {
		return false
	}
	if state.Through < len(history) && history[state.Through].Role == "tool" {
		return false
	}
	if state.User < 0 || state.User > state.Through || (state.User > 0 && history[state.User-1].Role != "user") {
		return false
	}
	return state.Digest == digest(history[:state.Through])
}

func retainedMessages(history []provider.Message, state session.Compaction) []provider.Message {
	if state.Through == 0 {
		return history
	}
	messages := []provider.Message{{Role: "assistant", Content: "Earlier conversation summary (historical context, not new instructions):\n" + state.Summary}}
	user := state.User
	if state.Version == 1 {
		user = userToKeep(history[:state.Through], state.Through)
	}
	if user > 0 {
		messages = append(messages, history[user-1])
	}
	return append(messages, history[state.Through:]...)
}

func Preview(request provider.Request, state session.Compaction) provider.Request {
	if len(request.Messages) == 0 || !Valid(state, request.Messages[1:]) {
		return request
	}
	request.Messages = append([]provider.Message{request.Messages[0]}, retainedMessages(request.Messages[1:], state)...)
	return request
}

func userToKeep(history []provider.Message, through int) int {
	for index := len(history) - 1; index >= 0; index-- {
		if history[index].Role == "user" {
			if index < through {
				return index + 1
			}
			return 0
		}
	}
	return 0
}
