package web

import (
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

type arguments struct {
	Operation string `json:"operation"`
	Query     string `json:"query"`
	URL       string `json:"url"`
}

func (c *Client) Prepare(raw json.RawMessage) (tool.Prepared, error) {
	var args arguments
	if err := tool.Decode(raw, &args); err != nil {
		return tool.Prepared{}, err
	}
	switch args.Operation {
	case "fetch", "image":
		if args.URL == "" || len(args.URL) > 8192 {
			return tool.Prepared{}, errors.New("A URL is required (up to 8192 characters)")
		}
	case "search":
		if strings.TrimSpace(args.Query) == "" || len(args.Query) > 2000 {
			return tool.Prepared{}, errors.New("A search query is required (up to 2000 characters)")
		}
	default:
		return tool.Prepared{}, errors.New("Unknown web operation")
	}
	return tool.Prepared{Action: permission.Action{Tool: "web", Operation: args.Operation, Query: args.Query, URL: args.URL}, Run: func(ctx context.Context) (tool.Result, error) {
		if args.Operation == "search" {
			return c.search(ctx, args.Query)
		}
		if args.Operation == "image" {
			data, _, err := c.image(ctx, args.URL)
			if err != nil {
				return tool.Result{}, err
			}
			address, _ := url.Parse(args.URL)
			return tool.Result{Text: "Image loaded for visual inspection: " + args.URL, ImageData: data, ImageName: address.Path, Sources: []tool.Source{{URL: args.URL, Title: "Image"}}}, nil
		}
		return c.fetch(ctx, args.URL)
	}}, nil
}
