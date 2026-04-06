package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WebFetch fetches a URL and returns cleaned text content.
func executeWebFetch(input json.RawMessage, _ string) (string, bool) {
	var args struct {
		URL    string `json:"url"`
		Prompt string `json:"prompt,omitempty"` // optional instruction
	}
	json.Unmarshal(input, &args)

	if args.URL == "" {
		return "URL is required", true
	}

	log.Printf("[WebFetch] Fetching: %s", args.URL)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", args.URL, nil)
	if err != nil {
		return fmt.Sprintf("Invalid URL: %v", err), true
	}
	req.Header.Set("User-Agent", "AniClew/1.0 (Web Fetch Tool)")
	req.Header.Set("Accept", "text/html, application/json, text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Fetch error: %v", err), true
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status), true
	}

	// Read body (limit 500KB)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024))
	if err != nil {
		return fmt.Sprintf("Read error: %v", err), true
	}

	content := string(body)
	contentType := resp.Header.Get("Content-Type")

	// JSON response — return as-is
	if strings.Contains(contentType, "json") {
		if len(content) > 10000 {
			content = content[:10000] + "\n... (truncated)"
		}
		return content, false
	}

	// HTML — strip tags
	if strings.Contains(contentType, "html") {
		content = htmlToText(content)
	}

	// Truncate
	if len(content) > 15000 {
		content = content[:15000] + "\n\n... (truncated, full page was larger)"
	}

	result := fmt.Sprintf("URL: %s\nStatus: %d\nContent-Type: %s\n\n%s",
		args.URL, resp.StatusCode, contentType, content)

	return result, false
}

// WebSearch performs a simple search (using DuckDuckGo HTML).
func executeWebSearch(input json.RawMessage, _ string) (string, bool) {
	var args struct {
		Query string `json:"query"`
	}
	json.Unmarshal(input, &args)

	if args.Query == "" {
		return "Query is required", true
	}

	url := "https://html.duckduckgo.com/html/?q=" + strings.ReplaceAll(args.Query, " ", "+")
	log.Printf("[WebSearch] Searching: %s", args.Query)

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "AniClew/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Search error: %v", err), true
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 200*1024))
	text := htmlToText(string(body))

	// Extract search results
	if len(text) > 5000 {
		text = text[:5000]
	}

	return fmt.Sprintf("Search: %s\n\n%s", args.Query, text), false
}

// htmlToText strips HTML tags and returns readable text.
func htmlToText(html string) string {
	// Remove script and style blocks
	html = regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<nav[^>]*>.*?</nav>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?s)<footer[^>]*>.*?</footer>`).ReplaceAllString(html, "")

	// Replace block elements with newlines
	html = regexp.MustCompile(`<br\s*/?\s*>`).ReplaceAllString(html, "\n")
	html = regexp.MustCompile(`</?(p|div|h[1-6]|li|tr|section|article)[^>]*>`).ReplaceAllString(html, "\n")

	// Strip remaining tags
	html = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(html, "")

	// Decode common entities
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&nbsp;", " ")

	// Collapse whitespace
	html = regexp.MustCompile(`[ \t]+`).ReplaceAllString(html, " ")
	html = regexp.MustCompile(`\n{3,}`).ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}
