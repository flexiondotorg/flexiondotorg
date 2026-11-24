package site

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
)

const (
	defaultBlueskyAPIBase  = "https://public.api.bsky.app"
	defaultDecapiBase      = "https://decapi.me"
	maxSocialResponseSize  = 1 << 20
	maxYouTubeResponseSize = 4 << 20
	socialUserAgent        = "vitrine-site/1.0 (+https://wimpys.world)"
	ytInitialDataPrefix    = "var ytInitialData = "
	ytInitialDataEndMarker = ";</script>"
)

var (
	reGTSFollowers = regexp.MustCompile(`(?is)<dd\b[^>]*class="[^"]*\bfolloweddd\b[^"]*"[^>]*>\s*([0-9,]+)\s*</dd>`)
	reGTSPosts     = regexp.MustCompile(`(?is)<dd\b[^>]*class="[^"]*\bpostsdd\b[^"]*"[^>]*>\s*([0-9,]+)\s*</dd>`)
	reYTSubs       = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?[kmb]?)\s+subscribers?$`)
	reYTVideos     = regexp.MustCompile(`(?i)^([0-9,]+(?:\.[0-9]+)?[kmb]?)\s+videos?$`)
)

// SocialFetcher retrieves unauthenticated public profile metrics for configured
// Fediverse (Mastodon / GoToSocial), Bluesky, Twitch, and YouTube link cards.
type SocialFetcher struct {
	HTTPClient        *http.Client
	BlueskyAPIBaseURL string
	DecapiBaseURL     string
}

// FetchAll fetches public social profile counts for supported link entries.
// Individual failures are returned as non-fatal errors so site builds succeed
// with graceful handle fallbacks.
func (f *SocialFetcher) FetchAll(ctx context.Context, links []config.SiteLink) (map[string]SocialProfile, []error) {
	if len(links) == 0 {
		return nil, nil
	}

	client := f.httpClient()
	profiles := make(map[string]SocialProfile)
	var errs []error

	for i, link := range links {
		key := linkLookupKey(link, i)
		icon := strings.ToLower(strings.TrimSpace(link.Icon))

		switch {
		case icon == "mastodon" || key == "mastodon":
			prof, err := f.fetchFediverseProfile(ctx, client, link.URL)
			if err != nil {
				errs = append(errs, fmt.Errorf("mastodon (%s): %w", link.URL, err))
				continue
			}
			if prof.Followers > 0 || prof.Posts > 0 {
				profiles[key] = prof
			}
		case icon == "bluesky" || key == "bluesky":
			prof, err := f.fetchBlueskyProfile(ctx, client, link.URL)
			if err != nil {
				errs = append(errs, fmt.Errorf("bluesky (%s): %w", link.URL, err))
				continue
			}
			if prof.Followers > 0 || prof.Posts > 0 {
				profiles[key] = prof
			}
		case icon == "twitch" || key == "twitch":
			prof, err := f.fetchTwitchProfile(ctx, client, link.URL)
			if err != nil {
				errs = append(errs, fmt.Errorf("twitch (%s): %w", link.URL, err))
				continue
			}
			if prof.Followers > 0 {
				profiles[key] = prof
			}
		case icon == "youtube" || key == "youtube":
			prof, err := f.fetchYouTubeProfile(ctx, client, link.URL)
			if err != nil {
				errs = append(errs, fmt.Errorf("youtube (%s): %w", link.URL, err))
				continue
			}
			if prof.Subscribers != "" || prof.Videos != "" {
				profiles[key] = prof
			}
		}
	}

	return profiles, errs
}

func (f *SocialFetcher) httpClient() *http.Client {
	if f != nil && f.HTTPClient != nil {
		return f.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (f *SocialFetcher) blueskyBase() string {
	if f != nil && strings.TrimSpace(f.BlueskyAPIBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(f.BlueskyAPIBaseURL), "/")
	}
	return defaultBlueskyAPIBase
}

func (f *SocialFetcher) decapiBase() string {
	if f != nil && strings.TrimSpace(f.DecapiBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(f.DecapiBaseURL), "/")
	}
	return defaultDecapiBase
}

func (f *SocialFetcher) fetchTwitchProfile(ctx context.Context, client *http.Client, rawURL string) (SocialProfile, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return SocialProfile{}, fmt.Errorf("parse twitch URL: %w", err)
	}
	channel := lastPathSegment(parsed.Path)
	if channel == "" {
		return SocialProfile{}, errors.New("could not extract Twitch channel from URL")
	}

	endpoint := f.decapiBase() + "/twitch/followcount/" + url.PathEscape(channel)
	body, err := fetchPublicBytes(ctx, client, endpoint, "text/plain", maxSocialResponseSize)
	if err != nil {
		return SocialProfile{}, err
	}
	followers := parseCommaInt(string(body))
	if followers == 0 {
		return SocialProfile{}, fmt.Errorf("unexpected followcount response %q", strings.TrimSpace(string(body)))
	}
	return SocialProfile{Followers: followers}, nil
}

func (f *SocialFetcher) fetchYouTubeProfile(ctx context.Context, client *http.Client, rawURL string) (SocialProfile, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return SocialProfile{}, errors.New("empty YouTube URL")
	}
	body, err := fetchPublicBytes(ctx, client, trimmed, "text/html", maxYouTubeResponseSize)
	if err != nil {
		return SocialProfile{}, err
	}
	prof := parseYouTubeChannelStats(string(body))
	if prof.Subscribers == "" && prof.Videos == "" {
		return SocialProfile{}, errors.New("no channel subscriber or video counts found")
	}
	return prof, nil
}

func (f *SocialFetcher) fetchBlueskyProfile(ctx context.Context, client *http.Client, rawURL string) (SocialProfile, error) {
	actor, ok := parseBlueskyActor(rawURL)
	if !ok {
		return SocialProfile{}, errors.New("could not extract Bluesky actor from URL")
	}

	endpoint := f.blueskyBase() + "/xrpc/app.bsky.actor.getProfile?actor=" + url.QueryEscape(actor)
	body, err := fetchPublicBytes(ctx, client, endpoint, "application/json", maxSocialResponseSize)
	if err != nil {
		return SocialProfile{}, err
	}

	var payload struct {
		FollowersCount int `json:"followersCount"`
		PostsCount     int `json:"postsCount"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return SocialProfile{}, fmt.Errorf("decode profile JSON: %w", err)
	}
	return SocialProfile{
		Followers: max(0, payload.FollowersCount),
		Posts:     max(0, payload.PostsCount),
	}, nil
}

