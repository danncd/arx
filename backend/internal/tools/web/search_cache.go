package web

import (
	tool "arx/internal/tools"
	"context"
	"time"
)

type searchEntry struct {
	result  tool.Result
	err     error
	expires time.Time
	pending chan struct{}
}

func (c *Client) cachedSearch(ctx context.Context, query string, run func() (tool.Result, error)) (tool.Result, error) {
	c.searchMutex.Lock()
	if c.searchCache == nil {
		c.searchCache = map[string]*searchEntry{}
	}
	if entry := c.searchCache[query]; entry != nil {
		if entry.pending != nil {
			pending := entry.pending
			c.searchMutex.Unlock()
			select {
			case <-ctx.Done():
				return tool.Result{}, ctx.Err()
			case <-pending:
				return entry.result, entry.err
			}
		}
		if time.Now().Before(entry.expires) {
			c.searchMutex.Unlock()
			return entry.result, entry.err
		}
	}
	if len(c.searchCache) >= 64 {
		for key, entry := range c.searchCache {
			if entry.pending == nil {
				delete(c.searchCache, key)
				break
			}
		}
	}
	if len(c.searchCache) >= 64 {
		c.searchMutex.Unlock()
		return run()
	}
	entry := &searchEntry{pending: make(chan struct{})}
	c.searchCache[query] = entry
	c.searchMutex.Unlock()
	result, err := run()
	c.searchMutex.Lock()
	entry.result, entry.err = result, err
	lifetime := 5 * time.Minute
	if err != nil {
		lifetime = 20 * time.Second
	}
	entry.expires = time.Now().Add(lifetime)
	close(entry.pending)
	entry.pending = nil
	if ctx.Err() != nil {
		delete(c.searchCache, query)
	}
	c.searchMutex.Unlock()
	return result, err
}
