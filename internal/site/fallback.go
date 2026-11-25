package site

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maxFallbackDataBytes = 10 << 20 // 10 MiB

// FallbackReport summarises which sections of a SiteData payload were
// backfilled from a previous cached or live data.json snapshot.
type FallbackReport struct {
	ProjectsRestored   int
	ProjectsEnriched   int
	PackagesRestored   bool
	StatsRestored      bool
	LinksRestored      int
	MediaKindsRestored []string
}

// Used reports whether any section relied on fallback data.
func (r FallbackReport) Used() bool {
	return r.ProjectsRestored > 0 ||
		r.ProjectsEnriched > 0 ||
		r.PackagesRestored ||
		r.StatsRestored ||
		r.LinksRestored > 0 ||
		len(r.MediaKindsRestored) > 0
}

// Summary formats a compact human-readable description of restored sections.
func (r FallbackReport) Summary() string {
	var parts []string
	if r.ProjectsRestored > 0 {
		parts = append(parts, fmt.Sprintf("%d projects restored", r.ProjectsRestored))
	}
	if r.ProjectsEnriched > 0 {
		parts = append(parts, fmt.Sprintf("%d projects backfilled", r.ProjectsEnriched))
	}
	if r.PackagesRestored {
		parts = append(parts, "packages restored")
	}
	if r.StatsRestored {
		parts = append(parts, "stats restored")
	}
	if r.LinksRestored > 0 {
		parts = append(parts, fmt.Sprintf("%d links restored", r.LinksRestored))
	}
	if len(r.MediaKindsRestored) > 0 {
		parts = append(parts, "media restored ("+strings.Join(r.MediaKindsRestored, ", ")+")")
	}
	return strings.Join(parts, ", ")
}

// LoadData loads a previously built SiteData JSON payload from an HTTP(S) URL
// or a local file path.
func LoadData(ctx context.Context, httpClient *http.Client, source string) (*SiteData, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, errors.New("site: fallback source must not be empty")
	}

	var raw []byte
	var err error
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		raw, err = fetchDataURL(ctx, httpClient, trimmed)
	} else {
		// #nosec G304 -- fallback file path is explicitly provided by the caller.
		raw, err = os.ReadFile(filepath.Clean(trimmed))
	}
	if err != nil {
		return nil, err
	}

	var data SiteData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("site: decode fallback data %q: %w", trimmed, err)
	}
	return &data, nil
}

func fetchDataURL(ctx context.Context, httpClient *http.Client, rawURL string) ([]byte, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("site: build fallback request %q: %w", rawURL, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "vitrine-site/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("site: fetch fallback %q: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("site: fetch fallback %q: unexpected HTTP %d", rawURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFallbackDataBytes))
	if err != nil {
		return nil, fmt.Errorf("site: read fallback %q: %w", rawURL, err)
	}
	return body, nil
}

// SaveDataCache persists a built SiteData payload to a local JSON cache file.
func SaveDataCache(path string, data *SiteData) error {
	if data == nil {
		return errors.New("site: data must not be nil")
	}
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return errors.New("site: cache path must not be empty")
	}

	if dir := filepath.Dir(cleaned); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("site: create cache dir %q: %w", dir, err)
		}
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("site: marshal cache data: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(cleaned, raw, 0o600); err != nil {
		return fmt.Errorf("site: write cache %q: %w", cleaned, err)
	}
	return nil
}

// MergeFallback backfills degraded or empty sections in current using fallback
// while always preserving configured profile copy, layout settings, and
// successfully fetched live entries.
//
// enabledMediaKinds limits media fallback to enabled feed kinds ("episode",
// "post", "video"); passing nil allows fallback for any missing kind.
func MergeFallback(current, fallback *SiteData, enabledMediaKinds map[string]bool) FallbackReport {
	var report FallbackReport
	if current == nil || fallback == nil {
		return report
	}

	haberdasherMissing := len(current.Packages) == 0 && len(fallback.Packages) > 0
	mergePackages(current, fallback, &report)
	mergeProjects(current, fallback, haberdasherMissing, &report)
	mergeStats(current, fallback, &report)
	mergeLinks(current, fallback, &report)
	mergeMedia(current, fallback, enabledMediaKinds, &report)
	return report
}

