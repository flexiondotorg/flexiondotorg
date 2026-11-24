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
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 15 * time.Second
	maxBodyBytes       = 4 << 20 // 4 MiB
	userAgent          = "vitrine/1.0"

	itunesNamespace  = "http://www.itunes.com/dtds/podcast-1.0.dtd"
	contentNamespace = "http://purl.org/rss/1.0/modules/content/"

	minReadTimeWords = 60
	wordsPerMinute   = 220
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
	Title           string
	URL             string
	PublishedAt     time.Time
	Summary         string
	Episode         string
	Duration        string
	Subtitle        string
	AudioURL        string
	ReadTimeMinutes int
}

type xmlText struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	PubDate     string         `xml:"pubDate"`
	Description string         `xml:"description"`
	Summaries   []xmlText      `xml:"summary"`
	Encoded     []xmlText      `xml:"encoded"`
	Episodes    []xmlText      `xml:"episode"`
	Durations   []xmlText      `xml:"duration"`
	Subtitles   []xmlText      `xml:"subtitle"`
	Enclosures  []rssEnclosure `xml:"enclosure"`
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

		subtitle := cleanText(selectNamespaceText(raw.Subtitles, itunesNamespace, true))
		if subtitle == "" {
			subtitle = cleanText(selectNamespaceText(raw.Summaries, "", false))
		}
		encoded := selectNamespaceText(raw.Encoded, contentNamespace, true)

		items = append(items, Item{
			Title:           cleanText(raw.Title),
			URL:             strings.TrimSpace(raw.Link),
			PublishedAt:     publishedAt,
			Summary:         cleanText(raw.Description),
			Episode:         cleanText(selectNamespaceText(raw.Episodes, itunesNamespace, true)),
			Duration:        formatDuration(selectNamespaceText(raw.Durations, itunesNamespace, true)),
			Subtitle:        subtitle,
			AudioURL:        selectAudioEnclosure(raw.Enclosures),
			ReadTimeMinutes: estimateReadTimeMinutes(encoded, raw.Description),
		})
	}
	return items, nil
}

func selectAudioEnclosure(enclosures []rssEnclosure) string {
	var fallback string
	for _, enc := range enclosures {
		rawURL := strings.TrimSpace(enc.URL)
		if err := validateHTTPURL(rawURL); err != nil {
			continue
		}
		mediaType := strings.ToLower(strings.TrimSpace(enc.Type))
		if strings.HasPrefix(mediaType, "audio/") {
			return rawURL
		}
		lowerURL := strings.ToLower(rawURL)
		if fallback == "" && (strings.HasSuffix(lowerURL, ".mp3") || strings.HasSuffix(lowerURL, ".m4a") || strings.HasSuffix(lowerURL, ".ogg") || strings.HasSuffix(lowerURL, ".opus")) {
			fallback = rawURL
		}
	}
	return fallback
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
			Subtitle:    cleanText(entry.Summary),
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

func selectNamespaceText(nodes []xmlText, preferredSpace string, fallbackAny bool) string {
	var fallback string
	for _, node := range nodes {
		if strings.TrimSpace(node.Value) == "" {
			continue
		}
		if node.XMLName.Space == preferredSpace {
			return node.Value
		}
		if fallbackAny && fallback == "" {
			fallback = node.Value
		}
	}
	return fallback
}

func formatDuration(raw string) string {
	totalSeconds, ok := parseDurationSeconds(raw)
	if !ok || totalSeconds <= 0 {
		return ""
	}

	totalMinutes := max(1, (totalSeconds+30)/60)
	if totalMinutes < 60 {
		return fmt.Sprintf("%d min", totalMinutes)
	}
	hours := totalMinutes / 60
	mins := totalMinutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func parseDurationSeconds(raw string) (int, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		return parseNonNegInt(parts[0])
	case 2:
		mins, ok1 := parseNonNegInt(parts[0])
		sec, ok2 := parseNonNegInt(parts[1])
		if !ok1 || !ok2 || sec >= 60 || mins > math.MaxInt/60-1 {
			return 0, false
		}
		return mins*60 + sec, true
	case 3:
		hours, ok1 := parseNonNegInt(parts[0])
		mins, ok2 := parseNonNegInt(parts[1])
		sec, ok3 := parseNonNegInt(parts[2])
		if !ok1 || !ok2 || !ok3 || mins >= 60 || sec >= 60 || hours > math.MaxInt/3600-1 {
			return 0, false
		}
		return hours*3600 + mins*60 + sec, true
	default:
		return 0, false
	}
}

func parseNonNegInt(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func estimateReadTimeMinutes(htmlBodies ...string) int {
	bestWords := 0
	for _, body := range htmlBodies {
		if strings.TrimSpace(body) == "" {
			continue
		}
		if words := countHTMLWords(body); words > bestWords {
			bestWords = words
		}
	}
	if bestWords < minReadTimeWords {
		return 0
	}
	return max(1, (bestWords+wordsPerMinute/2)/wordsPerMinute)
}

func countHTMLWords(raw string) int {
	plain := html.UnescapeString(stripHTMLTags(raw))
	return len(strings.Fields(plain))
}

func stripHTMLTags(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	lastByte := byte(0)
	for i := 0; i < len(s); {
		if s[i] == '<' {
			if next, ok := skipHTMLTag(s, i); ok {
				if needsTagSeparator(lastByte, s, next) {
					b.WriteByte(' ')
					lastByte = ' '
				}
				i = next
				continue
			}
		}
		b.WriteByte(s[i])
		lastByte = s[i]
		i++
	}
	return b.String()
}

func skipHTMLTag(s string, start int) (int, bool) {
	if start+1 >= len(s) {
		return 0, false
	}
	if strings.HasPrefix(s[start:], "<!--") {
		if end := strings.Index(s[start+4:], "-->"); end >= 0 {
			return start + 4 + end + 3, true
		}
		return len(s), true
	}
	if !isTagStartChar(s[start+1]) {
		return 0, false
	}

	var quote byte
	for i := start + 1; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '>' {
			return i + 1, true
		}
	}
	return 0, false
}

func isTagStartChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '/' || ch == '!' || ch == '?'
}

func needsTagSeparator(lastByte byte, s string, nextIdx int) bool {
	if lastByte == 0 || isASCIIWhitespace(lastByte) || nextIdx >= len(s) {
		return false
	}
	next := s[nextIdx]
	if isASCIIWhitespace(next) {
		return false
	}
	switch next {
	case '.', ',', ';', ':', '!', '?', ')', ']':
		return false
	default:
		return true
	}
}

func isASCIIWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v'
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
