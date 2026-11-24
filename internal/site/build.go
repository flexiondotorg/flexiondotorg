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
	links := buildLinks(cfg, in.GitHub, in.Discord, in.Social, in.Podcast, in.Blog, now)
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

func buildLinks(
	cfg *config.Config,
	ghData *gh.ProfileData,
	disc discord.Status,
	social map[string]SocialProfile,
	podcastItems, blogItems []rss.Item,
	now time.Time,
) []LinkCard {
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
		slug := linkLookupKey(link, i)
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

		enrichLinkCard(&card, link, cfg, ghData, disc, social[slug], podcastItems, blogItems, now)
		cards = append(cards, card)
	}
	return cards
}

func enrichLinkCard(
	card *LinkCard,
	link config.SiteLink,
	cfg *config.Config,
	ghData *gh.ProfileData,
	disc discord.Status,
	socialProf SocialProfile,
	podcastItems, blogItems []rss.Item,
	now time.Time,
) {
	icon := strings.ToLower(strings.TrimSpace(card.Icon))
	if len(link.Stats) > 0 {
		for _, s := range link.Stats {
			if len(card.Stats) >= 2 {
				break
			}
			card.Stats = append(card.Stats, StatRow{
				Label: strings.ToUpper(strings.TrimSpace(s.Label)),
				Value: strings.TrimSpace(s.Value),
			})
		}
		return
	}

	switch {
	case icon == "discord" || card.ID == "discord":
		enrichDiscordLinkCard(card, disc)
	case icon == "github" || card.ID == "github":
		enrichGitHubLinkCard(card, cfg, ghData)
	case icon == "heart" || card.ID == "github-sponsors" || strings.Contains(strings.ToLower(card.URL), "github.com/sponsors/"):
		enrichSponsorsLinkCard(card, cfg, ghData)
	case icon == "blog" || card.ID == "wimpys-world-blog":
		enrichBlogLinkCard(card, blogItems, now)
	case icon == "podcast" || card.ID == "linux-matters":
		enrichPodcastLinkCard(card, podcastItems, now)
	case icon == "mastodon" || card.ID == "mastodon":
		enrichSocialLinkCard(card, socialProf, deriveFediverseHandle(card.URL))
	case icon == "bluesky" || card.ID == "bluesky":
		enrichSocialLinkCard(card, socialProf, deriveBlueskyHandle(card.URL))
	case icon == "twitch" || card.ID == "twitch":
		enrichTwitchLinkCard(card, socialProf)
	case icon == "youtube" || card.ID == "youtube":
		enrichYouTubeLinkCard(card, socialProf)
	case icon == "steam" || card.ID == "steam":
		enrichSteamLinkCard(card, socialProf)
	case icon == "itchio" || card.ID == "itchio":
		enrichItchLinkCard(card, socialProf)
	default:
		card.Stats = []StatRow{deriveIdentityStat(icon, card.URL, card.HostLabel)}
	}
}

func enrichDiscordLinkCard(card *LinkCard, disc discord.Status) {
	if disc.Online > 0 {
		card.Badge = render.FormatInt(disc.Online) + " online"
		card.Meta = append(card.Meta, render.FormatInt(disc.Online)+" online")
		card.Stats = append(card.Stats, StatRow{
			Label:     "ONLINE",
			Value:     render.FormatInt(disc.Online),
			Highlight: true,
		})
	}
	if disc.Members > 0 {
		card.Meta = append(card.Meta, render.FormatInt(disc.Members)+" members")
		if card.Badge == "" {
			card.Badge = render.FormatInt(disc.Members) + " members"
		}
		card.Stats = append(card.Stats, StatRow{
			Label: "MEMBERS",
			Value: render.FormatInt(disc.Members),
		})
	}
	if len(card.Stats) == 0 {
		card.Stats = []StatRow{deriveIdentityStat("discord", card.URL, card.HostLabel)}
	}
}

