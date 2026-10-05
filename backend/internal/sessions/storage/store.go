package storage

import (
	session "arx/internal/sessions"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store struct {
	directory     string
	mutex         sync.Mutex
	file          *os.File
	conversations map[string]*conversationIndex
	order         []string
	recovered     bool
}

func Open(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(directory, "transcript.jsonl"), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	store := &Store{file: file, directory: directory, conversations: map[string]*conversationIndex{}}
	if err := store.load(); err != nil {
		file.Close()
		return nil, err
	}
	if err := store.restoreTitles(); err != nil {
		file.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}

func (s *Store) Recovered() bool { return s.recovered }

func (s *Store) Append(entry session.TranscriptChunk) error {
	entry.Conversation = strings.TrimSpace(entry.Conversation)
	if entry.Conversation == "" || entry.ID == "" {
		return errors.New("A conversation and message ID are required")
	}
	if entry.Offset < 0 || entry.ReasoningAt < 0 {
		return errors.New("Message offsets cannot be negative")
	}
	if entry.At == "" {
		entry.At = time.Now().UTC().Format(time.RFC3339Nano)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if len(line) > 8<<20 {
		return errors.New("Message chunk is too large")
	}
	var snapshot session.TranscriptChunk
	if err := json.Unmarshal(line, &snapshot); err != nil {
		return err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.file == nil {
		return errors.New("Session storage is closed")
	}
	start, err := s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	n, err := s.file.Write(line)
	if err == nil && n != len(line) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = s.file.Sync()
	}
	if err != nil {
		s.file.Truncate(start)
		return err
	}
	s.index(snapshot)
	return nil
}
