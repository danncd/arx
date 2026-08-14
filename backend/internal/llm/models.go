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

var httpClient = &http.Client{Timeout: 30 * time.Second}

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
}

type modelList struct {
	Data []Model `json:"data"`
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
			if or != nil {
				om, ok := or[normalizeModelID(m.ID)]
				if !ok || !strings.HasSuffix(om.Architecture.Modality, "->text") {
					continue
				}
				info.ContextWindow = om.ContextLength
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spec < out[j].Spec })
	return out, errors.Join(errs...)
}
