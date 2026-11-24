package site

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	"github.com/flexiondotorg/flexiondotorg/internal/discord"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/render"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
	"github.com/flexiondotorg/flexiondotorg/internal/svg"
)

const freshReleaseWindow = 14 * 24 * time.Hour

var reMarkdownLink = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+)\)`)

// Build transforms configuration and fetched live data into a SiteData payload
// ready for JSON serialisation and static site emission.
func Build(in Input) (*SiteData, error) {
	if in.Config == nil {
		return nil, errors.New("site: config must not be nil")
	}

	now := in.GeneratedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	cfg := in.Config
	defaultDeck := strings.ToLower(strings.TrimSpace(cfg.Site.DefaultDeck))
	switch defaultDeck {
	case config.SiteDeckProjects, config.SiteDeckLinks, config.SiteDeckMedia:
	default:
		defaultDeck = config.SiteDeckProjects
	}
	defaultSort := strings.ToLower(strings.TrimSpace(cfg.Haberdasher.Sort))
	if defaultSort == "" {
		defaultSort = config.HaberdasherSortStars
	}

	hostLogos := make(map[string]string)
	packages := buildPackages(in.Catalogue, hostLogos)
	projects := buildProjects(in.Catalogue, now)
	links := buildLinks(cfg, in.GitHub, in.Discord)
	media := buildMedia(cfg, in.Podcast, in.Blog, in.YouTube, now)

	profile := buildProfile(cfg)
	stats := buildStats(in.GitHub, in.Discord, len(projects), len(packages))

	return &SiteData{
		Profile:      profile,
		Stats:        stats,
		DefaultDeck:  defaultDeck,
		DefaultSort:  defaultSort,
		GeneratedAt:  now.Format(time.RFC3339),
		UpdatedLabel: now.Format("2 Jan 2006"),
		HostLogos:    hostLogos,
		Projects:     projects,
		Packages:     packages,
		Links:        links,
		Media:        media,
	}, nil
}

func buildProfile(cfg *config.Config) ProfileData {
	title := strings.TrimSpace(cfg.Site.Title)
	if title == "" {
		title = strings.TrimSpace(cfg.Profile.HeroTitle)
	}
	if title == "" {
		title = strings.TrimSpace(cfg.Profile.Name)
	}
	if title == "" {
		title = "Wimpy's World"
	}

	subtitle := strings.TrimSpace(cfg.Site.Description)
	if subtitle == "" {
		subtitle = strings.TrimSpace(cfg.Profile.HeroSubtitle)
	}

	domain := strings.TrimSpace(cfg.Site.Domain)
	siteURL := strings.TrimSpace(cfg.Site.URL)
	if siteURL == "" && domain != "" {
		siteURL = "https://" + domain
	}
	if siteURL == "" {
		siteURL = strings.TrimSpace(cfg.Profile.HeroURL)
	}

	handle := strings.TrimSpace(cfg.Profile.Handle)
	if handle == "" {
		handle = strings.TrimSpace(cfg.GitHub.Username)
	}
	ghURL := ""
	if handle != "" {
		ghURL = "https://github.com/" + handle
	}

	bioParas, bioPlain := parseBioParagraphs(cfg.Profile.Bio)

	return ProfileData{
		Name:         strings.TrimSpace(cfg.Profile.Name),
		Handle:       handle,
		Title:        title,
		Subtitle:     subtitle,
		Domain:       domain,
		URL:          siteURL,
		GitHubURL:    ghURL,
		SponsorsURL:  strings.TrimSpace(cfg.GitHub.SponsorsURL),
		Bio:          bioParas,
		BioPlainText: bioPlain,
	}
}

func buildStats(ghData *gh.ProfileData, disc discord.Status, projectCount, hostCount int) StatsData {
	stats := StatsData{
		ProjectCount:   projectCount,
		HostCount:      hostCount,
		DiscordOnline:  disc.Online,
		DiscordMembers: disc.Members,
	}
	if ghData != nil {
		stats.TotalStars = ghData.Stats.TotalStars
		stats.StarsFormatted = render.FormatInt(ghData.Stats.TotalStars)
		stats.Followers = ghData.Stats.Followers
		stats.PublicRepos = ghData.Stats.PublicRepos + ghData.Stats.OrgRepos
		stats.TotalCommits = ghData.Stats.TotalCommits
		stats.ActiveSponsors = ghData.SponsorCount
		if stats.ActiveSponsors == 0 {
			stats.ActiveSponsors = len(ghData.Sponsors)
		}
		stats.PastSponsors = len(ghData.PastSponsors)
	}
	return stats
}

func buildProjects(cat *haberdasher.Catalogue, now time.Time) []ProjectCard {
	if cat == nil || len(cat.Projects) == 0 {
		return []ProjectCard{}
	}

	cards := make([]ProjectCard, 0, len(cat.Projects))
	seen := make(map[string]int, len(cat.Projects))

	for i, proj := range cat.Projects {
		slug := slugify(proj.Slug)
		if slug == "" {
			slug = slugify(proj.Name)
		}
		if slug == "" {
			slug = fmt.Sprintf("project-%d", i+1)
		}
		seen[slug]++
		id := slug
		if seen[slug] > 1 {
			id = fmt.Sprintf("%s-%d", slug, seen[slug])
		}

		logoSVG, logoDataURI := SanitizeLogoArtwork(slug, proj.Name, proj.LogoSVG, proj.LogoDataURI)
		fresh := !proj.PublishedAt.IsZero() && now.Sub(proj.PublishedAt) <= freshReleaseWindow && now.Sub(proj.PublishedAt) >= 0

		hosts := make([]ProjectHostRef, 0, len(proj.Hosts))
		for _, host := range proj.Hosts {
			trimmedHost := strings.TrimSpace(host)
			if trimmedHost == "" {
				continue
			}
			hosts = append(hosts, ProjectHostRef{
				Name: subdomainName(trimmedHost),
				Host: trimmedHost,
				URL:  "https://" + trimmedHost + "/#" + slug,
			})
		}

		publishedISO := ""
		var publishedUnix int64
		if !proj.PublishedAt.IsZero() {
			publishedISO = proj.PublishedAt.UTC().Format(time.RFC3339)
			publishedUnix = proj.PublishedAt.UTC().Unix()
		}

		homepage := strings.TrimSpace(proj.Homepage)
		repoURL := strings.TrimSpace(proj.RepoURL)
		if repoURL == "" && proj.FullName != "" {
			repoURL = "https://github.com/" + proj.FullName
		}

		cards = append(cards, ProjectCard{
			ID:             id,
			FullName:       proj.FullName,
			Owner:          proj.Owner,
			Name:           proj.Name,
			Slug:           slug,
			Description:    strings.TrimSpace(proj.Description),
			Language:       strings.TrimSpace(proj.Language),
			LanguageColor:  languageColor(proj.Language),
			Stars:          proj.Stars,
			StarsFormatted: render.FormatInt(proj.Stars),
			StarsCompact:   svg.FormatCompactCount(proj.Stars),
			LatestTag:      strings.TrimSpace(proj.LatestTag),
			ReleaseURL:     strings.TrimSpace(proj.ReleaseURL),
			PublishedAt:    publishedISO,
			PublishedUnix:  publishedUnix,
			ReleaseAge:     render.FormatCompactAge(proj.PublishedAt, now),
			Fresh:          fresh,
			Homepage:       homepage,
			RepoURL:        repoURL,
			LogoSVG:        logoSVG,
			LogoDataURI:    logoDataURI,
			Hosts:          hosts,
		})
	}
	return cards
}

func buildPackages(cat *haberdasher.Catalogue, hostLogos map[string]string) []PackageHostCard {
	if cat == nil || len(cat.Hosts) == 0 {
		return []PackageHostCard{}
	}

	cards := make([]PackageHostCard, 0, len(cat.Hosts))
	for i, host := range cat.Hosts {
		name := strings.TrimSpace(host.Name)
		if name == "" {
			name = subdomainName(host.Host)
		}
		id := slugify(name)
		if id == "" {
			id = fmt.Sprintf("host-%d", i+1)
		}

		logoSVG, _ := SanitizeLogoArtwork(id, name, host.LogoSVG, "")
		if strings.TrimSpace(host.Host) != "" && logoSVG != "" {
			hostLogos[host.Host] = logoSVG
		}

		entries := make([]PackageEntry, 0, len(host.Repos))
		for _, fullName := range host.Repos {
			trimmed := strings.TrimSpace(fullName)
			if trimmed == "" {
				continue
			}
			pkgName := trimmed
			if _, after, ok := strings.Cut(trimmed, "/"); ok && strings.TrimSpace(after) != "" {
				pkgName = strings.TrimSpace(after)
			}
			pkgSlug := slugify(pkgName)
			if pkgSlug == "" {
				pkgSlug = strings.ToLower(pkgName)
			}
			entries = append(entries, PackageEntry{
				Name:     pkgName,
				Slug:     pkgSlug,
				FullName: trimmed,
				URL:      "https://" + host.Host + "/#" + pkgSlug,
			})
		}
		slices.SortFunc(entries, func(a, b PackageEntry) int {
			return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})

		pkgCount := host.PackageCount
		if pkgCount == 0 {
			pkgCount = len(entries)
		}
		displayName, desc, accent := describePackageHost(name)

		cards = append(cards, PackageHostCard{
			ID:           id,
			Name:         name,
			DisplayName:  displayName,
			Host:         host.Host,
			URL:          host.URL,
			Description:  desc,
			Accent:       accent,
			PackageCount: pkgCount,
			LogoSVG:      logoSVG,
			Packages:     entries,
		})
	}
	return cards
}

func describePackageHost(name string) (displayName, description, accent string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "apt":
		return "APT", "Debian and Ubuntu package repository.", "red"
	case "rpm":
		return "RPM", "Fedora, RHEL, and openSUSE RPM repository.", "blue"
	case "pkg":
		return "Arch PKG", "Arch Linux pacman package repository.", "sapphire"
	case "apk":
		return "Alpine APK", "Alpine Linux APK package repository.", "teal"
	case "flatpak":
		return "Flatpak", "Linux desktop Flatpak application repository.", "mauve"
	case "brew":
		return "Homebrew", "Homebrew tap for macOS and Linux.", "peach"
	case "scoop":
		return "Scoop", "Windows Scoop command-line bucket.", "green"
	case "nix":
		return "Nix", "Nix flake packages and repository catalogue.", "lavender"
	default:
		return strings.ToUpper(name), "First-party Haberdasher package repository.", "mauve"
	}
}

func buildLinks(cfg *config.Config, ghData *gh.ProfileData, disc discord.Status) []LinkCard {
	configured := cfg.Site.Links
	if len(configured) == 0 && len(cfg.Profile.Links) > 0 {
		configured = make([]config.SiteLink, 0, len(cfg.Profile.Links))
		for _, nav := range cfg.Profile.Links {
			configured = append(configured, config.SiteLink{
				Title:    nav.Label,
				Slug:     nav.Slug,
				Category: "Social",
				URL:      nav.URL,
				Icon:     nav.Icon,
				Accent:   nav.Accent,
			})
		}
	}

	cards := make([]LinkCard, 0, len(configured))
	seen := make(map[string]int, len(configured))

	for i, link := range configured {
		slug := slugify(link.Slug)
		if slug == "" {
			slug = slugify(link.Title)
		}
		if slug == "" {
			slug = fmt.Sprintf("link-%d", i+1)
		}
		seen[slug]++
		id := slug
		if seen[slug] > 1 {
			id = fmt.Sprintf("%s-%d", slug, seen[slug])
		}

		category := strings.TrimSpace(link.Category)
		if category == "" {
			category = "Links"
		}
		accent := normalizeAccent(link.Accent, link.Icon)

		card := LinkCard{
			ID:          id,
			Title:       link.Title,
			Category:    category,
			Description: link.Description,
			URL:         link.URL,
			HostLabel:   formatHostLabel(link.URL),
			Icon:        link.Icon,
			IconSVG:     BrandIconSVG(link.Icon),
			Accent:      accent,
		}

		enrichLinkCard(&card, ghData, disc)
		cards = append(cards, card)
	}
	return cards
}

func enrichLinkCard(card *LinkCard, ghData *gh.ProfileData, disc discord.Status) {
	icon := strings.ToLower(strings.TrimSpace(card.Icon))
	if icon == "discord" || card.ID == "discord" {
		if disc.Online > 0 {
			card.Badge = render.FormatInt(disc.Online) + " online"
			card.Meta = append(card.Meta, render.FormatInt(disc.Online)+" online")
		}
		if disc.Members > 0 {
			card.Meta = append(card.Meta, render.FormatInt(disc.Members)+" members")
			if card.Badge == "" {
				card.Badge = render.FormatInt(disc.Members) + " members"
			}
		}
		return
	}

	isSponsors := icon == "heart" || card.ID == "github-sponsors" || strings.Contains(strings.ToLower(card.URL), "github.com/sponsors/")
	if isSponsors && ghData != nil {
		active := ghData.SponsorCount
		if active == 0 {
			active = len(ghData.Sponsors)
		}
		past := len(ghData.PastSponsors)
		if active > 0 {
			card.Badge = fmt.Sprintf("%s active", render.FormatInt(active))
			card.Meta = append(card.Meta, fmt.Sprintf("%s active sponsors", render.FormatInt(active)))
		}
		if past > 0 {
			card.Meta = append(card.Meta, fmt.Sprintf("%s past sponsors", render.FormatInt(past)))
			if card.Badge == "" {
				card.Badge = fmt.Sprintf("%s past sponsors", render.FormatInt(past))
			}
		}
	}
}

func buildMedia(
	cfg *config.Config,
	podcastItems, blogItems, youtubeItems []rss.Item,
	now time.Time,
) []MediaCard {
	cards := make([]MediaCard, 0, 2+len(podcastItems)+len(blogItems)+len(youtubeItems))

	if pod := cfg.Feeds.Podcast; pod != nil && pod.Enabled {
		subLinks := buildSubscribeLinks(pod.Links)
		entries := make([]MediaEntry, 0, len(podcastItems))
		for _, item := range podcastItems {
			entries = append(entries, toMediaEntry(item, now))
		}

		title := strings.TrimSpace(pod.Title)
		if title == "" {
			title = "Linux Matters Podcast"
		}
		summary := "Recent episodes on Linux, open-source software, and the community."
		if len(entries) > 0 && entries[0].Subtitle != "" {
			summary = entries[0].Subtitle
		}
		badge := ""
		if len(entries) > 0 {
			badge = fmt.Sprintf("%d episodes", len(entries))
		}

		cards = append(cards, MediaCard{
			ID:             "podcast-hub",
			Kind:           "podcast-hub",
			Category:       "Podcast",
			Title:          title,
			Summary:        summary,
			URL:            fallbackURL(pod.SiteURL, pod.URL),
			SiteURL:        pod.SiteURL,
			CoverURL:       strings.TrimSpace(pod.CoverURL),
			Icon:           "podcast",
			IconSVG:        BrandIconSVG("podcast"),
			Accent:         "blue",
			Badge:          badge,
			SubscribeLinks: subLinks,
			Entries:        entries,
		})

		for i, item := range podcastItems {
			entry := toMediaEntry(item, now)
			epBadge := entry.Duration
			if entry.Episode != "" {
				epBadge = "#" + entry.Episode
			}
			cards = append(cards, MediaCard{
				ID:             fmt.Sprintf("podcast-ep-%d", i+1),
				Kind:           "episode",
				Category:       "Podcast",
				Title:          entry.Title,
				Subtitle:       entry.Subtitle,
				Summary:        fallbackText(entry.Subtitle, entry.Summary),
				URL:            fallbackURL(entry.URL, pod.SiteURL),
				SiteURL:        pod.SiteURL,
				CoverURL:       strings.TrimSpace(pod.CoverURL),
				Icon:           "podcast",
				IconSVG:        BrandIconSVG("podcast"),
				Accent:         "teal",
				Badge:          epBadge,
				Episode:        entry.Episode,
				Duration:       entry.Duration,
				PublishedAt:    entry.PublishedAt,
				DateLabel:      entry.DateLabel,
				Age:            entry.Age,
				SubscribeLinks: subLinks,
			})
		}
	}

	if blog := cfg.Feeds.Blog; blog != nil && blog.Enabled {
		entries := make([]MediaEntry, 0, len(blogItems))
		for _, item := range blogItems {
			entries = append(entries, toMediaEntry(item, now))
		}

		title := strings.TrimSpace(blog.Title)
		if title == "" || strings.EqualFold(title, "Blog") {
			title = "Wimpy's World Blog"
		}
		summary := "Recent posts on Linux, NixOS, open-source software, and tools."
		if len(entries) > 0 && entries[0].Summary != "" {
			summary = entries[0].Summary
		}
		badge := ""
		if len(entries) > 0 {
			badge = fmt.Sprintf("%d posts", len(entries))
		}

		blogLinks := buildSubscribeLinks(blog.Links)
		if len(blogLinks) == 0 && strings.TrimSpace(blog.URL) != "" {
			blogLinks = []SubscribeLink{
				{
					Label:   "RSS",
					URL:     blog.URL,
					Icon:    "rss",
					IconSVG: BrandIconSVG("rss"),
					Accent:  "peach",
				},
			}
		}

		cards = append(cards, MediaCard{
			ID:             "blog-hub",
			Kind:           "blog-hub",
			Category:       "Blog",
			Title:          title,
			Summary:        summary,
			URL:            fallbackURL(blog.SiteURL, blog.URL),
			SiteURL:        blog.SiteURL,
			Icon:           "blog",
			IconSVG:        BrandIconSVG("blog"),
			Accent:         "mauve",
			Badge:          badge,
			SubscribeLinks: blogLinks,
			Entries:        entries,
		})

		for i, item := range blogItems {
			entry := toMediaEntry(item, now)
			postBadge := entry.DateLabel
			if entry.ReadTimeMinutes > 0 {
				postBadge = fmt.Sprintf("%d min read", entry.ReadTimeMinutes)
			}
			cards = append(cards, MediaCard{
				ID:              fmt.Sprintf("blog-post-%d", i+1),
				Kind:            "post",
				Category:        "Blog",
				Title:           entry.Title,
				Summary:         entry.Summary,
				URL:             fallbackURL(entry.URL, blog.SiteURL),
				SiteURL:         blog.SiteURL,
				Icon:            "blog",
				IconSVG:         BrandIconSVG("blog"),
				Accent:          "peach",
				Badge:           postBadge,
				ReadTimeMinutes: entry.ReadTimeMinutes,
				PublishedAt:     entry.PublishedAt,
				DateLabel:       entry.DateLabel,
				Age:             entry.Age,
			})
		}
	}

	if yt := cfg.Feeds.YouTube; yt != nil && yt.Enabled {
		for i, item := range youtubeItems {
			entry := toMediaEntry(item, now)
			cards = append(cards, MediaCard{
				ID:          fmt.Sprintf("youtube-%d", i+1),
				Kind:        "video",
				Category:    "Video",
				Title:       entry.Title,
				Summary:     entry.Summary,
				URL:         fallbackURL(entry.URL, yt.SiteURL),
				SiteURL:     yt.SiteURL,
				Icon:        "youtube",
				IconSVG:     BrandIconSVG("youtube"),
				Accent:      "red",
				Badge:       entry.Age,
				PublishedAt: entry.PublishedAt,
				DateLabel:   entry.DateLabel,
				Age:         entry.Age,
			})
		}
	}

	return cards
}

func buildSubscribeLinks(links []config.NavLink) []SubscribeLink {
	out := make([]SubscribeLink, 0, len(links))
	for _, l := range links {
		out = append(out, SubscribeLink{
			Label:   l.Label,
			URL:     l.URL,
			Icon:    l.Icon,
			IconSVG: BrandIconSVG(l.Icon),
			Accent:  normalizeAccent(l.Accent, l.Icon),
		})
	}
	return out
}

func toMediaEntry(item rss.Item, now time.Time) MediaEntry {
	publishedISO := ""
	dateLabel := ""
	if !item.PublishedAt.IsZero() {
		utc := item.PublishedAt.UTC()
		publishedISO = utc.Format(time.RFC3339)
		dateLabel = utc.Format("2 Jan 2006")
	}
	return MediaEntry{
		Title:           strings.TrimSpace(item.Title),
		URL:             strings.TrimSpace(item.URL),
		PublishedAt:     publishedISO,
		DateLabel:       dateLabel,
		Age:             render.FormatCompactAge(item.PublishedAt, now),
		Episode:         strings.TrimSpace(item.Episode),
		Duration:        strings.TrimSpace(item.Duration),
		Subtitle:        cleanSummaryText(item.Subtitle),
		Summary:         cleanSummaryText(item.Summary),
		ReadTimeMinutes: item.ReadTimeMinutes,
	}
}

func cleanSummaryText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	// Strip simple HTML tags that RSS descriptions sometimes carry.
	var b strings.Builder
	b.Grow(len(trimmed))
	inTag := false
	for i := range len(trimmed) {
		ch := trimmed[i]
		if ch == '<' {
			inTag = true
			continue
		}
		if ch == '>' {
			inTag = false
			b.WriteByte(' ')
			continue
		}
		if !inTag {
			b.WriteByte(ch)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func parseBioParagraphs(raw string) ([]BioParagraph, string) {
	trimmed := strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	if trimmed == "" {
		return nil, ""
	}

	sep := "\n\n"
	if !strings.Contains(trimmed, sep) {
		sep = "\n"
	}
	chunks := strings.Split(trimmed, sep)
	paragraphs := make([]BioParagraph, 0, len(chunks))
	plainParts := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		line := strings.Join(strings.Fields(chunk), " ")
		if line == "" {
			continue
		}
		matches := reMarkdownLink.FindAllStringSubmatchIndex(line, -1)
		var segments []BioSegment
		var plainBuf strings.Builder
		cursor := 0
		for _, m := range matches {
			if m[0] > cursor {
				text := line[cursor:m[0]]
				segments = append(segments, BioSegment{Text: text})
				plainBuf.WriteString(text)
			}
			label := line[m[2]:m[3]]
			href := line[m[4]:m[5]]
			segments = append(segments, BioSegment{Text: label, URL: href})
			plainBuf.WriteString(label)
			cursor = m[1]
		}
		if cursor < len(line) {
			tail := line[cursor:]
			segments = append(segments, BioSegment{Text: tail})
			plainBuf.WriteString(tail)
		}
		paragraphs = append(paragraphs, BioParagraph{Segments: segments})
		plainParts = append(plainParts, plainBuf.String())
	}

	return paragraphs, strings.Join(plainParts, " ")
}

func normalizeAccent(accent, icon string) string {
	switch strings.ToLower(strings.TrimSpace(accent)) {
	case "mauve", "lavender", "blue", "sapphire", "teal", "green", "yellow", "peach", "red", "pink":
		return strings.ToLower(strings.TrimSpace(accent))
	}
	switch strings.ToLower(strings.TrimSpace(icon)) {
	case "bluesky":
		return "blue"
	case "linkedin", "steam":
		return "sapphire"
	case "spotify":
		return "green"
	case "youtube":
		return "red"
	case "rss", "itchio":
		return "peach"
	case "matrix":
		return "teal"
	case "discord":
		return "lavender"
	case "heart":
		return "pink"
	default:
		return "mauve"
	}
}

func formatHostLabel(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(rawURL)
	}
	host := strings.TrimPrefix(parsed.Host, "www.")
	path := strings.TrimRight(parsed.Path, "/")
	if path != "" && path != "/" && len(host)+len(path) <= 34 {
		return host + path
	}
	return host
}

func subdomainName(host string) string {
	prefix, _, ok := strings.Cut(strings.TrimSpace(host), ".")
	if ok && prefix != "" {
		return strings.ToLower(prefix)
	}
	return strings.ToLower(strings.TrimSpace(host))
}

func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func fallbackURL(primary, secondary string) string {
	if s := strings.TrimSpace(primary); s != "" {
		return s
	}
	return strings.TrimSpace(secondary)
}

func fallbackText(primary, secondary string) string {
	if s := strings.TrimSpace(primary); s != "" {
		return s
	}
	return strings.TrimSpace(secondary)
}
