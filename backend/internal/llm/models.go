package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// httpClient is for bounded metadata calls (/models, the OpenRouter
// catalog). Completions must NOT use it: 30s spans the entire body
// read, which a long generation legitimately exceeds.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// llmClient carries no overall timeout: a healthy completion or stream
// can legally run for minutes. The caller's ctx is the deadline
// authority; dial timeouts still apply via the default transport.
var llmClient = &http.Client{}

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

type ModelInfo struct {
	Spec          string
	Provider      string
	Model         string
	KeyEnv        string
	ContextWindow int
	Reasoning     bool // model has a reasoning mode (per OpenRouter)
	Tools         bool // model can call tools — required for the agent loop
}

type modelList struct {
	Data []Model `json:"data"`
}

// nonChatMarkers identify models that can't hold a chat conversation.
// Fallback filter for ids OpenRouter's catalog doesn't know; naming
// heuristics only, so it needs a new marker when a new modality ships.
var nonChatMarkers = []string{
	"embedding", "tts", "whisper", "audio", "transcribe", "image", "sora",
	"realtime", "moderation", "search-api", "babbage", "davinci", "instruct",
}

func chatCapable(id string) bool {
	for _, marker := range nonChatMarkers {
		if strings.Contains(id, marker) {
			return false
		}
	}
	return true
}

func ListModels(ctx context.Context, p Provider) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	if k := p.Key(); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("%s /models: HTTP %d: %s", p.Name, resp.StatusCode, body)
	}

	var list modelList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	return list.Data, nil
}

func LoadModels(ctx context.Context) ([]ModelInfo, error) {
	var out []ModelInfo
	var errs []error

	// OpenRouter's public catalog is the authority on which models are
	// chat-capable. When it is unreachable OR comes back empty, we skip
	// filtering rather than intersect against a hollow authority and
	// return an empty catalog (degrade open, report the error).
	or, orErr := fetchORCatalog(ctx)
	if orErr != nil {
		errs = append(errs, fmt.Errorf("openrouter catalog: %w", orErr))
	}

	for name, p := range Providers {
		if p.KeyEnv != "" && p.Key() == "" {
			continue
		}
		models, err := ListModels(ctx, p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		for _, m := range models {
			info := ModelInfo{
				Spec:     name + "/" + m.ID,
				Provider: name,
				Model:    m.ID,
				KeyEnv:   p.KeyEnv,
			}
			// Three tiers of truth. Known to OpenRouter: its modality
			// data decides, and the model gets enriched. Unknown (OR's
			// slug namespace misses real ids: deepseek-reasoner, ft:…
			// fine-tunes) or OR unreachable: the name heuristic decides
			// — a join miss must never disappear a real chat model, but
			// passing misses through unfiltered would readmit whisper,
			// tts and friends.
			om, known := orModel{}, false
			if len(or) > 0 {
				om, known = or[normalizeModelID(m.ID)]
			}
			if known {
				if !strings.HasSuffix(om.Architecture.Modality, "->text") {
					continue // the authority says not a chat model
				}
				info.ContextWindow = om.ContextLength
				info.Reasoning = contains(om.SupportedParameters, "reasoning")
				info.Tools = contains(om.SupportedParameters, "tools")
			} else if !chatCapable(m.ID) {
				continue // heuristic fallback: obviously non-chat by name
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spec < out[j].Spec })
	return out, errors.Join(errs...)
}
