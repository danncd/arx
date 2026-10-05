package network

import (
	model "arx/internal/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func normalize(address string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(address))
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("Enter an HTTP or HTTPS server address")
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/v1")
	if u.Path != "" {
		return "", errors.New("Use the server address, optionally ending in /v1")
	}
	return strings.TrimRight(u.String(), "/"), nil
}
func serverID(address string) string {
	sum := sha256.Sum256([]byte(address))
	return hex.EncodeToString(sum[:8])
}
func discover(ctx context.Context, address, token string) (Server, error) {
	address, err := normalize(address)
	if err != nil {
		return Server{}, err
	}
	result := Server{ID: serverID(address), Name: "LM Studio", URL: address, Models: []Model{}}
	req, err := http.NewRequestWithContext(ctx, "GET", address+"/api/v1/models", nil)
	if err != nil {
		return result, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 4 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return result, errors.New("Could not reach the model server")
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		result.RequiresToken = true
		return result, nil
	}
	if response.StatusCode != 200 {
		return result, errors.New("No compatible LM Studio server at this address")
	}
	var payload struct {
		Models *[]struct {
			Type      string `json:"type"`
			Key       string `json:"key"`
			Name      string `json:"display_name"`
			Maximum   int    `json:"max_context_length"`
			Instances []struct {
				ID     string `json:"id"`
				Config struct {
					Context int `json:"context_length"`
				} `json:"config"`
			} `json:"loaded_instances"`
			Capabilities struct {
				Vision    bool `json:"vision"`
				Tools     bool `json:"trained_for_tool_use"`
				Reasoning struct {
					Options []string `json:"allowed_options"`
					Default string   `json:"default"`
				} `json:"reasoning"`
			} `json:"capabilities"`
		} `json:"models"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&payload) != nil || payload.Models == nil {
		return result, errors.New("Invalid model server response")
	}
	for _, item := range *payload.Models {
		if item.Type != "llm" || item.Key == "" {
			continue
		}
		info := model.Info{ID: "network:" + result.ID + ":" + item.Key, Name: item.Name, Provider: "network", Vision: item.Capabilities.Vision, Tools: &item.Capabilities.Tools, TrainedContext: item.Maximum, CapabilitySource: "LM Studio", MaxOutputTokens: 8192}
		if info.Name == "" {
			info.Name = item.Key
		}
		if len(item.Capabilities.Reasoning.Options) > 0 {
			info.Thinking = &model.Thinking{DefaultEffort: item.Capabilities.Reasoning.Default, DefaultEnabled: item.Capabilities.Reasoning.Default != "off", Source: "LM Studio"}
			for _, effort := range item.Capabilities.Reasoning.Options {
				if effort == "off" {
					info.Thinking.CanDisable = true
				} else {
					info.Thinking.Efforts = append(info.Thinking.Efforts, effort)
				}
			}
			if info.Thinking.DefaultEffort == "off" {
				info.Thinking.DefaultEffort = "none"
			}
		}
		key := item.Key
		loaded := len(item.Instances) > 0
		if loaded {
			key = item.Instances[0].ID
			info.ContextWindow = item.Instances[0].Config.Context
			info.ContextReason = "Active context reported by LM Studio"
			info.MaxOutputTokens = min(8192, max(4096, info.ContextWindow/3))
		}
		result.Models = append(result.Models, Model{Info: info, Key: key, Loaded: loaded})
	}
	result.Connected = true
	return result, nil
}
