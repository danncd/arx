package web

import (
	"bytes"
	"golang.org/x/net/html"
	"net/url"
	"strings"
)

func pageText(body []byte, address string) string {
	base, _ := url.Parse(address)
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return string(body)
	}
	var output strings.Builder
	images := 0
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if attribute(node, "aria-hidden") == "true" {
				return
			}
			for _, attr := range node.Attr {
				if attr.Key == "hidden" {
					return
				}
			}
			switch node.Data {
			case "head", "script", "style", "noscript", "svg", "nav", "footer", "aside", "template":
				return
			}
			switch node.Data {
			case "p", "div", "section", "article", "br", "li", "h1", "h2", "h3", "h4", "pre", "tr":
				output.WriteString("\n")
			}
		}
		if node.Type == html.ElementNode && node.Data == "img" && base != nil && images < 12 {
			source, fallback, alt := "", "", ""
			for _, attribute := range node.Attr {
				switch attribute.Key {
				case "src":
					source = attribute.Val
				case "data-src":
					fallback = attribute.Val
				case "alt":
					alt = attribute.Val
				}
			}
			if source == "" || strings.HasPrefix(source, "data:") {
				source = fallback
			}
			if source != "" {
				link, err := url.Parse(source)
				if err == nil {
					link = base.ResolveReference(link)
					if validateURL(link) == nil && len(link.String()) <= 8192 {
						images++
						output.WriteString("\n[Image: " + shortText(strings.Join(strings.Fields(alt), " "), 200) + "] (" + link.String() + ")\n")
					}
				}
			}
		}
		if node.Type == html.TextNode {
			text := strings.Join(strings.Fields(node.Data), " ")
			if text != "" {
				output.WriteString(text)
				output.WriteByte(' ')
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if node.Type == html.ElementNode && node.Data == "a" && base != nil {
			for _, attribute := range node.Attr {
				if attribute.Key != "href" {
					continue
				}
				link, err := url.Parse(attribute.Val)
				if err != nil {
					continue
				}
				link = base.ResolveReference(link)
				if validateURL(link) == nil {
					output.WriteString("(" + link.String() + ") ")
				}
			}
		}
	}
	visit(contentRoot(document))
	lines := []string{}
	for _, line := range strings.Split(output.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func pageTitle(body []byte) string {
	tokens := html.NewTokenizer(bytes.NewReader(body))
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			return ""
		}
		if kind == html.StartTagToken {
			name, _ := tokens.TagName()
			if string(name) == "title" && tokens.Next() == html.TextToken {
				return shortText(strings.TrimSpace(string(tokens.Text())), 500)
			}
		}
	}
}

func shortText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}
