package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Model struct {
	Task      string   `json:"pipeline_tag"`
	Tags      []string `json:"tags"`
	Likes     int      `json:"likes"`
	ID        string   `json:"id"`
	Downloads int      `json:"downloads"`
	Gated     any      `json:"gated"`
}

type cached struct {
	data []byte
	next string
	at   time.Time
}
type Client struct {
	BaseURL string
	HTTP    *http.Client
	mutex   sync.Mutex
	cache   map[string]cached
}

func New() *Client {
	return &Client{BaseURL: "https://huggingface.co", HTTP: &http.Client{Timeout: 25 * time.Second}, cache: map[string]cached{}}
}

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)

func (c *Client) get(ctx context.Context, path string, target any) error {
	_, err := c.getPage(ctx, path, target)
	return err
}

func (c *Client) getPage(ctx context.Context, path string, target any) (string, error) {
	c.mutex.Lock()
	held, ok := c.cache[path]
	c.mutex.Unlock()
	if ok && time.Since(held.at) < 5*time.Minute {
		return held.next, json.Unmarshal(held.data, target)
	}
	request, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return "", err
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return "", errors.New("Could not reach Hugging Face")
	}
	defer response.Body.Close()
	if response.StatusCode == 429 {
		return "", errors.New("Hugging Face is busy. Try again shortly")
	}
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return "", errors.New("This model requires access through Hugging Face")
	}
	if response.StatusCode != 200 {
		return "", fmt.Errorf("Hugging Face request failed (HTTP %d)", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return "", err
	}
	if err = json.Unmarshal(data, target); err != nil {
		return "", errors.New("Invalid Hugging Face response")
	}
	next := ""
	for _, link := range strings.Split(response.Header.Get("Link"), ",") {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 || !strings.Contains(link, `rel="next"`) {
			continue
		}
		address, parseErr := url.Parse(strings.Trim(parts[0], "<> "))
		base, _ := url.Parse(c.BaseURL)
		if parseErr != nil || address.Host != base.Host || address.Scheme != base.Scheme || (address.Path != "/api/models" && !strings.HasPrefix(address.Path, "/api/models/")) {
			return "", errors.New("Invalid Hugging Face pagination")
		}
		next = address.RequestURI()
	}
	c.mutex.Lock()
	if len(c.cache) >= 64 {
		c.cache = map[string]cached{}
	}
	c.cache[path] = cached{data: data, next: next, at: time.Now()}
	c.mutex.Unlock()
	return next, nil
}

func chatTask(task, id string) bool {
	if task != "" && task != "text-generation" && task != "image-text-to-text" && task != "conversational" {
		return false
	}
	lower := strings.ToLower(id)
	for _, term := range []string{"embedding", "reranker", "-asr-", "-tts-", "whisper"} {
		if strings.Contains(lower, term) {
			return false
		}
	}
	return true
}
