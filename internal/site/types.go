// Package site builds the data payload and static web bundle for the
// https://wimpys.world portfolio site.
package site

import (
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
	"github.com/flexiondotorg/flexiondotorg/internal/discord"
	gh "github.com/flexiondotorg/flexiondotorg/internal/github"
	"github.com/flexiondotorg/flexiondotorg/internal/haberdasher"
	"github.com/flexiondotorg/flexiondotorg/internal/rss"
)

// SocialProfile carries live public counts fetched from a social, video, or gaming profile endpoint.
type SocialProfile struct {
	Followers   int
	Posts       int
	Subscribers string
	Videos      string
	Games       int
	Level       int
}

// Input aggregates all live and configured sources consumed by Build.
type Input struct {
	Config      *config.Config
	Catalogue   *haberdasher.Catalogue
	GitHub      *gh.ProfileData
	Discord     discord.Status
	Social      map[string]SocialProfile
	Podcast     []rss.Item
	Blog        []rss.Item
	YouTube     []rss.Item
	GeneratedAt time.Time
}

// SiteData is the complete JSON payload consumed by the static portfolio app.
//
//nolint:revive // Exported as SiteData to describe the top-level portfolio payload clearly.
type SiteData struct {
	Profile      ProfileData       `json:"profile"`
	Stats        StatsData         `json:"stats"`
	DefaultDeck  string            `json:"default_deck"`
	DefaultSort  string            `json:"default_sort"`
	GeneratedAt  string            `json:"generated_at"`
	UpdatedLabel string            `json:"updated_label"`
	HostLogos    map[string]string `json:"host_logos"`
	Projects     []ProjectCard     `json:"projects"`
	Packages     []PackageHostCard `json:"packages"`
	Links        []LinkCard        `json:"links"`
	Media        []MediaCard       `json:"media"`
}

// BioSegment represents either plain text or a link within a bio paragraph.
type BioSegment struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// BioParagraph holds an ordered sequence of plain text and link segments.
type BioParagraph struct {
	Segments []BioSegment `json:"segments"`
}

// ProfileData carries site identity, headings, links, and structured bio copy.
type ProfileData struct {
	Name         string         `json:"name"`
	Handle       string         `json:"handle"`
	Title        string         `json:"title"`
	Subtitle     string         `json:"subtitle"`
	Domain       string         `json:"domain"`
	URL          string         `json:"url"`
	GitHubURL    string         `json:"github_url"`
	SponsorsURL  string         `json:"sponsors_url"`
	Bio          []BioParagraph `json:"bio"`
	BioPlainText string         `json:"bio_plain_text"`
}

// StatsData summarises aggregate metrics displayed in the site header.
type StatsData struct {
	TotalStars     int    `json:"total_stars"`
	StarsFormatted string `json:"stars_formatted"`
	Followers      int    `json:"followers"`
	PublicRepos    int    `json:"public_repos"`
	TotalCommits   int    `json:"total_commits"`
	ProjectCount   int    `json:"project_count"`
	HostCount      int    `json:"host_count"`
	ActiveSponsors int    `json:"active_sponsors"`
	PastSponsors   int    `json:"past_sponsors"`
	DiscordOnline  int    `json:"discord_online"`
	DiscordMembers int    `json:"discord_members"`
}

// ProjectHostRef links a project to one Haberdasher package repository host.
type ProjectHostRef struct {
	Name string `json:"name"`
	Host string `json:"host"`
	URL  string `json:"url"`
}

// ProjectCard represents one card in the Projects deck.
type ProjectCard struct {
	ID             string           `json:"id"`
	FullName       string           `json:"full_name"`
	Owner          string           `json:"owner"`
	Name           string           `json:"name"`
	Slug           string           `json:"slug"`
	Description    string           `json:"description"`
	Language       string           `json:"language"`
	LanguageColor  string           `json:"language_color"`
	Stars          int              `json:"stars"`
	StarsFormatted string           `json:"stars_formatted"`
	StarsCompact   string           `json:"stars_compact"`
	LatestTag      string           `json:"latest_tag"`
	ReleaseURL     string           `json:"release_url"`
	PublishedAt    string           `json:"published_at,omitempty"`
	PublishedUnix  int64            `json:"published_unix"`
	ReleaseAge     string           `json:"release_age"`
	Fresh          bool             `json:"fresh"`
	Homepage       string           `json:"homepage"`
	RepoURL        string           `json:"repo_url"`
	LogoSVG        string           `json:"logo_svg,omitempty"`
	LogoDataURI    string           `json:"logo_data_uri,omitempty"`
	Hosts          []ProjectHostRef `json:"hosts"`
}