func enrichGitHubLinkCard(card *LinkCard, cfg *config.Config, ghData *gh.ProfileData) {
	handle := ""
	if cfg != nil {
		handle = strings.TrimSpace(cfg.Profile.Handle)
		if handle == "" {
			handle = strings.TrimSpace(cfg.GitHub.Username)
		}
	}

	if ghData != nil {
		if ghData.Stats.TotalStars > 0 {
			stars := render.FormatInt(ghData.Stats.TotalStars)
			card.Stats = append(card.Stats, StatRow{
				Label: "STARS",
				Value: stars,
			})
			card.Meta = append(card.Meta, stars+" stars")
		}
		if ghData.Stats.Followers > 0 {
			followers := render.FormatInt(ghData.Stats.Followers)
			card.Stats = append(card.Stats, StatRow{
				Label: "FOLLOWERS",
				Value: followers,
			})
			card.Meta = append(card.Meta, followers+" followers")
		}
		totalRepos := ghData.Stats.PublicRepos + ghData.Stats.OrgRepos
		if totalRepos > 0 {
			card.Meta = append(card.Meta, render.FormatInt(totalRepos)+" repos")
		}
		if ghData.Stats.TotalCommits > 0 {
			card.Meta = append(card.Meta, render.FormatInt(ghData.Stats.TotalCommits)+" commits")
		}
		card.Sponsors = buildSponsorRefs(ghData.Sponsors)
		card.PastSponsors = buildSponsorRefs(ghData.PastSponsors)
	}

	if len(card.Stats) == 0 {
		if handle != "" {
			card.Stats = []StatRow{{Label: "HANDLE", Value: "@" + handle}}
		} else {
			card.Stats = []StatRow{deriveIdentityStat("github", card.URL, card.HostLabel)}
		}
	}
}

func enrichSponsorsLinkCard(card *LinkCard, cfg *config.Config, ghData *gh.ProfileData) {
	handle := ""
	if cfg != nil {
		handle = strings.TrimSpace(cfg.Profile.Handle)
		if handle == "" {
			handle = strings.TrimSpace(cfg.GitHub.Username)
		}
	}

	if ghData != nil {
		active := ghData.SponsorCount
		if active == 0 {
			active = len(ghData.Sponsors)
		}
		past := len(ghData.PastSponsors)
		if active > 0 {
			card.Badge = fmt.Sprintf("%s active", render.FormatInt(active))
			card.Meta = append(card.Meta, fmt.Sprintf("%s active sponsors", render.FormatInt(active)))
			card.Stats = append(card.Stats, StatRow{
				Label:     "ACTIVE",
				Value:     render.FormatInt(active),
				Highlight: true,
			})
		}
		if past > 0 {
			card.Meta = append(card.Meta, fmt.Sprintf("%s past sponsors", render.FormatInt(past)))
			if card.Badge == "" {
				card.Badge = fmt.Sprintf("%s past sponsors", render.FormatInt(past))
			}
			card.Stats = append(card.Stats, StatRow{
				Label: "PAST SPONSORS",
				Value: render.FormatInt(past),
			})
		}
		card.Sponsors = buildSponsorRefs(ghData.Sponsors)
		card.PastSponsors = buildSponsorRefs(ghData.PastSponsors)
	}

	if len(card.Stats) < 2 && handle != "" {
		card.Stats = append(card.Stats, StatRow{
			Label: "GITHUB",
			Value: "@" + handle,
		})
	}
	if len(card.Stats) == 0 {
		card.Stats = []StatRow{{Label: "SITE", Value: card.HostLabel}}
	}
}

func enrichBlogLinkCard(card *LinkCard, blogItems []rss.Item, now time.Time) {
	if len(blogItems) > 0 {
		latest := toMediaEntry(blogItems[0], now)
		latestVal := fallbackText(latest.Age, latest.DateLabel)
		if latestVal != "" {
			card.Stats = append(card.Stats, StatRow{Label: "LATEST", Value: latestVal})
			card.Meta = append(card.Meta, "Latest post "+latestVal)
		}
		card.Meta = append(card.Meta, fmt.Sprintf("%d recent posts", len(blogItems)))
	}
	card.Stats = append(card.Stats, StatRow{Label: "SITE", Value: card.HostLabel})
}

func enrichPodcastLinkCard(card *LinkCard, podcastItems []rss.Item, now time.Time) {
	if len(podcastItems) > 0 {
		latest := toMediaEntry(podcastItems[0], now)
		if latest.Episode != "" {
			card.Stats = append(card.Stats, StatRow{Label: "LATEST", Value: "#" + latest.Episode})
			card.Meta = append(card.Meta, "Episode #"+latest.Episode)
		}
		if age := fallbackText(latest.Age, latest.DateLabel); age != "" {
			card.Stats = append(card.Stats, StatRow{Label: "PUBLISHED", Value: age})
			card.Meta = append(card.Meta, "Published "+age)
		}
	}
	if len(card.Stats) == 0 {
		card.Stats = []StatRow{{Label: "SITE", Value: card.HostLabel}}
	}
}

