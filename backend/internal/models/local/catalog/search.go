package catalog

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

type SearchOptions struct {
	Query  string `json:"query"`
	Sort   string `json:"sort,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}
type SearchPage struct {
	Models []Model `json:"models"`
	Next   string  `json:"next"`
}

func (c *Client) SearchPage(ctx context.Context, options SearchOptions) (SearchPage, error) {
	result := SearchPage{Models: []Model{}}
	query := strings.TrimSpace(options.Query)
	if len(query) > 160 || len(options.Cursor) > 8192 {
		return result, errors.New("Invalid model search")
	}
	sort := options.Sort
	if sort == "" {
		sort = "downloads"
	}
	switch sort {
	case "downloads", "trendingScore", "likes", "lastModified":
	default:
		return result, errors.New("Invalid model sort")
	}
	values := url.Values{"search": {query}, "filter": {"gguf"}, "sort": {sort}, "direction": {"-1"}, "limit": {"20"}, "full": {"true"}}
	if options.Cursor != "" {
		values.Set("cursor", options.Cursor)
	}
	var models []Model
	next, err := c.getPage(ctx, "/api/models?"+values.Encode(), &models)
	if err != nil {
		return result, err
	}
	for _, item := range models {
		if chatTask(item.Task, item.ID) {
			result.Models = append(result.Models, item)
		}
	}
	if next != "" {
		address, err := url.Parse(next)
		if err != nil || address.Path != "/api/models" {
			return result, errors.New("Invalid model pagination")
		}
		result.Next = address.Query().Get("cursor")
	}
	return result, nil
}
