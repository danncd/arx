package web

import (
	tool "arx/internal/tools"
	"bytes"
	"context"
	"errors"
	"golang.org/x/net/html"
	"net/url"
	"strings"
)

func (c *Client) duckSearch(ctx context.Context, query string) ([]tool.Source, error) {
	body, _, _, err := c.get(ctx, c.duckURL+"?"+url.Values{"q": {query}}.Encode())
	if err != nil {
		return nil, err
	}
	return duckResults(body)
}

func duckResults(body []byte) ([]tool.Source, error) {
	raw := strings.ToLower(string(body))
	if strings.Contains(raw, "challenge-form") || strings.Contains(raw, "anomaly.js") || strings.Contains(raw, "captcha") {
		return nil, errors.New("DuckDuckGo requires browser verification")
	}
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("DuckDuckGo returned unreadable results")
	}
	results := []tool.Source{}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if hasClass(node, "result") {
			var result tool.Source
			var read func(*html.Node)
			read = func(child *html.Node) {
				if hasClass(child, "result__a") {
					result.Title = nodeText(child)
					result.URL = duckLink(attribute(child, "href"))
				}
				if hasClass(child, "result__snippet") {
					result.Snippet = nodeText(child)
				}
				for next := child.FirstChild; next != nil; next = next.NextSibling {
					read(next)
				}
			}
			read(node)
			if result.URL != "" {
				results = append(results, result)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if len(results) == 0 && !strings.Contains(raw, "no-results") {
		return nil, errors.New("DuckDuckGo returned no recognizable search results")
	}
	return results, nil
}

func hasClass(node *html.Node, class string) bool {
	for _, value := range strings.Fields(attribute(node, "class")) {
		if value == class {
			return true
		}
	}
	return false
}

func nodeText(node *html.Node) string {
	var output strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			output.WriteString(node.Data)
			output.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return strings.Join(strings.Fields(output.String()), " ")
}

func duckLink(value string) string {
	address, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if address.Scheme == "" {
		address.Scheme = "https"
	}
	if address.Hostname() == "duckduckgo.com" || strings.HasSuffix(address.Hostname(), ".duckduckgo.com") {
		value = address.Query().Get("uddg")
		address, err = url.Parse(value)
		if err != nil {
			return ""
		}
	}
	if validateURL(address) != nil {
		return ""
	}
	return address.String()
}