func enrichSocialLinkCard(card *LinkCard, prof SocialProfile, fallbackHandle string) {
	if prof.Followers > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "FOLLOWERS",
			Value: render.FormatInt(prof.Followers),
		})
		card.Meta = append(card.Meta, render.FormatInt(prof.Followers)+" followers")
	}
	if prof.Posts > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "POSTS",
			Value: render.FormatInt(prof.Posts),
		})
		card.Meta = append(card.Meta, render.FormatInt(prof.Posts)+" posts")
	}
	if len(card.Stats) == 0 {
		val := fallbackText(fallbackHandle, card.HostLabel)
		card.Stats = []StatRow{{Label: "HANDLE", Value: val}}
	}
}

func enrichTwitchLinkCard(card *LinkCard, prof SocialProfile) {
	if prof.Followers > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "FOLLOWERS",
			Value: render.FormatInt(prof.Followers),
		})
		card.Meta = append(card.Meta, render.FormatInt(prof.Followers)+" followers")
	}
	card.Stats = append(card.Stats, deriveIdentityStat("twitch", card.URL, card.HostLabel))
}

func enrichYouTubeLinkCard(card *LinkCard, prof SocialProfile) {
	if prof.Subscribers != "" {
		card.Stats = append(card.Stats, StatRow{
			Label: "SUBSCRIBERS",
			Value: prof.Subscribers,
		})
		card.Meta = append(card.Meta, prof.Subscribers+" subscribers")
	}
	if prof.Videos != "" {
		card.Stats = append(card.Stats, StatRow{
			Label: "VIDEOS",
			Value: prof.Videos,
		})
		card.Meta = append(card.Meta, prof.Videos+" videos")
	}
	if len(card.Stats) == 0 {
		card.Stats = []StatRow{deriveIdentityStat("youtube", card.URL, card.HostLabel)}
	}
}

func enrichSteamLinkCard(card *LinkCard, prof SocialProfile) {
	if prof.Games > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "GAMES",
			Value: render.FormatInt(prof.Games),
		})
		card.Meta = append(card.Meta, render.FormatInt(prof.Games)+" games")
	}
	if prof.Level > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "LEVEL",
			Value: render.FormatInt(prof.Level),
		})
		card.Meta = append(card.Meta, "Level "+render.FormatInt(prof.Level))
	}
	if len(card.Stats) == 0 {
		card.Stats = []StatRow{deriveIdentityStat("steam", card.URL, card.HostLabel)}
	}
}

func enrichItchLinkCard(card *LinkCard, prof SocialProfile) {
	if prof.Games > 0 {
		card.Stats = append(card.Stats, StatRow{
			Label: "GAMES",
			Value: render.FormatInt(prof.Games),
		})
		unit := "published games"
		if prof.Games == 1 {
			unit = "published game"
		}
		card.Meta = append(card.Meta, render.FormatInt(prof.Games)+" "+unit)
	}
	card.Stats = append(card.Stats, deriveIdentityStat("itchio", card.URL, card.HostLabel))
}

func deriveFediverseHandle(rawURL string) string {
	origin, acct, ok := parseFediverseURL(rawURL)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return "@" + acct
	}
	return "@" + acct + "@" + parsed.Host
}

func deriveBlueskyHandle(rawURL string) string {
	actor, ok := parseBlueskyActor(rawURL)
	if !ok {
		return ""
	}
	return "@" + actor
}

