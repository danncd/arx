package web

import (
	tool "arx/internal/tools"
	"context"
	"fmt"
	"strings"
	"time"
)

func (c *Client) search(ctx context.Context, query string) (tool.Result, error) {
	query = strings.TrimSpace(query)
	return c.cachedSearch(ctx, query, func() (tool.Result, error) { return c.searchProviders(ctx, query) })
}

func (c *Client) searchProviders(ctx context.Context, query string) (tool.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()
	type searchProvider struct {
		name string
		run  func(context.Context, string) ([]tool.Source, error)
	}
	providers := []searchProvider{}
	if c.duckURL != "" {
		providers = append(providers, searchProvider{"DuckDuckGo", c.duckSearch})
	}
	providers = append(providers, searchProvider{"Bing", c.bingSearch})
	failures := []string{}
	for _, provider := range providers {
		if err := ctx.Err(); err != nil {
			return tool.Result{}, err
		}
		attempt, stop := context.WithTimeout(ctx, 8*time.Second)
		candidates, err := provider.run(attempt, query)
		stop()
		if err != nil {
			failures = append(failures, provider.name+": "+err.Error())
			continue
		}
		result := searchResult(query, candidates)
		matching := []tool.Source{}
		for _, source := range result.Sources {
			if matchesQuery(query, source.Title+" "+source.Snippet+" "+source.URL) {
				matching = append(matching, source)
			}
		}
		if len(matching) == 0 {
			failures = append(failures, provider.name+": no sufficiently matching results")
			continue
		}
		result = searchResult(query, matching)
		result.Text = "Search provider: " + provider.name + "\n" + result.Text
		return result, nil
	}
	return tool.Result{}, fmt.Errorf("Search could not retrieve reliable results for %q. %s. Do not substitute unrelated results or conclude that no matching pages exist. Try a more specific query or fetch a known source URL.", query, strings.Join(failures, "; "))
}
