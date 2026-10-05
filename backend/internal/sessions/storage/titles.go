package storage

import session "arx/internal/sessions"

func (s *Store) Conversation(id string) session.Conversation {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if held := s.conversations[id]; held != nil {
		return held.summary
	}
	return session.Conversation{ID: id}
}

func (s *Store) restoreTitles() error {
	for _, id := range s.order {
		held := s.conversations[id]
		if !held.titled || held.summary.Title != "" {
			continue
		}
		for _, entry := range held.entries {
			if entry.Role != "user" {
				continue
			}
			title := firstLine(entry.Text)
			if title == "" {
				title = "Image conversation"
			}
			if err := s.Append(session.TranscriptChunk{Conversation: id, ID: "title:" + entry.ID, Role: "usage", Title: &title}); err != nil {
				return err
			}
			break
		}
	}
	return nil
}
