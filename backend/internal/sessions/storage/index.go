package storage

import (
	session "arx/internal/sessions"
	"sort"
	"strings"
)

type conversationIndex struct {
	entries   []session.TranscriptChunk
	messages  map[string][]session.TranscriptChunk
	order     []string
	positions map[string]int
	summary   session.Conversation
	visible   bool
	titled    bool
}

func (s *Store) index(entry session.TranscriptChunk) {
	held := s.conversations[entry.Conversation]
	if held == nil {
		held = &conversationIndex{messages: map[string][]session.TranscriptChunk{}, positions: map[string]int{}, summary: session.Conversation{ID: entry.Conversation, Status: "idle"}}
		s.conversations[entry.Conversation] = held
		s.order = append(s.order, entry.Conversation)
	}
	if entry.Title != nil {
		held.summary.Title = *entry.Title
		held.titled = true
	}
	held.entries = append(held.entries, entry)
	if entry.Role != "usage" {
		if _, exists := held.positions[entry.ID]; !exists {
			held.positions[entry.ID] = len(held.order)
			held.order = append(held.order, entry.ID)
		}
		held.messages[entry.ID] = append(held.messages[entry.ID], entry)
	}
	if entry.Role == "system" || entry.Role == "usage" {
		return
	}
	held.visible = true
	if !held.titled && held.summary.Title == "" && entry.Role == "user" {
		held.summary.Title = firstLine(entry.Text)
		if held.summary.Title == "" && len(entry.Images) > 0 {
			held.summary.Title = "Image"
		}
	}
	if entry.At > held.summary.Updated {
		held.summary.Updated = entry.At
	}
}

func (s *Store) Conversations() []session.Conversation {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	out := []session.Conversation{}
	for _, id := range s.order {
		held := s.conversations[id]
		if !held.visible {
			continue
		}
		item := held.summary
		if item.Title == "" && !held.titled {
			item.Title = "New conversation"
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Updated > out[j].Updated })
	return out
}

func (s *Store) Entries(conversation string) []session.TranscriptChunk {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if held := s.conversations[conversation]; held != nil {
		return append([]session.TranscriptChunk{}, held.entries...)
	}
	return []session.TranscriptChunk{}
}

func firstLine(text string) string {
	text = strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	runes := []rune(text)
	if len(runes) > 80 {
		text = string(runes[:80])
	}
	return text
}
