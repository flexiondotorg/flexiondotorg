package rss

import (
	"context"
	"fmt"
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

func TestParsePodcastRSSMetadata(t *testing.T) {
	t.Parallel()

	podcastFeed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd">
  <channel>
    <title>Linux Matters Podcast</title>
    <item>
      <title>Episode 94</title>
      <link>https://linuxmatters.sh/94/</link>
      <pubDate>Fri, 14 Nov 2025 10:00:00 +0000</pubDate>
      <description>&lt;p&gt;Short show notes.&lt;/p&gt;</description>
      <itunes:episode> 94 </itunes:episode>
      <itunes:duration>00:35:00</itunes:duration>
      <itunes:subtitle>Containers, Nix &amp; desktop workflows</itunes:subtitle>
      <itunes:summary>Longer podcast summary that must not override subtitle.</itunes:summary>
    </item>
    <item>
      <title>Episode 93</title>
      <link>https://linuxmatters.sh/93/</link>
      <pubDate>Fri, 07 Nov 2025 10:00:00 +0000</pubDate>
      <itunes:episode>93</itunes:episode>
      <itunes:duration>00:41:57</itunes:duration>
      <itunes:subtitle>Rounding up &amp; &#8217;testing&#8217;</itunes:subtitle>
    </item>
    <item>
      <title>Episode 92</title>
      <link>https://linuxmatters.sh/92/</link>
      <pubDate>Fri, 31 Oct 2025 10:00:00 +0000</pubDate>
      <itunes:episode>92</itunes:episode>
      <itunes:duration>01:05:10</itunes:duration>
      <summary>Fallback summary when itunes subtitle is absent</summary>
    </item>
    <item>
      <title>Episode 91</title>
      <link>https://linuxmatters.sh/91/</link>
      <pubDate>Fri, 24 Oct 2025 10:00:00 +0000</pubDate>
      <itunes:episode>91</itunes:episode>
      <itunes:duration>3600</itunes:duration>
      <itunes:summary>iTunes summary alone stays separate from bare summary</itunes:summary>
    </item>
    <item>
      <title>Episode 90</title>
      <link>https://linuxmatters.sh/90/</link>
      <pubDate>Fri, 17 Oct 2025 10:00:00 +0000</pubDate>
      <itunes:episode>90</itunes:episode>
      <itunes:duration>2400</itunes:duration>
    </item>
    <item>
      <title>Episode 89</title>
      <link>https://linuxmatters.sh/89/</link>
      <pubDate>Fri, 10 Oct 2025 10:00:00 +0000</pubDate>
      <itunes:episode>89</itunes:episode>
      <itunes:duration>41:57</itunes:duration>
    </item>
  </channel>
</rss>`

	items, err := Parse([]byte(podcastFeed), 0)
	if err != nil {
		t.Fatalf("Parse podcast RSS failed: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("len(items) = %d, want 6", len(items))
	}

	want := []struct {
		episode  string
		duration string
		subtitle string
	}{
		{episode: "94", duration: "35 min", subtitle: "Containers, Nix & desktop workflows"},
		{episode: "93", duration: "42 min", subtitle: "Rounding up & ’testing’"},
		{episode: "92", duration: "1h 5m", subtitle: "Fallback summary when itunes subtitle is absent"},
		{episode: "91", duration: "1h", subtitle: ""},
		{episode: "90", duration: "40 min", subtitle: ""},
		{episode: "89", duration: "42 min", subtitle: ""},
	}

	for i, tc := range want {
		if items[i].Episode != tc.episode {
			t.Errorf("items[%d].Episode = %q, want %q", i, items[i].Episode, tc.episode)
		}
		if items[i].Duration != tc.duration {
			t.Errorf("items[%d].Duration = %q, want %q", i, items[i].Duration, tc.duration)
		}
		if items[i].Subtitle != tc.subtitle {
			t.Errorf("items[%d].Subtitle = %q, want %q", i, items[i].Subtitle, tc.subtitle)
		}
	}
}

func TestFormatDurationEdgeCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw  string
		want string
	}{
		{raw: "00:35:00", want: "35 min"},
		{raw: "00:41:57", want: "42 min"},
		{raw: "01:05:10", want: "1h 5m"},
		{raw: "3600", want: "1h"},
		{raw: "2400", want: "40 min"},
		{raw: "35:00", want: "35 min"},
		{raw: "75:30", want: "1h 16m"},
		{raw: "00:59:45", want: "1h"},
		{raw: "00:00:15", want: "1 min"},
		{raw: "20", want: "1 min"},
		{raw: "0", want: ""},
		{raw: "00:00:00", want: ""},
		{raw: "", want: ""},
		{raw: "   ", want: ""},
		{raw: "not-a-duration", want: ""},
		{raw: "01:99:00", want: ""},
		{raw: "01:02:03:04", want: ""},
		{raw: "-100", want: ""},
		{raw: "10:-5", want: ""},
	}

	for _, tc := range cases {
		if got := formatDuration(tc.raw); got != tc.want {
			t.Errorf("formatDuration(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestParseBlogRSSReadTimeAndSubtitle(t *testing.T) {
	t.Parallel()

	makeWords := func(n int) string {
		words := make([]string, n)
		for i := range n {
			words[i] = fmt.Sprintf("w%d", i+1)
		}
		return strings.Join(words, " ")
	}

	// 440 words -> (440+110)/220 = 2 min.
	longHTML := "<article><h1>Post Heading</h1><p>" + makeWords(200) +
		"</p><p>" + makeWords(238) + `</p><!-- hidden comment --><p><a href="https://example.com" title="a > b">link</a>.</p></article>`
	// 60 words -> minimum threshold -> max(1, (60+110)/220) = 1 min.
	thresholdHTML := "<p>" + makeWords(60) + "</p>"
	// 59 words -> below 60-word threshold -> 0 min.
	shortHTML := "<p>" + makeWords(59) + "</p>"
	// 660 words in content:encoded -> (660+110)/220 = 3 min (longer than short description).
	encodedHTML := "<p>" + makeWords(660) + "</p>"

	blogFeed := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Wimpy's World</title>
    <item>
      <title>Full HTML Description Post</title>
      <link>https://wimpysworld.com/posts/full-html/</link>
      <pubDate>Thu, 20 Nov 2025 12:00:00 +0000</pubDate>
      <summary>A crisp one-line summary &amp; overview.</summary>
      <description><![CDATA[%s]]></description>
    </item>
    <item>
      <title>Content Encoded Post</title>
      <link>https://wimpysworld.com/posts/content-encoded/</link>
      <pubDate>Wed, 19 Nov 2025 12:00:00 +0000</pubDate>
      <summary>Short summary for encoded post.</summary>
      <description>&lt;p&gt;Brief teaser.&lt;/p&gt;</description>
      <content:encoded><![CDATA[%s]]></content:encoded>
    </item>
    <item>
      <title>Threshold 60 Words</title>
      <link>https://wimpysworld.com/posts/threshold/</link>
      <pubDate>Tue, 18 Nov 2025 12:00:00 +0000</pubDate>
      <summary>Exactly sixty words.</summary>
      <description><![CDATA[%s]]></description>
    </item>
    <item>
      <title>Too Short 59 Words</title>
      <link>https://wimpysworld.com/posts/short/</link>
      <pubDate>Mon, 17 Nov 2025 12:00:00 +0000</pubDate>
      <summary>Just under sixty words.</summary>
      <description><![CDATA[%s]]></description>
    </item>
  </channel>
</rss>`, longHTML, encodedHTML, thresholdHTML, shortHTML)

	items, err := Parse([]byte(blogFeed), 0)
	if err != nil {
		t.Fatalf("Parse blog RSS failed: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(items))
	}

	want := []struct {
		title    string
		subtitle string
		readTime int
	}{
		{
			title:    "Full HTML Description Post",
			subtitle: "A crisp one-line summary & overview.",
			readTime: 2,
		},
		{
			title:    "Content Encoded Post",
			subtitle: "Short summary for encoded post.",
			readTime: 3,
		},
		{
			title:    "Threshold 60 Words",
			subtitle: "Exactly sixty words.",
			readTime: 1,
		},
		{
			title:    "Too Short 59 Words",
			subtitle: "Just under sixty words.",
			readTime: 0,
		},
	}

	for i, tc := range want {
		if items[i].Title != tc.title {
			t.Errorf("items[%d].Title = %q, want %q", i, items[i].Title, tc.title)
		}
		if items[i].Subtitle != tc.subtitle {
			t.Errorf("items[%d].Subtitle = %q, want %q", i, items[i].Subtitle, tc.subtitle)
		}
		if items[i].ReadTimeMinutes != tc.readTime {
			t.Errorf("items[%d].ReadTimeMinutes = %d, want %d", i, items[i].ReadTimeMinutes, tc.readTime)
		}
	}

	// Ensure <summary> stays distinct from full HTML <description> stored in Summary.
	if !strings.Contains(items[0].Summary, "<h1>Post Heading</h1>") {
		t.Errorf("expected items[0].Summary to keep description content, got %q", items[0].Summary)
	}
}
