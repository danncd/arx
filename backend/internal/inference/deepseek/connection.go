package deepseek

import (
	provider "arx/internal/inference"
	model "arx/internal/models"
	credentials "arx/internal/platform/keychain"
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type Discovery interface {
	Models(context.Context, string) ([]model.Info, error)
}

type Connection struct {
	Configured bool         `json:"configured"`
	Connected  bool         `json:"connected"`
	Models     []model.Info `json:"models"`
	Error      string       `json:"error"`
}

type DeepSeek struct {
	mutex       sync.Mutex
	credentials credentials.Store
	provider    Discovery
	key         string
	loaded      bool
	models      []model.Info
	checked     time.Time
}

func NewConnection(store credentials.Store, provider Discovery) *DeepSeek {
	return &DeepSeek{credentials: store, provider: provider, models: []model.Info{}}
}

func (d *DeepSeek) Status(ctx context.Context, refresh bool) (Connection, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if !d.loaded {
		key, err := d.credentials.Load(ctx)
		if err != nil {
			return Connection{}, err
		}
		d.key, d.loaded = key, true
	}
	if d.key == "" {
		return Connection{Models: []model.Info{}}, nil
	}
	if !refresh && !d.checked.IsZero() && time.Since(d.checked) < 10*time.Minute {
		return Connection{Configured: true, Connected: true, Models: d.models}, nil
	}
	models, err := d.provider.Models(ctx, d.key)
	if err != nil {
		d.checked = time.Time{}
		return Connection{Configured: true, Models: d.models, Error: err.Error()}, nil
	}
	d.models, d.checked = models, time.Now()
	return Connection{Configured: true, Connected: true, Models: d.models}, nil
}

func (d *DeepSeek) Connect(ctx context.Context, key string) (Connection, error) {
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 4096 || strings.ContainsAny(key, "\r\n\t ") {
		return Connection{}, errors.New("Enter a valid API key")
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	models, err := d.provider.Models(ctx, key)
	if err != nil {
		return Connection{}, err
	}
	if err := d.credentials.Save(ctx, key); err != nil {
		return Connection{}, err
	}
	d.key, d.loaded, d.models, d.checked = key, true, models, time.Now()
	return Connection{Configured: true, Connected: true, Models: models}, nil
}

func (d *DeepSeek) Disconnect(ctx context.Context) (Connection, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if err := d.credentials.Delete(ctx); err != nil {
		return Connection{}, err
	}
	d.key, d.loaded, d.models, d.checked = "", true, []model.Info{}, time.Time{}
	return Connection{Models: []model.Info{}}, nil
}

func (d *DeepSeek) Validate(id, effort string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, item := range d.models {
		if item.ID != id {
			continue
		}
		if item.Thinking == nil {
			if effort == "" {
				return nil
			}
			return errors.New("Thinking capabilities are unavailable for this model")
		}
		if effort == "none" && item.Thinking.CanDisable {
			return nil
		}
		for _, level := range item.Thinking.Efforts {
			if level == effort {
				return nil
			}
		}
		return errors.New("This model does not support that thinking effort")
	}
	return errors.New("Choose an available DeepSeek model")
}

func (d *DeepSeek) Complete(ctx context.Context, request provider.Request, emit func(provider.Delta) error) (provider.Response, error) {
	d.mutex.Lock()
	key := d.key
	client, ok := d.provider.(provider.AuthenticatedCompleter)
	d.mutex.Unlock()
	if key == "" {
		return provider.Response{}, errors.New("Connect DeepSeek in Settings")
	}
	if !ok {
		return provider.Response{}, errors.New("DeepSeek chat is unavailable")
	}
	return client.Complete(ctx, key, request, emit)
}

func (d *DeepSeek) Model(id string) (model.Info, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, info := range d.models {
		if info.ID == id {
			return info, nil
		}
	}
	return model.Info{}, errors.New("Choose an available DeepSeek model")
}
