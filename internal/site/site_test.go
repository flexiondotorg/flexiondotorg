package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	"github.com/flexiondotorg/flexiondotorg/internal/discord"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
)

func TestBrandIconSVG(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"mastodon", "bluesky", "linkedin", "discord", "matrix",
		"twitch", "youtube", "steam", "itchio", "apple-podcasts",
		"spotify", "rss", "github", "podcast", "blog", "globe", "heart",
	} {
		out := BrandIconSVG(key)
		if !strings.Contains(out, `viewBox="0 0 24 24"`) {
			t.Errorf("BrandIconSVG(%q) missing viewBox: %s", key, out)
		}
		if !strings.Contains(out, `fill="currentColor"`) {
			t.Errorf("BrandIconSVG(%q) missing currentColor: %s", key, out)
		}
		if strings.Contains(out, fallbackStarPath) {
			t.Errorf("BrandIconSVG(%q) unexpectedly used fallback path", key)
		}
	}

	unknown := BrandIconSVG("unknown-brand")
	if !strings.Contains(unknown, fallbackStarPath) {
		t.Errorf("BrandIconSVG(unknown) did not use fallback path: %s", unknown)
	}
}

func TestSanitizeLogoArtwork(t *testing.T) {
	t.Parallel()

	// Unsafe script and event handler stripped from inline SVG.
	raw := `<svg viewBox="0 0 32 32"><script>alert(1)</script><defs><linearGradient id="g"><stop offset="0%"/></linearGradient></defs><rect width="32" height="32" fill="url(#g)" onclick="evil()"/></svg>`
	cleaned, dataURI := SanitizeLogoArtwork("demo", "Demo", raw, "")
	if dataURI != "" {
		t.Fatalf("unexpected dataURI %q for valid inline SVG", dataURI)
	}
	if strings.Contains(cleaned, "<script") || strings.Contains(cleaned, "onclick") {
		t.Fatalf("SanitizeLogoArtwork leaked unsafe content: %s", cleaned)
	}
	if !strings.Contains(cleaned, "vlogo-") {
		t.Fatalf("SanitizeLogoArtwork did not namespace internal IDs: %s", cleaned)
	}

	// PNG data URI accepted, unsafe scheme rejected.
	validPNG := "data:image/png;base64,iVBORw0KGgo="
	svgOut, uriOut := SanitizeLogoArtwork("tool", "Tool", "", validPNG)
	if svgOut != "" || uriOut != validPNG {
		t.Fatalf("PNG data URI = (%q, %q), want (\"\", %q)", svgOut, uriOut, validPNG)
	}

	svgOut, uriOut = SanitizeLogoArtwork("tool", "Tool", "", "javascript:alert(1)")
	if uriOut != "" || !strings.Contains(svgOut, ">TO</text>") {
		t.Fatalf("unsafe data URI should fall back to monogram, got (%q, %q)", svgOut, uriOut)
	}

	// jive-* fallback renders waveform bars.
	jiveSVG, _ := SanitizeLogoArtwork("jive-talkin", "jive-talkin", "", "")
	if strings.Count(jiveSVG, "<rect ") != 5 {
		t.Fatalf("expected 5 waveform bars for jive-* slug, got: %s", jiveSVG)
	}

	// Multi-word monogram derives two initials.
	monoSVG, _ := SanitizeLogoArtwork("deb-get", "deb-get", "", "")
	if !strings.Contains(monoSVG, ">DG</text>") {
		t.Fatalf("expected DG initials in monogram, got: %s", monoSVG)
	}
}

