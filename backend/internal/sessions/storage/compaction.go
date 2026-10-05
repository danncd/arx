package storage

import (
	"arx/internal/platform/atomicfile"
	session "arx/internal/sessions"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

func (s *Store) compactionPath(conversation string) string {
	hash := sha256.Sum256([]byte(conversation))
	return filepath.Join(s.directory, "summaries", hex.EncodeToString(hash[:])+".json")
}

func (s *Store) Compaction(conversation string) (session.Compaction, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	file, err := os.Open(s.compactionPath(conversation))
	if errors.Is(err, os.ErrNotExist) {
		return session.Compaction{}, nil
	}
	if err != nil {
		return session.Compaction{}, err
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 128<<10))
	if err != nil {
		return session.Compaction{}, err
	}
	var state session.Compaction
	if json.Unmarshal(body, &state) != nil {
		return session.Compaction{}, nil
	}
	return state, nil
}

func (s *Store) SaveCompaction(conversation string, state session.Compaction) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.file == nil {
		return errors.New("Session storage is closed")
	}
	if (state.Version != 1 && state.Version != 2) || state.User < 0 || state.User > state.Through || state.Through <= 0 || len(state.Digest) != 64 || state.Summary == "" || len(state.Summary) > 64<<10 {
		return errors.New("Invalid conversation summary")
	}
	path := s.compactionPath(conversation)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	body, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return atomicfile.Replace(path, body, true)
}
