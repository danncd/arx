package network

import (
	provider "arx/internal/inference"
	compatible "arx/internal/inference/chatcompletions"
	model "arx/internal/models"
	"context"
	"errors"
	"strings"
)

func (m *Manager) resolve(ctx context.Context, id string) (Model, savedServer, error) {
	for _, entry := range m.entries() {
		if !strings.HasPrefix(id, "network:"+entry.ID+":") {
			continue
		}
		server, err := m.metadata(ctx, entry, false)
		if err != nil {
			return Model{}, entry, err
		}
		if !server.Connected {
			return Model{}, entry, errors.New("Reconnect this server in Settings > Connections")
		}
		for _, item := range server.Models {
			if item.Info.ID == id {
				if !item.Loaded || item.Info.ContextWindow <= 0 {
					return Model{}, entry, errors.New("Load this model in LM Studio first")
				}
				return item, entry, nil
			}
		}
	}
	return Model{}, savedServer{}, errors.New("Network model is unavailable")
}
func (m *Manager) Model(id string) (model.Info, error) {
	item, _, err := m.resolve(context.Background(), id)
	return item.Info, err
}
func (m *Manager) Complete(ctx context.Context, input provider.Request, readImage func(string) ([]byte, error), emit func(provider.Delta) error) (provider.Response, error) {
	item, entry, err := m.resolve(ctx, input.Model)
	if err != nil {
		return provider.Response{}, err
	}
	if item.Info.Tools == nil || !*item.Info.Tools {
		input.Tools = nil
	}
	token, err := m.token(ctx, entry)
	if err != nil {
		return provider.Response{}, err
	}
	client := compatible.Client{Profile: compatible.NetworkWire, URL: entry.URL, Key: token, Info: item.Info, ReadImage: readImage, RemoteModel: item.Key}
	return client.Complete(ctx, input, emit)
}
