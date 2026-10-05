package local

import (
	model "arx/internal/models"
	"arx/internal/models/local/library"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func modelID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "local:" + hex.EncodeToString(sum[:16])
}
func (m *Manager) inspect(entry *library.Entry) error {
	if err := library.ValidateChatFile(entry.Path); err != nil {
		return err
	}
	size, err := entry.FileSize()
	if err != nil {
		return err
	}
	entry.Size = size
	metadata, err := library.Inspect(entry.Path)
	if err != nil {
		return err
	}
	if metadata.Context < 4096 {
		return errors.New("This model has no usable chat context metadata")
	}
	context, reason := m.state.Hardware.Context(metadata.Context, entry.Size, metadata.CacheBytesPerToken())
	previous := entry.Info
	entry.Info = model.Info{ID: entry.ID, Provider: "local", Name: entry.Name, TrainedContext: metadata.Context, ContextWindow: context, MaxOutputTokens: min(32768, context/2), CapabilitySource: "metadata", ContextReason: reason, Vision: entry.Projector != ""}
	if previous.CapabilitySource == "runtime" {
		entry.Info.Tools = previous.Tools
		entry.Info.Thinking = previous.Thinking
		entry.Info.Vision = previous.Vision
		entry.Info.CapabilitySource = "runtime"
	}
	return nil
}
func (m *Manager) Import(path, projector string) (string, error) {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".gguf") {
		return "", errors.New("Choose a GGUF model file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !stat.Mode().IsRegular() {
		return "", errors.New("Choose a model file")
	}
	if projector != "" {
		if !filepath.IsAbs(projector) {
			return "", errors.New("Choose a projector file")
		}
		projector, err = filepath.EvalSymlinks(projector)
		if err != nil {
			return "", err
		}
		if _, err := library.Inspect(projector); err != nil {
			return "", err
		}
	}
	entry := library.Entry{ID: modelID(resolved), Name: strings.TrimSuffix(filepath.Base(resolved), filepath.Ext(resolved)), Path: resolved, Projector: projector, Imported: true, Size: stat.Size(), Status: "installed"}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.closed {
		return "", errors.New("Backend is stopping")
	}
	if err := m.inspect(&entry); err != nil {
		return "", err
	}
	if m.index(entry.ID) >= 0 {
		return entry.ID, nil
	}
	m.state.Models = append(m.state.Models, entry)
	m.publishLocked(true)
	return entry.ID, nil
}
func (m *Manager) Model(id string) (model.Info, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	index := m.index(id)
	if index < 0 {
		return model.Info{}, errors.New("Local model was not found")
	}
	entry := m.state.Models[index]
	if entry.Status != "installed" && entry.Status != "loaded" {
		return model.Info{}, errors.New("Finish downloading or loading this model first")
	}
	return entry.Info, nil
}
