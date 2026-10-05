package web

import (
	tool "arx/internal/tools"
	"context"
	"encoding/xml"
	"errors"
	"net/url"
)

func (c *Client) bingSearch(ctx context.Context, query string) ([]tool.Source, error) {
	address := c.searchURL + "?" + url.Values{"q": {query}, "format": {"rss"}}.Encode()
	body, _, _, err := c.get(ctx, address)
	if err != nil {
		return nil, err
	}
	var feed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, errors.New("Search provider returned an unreadable response")
	}

	sources := []tool.Source{}
	for _, item := range feed.Channel.Items {
		sources = append(sources, tool.Source{URL: item.Link, Title: item.Title, Snippet: pageText([]byte(item.Description), item.Link)})
	}
	return sources, nil
}
