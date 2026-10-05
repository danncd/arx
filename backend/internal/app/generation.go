package app

import (
	generation "arx/internal/generation"
	jobs "arx/internal/generation/jobs"
	models "arx/internal/generation/models"
	engines "arx/internal/generation/runtime"
	context "context"
	time "time"
)

type GenerationEvent struct {
	Library *models.State `json:"library,omitempty"`
	Job     *jobs.Job     `json:"job,omitempty"`
}

func (s *Service) Generation() (*generation.Service, error) {
	s.generationMutex.Lock()
	defer s.generationMutex.Unlock()
	if s.generation != nil {
		return s.generation, nil
	}
	assets, err := engines.WorkerDirectory()
	if err != nil {
		return nil, err
	}
	manager, err := generation.Open(s.directory, assets)
	if err != nil {
		return nil, err
	}
	manager.Acquire = s.generationMemory
	manager.Models.Subscribe(func(state models.State) { s.emitGeneration(GenerationEvent{Library: &state}) })
	manager.Jobs.Subscribe(func(job jobs.Job) { s.emitGeneration(GenerationEvent{Job: &job}) })
	s.generation = manager
	return manager, nil
}

func (s *Service) SubscribeGeneration(notify func(GenerationEvent)) {
	s.generationNotifyMutex.Lock()
	defer s.generationNotifyMutex.Unlock()
	s.generationNotify = notify
}
func (s *Service) emitGeneration(event GenerationEvent) {
	s.generationNotifyMutex.Lock()
	notify := s.generationNotify
	s.generationNotifyMutex.Unlock()
	if notify != nil {
		notify(event)
	}
}

func (s *Service) generationMemory(ctx context.Context) (func() error, error) {
	s.localMutex.Lock()
	local := s.local
	s.localMutex.Unlock()
	if local == nil {
		return func() error { return nil }, nil
	}
	state := local.Snapshot()
	if state.Runtime.State != "ready" {
		return func() error { return nil }, nil
	}
	if err := local.Unload(); err != nil {
		return nil, err
	}
	return func() error {
		closed := s.Service.Closed()
		if closed {
			return nil
		}
		restore, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Minute)
		defer cancel()
		_, err := local.Ensure(restore, state.Runtime.Model)
		if err != nil {
			local.Unload()
		}
		return err
	}, nil
}
func (s *Service) generationGuidance() string {
	manager, err := s.Generation()
	if err != nil {
		return "Generation settings could not be read. Do not assume a model is configured."
	}
	return generation.Describe(manager.Models.Snapshot())
}
