package network

import model "arx/internal/models"

type Model struct {
	Info   model.Info `json:"info"`
	Key    string     `json:"key"`
	Loaded bool       `json:"loaded"`
}
type Server struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	URL           string  `json:"url"`
	Connected     bool    `json:"connected"`
	RequiresToken bool    `json:"requiresToken"`
	Error         string  `json:"error,omitempty"`
	Models        []Model `json:"models"`
}
type savedServer struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	HasToken bool   `json:"hasToken"`
}
