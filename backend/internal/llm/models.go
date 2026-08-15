package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

/* Discovery requests stop after 30 seconds. */

var httpClient = &http.Client{Timeout: 30 * time.Second}

const maxCatalogBytes int64 = 8 << 20

/* Provider models and their discovered capabilities. */

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
	MaxOutput     int // provider's completion-token ceiling, 0 = unknown
	Reasoning     bool
	Tools         bool
	ToolsKnown    bool
}

type modelList struct {
	Data []Model `json:"data"`
}

/* Fallback markers for models missing from OpenRouter. */

var nonChatMarkers = []string{
	"embedding", "tts", "whisper", "audio", "transcribe", "image", "sora",
	"realtime", "moderation", "search-api", "babbage", "davinci", "instruct",
	"dall-e", "codex-mini", "computer-use",
}

func providerChatCapable(p Provider, id string) bool {
	if p.Name == "openai" {
		bare := normalizeModelID(id)
		// Keep current Chat Completions families.
		if !strings.HasPrefix(bare, "gpt-5") {
			return false
		}
		if strings.Contains(bare, "-codex") {
			return false
		}
	}
	return chatCapable(id)
}

func chatCapable(id string) bool {
	id = strings.ToLower(id)
	for _, marker := range nonChatMarkers {
		if strings.Contains(id, marker) {
			return false
		}
	}
	return true
}

/* Lists one provider's models. */

func ListModels(ctx context.Context, p Provider) ([]Model, error) {
	endpoint, err := url.JoinPath(p.BaseURL, "models")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
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
	if err := decodeJSON(resp.Body, &list); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}
	if list.Data == nil {
		return nil, fmt.Errorf("%s /models: response carried no data field", p.Name)
	}
	models := make([]Model, 0, len(list.Data))
	for _, m := range list.Data {
		if strings.TrimSpace(m.ID) == "" {
			continue
		}
		models = append(models, m)
	}
	if len(list.Data) > 0 && len(models) == 0 {
		return nil, fmt.Errorf("%s /models: every model entry is missing an id", p.Name)
	}
	return models, nil
}

/* Finds chat models for every configured provider. */

func LoadModels(ctx context.Context) ([]ModelInfo, error) {
	var out []ModelInfo
	var errs []error
	var names []string
	for name, p := range Providers {
		if p.KeyEnv == "" || p.Key() != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	sort.Strings(names)

	// OpenRouter supplies cross-provider capabilities.
	or, orErr := fetchORCatalog(ctx)
	if orErr != nil {
		errs = append(errs, fmt.Errorf("openrouter catalog: %w", orErr))
	}

	for _, name := range names {
		p := Providers[name]
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

			if !providerChatCapable(p, m.ID) {
				continue
			}

			om, known := or[normalizeModelID(m.ID)]
			if known {
				if mod := om.Architecture.Modality; mod != "" && !strings.HasSuffix(mod, "->text") {
					continue
				}
				info.ContextWindow = om.ContextLength
				info.MaxOutput = om.TopProvider.MaxCompletionTokens
				info.Reasoning = contains(om.SupportedParameters, "reasoning")
				if om.SupportedParameters != nil {
					info.Tools = contains(om.SupportedParameters, "tools")
					info.ToolsKnown = true
				}
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Spec < out[j].Spec })
	return out, errors.Join(errs...)
}

func decodeJSON(r io.Reader, v any) error {
	raw, err := io.ReadAll(io.LimitReader(r, maxCatalogBytes+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > maxCatalogBytes {
		return fmt.Errorf("response exceeded %d bytes", maxCatalogBytes)
	}
	if !utf8.Valid(raw) {
		return fmt.Errorf("response contains invalid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra json.RawMessage
	err = dec.Decode(&extra)
	if err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}
