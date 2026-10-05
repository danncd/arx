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

func (c *Client) Icon(ctx context.Context, source string) (string, error) {
	if len(source) > 8192 {
		return "", errors.New("URL is too long")
	}
	address, err := url.Parse(source)
	if err != nil || validateURL(address) != nil {
		return "", errors.New("Invalid site URL")
	}
	address.Path, address.RawPath, address.RawQuery, address.Fragment = "/favicon.ico", "", "", ""
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address.String(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "image/png,image/jpeg,image/gif,image/webp,image/x-icon")
	response, err := c.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("Site icon is unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (256<<10)+1))
	if err != nil || len(body) > 256<<10 {
		return "", errors.New("Site icon is too large")
	}
	kind := http.DetectContentType(body)
	switch kind {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/x-icon", "image/vnd.microsoft.icon":
		return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(body), nil
	default:
		return "", errors.New("Unsupported site icon")
	}
}
