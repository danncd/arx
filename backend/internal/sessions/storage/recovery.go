package storage

import (
	session "arx/internal/sessions"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func (s *Store) load() error {
	reader := bufio.NewReader(s.file)
	for {
		var line []byte
		for {
			part, err := reader.ReadSlice('\n')
			line = append(line, part...)
			if len(line) > (8<<20)+1 {
				return errors.New("Transcript line is too large")
			}
			if err == bufio.ErrBufferFull {
				continue
			}
			if err != nil && err != io.EOF {
				return err
			}
			if len(bytes.TrimSpace(line)) > 0 {
				var entry session.TranscriptChunk
				if json.Unmarshal(line, &entry) != nil || entry.Conversation == "" || entry.ID == "" {
					s.recovered = true
				} else {
					s.index(entry)
				}
			}
			if err == io.EOF {
				if len(line) > 0 && line[len(line)-1] != '\n' {
					if _, err := s.file.Write([]byte{'\n'}); err != nil {
						return err
					}
					if err := s.file.Sync(); err != nil {
						return err
					}
				}
				return nil
			}
			break
		}
	}
}