func TestBuildAndWrite(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	cfg, err := config.Parse([]byte(`
profile:
  name: "Martin Wimpress"
  handle: "flexiondotorg"
  hero_title: "MARTIN WIMPRESS"
  hero_subtitle: "Linux enthusiast. Nix wrangler."
  bio: >-
    Hello from [Oval Tutu](https://oval-tutu.com) and [Linux Matters](https://linuxmatters.sh).

    Second paragraph with plain text.
github:
  username: "flexiondotorg"
  sponsors_url: "https://github.com/sponsors/flexiondotorg"
haberdasher:
  enabled: true
  sort: stars
feeds:
  podcast:
    enabled: true
    title: "Linux Matters Podcast"
    url: "https://linuxmatters.sh/episode/index.xml"
    site_url: "https://linuxmatters.sh"
    cover_url: "https://example.com/cover.png"
    links:
      - label: "Apple Podcasts"
        url: "https://podcasts.apple.com/example"
        icon: "apple-podcasts"
      - label: "RSS"
        url: "https://linuxmatters.sh/episode/index.xml"
        icon: "rss"
  blog:
    enabled: true
    title: "Blog"
    url: "https://wimpysworld.com/posts/rss.xml"
    site_url: "https://wimpysworld.com"
site:
  title: "Wimpy's World"
  description: "Linux enthusiast. Nix wrangler."
  domain: "wimpys.world"
  url: "https://wimpys.world"
  default_deck: "projects"
  links:
    - title: "Wimpy's World Discord"
      slug: "discord"
      category: "Community"
      description: "Friendly Linux hub."
      url: "https://wimpysworld.io/discord"
      icon: "discord"
      accent: "lavender"
    - title: "Matrix"
      slug: "matrix"
      category: "Community"
      description: "Federated chat."
      url: "https://matrix.to/#/@wimpress:matrix.org"
      icon: "matrix"
      accent: "teal"
    - title: "GitHub"
      slug: "github"
      category: "Code"
      description: "Open-source repositories on GitHub."
      url: "https://github.com/flexiondotorg"
      icon: "github"
      accent: "mauve"
    - title: "GitHub Sponsors"
      slug: "github-sponsors"
      category: "Support"
      description: "Sponsor open-source work."
      url: "https://github.com/sponsors/flexiondotorg"
      icon: "heart"
      accent: "pink"
    - title: "Wimpy's World Blog"
      slug: "wimpys-world-blog"
      category: "Sites"
      description: "Articles on Linux and NixOS."
      url: "https://wimpysworld.com"
      icon: "blog"
      accent: "mauve"
    - title: "Linux Matters"
      slug: "linux-matters"
      category: "Sites"
      description: "Fortnightly Linux podcast."
      url: "https://linuxmatters.sh"
      icon: "podcast"
      accent: "blue"
    - title: "Mastodon"
      slug: "mastodon"
      category: "Social"
      description: "Fediverse updates."
      url: "https://wimpysworld.social/@martin"
      icon: "mastodon"
      accent: "mauve"
    - title: "Bluesky"
      slug: "bluesky"
      category: "Social"
      description: "Short-form notes."
      url: "https://bsky.app/profile/wimpys.world"
      icon: "bluesky"
      accent: "blue"
    - title: "Twitch"
      slug: "twitch"
      category: "Video"
      description: "Live Linux streams."
      url: "https://twitch.tv/WimpysWorld"
      icon: "twitch"
      accent: "mauve"
    - title: "YouTube"
      slug: "youtube"
      category: "Video"
      description: "Linux walkthroughs."
      url: "https://youtube.com/WimpysWorld"
      icon: "youtube"
      accent: "red"
    - title: "Steam"
      slug: "steam"
      category: "Gaming"
      description: "PC gaming profile."
      url: "https://steamcommunity.com/id/wimpress/"
      icon: "steam"
      accent: "sapphire"
    - title: "itch.io"
      slug: "itchio"
      category: "Gaming"
      description: "Indie game releases."
      url: "https://wimpress.itch.io/"
      icon: "itchio"
      accent: "peach"
    - title: "Custom Stat Link"
      slug: "custom-stat"
      category: "Sites"
      description: "Custom stats override."
      url: "https://oval-tutu.com"
      icon: "globe"
      stats:
        - label: "Focus"
          value: "Indie Games"
`))
	if err != nil {
		t.Fatalf("config.Parse failed: %v", err)
	}

	cat := &haberdasher.Catalogue{
		Projects: []haberdasher.Project{
			{
				FullName:    "wimpysworld/deb-get",
				Owner:       "wimpysworld",
				Name:        "deb-get",
				Slug:        "deb-get",
				Description: "apt-get functionality for third-party .debs",
				Homepage:    "https://wimpysworld.github.io/deb-get",
				RepoURL:     "https://github.com/wimpysworld/deb-get",
				Language:    "Shell",
				Stars:       4200,
				LatestTag:   "v0.4.5",
				ReleaseURL:  "https://github.com/wimpysworld/deb-get/releases/tag/v0.4.5",
				PublishedAt: now.Add(-3 * 24 * time.Hour),
				Hosts:       []string{"apt.wimpys.world", "nix.wimpys.world"},
				LogoSVG:     `<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`,
			},
			{
				FullName:    "linuxmatters/jive-talkin",
				Owner:       "linuxmatters",
				Name:        "jive-talkin",
				Slug:        "jive-talkin",
				Description: "Audio mastering helper </script><script>alert(1)</script>",
				RepoURL:     "https://github.com/linuxmatters/jive-talkin",
				Language:    "Go",
				Stars:       120,
				LatestTag:   "v1.0.0",
				PublishedAt: now.Add(-60 * 24 * time.Hour),
			},
		},
		Hosts: []haberdasher.PackageHost{
			{
				Name:         "apt",
				Host:         "apt.wimpys.world",
				URL:          "https://apt.wimpys.world",
				LogoSVG:      `<svg viewBox="0 0 24 24"><rect width="24" height="24"/></svg>`,
				PackageCount: 2,
				Repos:        []string{"wimpysworld/ia-get", "wimpysworld/deb-get"},
			},
			{
				Name:         "nix",
				Host:         "nix.wimpys.world",
				URL:          "https://nix.wimpys.world",
				PackageCount: 1,
				Repos:        []string{"wimpysworld/deb-get"},
			},
		},
	}

	ghData := &gh.ProfileData{
		Stats: gh.Stats{
			Username:     "flexiondotorg",
			TotalStars:   12345,
			Followers:    800,
			PublicRepos:  40,
			OrgRepos:     25,
			TotalCommits: 9001,
		},
		Sponsors:     []gh.Sponsor{{Login: "alice"}, {Login: "bob"}},
		PastSponsors: []gh.Sponsor{{Login: "carol"}},
		SponsorCount: 2,
	}

	disc := discord.Status{
		Name:    "Wimpy's World Discord",
		Online:  184,
		Members: 1520,
		URL:     "https://wimpysworld.io/discord",
	}

	podcast := []rss.Item{
		{
			Title:       "NixOS All The Way",
			URL:         "https://linuxmatters.sh/episode/94",
			PublishedAt: now.Add(-2 * 24 * time.Hour),
			Episode:     "94",
			Duration:    "42 min",
			AudioURL:    "https://audio.linuxmatters.net/LMP94.mp3",
			Subtitle:    "Talking NixOS flakes and Haberdasher.",
		},
	}
	blog := []rss.Item{
		{
			Title:           "Building Static Sites in Go",
			URL:             "https://wimpysworld.com/posts/static-go",
			PublishedAt:     now.Add(-5 * 24 * time.Hour),
			Summary:         "<p>How <b>cmd/site</b> works.</p><pre><code>fmt.Println(1)</code></pre><p>Fast and simple.</p>",
			ReadTimeMinutes: 6,
		},
	}

	data, err := Build(Input{
		Config:    cfg,
		Catalogue: cat,
		GitHub:    ghData,
		Discord:   disc,
		Social: map[string]SocialProfile{
			"mastodon": {Followers: 3282, Posts: 327},
			"bluesky":  {Followers: 821, Posts: 144},
			"twitch":   {Followers: 1533},
			"youtube":  {Subscribers: "4.48K", Videos: "144"},
			"steam":    {Games: 296, Level: 15},
			"itchio":   {Games: 2},
		},
		Podcast:     podcast,
		Blog:        blog,
		GeneratedAt: now,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if data.DefaultDeck != "projects" || data.DefaultSort != "stars" {
		t.Fatalf("defaults = (%q, %q), want (projects, stars)", data.DefaultDeck, data.DefaultSort)
	}
	if len(data.Profile.Bio) != 2 || len(data.Profile.Bio[0].Segments) < 3 {
		t.Fatalf("unexpected parsed bio: %+v", data.Profile.Bio)
	}
	if data.Stats.TotalStars != 12345 || data.Stats.ActiveSponsors != 2 || data.Stats.PastSponsors != 1 {
		t.Fatalf("unexpected stats: %+v", data.Stats)
	}
	if len(data.Projects) != 2 || !data.Projects[0].Fresh || data.Projects[1].Fresh {
		t.Fatalf("unexpected projects freshness: %+v", data.Projects)
	}
	if len(data.Projects[0].Hosts) != 2 || data.Projects[0].Hosts[0].URL != "https://apt.wimpys.world/#deb-get" {
		t.Fatalf("unexpected project hosts: %+v", data.Projects[0].Hosts)
	}
	if len(data.HostLogos) != 2 || data.HostLogos["apt.wimpys.world"] == "" {
		t.Fatalf("expected HostLogos populated for both hosts: %+v", data.HostLogos)
	}
	for host, logo := range data.HostLogos {
		if !strings.Contains(logo, `xmlns="http://www.w3.org/2000/svg"`) {
			t.Errorf("HostLogos[%q] missing SVG namespace: %s", host, logo)
		}
	}
	for _, proj := range data.Projects {
		if proj.LogoSVG != "" && !strings.Contains(proj.LogoSVG, `xmlns="http://www.w3.org/2000/svg"`) {
			t.Errorf("Project %q LogoSVG missing SVG namespace: %s", proj.Name, proj.LogoSVG)
		}
	}
	for _, link := range data.Links {
		if !strings.Contains(link.IconSVG, `xmlns="http://www.w3.org/2000/svg"`) {
			t.Errorf("Link %q IconSVG missing SVG namespace: %s", link.Title, link.IconSVG)
		}
	}
	if len(data.Packages) != 2 || len(data.Packages[0].Packages) != 2 {
		t.Fatalf("unexpected packages: %+v", data.Packages)
	}
	if data.Packages[0].Packages[0].Name != "deb-get" || data.Packages[0].Packages[0].URL != "https://apt.wimpys.world/#deb-get" {
		t.Fatalf("packages not sorted or formatted cleanly: %+v", data.Packages[0].Packages)
	}
	if len(data.Links) != 13 {
		t.Fatalf("len(Links) = %d, want 13", len(data.Links))
	}
	if data.Links[0].Badge != "184 online" || len(data.Links[0].Meta) != 2 || len(data.Links[0].Stats) != 2 || data.Links[0].Stats[0].Label != "ONLINE" || !data.Links[0].Stats[0].Highlight {
		t.Fatalf("discord link not enriched: %+v", data.Links[0])
	}
	if len(data.Links[1].Stats) != 1 || data.Links[1].Stats[0].Label != "HANDLE" || data.Links[1].Stats[0].Value != "@wimpress:matrix.org" {
		t.Fatalf("matrix link identity stat wrong: %+v", data.Links[1].Stats)
	}
	if len(data.Links[2].Stats) != 2 || data.Links[2].Stats[0].Label != "STARS" || data.Links[2].Stats[0].Value != "12,345" || data.Links[2].Stats[1].Label != "FOLLOWERS" || data.Links[2].Stats[1].Value != "800" {
		t.Fatalf("github link stats wrong: %+v", data.Links[2].Stats)
	}
	if len(data.Links[2].Meta) != 4 || len(data.Links[2].Sponsors) != 2 || len(data.Links[2].PastSponsors) != 1 {
		t.Fatalf("github link meta/sponsors not enriched: %+v", data.Links[2])
	}
	if data.Links[3].Badge != "2 active" || len(data.Links[3].Meta) != 2 || len(data.Links[3].Stats) != 2 {
		t.Fatalf("sponsors link not enriched: %+v", data.Links[3])
	}
	if len(data.Links[4].Stats) != 2 || data.Links[4].Stats[0].Label != "LATEST" || data.Links[4].Stats[1].Label != "SITE" {
		t.Fatalf("blog link stats wrong: %+v", data.Links[4].Stats)
	}
	if len(data.Links[5].Stats) != 2 || data.Links[5].Stats[0].Value != "#94" || data.Links[5].Stats[1].Label != "PUBLISHED" {
		t.Fatalf("podcast link stats wrong: %+v", data.Links[5].Stats)
	}
	if len(data.Links[6].Stats) != 2 || data.Links[6].Stats[0].Value != "3,282" || data.Links[6].Stats[1].Value != "327" {
		t.Fatalf("mastodon link live stats wrong: %+v", data.Links[6].Stats)
	}
	if len(data.Links[7].Stats) != 2 || data.Links[7].Stats[0].Value != "821" || data.Links[7].Stats[1].Value != "144" {
		t.Fatalf("bluesky link live stats wrong: %+v", data.Links[7].Stats)
	}
	if len(data.Links[8].Stats) != 2 || data.Links[8].Stats[0].Value != "1,533" || data.Links[8].Stats[1].Value != "WimpysWorld" {
		t.Fatalf("twitch link live stats wrong: %+v", data.Links[8].Stats)
	}
	if len(data.Links[9].Stats) != 2 || data.Links[9].Stats[0].Value != "4.48K" || data.Links[9].Stats[1].Value != "144" {
		t.Fatalf("youtube link live stats wrong: %+v", data.Links[9].Stats)
	}
	if len(data.Links[10].Stats) != 2 || data.Links[10].Stats[0].Value != "296" || data.Links[10].Stats[1].Value != "15" {
		t.Fatalf("steam link live stats wrong: %+v", data.Links[10].Stats)
	}
	if len(data.Links[11].Stats) != 2 || data.Links[11].Stats[0].Value != "2" || data.Links[11].Stats[1].Value != "wimpress.itch.io" {
		t.Fatalf("itch.io link live stats wrong: %+v", data.Links[11].Stats)
	}
	if len(data.Links[12].Stats) != 1 || data.Links[12].Stats[0].Label != "FOCUS" || data.Links[12].Stats[0].Value != "Indie Games" {
		t.Fatalf("custom link stats wrong: %+v", data.Links[12].Stats)
	}
	if len(data.Links[3].Sponsors) != 2 || data.Links[3].Sponsors[0].Login != "alice" || data.Links[3].Sponsors[0].URL != "https://github.com/alice" {
		t.Fatalf("sponsors handles not populated: %+v", data.Links[3].Sponsors)
	}
	if len(data.Links[3].PastSponsors) != 1 || data.Links[3].PastSponsors[0].Login != "carol" {
		t.Fatalf("past sponsors handles not populated: %+v", data.Links[3].PastSponsors)
	}
	if len(data.Media) != 2 {
		t.Fatalf("len(Media) = %d, want 2 (1 episode + 1 post)", len(data.Media))
	}
	if data.Media[0].Kind != "episode" || len(data.Media[0].SubscribeLinks) != 2 || data.Media[0].AudioURL != "https://audio.linuxmatters.net/LMP94.mp3" {
		t.Fatalf("unexpected podcast episode card: %+v", data.Media[0])
	}
	if data.Media[1].Kind != "post" || strings.Contains(data.Media[1].Summary, "<p>") || strings.Contains(data.Media[1].Summary, "<b>") {
		t.Fatalf("blog summary did not strip HTML tags: %q", data.Media[1].Summary)
	}
	if !strings.Contains(data.Media[1].SpeechText, "Building Static Sites in Go.") || strings.Contains(data.Media[1].SpeechText, "fmt.Println") {
		t.Fatalf("blog speech text did not strip code blocks cleanly: %q", data.Media[1].SpeechText)
	}

	outDir := t.TempDir()
	if err := Write(outDir, data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	for _, name := range []string{
		"index.html",
		"data.json",
		"style.css",
		"app.js",
		"icon.svg",
		"vendor/pixi.min.mjs",
		"vendor/pixi.LICENSE",
		".nojekyll",
		"CNAME",
	} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected output file %q: %v", name, err)
		}
	}

	cname, err := os.ReadFile(filepath.Join(outDir, "CNAME"))
	if err != nil {
		t.Fatalf("read CNAME: %v", err)
	}
	if string(cname) != "wimpys.world\n" {
		t.Errorf("CNAME = %q, want %q", string(cname), "wimpys.world\n")
	}

	indexBytes, err := os.ReadFile(filepath.Join(outDir, "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	indexHTML := string(indexBytes)
	if strings.Contains(indexHTML, "</script><script>alert(1)</script>") {
		t.Fatal("index.html did not escape </script> inside inline JSON payload")
	}

	// Verify embedded JSON in #site-data parses cleanly back into SiteData.
	const marker = `<script id="site-data" type="application/json">`
	start := strings.Index(indexHTML, marker)
	if start == -1 {
		t.Fatal("index.html missing #site-data script element")
	}
	rest := indexHTML[start+len(marker):]
	end := strings.Index(rest, "</script>")
	if end == -1 {
		t.Fatal("index.html missing closing </script> for #site-data")
	}
	var roundTrip SiteData
	if err := json.Unmarshal([]byte(rest[:end]), &roundTrip); err != nil {
		t.Fatalf("unmarshal embedded #site-data JSON failed: %v", err)
	}
	if len(roundTrip.Projects) != 2 || roundTrip.Profile.Title != "Wimpy's World" {
		t.Fatalf("unexpected round-tripped SiteData: %+v", roundTrip.Profile)
	}
	if !strings.Contains(indexHTML, `<script type="module" src="./app.js"></script>`) {
		t.Fatal("index.html missing ES module script tag for app.js")
	}
	for _, wantID := range []string{
		"site-header",
		"btn-theme-system",
		"btn-theme-light",
		"btn-theme-dark",
		"pixi-root",
		"intro-bio",
		"scroll-spacer",
		"site-footer",
		"card-modal",
		"flip-float-wrap",
		"flip-scene",
		"flip-card",
		"flip-face-front",
		"flip-face-back",
	} {
		if !strings.Contains(indexHTML, `id="`+wantID+`"`) {
			t.Errorf("index.html missing expected element id %q", wantID)
		}
	}
	for _, wantHeader := range []string{
		`class="header-inner"`,
		`<a class="brand-block" href="./">`,
		`aria-label="Site sections"`,
		`href="./" aria-current="page"`,
		`href="https://blog.wimpys.world"`,
		`class="header-utility header-github-btn"`,
		`href="https://github.com/flexiondotorg"`,
	} {
		if !strings.Contains(indexHTML, wantHeader) {
			t.Errorf("index.html header missing expected markup %q", wantHeader)
		}
	}
	if strings.Index(indexHTML, `id="intro-bio"`) > strings.Index(indexHTML, `id="scroll-spacer"`) {
		t.Errorf("index.html should place #intro-bio above #scroll-spacer")
	}
	for _, wantText := range []string{
		"© 2026 Martin Wimpress",
		"BlueOak-1.0.0",
		"Built with Go, PixiJS, and Catppuccin",
		"Updated <time",
	} {
		if !strings.Contains(indexHTML, wantText) {
			t.Errorf("index.html footer missing expected text %q", wantText)
		}
	}
	for _, wantAttr := range []string{`role="dialog"`, `aria-modal="true"`, `aria-labelledby="modal-title"`} {
		if !strings.Contains(indexHTML, wantAttr) {
			t.Errorf("index.html missing expected dialog attribute %q", wantAttr)
		}
	}
	for _, removedID := range []string{"btn-view-floating", "btn-view-grid", "btn-drift-toggle", "header-stats", "deck-nav", "sort-controls"} {
		if strings.Contains(indexHTML, `id="`+removedID+`"`) {
			t.Errorf("index.html still contains removed chrome id %q", removedID)
		}
	}
}

func TestBuildAndWriteValidationErrors(t *testing.T) {
	t.Parallel()

	if _, err := Build(Input{}); err == nil {
		t.Fatal("expected error for nil config in Build")
	}
	if err := Write(t.TempDir(), nil); err == nil {
		t.Fatal("expected error for nil data in Write")
	}
	if err := Write("", &SiteData{}); err == nil {
		t.Fatal("expected error for empty outDir in Write")
	}
}

func TestBuildFallbacksAndAllPackageHosts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	cfg, err := config.Parse([]byte(`
profile:
  name: "Fallback User"
  handle: "fallback"
  links:
    - label: "Mastodon"
      url: "https://example.social/@fallback"
      icon: "mastodon"
feeds:
  youtube:
    enabled: true
    title: "YouTube"
    url: "https://www.youtube.com/feeds/videos.xml?channel_id=123"
    site_url: "https://youtube.com/@fallback"
`))
	if err != nil {
		t.Fatalf("config.Parse failed: %v", err)
	}

	hosts := make([]haberdasher.PackageHost, 0, len(haberdasher.CanonicalHosts))
	for _, h := range haberdasher.CanonicalHosts {
		hosts = append(hosts, haberdasher.PackageHost{
			Name:  subdomainName(h),
			Host:  h,
			URL:   "https://" + h,
			Repos: []string{"wimpysworld/deb-get"},
		})
	}

	data, err := Build(Input{
		Config:    cfg,
		Catalogue: &haberdasher.Catalogue{Hosts: hosts},
		YouTube: []rss.Item{
			{
				Title:       "Demo Video",
				URL:         "https://youtube.com/watch?v=demo",
				PublishedAt: now.Add(-24 * time.Hour),
				Summary:     "Walkthrough video.",
			},
		},
		GeneratedAt: now,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(data.Packages) != 8 {
		t.Fatalf("len(Packages) = %d, want 8", len(data.Packages))
	}
	if len(data.Links) != 1 || data.Links[0].Title != "Mastodon" {
		t.Fatalf("expected fallback to Profile.Links, got: %+v", data.Links)
	}
	if len(data.Media) != 1 || data.Media[0].Kind != "video" {
		t.Fatalf("expected YouTube video card, got: %+v", data.Media)
	}
}