func mergePackages(current, fallback *SiteData, report *FallbackReport) {
	if current.HostLogos == nil {
		current.HostLogos = make(map[string]string)
	}
	for host, logo := range fallback.HostLogos {
		if current.HostLogos[host] == "" && logo != "" {
			current.HostLogos[host] = logo
		}
	}
	if len(current.Packages) == 0 && len(fallback.Packages) > 0 {
		current.Packages = slices.Clone(fallback.Packages)
		current.Stats.HostCount = len(current.Packages)
		report.PackagesRestored = true
	}
}

func mergeProjects(current, fallback *SiteData, haberdasherMissing bool, report *FallbackReport) {
	if len(fallback.Projects) == 0 {
		return
	}
	if len(current.Projects) == 0 {
		current.Projects = slices.Clone(fallback.Projects)
		current.Stats.ProjectCount = len(current.Projects)
		report.ProjectsRestored = len(current.Projects)
		return
	}

	byKey := make(map[string]ProjectCard, len(fallback.Projects)*2)
	for _, prev := range fallback.Projects {
		if k := projectKey(prev); k != "" {
			byKey[k] = prev
		}
		if prev.Slug != "" {
			byKey["slug:"+strings.ToLower(prev.Slug)] = prev
		}
	}

	present := make(map[string]bool, len(current.Projects)*2)
	for i := range current.Projects {
		cur := &current.Projects[i]
		if k := projectKey(*cur); k != "" {
			present[k] = true
		}
		if cur.Slug != "" {
			present["slug:"+strings.ToLower(cur.Slug)] = true
		}
		prev, ok := lookupFallbackProject(*cur, byKey)
		if ok && backfillProject(cur, prev) {
			report.ProjectsEnriched++
		}
	}

	if haberdasherMissing {
		for _, prev := range fallback.Projects {
			if isProjectPresent(prev, present) {
				continue
			}
			current.Projects = append(current.Projects, prev)
			report.ProjectsRestored++
		}
		if report.ProjectsRestored > 0 && current.DefaultSort == "stars" {
			slices.SortStableFunc(current.Projects, func(a, b ProjectCard) int {
				if c := cmp.Compare(b.Stars, a.Stars); c != 0 {
					return c
				}
				return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
			})
		}
	}
	current.Stats.ProjectCount = len(current.Projects)
}

func projectKey(p ProjectCard) string {
	if k := strings.ToLower(strings.TrimSpace(p.FullName)); k != "" {
		return "repo:" + k
	}
	return ""
}

func lookupFallbackProject(cur ProjectCard, byKey map[string]ProjectCard) (ProjectCard, bool) {
	if k := projectKey(cur); k != "" {
		if prev, ok := byKey[k]; ok {
			return prev, true
		}
	}
	if cur.Slug != "" {
		if prev, ok := byKey["slug:"+strings.ToLower(cur.Slug)]; ok {
			return prev, true
		}
	}
	return ProjectCard{}, false
}

func isProjectPresent(p ProjectCard, present map[string]bool) bool {
	if k := projectKey(p); k != "" && present[k] {
		return true
	}
	return p.Slug != "" && present["slug:"+strings.ToLower(p.Slug)]
}

func backfillProject(cur *ProjectCard, prev ProjectCard) bool {
	changed := false
	if cur.Stars == 0 && prev.Stars > 0 {
		cur.Stars = prev.Stars
		cur.StarsFormatted = prev.StarsFormatted
		cur.StarsCompact = prev.StarsCompact
		changed = true
	}
	if cur.Description == "" && prev.Description != "" {
		cur.Description = prev.Description
		changed = true
	}
	if cur.LatestTag == "" && prev.LatestTag != "" {
		cur.LatestTag = prev.LatestTag
		cur.ReleaseURL = prev.ReleaseURL
		cur.PublishedAt = prev.PublishedAt
		cur.PublishedUnix = prev.PublishedUnix
		cur.ReleaseAge = prev.ReleaseAge
		cur.Fresh = prev.Fresh
		changed = true
	}
	if cur.Language == "" && prev.Language != "" {
		cur.Language = prev.Language
		cur.LanguageColor = prev.LanguageColor
		changed = true
	}
	if cur.Homepage == "" && prev.Homepage != "" {
		cur.Homepage = prev.Homepage
		changed = true
	}
	if cur.LogoSVG == "" && cur.LogoDataURI == "" && (prev.LogoSVG != "" || prev.LogoDataURI != "") {
		cur.LogoSVG = prev.LogoSVG
		cur.LogoDataURI = prev.LogoDataURI
		changed = true
	}
	if len(cur.Hosts) == 0 && len(prev.Hosts) > 0 {
		cur.Hosts = slices.Clone(prev.Hosts)
		changed = true
	}
	return changed
}

