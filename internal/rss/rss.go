// Package rss fetches and parses RSS 2.0 and Atom feeds (including YouTube Atom feeds).
package rss

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 15 * time.Second
	maxBodyBytes       = 4 << 20 // 4 MiB
	userAgent          = "vitrine/1.0"
)

var dateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
	time.RFC822Z,
	time.RFC822,
	time.RFC3339Nano,
	time.RFC3339,
	"2 Jan 2006 15:04:05 -0700",
}

// Item represents a single normalised entry from an RSS 2.0 or Atom feed.
type Item struct {
	Title       string
	URL         string
	PublishedAt time.Time
	Summary     string
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

type atomDocument struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	Links      []atomLink     `xml:"link"`
	Published  string         `xml:"published"`
	Summary    string         `xml:"summary"`
	MediaGroup atomMediaGroup `xml:"http://search.yahoo.com/mrss/ group"`
}

type atomMediaGroup struct {
	Description string `xml:"http://search.yahoo.com/mrss/ description"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

// Fetch downloads and parses an RSS 2.0 or Atom feed from feedURL.
func Fetch(ctx context.Context, client *http.Client, feedURL string, limit int) ([]Item, error) {
	if err := validateHTTPURL(feedURL); err != nil {
		return nil, fmt.Errorf("invalid feed url: %w", err)
	}

	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create feed request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	// #nosec G704 -- feed URL is validated to http/https and supplied via configuration.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch feed %q: %w", feedURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch feed %q: unexpected status %d", feedURL, resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read feed %q: %w", feedURL, err)
	}
	if len(body) > maxBodyBytes {
		return nil, fmt.Errorf("read feed %q: response exceeds %d bytes", feedURL, maxBodyBytes)
	}

	items, err := Parse(body, limit)
	if err != nil {
		return nil, fmt.Errorf("parse feed %q: %w", feedURL, err)
	}
	return items, nil
}

// Parse decodes RSS 2.0 or Atom XML bytes, sorts items newest-first, and applies limit.
func Parse(data []byte, limit int) ([]Item, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("feed data is empty")
	}

	rootName, err := detectRootElement(trimmed)
	if err != nil {
		return nil, err
	}

	var items []Item
	switch strings.ToLower(rootName) {
	case "rss":
		items, err = parseRSS(trimmed)
	case "feed":
		items, err = parseAtom(trimmed)
	default:
		return nil, fmt.Errorf("unsupported feed root element %q", rootName)
	}
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(items, func(a, b Item) int {
		if a.PublishedAt.After(b.PublishedAt) {
			return -1
		}
		if a.PublishedAt.Before(b.PublishedAt) {
			return 1
		}
		return 0
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func detectRootElement(data []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("decode xml root: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name.Local, nil
		}
	}
}

func parseRSS(data []byte) ([]Item, error) {
	var doc rssDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal rss: %w", err)
	}

	items := make([]Item, 0, len(doc.Channel.Items))
	for _, raw := range doc.Channel.Items {
		publishedAt, err := parseDate(raw.PubDate)
		if err != nil {
			return nil, fmt.Errorf("parse rss pubDate %q: %w", raw.PubDate, err)
		}
		items = append(items, Item{
			Title:       cleanText(raw.Title),
			URL:         strings.TrimSpace(raw.Link),
			PublishedAt: publishedAt,
			Summary:     cleanText(raw.Description),
		})
	}
	return items, nil
}

func parseAtom(data []byte) ([]Item, error) {
	var doc atomDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal atom: %w", err)
	}

	items := make([]Item, 0, len(doc.Entries))
	for _, entry := range doc.Entries {
		// Strictly parse <published> and never fall back to <updated>,
		// because YouTube bumps <updated> on metadata edits to old videos.
		publishedAt, err := parseDate(entry.Published)
		if err != nil {
			return nil, fmt.Errorf("parse atom published %q: %w", entry.Published, err)
		}

		summary := entry.Summary
		if strings.TrimSpace(summary) == "" {
			summary = entry.MediaGroup.Description
		}

		items = append(items, Item{
			Title:       cleanText(entry.Title),
			URL:         selectAtomLink(entry.Links),
			PublishedAt: publishedAt,
			Summary:     cleanText(summary),
		})
	}
	return items, nil
}

func selectAtomLink(links []atomLink) string {
	var firstHref string
	var noRelHref string
	for _, link := range links {
		href := strings.TrimSpace(link.Href)
		if href == "" {
			continue
		}
		if firstHref == "" {
			firstHref = href
		}
		rel := strings.TrimSpace(link.Rel)
		if strings.EqualFold(rel, "alternate") {
			return href
		}
		if rel == "" && noRelHref == "" {
			noRelHref = href
		}
	}
	if noRelHref != "" {
		return noRelHref
	}
	return firstHref
}

func parseDate(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, nil
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format %q", s)
}

func cleanText(s string) string {
	return strings.TrimSpace(html.UnescapeString(s))
}

func validateHTTPURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return errors.New("url must not be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return errors.New("url host must not be empty")
	}
	return nil
}