// PackageEntry represents a single published package listed on a Haberdasher host.
type PackageEntry struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	FullName string `json:"full_name"`
	URL      string `json:"url"`
}

// PackageHostCard represents one Haberdasher package repository host record
// used to supply package host metadata and inline SVG logos for project cards.
type PackageHostCard struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Host         string         `json:"host"`
	URL          string         `json:"url"`
	Description  string         `json:"description"`
	Accent       string         `json:"accent"`
	PackageCount int            `json:"package_count"`
	LogoSVG      string         `json:"logo_svg"`
	Packages     []PackageEntry `json:"packages"`
}

// SponsorRef represents a public GitHub sponsor profile link on the Sponsors card.
type SponsorRef struct {
	Login string `json:"login"`
	URL   string `json:"url"`
}

// StatRow represents one key/value stat row displayed at the bottom of a card.
type StatRow struct {
	Label     string `json:"label"`
	Value     string `json:"value"`
	Highlight bool   `json:"highlight,omitempty"`
}

// LinkCard represents one curated site, community, social, or support link in the Links deck.
type LinkCard struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Category     string       `json:"category"`
	Description  string       `json:"description"`
	URL          string       `json:"url"`
	HostLabel    string       `json:"host_label"`
	Icon         string       `json:"icon"`
	IconSVG      string       `json:"icon_svg"`
	Accent       string       `json:"accent"`
	Badge        string       `json:"badge,omitempty"`
	Stats        []StatRow    `json:"stats,omitempty"`
	Meta         []string     `json:"meta,omitempty"`
	Sponsors     []SponsorRef `json:"sponsors,omitempty"`
	PastSponsors []SponsorRef `json:"past_sponsors,omitempty"`
}

// SubscribeLink describes a podcast or blog subscription pill with an inline vector icon.
type SubscribeLink struct {
	Label   string `json:"label"`
	URL     string `json:"url"`
	Icon    string `json:"icon"`
	IconSVG string `json:"icon_svg"`
	Accent  string `json:"accent"`
}

// MediaEntry represents a single episode or article within a media hub card.
type MediaEntry struct {
	Title           string `json:"title"`
	URL             string `json:"url"`
	PublishedAt     string `json:"published_at,omitempty"`
	DateLabel       string `json:"date_label,omitempty"`
	Age             string `json:"age,omitempty"`
	Episode         string `json:"episode,omitempty"`
	Duration        string `json:"duration,omitempty"`
	AudioURL        string `json:"audio_url,omitempty"`
	Subtitle        string `json:"subtitle,omitempty"`
	Summary         string `json:"summary,omitempty"`
	ReadTimeMinutes int    `json:"read_time_minutes,omitempty"`
}

// MediaCard represents either a media hub card or an individual episode, post, or video card.
type MediaCard struct {
	ID              string          `json:"id"`
	Kind            string          `json:"kind"`
	Category        string          `json:"category"`
	Title           string          `json:"title"`
	Subtitle        string          `json:"subtitle,omitempty"`
	Summary         string          `json:"summary,omitempty"`
	SpeechText      string          `json:"speech_text,omitempty"`
	URL             string          `json:"url"`
	SiteURL         string          `json:"site_url,omitempty"`
	CoverURL        string          `json:"cover_url,omitempty"`
	AudioURL        string          `json:"audio_url,omitempty"`
	Icon            string          `json:"icon"`
	IconSVG         string          `json:"icon_svg"`
	Accent          string          `json:"accent"`
	Badge           string          `json:"badge,omitempty"`
	Episode         string          `json:"episode,omitempty"`
	Duration        string          `json:"duration,omitempty"`
	ReadTimeMinutes int             `json:"read_time_minutes,omitempty"`
	PublishedAt     string          `json:"published_at,omitempty"`
	DateLabel       string          `json:"date_label,omitempty"`
	Age             string          `json:"age,omitempty"`
	SubscribeLinks  []SubscribeLink `json:"subscribe_links,omitempty"`
	Entries         []MediaEntry    `json:"entries,omitempty"`
}