func mergeStats(current, fallback *SiteData, report *FallbackReport) {
	if current.Stats.TotalStars == 0 && fallback.Stats.TotalStars > 0 {
		current.Stats.TotalStars = fallback.Stats.TotalStars
		current.Stats.StarsFormatted = fallback.Stats.StarsFormatted
		current.Stats.Followers = fallback.Stats.Followers
		current.Stats.PublicRepos = fallback.Stats.PublicRepos
		current.Stats.TotalCommits = fallback.Stats.TotalCommits
		report.StatsRestored = true
	}
	if current.Stats.ActiveSponsors == 0 && fallback.Stats.ActiveSponsors > 0 {
		current.Stats.ActiveSponsors = fallback.Stats.ActiveSponsors
		current.Stats.PastSponsors = fallback.Stats.PastSponsors
		report.StatsRestored = true
	}
	if current.Stats.DiscordMembers == 0 && fallback.Stats.DiscordMembers > 0 {
		current.Stats.DiscordOnline = fallback.Stats.DiscordOnline
		current.Stats.DiscordMembers = fallback.Stats.DiscordMembers
		report.StatsRestored = true
	}
}

func mergeLinks(current, fallback *SiteData, report *FallbackReport) {
	if len(current.Links) == 0 || len(fallback.Links) == 0 {
		return
	}
	byID := make(map[string]LinkCard, len(fallback.Links))
	for _, prev := range fallback.Links {
		if prev.ID != "" {
			byID[prev.ID] = prev
		}
	}
	for i := range current.Links {
		cur := &current.Links[i]
		prev, ok := byID[cur.ID]
		if !ok || !linkNeedsFallback(*cur, prev) {
			continue
		}
		if cur.Badge == "" && prev.Badge != "" {
			cur.Badge = prev.Badge
		}
		if isStaticFallbackStats(cur.Stats) && !isStaticFallbackStats(prev.Stats) {
			cur.Stats = slices.Clone(prev.Stats)
		}
		if len(cur.Meta) == 0 && len(prev.Meta) > 0 {
			cur.Meta = slices.Clone(prev.Meta)
		}
		if len(cur.Sponsors) == 0 && len(prev.Sponsors) > 0 {
			cur.Sponsors = slices.Clone(prev.Sponsors)
		}
		if len(cur.PastSponsors) == 0 && len(prev.PastSponsors) > 0 {
			cur.PastSponsors = slices.Clone(prev.PastSponsors)
		}
		report.LinksRestored++
	}
}

func linkNeedsFallback(cur, prev LinkCard) bool {
	if len(cur.Sponsors) == 0 && len(prev.Sponsors) > 0 {
		return true
	}
	if cur.Badge == "" && prev.Badge != "" {
		return true
	}
	return isStaticFallbackStats(cur.Stats) && !isStaticFallbackStats(prev.Stats)
}

func isStaticFallbackStats(stats []StatRow) bool {
	if len(stats) == 0 {
		return true
	}
	for _, st := range stats {
		switch strings.ToUpper(strings.TrimSpace(st.Label)) {
		case "SITE", "HANDLE", "PROFILE", "GITHUB":
		default:
			return false
		}
	}
	return true
}

func mergeMedia(current, fallback *SiteData, enabledKinds map[string]bool, report *FallbackReport) {
	if len(fallback.Media) == 0 {
		return
	}
	haveKind := make(map[string]bool)
	for _, item := range current.Media {
		haveKind[item.Kind] = true
	}

	restoredKinds := make(map[string]bool)
	for _, prev := range fallback.Media {
		if prev.Kind == "" || haveKind[prev.Kind] {
			continue
		}
		if enabledKinds != nil && !enabledKinds[prev.Kind] {
			continue
		}
		current.Media = append(current.Media, prev)
		if !restoredKinds[prev.Kind] {
			restoredKinds[prev.Kind] = true
			report.MediaKindsRestored = append(report.MediaKindsRestored, prev.Kind)
		}
	}
}
