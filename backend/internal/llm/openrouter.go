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
	for _, m := range list.Data {
		key := normalizeModelID(m.ID)
		// Base ids and dated/":free" variants can collide after
		// normalization; first entry wins so the choice is deterministic.
		if _, exists := index[key]; !exists {
			index[key] = m
		}
	}
	return index, nil
}

var datedSnapshot = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`)

func normalizeModelID(id string) string {
	if _, bare, ok := strings.Cut(id, "/"); ok {
		id = bare
	}
	if i := strings.IndexByte(id, ':'); i >= 0 {
		id = id[:i]
	}
	id = datedSnapshot.ReplaceAllString(id, "")
	return strings.ToLower(id)
}
