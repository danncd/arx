package web

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) Image(ctx context.Context, source string) (string, error) {
	data, kind, err := c.image(ctx, source)
	if err != nil {
		return "", err
	}
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (c *Client) image(ctx context.Context, source string) ([]byte, string, error) {
	if len(source) > 8192 {
		return nil, "", errors.New("Image URL is too long")
	}
	address, err := url.Parse(source)
	if err != nil || validateURL(address) != nil {
		return nil, "", errors.New("Invalid image URL")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address.String(), nil)
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/webp")
	request.Header.Set("User-Agent", "Arx/0.1")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", errors.New("Image request failed: " + response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(body) > 8<<20 {
		return nil, "", errors.New("Image exceeds 8 MiB")
	}
	kind := http.DetectContentType(body)
	switch kind {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return body, kind, nil
	default:
		return nil, "", errors.New("Unsupported image format")
	}
}
