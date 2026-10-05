package app

import (
	inference "arx/internal/inference"
	catalog "arx/internal/inference/deepseek"
	models "arx/internal/models"
	localruntime "arx/internal/models/local"
	network "arx/internal/models/network"
	settings "arx/internal/settings"
	context "context"
	errors "errors"
	os "os"
	filepath "path/filepath"
)

func (s *Service) newRouter() *inference.Router {
	return &inference.Router{
		CloudModel:    func(id string) (models.Info, error) { return s.deepseek.Model(id) },
		CloudValidate: func(id, effort string) error { return s.deepseek.Validate(id, effort) },
		CloudComplete: func(ctx context.Context, req inference.Request, emit func(inference.Delta) error) (inference.Response, error) {
			return s.deepseek.Complete(ctx, req, emit)
		},
		LocalModel: func(id string) (models.Info, error) {
			m, err := s.LocalModels()
			if err != nil {
				return models.Info{}, err
			}
			return m.Model(id)
		},
		NetworkModel: func(id string) (models.Info, error) {
			m, err := s.NetworkModels()
			if err != nil {
				return models.Info{}, err
			}
			return m.Model(id)
		},
		LocalComplete: func(ctx context.Context, req inference.Request, emit func(inference.Delta) error) (inference.Response, error) {
			m, err := s.LocalModels()
			if err != nil {
				return inference.Response{}, err
			}
			return m.Complete(ctx, req, emit)
		},
		NetworkComplete: func(ctx context.Context, req inference.Request, emit func(inference.Delta) error) (inference.Response, error) {
			m, err := s.NetworkModels()
			if err != nil {
				return inference.Response{}, err
			}
			return m.Complete(ctx, req, s.images.Read, emit)
		},
	}
}
func (s *Service) LocalModels() (*localruntime.Manager, error) {
	s.localMutex.Lock()
	defer s.localMutex.Unlock()
	if s.local != nil {
		return s.local, nil
	}
	directory := os.Getenv("ARX_MODELS_DIR")
	if directory == "" {
		directory = filepath.Join(s.directory, "local-models")
	}
	manager, err := localruntime.Open(directory)
	if err != nil {
		return nil, err
	}
	manager.ReadImage = s.images.Read
	manager.SetIdleMinutes(s.preferences.Values().LocalIdleMinutes)
	manager.Subscribe(s.localNotify)
	s.local = manager
	return manager, nil
}

func (s *Service) ConfigureLocalIdleMinutes(minutes int) (settings.Values, error) {
	values, err := s.preferences.ConfigureLocalIdleMinutes(minutes)
	if err != nil {
		return values, err
	}
	s.localMutex.Lock()
	if s.local != nil {
		s.local.SetIdleMinutes(minutes)
	}
	s.localMutex.Unlock()
	return values, nil
}
func (s *Service) SubscribeLocal(notify func(localruntime.State)) {
	s.localMutex.Lock()
	defer s.localMutex.Unlock()
	s.localNotify = notify
	if s.local != nil {
		s.local.Subscribe(notify)
	}
}
func (s *Service) LocalAction(action, id string, deleteFiles bool) error {
	return s.Service.WithState(func(active bool, conversation string, closed bool) error {
		if closed {
			return errors.New("Backend is stopping")
		}
		if active && (action == "load" || action == "unload" || action == "remove") {
			return errors.New("Stop the current reply before changing local models")
		}
		manager, err := s.LocalModels()
		if err != nil {
			return err
		}
		switch action {
		case "load":
			return manager.Load(id)
		case "unload":
			return manager.Unload()
		case "pause":
			return manager.Pause(id)
		case "resume":
			return manager.Resume(id)
		case "cancel":
			return manager.Cancel(id)
		case "remove":
			if manager.Snapshot().Runtime.Model == id {
				if err := manager.Unload(); err != nil {
					return err
				}
			}
			return manager.Remove(id, deleteFiles)
		}
		return errors.New("Unknown local model action")
	})
}
func (s *Service) NetworkModels() (*network.Manager, error) {
	s.networkMutex.Lock()
	defer s.networkMutex.Unlock()
	if s.network != nil {
		return s.network, nil
	}
	manager, err := network.Open(filepath.Join(s.directory, "network-models"))
	if err == nil {
		s.network = manager
	}
	return manager, err
}
func (s *Service) DeepSeekStatus(ctx context.Context, refresh bool) (catalog.Connection, error) {
	return s.deepseek.Status(ctx, refresh)
}
func (s *Service) ConnectDeepSeek(ctx context.Context, key string) (catalog.Connection, error) {
	return s.deepseek.Connect(ctx, key)
}
func (s *Service) DisconnectDeepSeek(ctx context.Context) (catalog.Connection, error) {
	return s.deepseek.Disconnect(ctx)
}
