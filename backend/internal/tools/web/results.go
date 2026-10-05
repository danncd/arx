package web

import (
	tool "arx/internal/tools"
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

func searchResult(query string, candidates []tool.Source) tool.Result {
	var output strings.Builder
	fmt.Fprintf(&output, "Search query: %s\nSearch snippets are previews; fetch relevant sources before relying on their contents.\n\n", query)
	sources := []tool.Source{}
	seen := map[string]bool{}
	relevant := false
	for _, item := range candidates {
		address, err := url.Parse(strings.TrimSpace(item.URL))
		if err != nil || validateURL(address) != nil || strings.TrimSpace(item.Title) == "" {
			continue
		}
		address.Fragment = ""
		key := strings.ToLower(address.Host) + strings.TrimRight(address.EscapedPath(), "/") + "?" + address.RawQuery
		if seen[key] || !matchesSite(query, address.Hostname()) {
			continue
		}
		seen[key] = true
		item.URL = address.String()
		item.Title = shortText(strings.Join(strings.Fields(item.Title), " "), 500)
		item.Snippet = shortText(strings.Join(strings.Fields(item.Snippet), " "), 1000)
		relevant = relevant || matchesQuery(query, item.Title+" "+item.Snippet+" "+item.URL)
		sources = append(sources, item)
		fmt.Fprintf(&output, "%d. %s\n%s\n%s\n\n", len(sources), item.Title, item.URL, item.Snippet)
		if len(sources) == 10 {
			break
		}
	}
	if len(sources) == 0 {
		output.WriteString("No usable results returned for this query. This does not establish that no matching pages exist.")
	} else if !relevant {
		output.WriteString("Results may not match the full query. Refine the search or use a known primary-source URL; do not treat these snippets as an answer.")
	}
	result := tool.Output(output.String())
	result.Sources = sources
	return result
}

func matchesSite(query, host string) bool {
	host = strings.ToLower(host)
	for _, word := range strings.Fields(strings.ToLower(query)) {
		if !strings.HasPrefix(word, "site:") {
			continue
		}
		site := strings.Trim(strings.TrimPrefix(word, "site:"), "\"/")
		if site == "" {
			continue
		}
		site = strings.Split(site, "/")[0]
		if host != site && !strings.HasSuffix(host, "."+site) {
			return false
		}
	}
	return true
}

func matchesQuery(query, text string) bool {
	text = strings.ToLower(text)
	total, matched := 0, 0
	stopwords := " a an the is are of for to in on and or with what how latest news "
	for _, word := range strings.Fields(strings.ToLower(query)) {
		if strings.HasPrefix(word, "site:") || strings.HasPrefix(word, "-") {
			continue
		}
		word = strings.TrimFunc(word, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if len(word) < 2 || strings.Contains(stopwords, " "+word+" ") {
			continue
		}
		total++
		hasLetter, hasDigit := false, false
		for _, char := range word {
			hasLetter = hasLetter || unicode.IsLetter(char)
			hasDigit = hasDigit || unicode.IsDigit(char)
		}
		if hasLetter && hasDigit && !strings.Contains(text, word) {
			return false
		}
		if strings.Contains(text, word) {
			matched++
		}
	}
	return total == 0 || matched*5 >= total*3
}
