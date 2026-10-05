package web

import (
	"golang.org/x/net/html"
	"strings"
)

func contentRoot(document *html.Node) *html.Node {
	var main, article *html.Node
	articles := 0
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if main == nil && (node.Data == "main" || attribute(node, "role") == "main") {
				main = node
			}
			if node.Data == "article" {
				articles++
				article = node
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	if main != nil {
		return main
	}
	if articles == 1 {
		return article
	}
	return document
}

func attribute(node *html.Node, key string) string {
	for _, value := range node.Attr {
		if value.Key == key {
			return value.Val
		}
	}
	return ""
}

func blockedPage(body []byte) bool {
	title := strings.ToLower(pageTitle(body))
	for _, marker := range []string{"just a moment", "access denied", "verify you are human", "attention required!", "robot or human"} {
		if strings.Contains(title, marker) {
			return true
		}
	}
	return false
}
