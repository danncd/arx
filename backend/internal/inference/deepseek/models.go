package deepseek

import (
	"arx/internal/inference"
	model "arx/internal/models"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	ReadImage func(string) ([]byte, error)
	client    *http.Client
	baseURL   string
}

func NewClient() *Client {
	return &Client{baseURL: "https://api.deepseek.com", client: inference.NewHTTPClient(15 * time.Second)}
}

func (c *Client) Models(ctx context.Context, key string) ([]model.Info, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, errors.New("Could not prepare DeepSeek request")
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return nil, errors.New("Couldn’t reach DeepSeek. Try again")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, responseError(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return nil, errors.New("Could not read DeepSeek models")
	}
	var data struct {
		Data []wireModel `json:"data"`
	}
	if json.Unmarshal(body, &data) != nil {
		return nil, errors.New("DeepSeek returned invalid model data")
	}
	models := []model.Info{}
	seen := map[string]bool{}
	for _, entry := range data.Data {
		if entry.ID == "" || seen[entry.ID] {
			continue
		}
		seen[entry.ID] = true
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = entry.ID
		}
		info := model.Info{Vision: SupportsVision(entry.ID), ID: entry.ID, Name: name, ContextWindow: max(0, entry.ContextWindow), MaxOutputTokens: max(0, entry.MaxOutputTokens)}
		if len(entry.Effort.Levels) > 0 {
			efforts := []string{}
			seenEfforts := map[string]bool{}
			for _, level := range entry.Effort.Levels {
				if level != "" && level != "none" && !seenEfforts[level] {
					efforts = append(efforts, level)
					seenEfforts[level] = true
				}
			}
			if len(efforts) > 0 {
				defaultEffort := entry.Effort.Default
				if !seenEfforts[defaultEffort] {
					defaultEffort = efforts[0]
				}
				info.Thinking = &model.Thinking{Efforts: efforts, DefaultEffort: defaultEffort, DefaultEnabled: true, CanDisable: true, Source: "deepseek"}
			}
		}
		models = append(models, info)
	}
	if len(models) == 0 {
		return nil, errors.New("DeepSeek returned no available models")
	}
	return models, nil
}

type wireModel struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ContextWindow   int    `json:"context_window"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Effort          struct {
		Levels  []string `json:"supported_levels"`
		Default string   `json:"default_level"`
	} `json:"effort"`
}

func responseError(status int) error {
	switch status {
	case 401, 403:
		return errors.New("DeepSeek rejected this API key")
	case 402:
		return errors.New("DeepSeek account has insufficient balance")
	case 429:
		return errors.New("DeepSeek rate limit reached. Try again shortly")
	default:
		return errors.New("DeepSeek is unavailable. Try again")
	}
}