func deriveIdentityStat(icon, rawURL, hostLabel string) StatRow {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err == nil {
		switch icon {
		case "matrix":
			if frag := strings.TrimPrefix(strings.TrimSpace(parsed.Fragment), "/"); frag != "" {
				return StatRow{Label: "HANDLE", Value: frag}
			}
		case "youtube":
			if seg := lastPathSegment(parsed.Path); seg != "" {
				if !strings.HasPrefix(seg, "@") {
					seg = "@" + seg
				}
				return StatRow{Label: "CHANNEL", Value: seg}
			}
		case "twitch":
			if seg := lastPathSegment(parsed.Path); seg != "" {
				return StatRow{Label: "CHANNEL", Value: seg}
			}
		case "linkedin", "steam":
			if seg := lastPathSegment(parsed.Path); seg != "" {
				return StatRow{Label: "PROFILE", Value: seg}
			}
		case "github":
			if seg := lastPathSegment(parsed.Path); seg != "" {
				return StatRow{Label: "HANDLE", Value: "@" + strings.TrimPrefix(seg, "@")}
			}
		case "itchio":
			if host := strings.TrimPrefix(parsed.Host, "www."); host != "" {
				return StatRow{Label: "CREATOR", Value: host}
			}
		}
	}
	return StatRow{Label: "SITE", Value: fallbackText(hostLabel, strings.TrimSpace(rawURL))}
}

func lastPathSegment(rawPath string) string {
	trimmed := strings.Trim(strings.TrimSpace(rawPath), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return strings.TrimSpace(parts[len(parts)-1])
}

func buildSponsorRefs(list []gh.Sponsor) []SponsorRef {
	if len(list) == 0 {
		return nil
	}
	refs := make([]SponsorRef, 0, len(list))
	for _, sp := range list {
		login := strings.TrimSpace(sp.Login)
		if login == "" {
			continue
		}
		profileURL := strings.TrimSpace(sp.URL)
		if profileURL == "" {
			profileURL = "https://github.com/" + login
		}
		refs = append(refs, SponsorRef{
			Login: login,
			URL:   profileURL,
		})
	}
	if len(refs) == 0 {
		return nil
	}
	return refs
}

func buildMedia(
	cfg *config.Config,
	podcastItems, blogItems, youtubeItems []rss.Item,
	now time.Time,
) []MediaCard {
	cards := make([]MediaCard, 0, len(podcastItems)+len(blogItems)+len(youtubeItems))

	if pod := cfg.Feeds.Podcast; pod != nil && pod.Enabled {
		subLinks := buildSubscribeLinks(pod.Links)
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
				AudioURL:       entry.AudioURL,
				Icon:           "podcast",
				IconSVG:        BrandIconSVG("podcast"),
				Accent:         "yellow",
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
				Summary:         truncateExcerpt(entry.Summary, 360),
				SpeechText:      cleanSpeechText(entry.Title, item.Summary),
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
		AudioURL:        strings.TrimSpace(item.AudioURL),
		Subtitle:        cleanSummaryText(item.Subtitle),
		Summary:         cleanSummaryText(item.Summary),
		ReadTimeMinutes: item.ReadTimeMinutes,
	}
}

var (
	reCodeBlock   = regexp.MustCompile(`(?is)<(?:pre|script|style)\b[^>]*>[\s\S]*?</(?:pre|script|style)>`)
	reBlockBreak  = regexp.MustCompile(`(?i)</(?:p|div|h[1-6]|li|blockquote|section|article|br)>`)
	reInlineSpace = regexp.MustCompile(`[ \t]+`)
)

func cleanSpeechText(title, rawHTML string) string {
	stripped := reCodeBlock.ReplaceAllString(rawHTML, " ")
	stripped = reBlockBreak.ReplaceAllString(stripped, ". ")
	body := cleanSummaryText(stripped)
	body = reInlineSpace.ReplaceAllString(body, " ")
	body = strings.ReplaceAll(body, "..", ".")
	body = strings.ReplaceAll(body, " .", ".")
	body = strings.TrimSpace(body)

	t := strings.TrimSpace(title)
	switch {
	case t != "" && body != "":
		if !strings.HasSuffix(t, ".") && !strings.HasSuffix(t, "!") && !strings.HasSuffix(t, "?") {
			t += "."
		}
		return t + " " + body
	case body != "":
		return body
	default:
		return t
	}
}

func truncateExcerpt(raw string, maxRunes int) string {
	trimmed := strings.TrimSpace(raw)
	if maxRunes <= 0 {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	cut := string(runes[:maxRunes])
	if idx := strings.LastIndexByte(cut, ' '); idx > maxRunes/2 {
		cut = cut[:idx]
	}
	return strings.TrimRight(cut, " ,;:-") + "…"
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
