package context

import (
	provider "arx/internal/inference"
	session "arx/internal/sessions"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func fingerprint(request provider.Request) string {
	body, _ := json.Marshal(struct {
		Messages []provider.Message
		Tools    any
	}{request.Messages, request.Tools})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func Measure(request provider.Request, input int) *session.ContextUsage {
	if input <= 0 {
		return nil
	}
	return &session.ContextUsage{Model: request.Model, Effort: request.Effort, Messages: len(request.Messages), Digest: fingerprint(request), Input: input}
}

func Count(request provider.Request, usage *session.ContextUsage) int {
	if usage == nil || usage.Input <= 0 || usage.Model != request.Model || usage.Effort != request.Effort || usage.Messages <= 0 || usage.Messages > len(request.Messages) {
		return Estimate(request)
	}
	prefix := request
	prefix.Messages = request.Messages[:usage.Messages]
	if fingerprint(prefix) != usage.Digest {
		return Estimate(request)
	}
	return usage.Input + Messages(request.Messages[usage.Messages:])
}

func Latest(entries []session.TranscriptChunk) *session.ContextUsage {
	for index := len(entries) - 1; index >= 0; index-- {
		if entries[index].Context != nil {
			return entries[index].Context
		}
	}
	return nil
}
