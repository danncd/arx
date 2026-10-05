package web

import (
	tool "arx/internal/tools"
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

func (c *Client) fetch(ctx context.Context, address string) (tool.Result, error) {
	body, address, kind, err := c.get(ctx, address)
	if err != nil {
		return tool.Result{}, err
	}
	if !utf8.Valid(body) || strings.IndexByte(string(body), 0) >= 0 {
		return tool.Result{}, errors.New("Page is not UTF-8 text")
	}
	var text string
	if strings.Contains(kind, "html") {
		if blockedPage(body) {
			return tool.Result{}, errors.New("This page requires browser verification or denies access; its content was not retrieved")
		}
		text = pageText(body, address)
		if strings.TrimSpace(text) == "" {
			return tool.Result{}, errors.New("This page contains no readable text; it may require JavaScript")
		}
	} else if strings.HasPrefix(kind, "text/") || strings.Contains(kind, "json") || strings.Contains(kind, "xml") {
		text = string(body)
	} else {
		return tool.Result{}, errors.New("This page type is not supported")
	}
	result := tool.Output("Source: " + address + "\n\n" + text)
	title := address
	if strings.Contains(kind, "html") {
		if value := pageTitle(body); value != "" {
			title = value
		}
	}
	result.Sources = []tool.Source{{URL: address, Title: title}}
	return result, nil
}
