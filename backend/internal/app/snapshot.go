package app

import (
	session "arx/internal/sessions"
	"arx/internal/settings"
)

type Snapshot struct {
	Conversations []session.Conversation `json:"conversations"`
	Settings      settings.Values        `json:"settings"`
	Views         map[string]string      `json:"views"`
	Recovered     bool                   `json:"recovered"`
}

type History struct {
	Chunks []session.TranscriptChunk `json:"chunks"`
	Before string                    `json:"before"`
	More   bool                      `json:"more"`
}

func (s *Service) Snapshot() Snapshot {
	return Snapshot{s.sessions.Conversations(), s.preferences.Values(), s.preferences.Views(), s.sessions.Recovered()}
}
func (s *Service) History(conversation, before string, limit int) (History, error) {
	chunks, cursor, more, err := s.sessions.History(conversation, before, limit)
	return History{chunks, cursor, more}, err
}
func (s *Service) Configure(run settings.Run, directory string, autoContinue ...*bool) (settings.Values, error) {
	if run != s.preferences.Values().Run {
		if err := s.Service.Inference.Validate(run); err != nil {
			return s.preferences.Values(), err
		}
	}
	return s.preferences.Configure(run, directory, autoContinue...)
}
func (s *Service) SaveView(key, value string) error           { return s.preferences.SaveView(key, value) }
func (s *Service) Record(entry session.TranscriptChunk) error { return s.sessions.Append(entry) }

func (s *Service) SaveViews(values map[string]string) error { return s.preferences.SaveViews(values) }
