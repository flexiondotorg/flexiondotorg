package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRSS20FormatsSortingAndEntities(t *testing.T) {
	t.Parallel()

	feed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Older &amp; Classic &#8212; RFC822Z</title>
      <link>https://example.com/1</link>
      <pubDate>01 Jan 24 10:00 +0000</pubDate>
      <description>Summary &lt;1&gt;</description>
    </item>
    <item>
      <title>Newest &quot;Episode&quot;</title>
      <link>https://example.com/3</link>
      <pubDate>2025-05-10T18:30:00Z</pubDate>
      <description>Latest summary</description>
    </item>
    <item>
      <title>Middle &#39;Post&#39;</title>
      <link>https://example.com/2</link>
      <pubDate>Wed, 15 Jan 2025 12:00:00 GMT</pubDate>
      <description>Middle summary</description>
    </item>
    <item>
      <title>RFC1123Z Item</title>
      <link>https://example.com/4</link>
      <pubDate>Thu, 20 Feb 2025 09:00:00 +0100</pubDate>
      <description>Fourth summary</description>
    </item>
  </channel>
</rss>`

	items, err := Parse([]byte(feed), 3)
	if err != nil {
		t.Fatalf("Parse RSS failed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	if items[0].Title != `Newest "Episode"` || items[0].URL != "https://example.com/3" {
		t.Errorf("items[0] = %+v", items[0])
	}
	if items[1].Title != "RFC1123Z Item" || !items[1].PublishedAt.Equal(time.Date(2025, 2, 20, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("items[1] = %+v", items[1])
	}
	if items[2].Title != "Middle 'Post'" || items[2].URL != "https://example.com/2" {
		t.Errorf("items[2] = %+v", items[2])
	}

	allItems, err := Parse([]byte(feed), 0)
	if err != nil {
		t.Fatalf("Parse RSS without limit failed: %v", err)
	}
	if len(allItems) != 4 {
		t.Fatalf("len(allItems) = %d, want 4", len(allItems))
	}
	if allItems[3].Title != "Older & Classic — RFC822Z" || allItems[3].Summary != "Summary <1>" {
		t.Errorf("entity unescaping failed: %+v", allItems[3])
	}
}

func TestParseYouTubeAtomUsesPublishedNotUpdated(t *testing.T) {
	t.Parallel()

	// Regression test: an archive video has its <updated> timestamp bumped to 2026
	// by a YouTube description edit, while a genuinely new video was published in 2025.
	// Parse must strictly use <published> and ignore <updated>.
	youtubeFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns:yt="http://www.youtube.com/xml/schemas/2015"
      xmlns:media="http://search.yahoo.com/mrss/"
      xmlns="http://www.w3.org/2005/Atom">
  <title>Example Channel</title>
  <entry>
    <title>Archive Video With Edited Metadata &amp; Notes</title>
    <link rel="self" href="https://www.youtube.com/api/self/old"/>
    <link rel="alternate" href="https://www.youtube.com/watch?v=oldvideo"/>
    <published>2021-03-14T10:00:00+00:00</published>
    <updated>2026-04-01T23:59:59+00:00</updated>
    <media:group>
      <media:description>Archive description</media:description>
    </media:group>
  </entry>
  <entry>
    <title>Brand New Release</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=newvideo"/>
    <published>2025-11-20T15:30:00+00:00</published>
    <updated>2025-11-20T15:35:00+00:00</updated>
    <media:group>
      <media:description>Fresh video &amp; walkthrough</media:description>
    </media:group>
  </entry>
</feed>`

	items, err := Parse([]byte(youtubeFeed), 0)
	if err != nil {
		t.Fatalf("Parse YouTube Atom failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	if items[0].URL != "https://www.youtube.com/watch?v=newvideo" {
		t.Fatalf("expected newest published video first, got %q", items[0].URL)
	}
	wantNew := time.Date(2025, 11, 20, 15, 30, 0, 0, time.UTC)
	if !items[0].PublishedAt.Equal(wantNew) {
		t.Errorf("items[0].PublishedAt = %v, want %v", items[0].PublishedAt, wantNew)
	}
	if items[0].Summary != "Fresh video & walkthrough" {
		t.Errorf("items[0].Summary = %q", items[0].Summary)
	}

	if items[1].URL != "https://www.youtube.com/watch?v=oldvideo" {
		t.Fatalf("expected archive video second (with rel=alternate link), got %q", items[1].URL)
	}
	wantOld := time.Date(2021, 3, 14, 10, 0, 0, 0, time.UTC)
	if !items[1].PublishedAt.Equal(wantOld) {
		t.Errorf("items[1].PublishedAt = %v, want %v (must ignore <updated>)", items[1].PublishedAt, wantOld)
	}
}

func TestParseAtomLinkFallbackAndMissingPublished(t *testing.T) {
	t.Parallel()

	atomFeed := `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <entry>
    <title>No Rel Link</title>
    <link href="https://example.com/first"/>
    <updated>2026-01-01T00:00:00Z</updated>
    <summary>Atom summary</summary>
  </entry>
</feed>`

	items, err := Parse([]byte(atomFeed), 5)
	if err != nil {
		t.Fatalf("Parse Atom failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].URL != "https://example.com/first" {
		t.Errorf("items[0].URL = %q", items[0].URL)
	}
	if !items[0].PublishedAt.IsZero() {
		t.Errorf("items[0].PublishedAt = %v, want zero time when <published> is absent", items[0].PublishedAt)
	}
	if items[0].Summary != "Atom summary" {
		t.Errorf("items[0].Summary = %q", items[0].Summary)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()

	if _, err := Parse(nil, 5); err == nil {
		t.Fatal("expected error for empty input")
	}
	if _, err := Parse([]byte(`<html></html>`), 5); err == nil {
		t.Fatal("expected error for unsupported root element")
	}
	if _, err := Parse([]byte(`<rss><channel><item><pubDate>not-a-date</pubDate></item></channel></rss>`), 5); err == nil {
		t.Fatal("expected error for invalid pubDate")
	}
}

func TestFetch(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss version="2.0"><channel><item><title>Hello &amp; World</title><link>https://example.com/post</link><pubDate>Mon, 02 Jan 2025 15:04:05 +0000</pubDate></item></channel></rss>`))
		case "/fail":
			http.Error(w, "boom", http.StatusBadGateway)
		case "/huge":
			_, _ = w.Write([]byte(strings.Repeat("a", maxBodyBytes+10)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := context.Background()
	items, err := Fetch(ctx, srv.Client(), srv.URL+"/ok.xml", 5)
	if err != nil {
		t.Fatalf("Fetch ok failed: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Hello & World" {
		t.Fatalf("unexpected items: %+v", items)
	}

	if _, err := Fetch(ctx, srv.Client(), srv.URL+"/fail", 5); err == nil {
		t.Fatal("expected error for HTTP 502")
	}
	if _, err := Fetch(ctx, srv.Client(), srv.URL+"/huge", 5); err == nil {
		t.Fatal("expected error for oversized response")
	}
	if _, err := Fetch(ctx, srv.Client(), "file:///etc/passwd", 5); err == nil {
		t.Fatal("expected error for non-http scheme")
	}
}
