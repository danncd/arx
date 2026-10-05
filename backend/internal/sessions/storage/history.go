package storage

import (
	session "arx/internal/sessions"
	"encoding/json"
	"errors"
	"sort"
)

func (s *Store) History(conversation, before string, limit int) ([]session.TranscriptChunk, string, bool, error) {
	if limit <= 0 || limit > 200 {
		limit = 60
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	held := s.conversations[conversation]
	if held == nil {
		if before != "" {
			return nil, "", false, errors.New("History cursor was not found")
		}
		return []session.TranscriptChunk{}, "", false, nil
	}
	end := len(held.order)
	if before != "" {
		var found bool
		end, found = held.positions[before]
		if !found {
			end = -1
		}
	}

	if end < 0 {
		return nil, "", false, errors.New("History cursor was not found")
	}
	start := end - limit
	more := false
	if start < 0 {
		start = 0
	} else {
		more = start > 0
	}

	size := 0
	for index := end - 1; index >= start; index-- {
		body, err := json.Marshal(held.messages[held.order[index]])
		if err != nil {
			return nil, "", false, err
		}
		if size+len(body) > 12<<20 {
			if index == end-1 {
				return nil, "", false, errors.New("Message exceeds the history page size")
			}
			start = index + 1
			more = true
			break
		}
		size += len(body)
	}
	out := []session.TranscriptChunk{}
	for _, id := range held.order[start:end] {
		message := append([]session.TranscriptChunk{}, held.messages[id]...)
		sort.SliceStable(message, func(i, j int) bool { return message[i].Offset < message[j].Offset })
		out = append(out, message...)
	}
	cursor := ""
	if start > 0 {
		cursor = held.order[start]
	}
	return out, cursor, more, nil
}