func (f *SocialFetcher) fetchFediverseProfile(ctx context.Context, client *http.Client, rawURL string) (SocialProfile, error) {
	origin, acct, ok := parseFediverseURL(rawURL)
	if !ok {
		return SocialProfile{}, errors.New("could not extract Fediverse account from URL")
	}

	// Primary path: standard Mastodon public account lookup API.
	apiURL := origin + "/api/v1/accounts/lookup?acct=" + url.QueryEscape(acct)
	if body, err := fetchPublicBytes(ctx, client, apiURL, "application/json", maxSocialResponseSize); err == nil {
		var payload struct {
			FollowersCount int `json:"followers_count"`
			StatusesCount  int `json:"statuses_count"`
		}
		if jsonErr := json.Unmarshal(body, &payload); jsonErr == nil && (payload.FollowersCount > 0 || payload.StatusesCount > 0) {
			return SocialProfile{
				Followers: max(0, payload.FollowersCount),
				Posts:     max(0, payload.StatusesCount),
			}, nil
		}
	}

	// Fallback path: public GoToSocial / Fediverse HTML profile page stats.
	htmlBody, err := fetchPublicBytes(ctx, client, strings.TrimSpace(rawURL), "text/html", maxSocialResponseSize)
	if err != nil {
		return SocialProfile{}, err
	}
	prof := parseGoToSocialHTMLStats(string(htmlBody))
	if prof.Followers == 0 && prof.Posts == 0 {
		return SocialProfile{}, errors.New("no public profile counts found")
	}
	return prof, nil
}

func fetchPublicBytes(ctx context.Context, client *http.Client, endpoint, accept string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", socialUserAgent)
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return body, nil
}

func parseYouTubeChannelStats(htmlDoc string) SocialProfile {
	start := strings.Index(htmlDoc, ytInitialDataPrefix)
	if start < 0 {
		return SocialProfile{}
	}
	rest := htmlDoc[start+len(ytInitialDataPrefix):]
	end := strings.Index(rest, ytInitialDataEndMarker)
	if end < 0 {
		return SocialProfile{}
	}

	var payload struct {
		Header struct {
			PageHeaderRenderer struct {
				Content struct {
					PageHeaderViewModel struct {
						Metadata struct {
							ContentMetadataViewModel struct {
								MetadataRows []struct {
									MetadataParts []struct {
										Text struct {
											Content string `json:"content"`
										} `json:"text"`
									} `json:"metadataParts"`
								} `json:"metadataRows"`
							} `json:"contentMetadataViewModel"`
						} `json:"metadata"`
					} `json:"pageHeaderViewModel"`
				} `json:"content"`
			} `json:"pageHeaderRenderer"`
		} `json:"header"`
	}
	if err := json.Unmarshal([]byte(rest[:end]), &payload); err != nil {
		return SocialProfile{}
	}

	var prof SocialProfile
	for _, row := range payload.Header.PageHeaderRenderer.Content.PageHeaderViewModel.Metadata.ContentMetadataViewModel.MetadataRows {
		for _, part := range row.MetadataParts {
			content := strings.TrimSpace(part.Text.Content)
			if prof.Subscribers == "" {
				if m := reYTSubs.FindStringSubmatch(content); len(m) == 2 {
					prof.Subscribers = strings.ToUpper(m[1])
				}
			}
			if prof.Videos == "" {
				if m := reYTVideos.FindStringSubmatch(content); len(m) == 2 {
					prof.Videos = strings.ToUpper(m[1])
				}
			}
		}
	}
	return prof
}

func parseFediverseURL(rawURL string) (origin, acct string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", false
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for _, seg := range segments {
		if strings.HasPrefix(seg, "@") && len(seg) > 1 {
			return parsed.Scheme + "://" + parsed.Host, strings.TrimPrefix(seg, "@"), true
		}
	}
	return "", "", false
}

func parseBlueskyActor(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return "", false
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	for i := 0; i+1 < len(segments); i++ {
		if strings.EqualFold(segments[i], "profile") && strings.TrimSpace(segments[i+1]) != "" {
			return strings.TrimSpace(segments[i+1]), true
		}
	}
	return "", false
}

func parseGoToSocialHTMLStats(htmlDoc string) SocialProfile {
	var prof SocialProfile
	if m := reGTSFollowers.FindStringSubmatch(htmlDoc); len(m) == 2 {
		prof.Followers = parseCommaInt(m[1])
	}
	if m := reGTSPosts.FindStringSubmatch(htmlDoc); len(m) == 2 {
		prof.Posts = parseCommaInt(m[1])
	}
	return prof
}

func parseCommaInt(raw string) int {
	clean := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	v, err := strconv.Atoi(clean)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func linkLookupKey(link config.SiteLink, idx int) string {
	if s := slugify(link.Slug); s != "" {
		return s
	}
	if s := slugify(link.Title); s != "" {
		return s
	}
	return fmt.Sprintf("link-%d", idx+1)
}
