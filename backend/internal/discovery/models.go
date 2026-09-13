package discovery

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

	"arx/internal/provider"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

const maxCatalogBytes int64 = 8 << 20

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

type ModelInfo struct {
	Spec       string
	Provider   string
	Model      string
	MaxOutput  int
	Tools      bool
	ToolsKnown bool
}

type modelList struct {
	Data []Model `json:"data"`
}

var nonChatMarkers = []string{
	"embedding", "tts", "whisper", "audio", "transcribe", "image", "sora",
	"realtime", "moderation", "search-api", "babbage", "davinci", "instruct",
	"dall-e", "codex-mini", "computer-use",
}

func providerChatCapable(p provider.Provider, id string) bool {
	if p.Name == "openai" {
		bare := normalizeModelID(id)
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

func ListModels(ctx context.Context, p provider.Provider) ([]Model, error) {
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

func LoadModels(ctx context.Context) ([]ModelInfo, error) {
	var out []ModelInfo
	var errs []error
	var names []string
	for name, p := range provider.Providers {
		if p.KeyEnv == "" || p.Key() != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	sort.Strings(names)

	catalog, catalogErr := fetchORCatalog(ctx)
	if catalogErr != nil {
		errs = append(errs, fmt.Errorf("openrouter catalog: %w", catalogErr))
	}

	for _, name := range names {
		p := provider.Providers[name]
		models, err := ListModels(ctx, p)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}

		for _, m := range models {
			if !providerChatCapable(p, m.ID) {
				continue
			}
			info := ModelInfo{
				Spec:     name + "/" + m.ID,
				Provider: name,
				Model:    m.ID,
			}
			om, known := catalog[normalizeModelID(m.ID)]
			if known {
				if mod := om.Architecture.Modality; mod != "" && !strings.HasSuffix(mod, "->text") {
					continue
				}
				info.MaxOutput = om.TopProvider.MaxCompletionTokens
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
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("multiple JSON values")
	}
	return fmt.Errorf("trailing JSON data: %w", err)
}
