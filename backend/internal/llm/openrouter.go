package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var orModelsURL = "https://openrouter.ai/api/v1/models"

type orModel struct {
	ID                  string   `json:"id"`
	ContextLength       int      `json:"context_length"`
	SupportedParameters []string `json:"supported_parameters"`
	Architecture        struct {
		Modality string `json:"modality"`
	} `json:"architecture"`
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

type orList struct {
	Data []orModel `json:"data"`
}

func fetchORCatalog(ctx context.Context) (map[string]orModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", orModelsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter /models: HTTP %d", resp.StatusCode)
	}
	var list orList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode openrouter catalog: %w", err)
	}
	index := make(map[string]orModel, len(list.Data))
	isVariant := map[string]bool{}
	for _, m := range list.Data {
		id := m.ID
		variant := false
		// ":free"/":nitro" routing variants share the base model's key
		// but can differ in context length and tool support — the base
		// entry's metadata must win regardless of response order.
		if i := strings.IndexByte(id, ':'); i >= 0 {
			id, variant = id[:i], true
		}
		key := normalizeModelID(id)
		if _, exists := index[key]; !exists || (isVariant[key] && !variant) {
			index[key] = m
			isVariant[key] = variant
		}
	}
	return index, nil
}

var datedSnapshot = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`)

// normalizeModelID reduces a model id to a joinable bare name: vendor
// prefix off, dated snapshot off, lowercased. Colons are deliberately
// NOT stripped here — OpenAI fine-tune ids ("ft:gpt-4o:org::abc")
// contain them structurally, and collapsing those to "ft" merged every
// fine-tune onto one key. OpenRouter's ":free" variant suffixes are
// handled at index-build time in fetchORCatalog instead.
func normalizeModelID(id string) string {
	if _, bare, ok := strings.Cut(id, "/"); ok {
		id = bare
	}
	id = datedSnapshot.ReplaceAllString(id, "")
	return strings.ToLower(id)
}
