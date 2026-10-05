package inference

import (
	model "arx/internal/models"
	"arx/internal/settings"
	"context"
	"errors"
	"strings"
)

type Router struct {
	CloudModel      func(string) (model.Info, error)
	LocalModel      func(string) (model.Info, error)
	NetworkModel    func(string) (model.Info, error)
	CloudValidate   func(string, string) error
	CloudComplete   func(context.Context, Request, func(Delta) error) (Response, error)
	LocalComplete   func(context.Context, Request, func(Delta) error) (Response, error)
	NetworkComplete func(context.Context, Request, func(Delta) error) (Response, error)
}

func ProviderFor(run settings.Run) (string, error) {
	provider := "deepseek"
	if strings.HasPrefix(run.Model, "local:") {
		provider = "local"
	}
	if strings.HasPrefix(run.Model, "network:") {
		provider = "network"
	}
	if run.Provider != "" && run.Provider != provider {
		return "", errors.New("Selected model does not match its provider")
	}
	return provider, nil
}
func IsLocal(run settings.Run) bool { p, _ := ProviderFor(run); return p == "local" }
func (s *Router) Model(run settings.Run) (model.Info, error) {
	source, err := ProviderFor(run)
	if err != nil {
		return model.Info{}, err
	}
	if source == "network" {
		return s.NetworkModel(run.Model)
	}
	if IsLocal(run) {
		return s.LocalModel(run.Model)
	}
	return s.CloudModel(run.Model)
}
func (s *Router) Validate(run settings.Run) error {
	if _, err := ProviderFor(run); err != nil {
		return err
	}
	if !IsLocal(run) && !strings.HasPrefix(run.Model, "network:") {
		return s.CloudValidate(run.Model, run.Effort)
	}
	info, err := s.Model(run)
	if err != nil {
		return err
	}
	if run.Effort == "" {
		return nil
	}
	if info.Thinking != nil {
		if run.Effort == "none" && info.Thinking.CanDisable {
			return nil
		}
		for _, effort := range info.Thinking.Efforts {
			if run.Effort == effort {
				return nil
			}
		}
	}
	return errors.New("This model does not support that thinking setting")
}
func (s *Router) Complete(ctx context.Context, request Request, emit func(Delta) error) (Response, error) {
	if strings.HasPrefix(request.Model, "network:") {
		return s.NetworkComplete(ctx, request, emit)
	}
	if strings.HasPrefix(request.Model, "local:") {
		return s.LocalComplete(ctx, request, emit)
	}
	return s.CloudComplete(ctx, request, emit)
}
