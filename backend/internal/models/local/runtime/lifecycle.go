package engine

import (
	model "arx/internal/models"
	"errors"
	"os"
	"syscall"
	"time"
)

func (s *Server) Connection() (string, string, model.Info, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.command == nil {
		return "", "", model.Info{}, errors.New("Load a local model in Settings")
	}
	select {
	case <-s.done:
		return "", "", model.Info{}, errors.New("The local engine stopped. Reload the model")
	default:
	}
	return s.url, s.key, s.info, nil
}
func (s *Server) Stop() {
	s.mutex.Lock()
	command, done := s.command, s.done
	recordPath, keyPath := s.recordPath, s.keyPath
	s.command = nil
	s.info = model.Info{}
	s.mutex.Unlock()
	if command == nil {
		return
	}
	defer os.Remove(recordPath)
	defer os.Remove(keyPath)
	select {
	case <-done:
		return
	default:
	}
	syscall.Kill(-command.Process.Pid, syscall.SIGTERM)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		<-done
	}
}
