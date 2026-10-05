package library

import (
	model "arx/internal/models"
	download "arx/internal/platform/download"
)

type Entry struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Repository   string          `json:"repository,omitempty"`
	Revision     string          `json:"revision,omitempty"`
	Quantization string          `json:"quantization,omitempty"`
	Files        []download.File `json:"files,omitempty"`
	Path         string          `json:"path"`
	Projector    string          `json:"projector,omitempty"`
	Imported     bool            `json:"imported"`
	Size         int64           `json:"size"`
	Status       string          `json:"status"`
	Received     int64           `json:"received"`
	Error        string          `json:"error,omitempty"`
	Info         model.Info      `json:"info"`
}
