// Package github fetches public GitHub profile, repository, release, and
// sponsor data using REST endpoints and public web pages without a PAT.
package github

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultAPIBaseURL is the default GitHub REST API origin.
	DefaultAPIBaseURL = "https://api.github.com"
	// DefaultRawBaseURL is the default raw GitHub content origin.
	DefaultRawBaseURL = "https://raw.githubusercontent.com"
	// DefaultWebBaseURL is the default GitHub web origin.
	DefaultWebBaseURL = "https://github.com"
	// DefaultUserAgent is sent when Client.UserAgent is empty.
	DefaultUserAgent = "vitrine/1.0"

	acceptJSON     = "application/vnd.github+json"
	acceptStarJSON = "application/vnd.github.star+json"
	apiVersion     = "2022-11-28"

	defaultHTTPTimeout             = 30 * time.Second
	defaultProfileMaxItem          = 5
	defaultContributionsWindowDays = 180
	maxPages                       = 50
	maxRawFileBytes                = 2 << 20
	maxJSONBodyBytes               = 10 << 20
	maxHTMLBodyBytes               = 4 << 20
	minReleaseCandidates           = 15
)

var (
	// ErrNotFound reports an HTTP 404 response from GitHub.
	ErrNotFound = errors.New("github: resource not found")
	// ErrInvalidRepo reports a repository identifier not in owner/repo form.
	ErrInvalidRepo = errors.New("github: invalid repository name")

	defaultHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}

	sponsorAltRe = regexp.MustCompile(`alt="@([A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)"`)
	linkNextRe   = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

	contributionEventTypes = map[string]bool{
		"PushEvent":              true,
		"PullRequestEvent":       true,
		"IssuesEvent":            true,
		"PullRequestReviewEvent": true,
		"CreateEvent":            true,
		"ReleaseEvent":           true,
	}
)

// Client calls GitHub REST API, raw content, and public web endpoints.
type Client struct {
	HTTPClient *http.Client
	APIBaseURL string
	RawBaseURL string
	WebBaseURL string
	Token      string
	UserAgent  string
	now        func() time.Time
}

// NewClient constructs a Client with default GitHub base URLs.
func NewClient(token string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
		APIBaseURL: DefaultAPIBaseURL,
		RawBaseURL: DefaultRawBaseURL,
		WebBaseURL: DefaultWebBaseURL,
		Token:      token,
		UserAgent:  DefaultUserAgent,
	}
}

// Stats summarises public account and organisation metrics.
type Stats struct {
	Username      string
	Followers     int
	PublicRepos   int
	PersonalStars int
	OrgRepos      int
	OrgStars      int
	TotalStars    int
	TotalCommits  int
	PublicPRs     int
	PublicIssues  int
}

// Contribution records the most recent public contribution event in a repository.
type Contribution struct {
	RepoFullName string
	RepoURL      string
	Description  string
	OccurredAt   time.Time
}

// PullRequest records a public pull request authored by a user.
type PullRequest struct {
	Title        string
	URL          string
	RepoFullName string
	RepoURL      string
	State        string
	CreatedAt    time.Time
}

// Release records a published GitHub release for a repository.
type Release struct {
	RepoFullName string
	RepoURL      string
	Description  string
	TagName      string
	ReleaseURL   string
	PublishedAt  time.Time
}

// StarredRepo records a repository starred by a user.
type StarredRepo struct {
	RepoFullName string
	RepoURL      string
	Description  string
	Stars        int
	StarredAt    time.Time
}

// Sponsor records a public GitHub sponsor account.
type Sponsor struct {
	Login string
	URL   string
}

// RepoMeta holds repository metadata and its optional latest release.
type RepoMeta struct {
	FullName      string
	Owner         string
	Name          string
	Description   string
	Homepage      string
	HTMLURL       string
	DefaultBranch string
	Language      string
	Stars         int
	LatestRelease *Release
}

// ProfileData aggregates all GitHub profile sections for rendering.
type ProfileData struct {
	Stats         Stats
	Contributions []Contribution
	PullRequests  []PullRequest
	Releases      []Release
	Starred       []StarredRepo
	Sponsors      []Sponsor
	PastSponsors  []Sponsor
	SponsorCount  int
}

// ProfileOptions configures optional inputs for FetchProfile.
type ProfileOptions struct {
	Orgs                    []string
	MaxItems                int
	ContributionsWindowDays int
	FallbackSponsors        []Sponsor
	ExtraReleaseRepos       []string
	ExcludeOwners           []string
}

type apiUser struct {
	Login       string `json:"login"`
	Followers   int    `json:"followers"`
	PublicRepos int    `json:"public_repos"`
}

type apiOwner struct {
	Login string `json:"login"`
}

type apiRepo struct {
	FullName      string    `json:"full_name"`
	Name          string    `json:"name"`
	Owner         apiOwner  `json:"owner"`
	Description   string    `json:"description"`
	Homepage      string    `json:"homepage"`
	HTMLURL       string    `json:"html_url"`
	DefaultBranch string    `json:"default_branch"`
	Language      string    `json:"language"`
	Stargazers    int       `json:"stargazers_count"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	PushedAt      time.Time `json:"pushed_at"`
}

type apiRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type apiEvent struct {
	Type      string       `json:"type"`
	Repo      apiEventRepo `json:"repo"`
	CreatedAt time.Time    `json:"created_at"`
}

type apiEventRepo struct {
	Name string `json:"name"`
}

type apiSearchCountResponse struct {
	TotalCount int `json:"total_count"`
}

type apiSearchIssuesResponse struct {
	TotalCount int              `json:"total_count"`
	Items      []apiSearchIssue `json:"items"`
}

type apiSearchIssue struct {
	Title         string          `json:"title"`
	HTMLURL       string          `json:"html_url"`
	RepositoryURL string          `json:"repository_url"`
	State         string          `json:"state"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	PullRequest   *apiPRTimestamp `json:"pull_request"`
}

type apiPRTimestamp struct {
	MergedAt *time.Time `json:"merged_at"`
}

type apiStarredItem struct {
	StarredAt time.Time `json:"starred_at"`
	Repo      apiRepo   `json:"repo"`
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return defaultHTTPClient
}

func (c *Client) apiBaseURL() string {
	if c != nil && strings.TrimSpace(c.APIBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(c.APIBaseURL), "/")
	}
	return DefaultAPIBaseURL
}

func (c *Client) rawBaseURL() string {
	if c != nil && strings.TrimSpace(c.RawBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(c.RawBaseURL), "/")
	}
	return DefaultRawBaseURL
}

func (c *Client) webBaseURL() string {
	if c != nil && strings.TrimSpace(c.WebBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(c.WebBaseURL), "/")
	}
	return DefaultWebBaseURL
}

func (c *Client) userAgent() string {
	if c != nil && strings.TrimSpace(c.UserAgent) != "" {
		return strings.TrimSpace(c.UserAgent)
	}
	return DefaultUserAgent
}

func (c *Client) token() string {
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Token)
}

func (c *Client) nowUTC() time.Time {
	if c != nil && c.now != nil {
		return c.now().UTC()
	}
	return time.Now().UTC()
}

type apiStatusError struct {
	Endpoint   string
	StatusCode int
	Body       string
}

func (e *apiStatusError) Error() string {
	return fmt.Sprintf("github API %s returned HTTP %d: %s", e.Endpoint, e.StatusCode, e.Body)
}

func isSearchDegradableError(err error) bool {
	var statusErr *apiStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	switch statusErr.StatusCode {
	case http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func (c *Client) getAPIJSON(ctx context.Context, endpoint, accept string, out any) (http.Header, error) {
	if accept == "" {
		accept = acceptJSON
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", endpoint, err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if tok := c.token(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return resp.Header, fmt.Errorf("%s: %w", endpoint, ErrNotFound)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return resp.Header, &apiStatusError{
			Endpoint:   endpoint,
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONBodyBytes+1))
	if err != nil {
		return resp.Header, fmt.Errorf("read response from %s: %w", endpoint, err)
	}
	if len(data) > maxJSONBodyBytes {
		return resp.Header, fmt.Errorf("response from %s exceeds %d bytes", endpoint, maxJSONBodyBytes)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return resp.Header, fmt.Errorf("decode response from %s: %w", endpoint, err)
	}
	return resp.Header, nil
}

func (c *Client) listRepos(ctx context.Context, firstURL string) ([]apiRepo, error) {
	var all []apiRepo
	nextURL := firstURL
	for page := 0; nextURL != "" && page < maxPages; page++ {
		var batch []apiRepo
		hdr, err := c.getAPIJSON(ctx, nextURL, acceptJSON, &batch)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		nextURL = parseNextLink(hdr.Get("Link"), c.apiBaseURL())
	}
	return all, nil
}

func parseNextLink(headerVal, allowedBase string) string {
	if headerVal == "" {
		return ""
	}
	m := linkNextRe.FindStringSubmatch(headerVal)
	if len(m) < 2 {
		return ""
	}
	candidate := strings.TrimSpace(m[1])
	if !strings.HasPrefix(candidate, allowedBase+"/") && candidate != allowedBase {
		return ""
	}
	return candidate
}

func splitRepoFullName(fullName string) (owner, repo string, err error) {
	trimmed := strings.Trim(strings.TrimSpace(fullName), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%w: %q", ErrInvalidRepo, fullName)
	}
	return parts[0], parts[1], nil
}

// FetchUserStats retrieves follower, repository, deduplicated star, commit,
// public PR, and public issue counts for username, the given organisations,
// and any extra repositories not already covered.
func (c *Client) FetchUserStats(ctx context.Context, username string, orgs, extraStarRepos []string) (Stats, error) {
	stats, _, err := c.fetchUserStatsWithRepos(ctx, username, orgs, extraStarRepos)
	return stats, err
}

func (c *Client) fetchUserStatsWithRepos(
	ctx context.Context,
	username string,
	orgs []string,
	extraStarRepos []string,
) (Stats, []apiRepo, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return Stats{}, nil, errors.New("github: username must not be empty")
	}

	var u apiUser
	userURL := fmt.Sprintf("%s/users/%s", c.apiBaseURL(), url.PathEscape(user))
	if _, err := c.getAPIJSON(ctx, userURL, acceptJSON, &u); err != nil {
		return Stats{}, nil, fmt.Errorf("fetch user %s: %w", user, err)
	}

	userReposURL := fmt.Sprintf("%s/users/%s/repos?type=owner&per_page=100", c.apiBaseURL(), url.PathEscape(user))
	userRepos, err := c.listRepos(ctx, userReposURL)
	if err != nil {
		return Stats{}, nil, fmt.Errorf("fetch repos for %s: %w", user, err)
	}

	seenRepos := make(map[string]bool, len(userRepos))
	personalStars := 0
	allRepos := make([]apiRepo, 0, len(userRepos))
	for _, r := range userRepos {
		if r.Private || r.Fork {
			continue
		}
		if key := repoDedupKey(r.FullName); key != "" {
			if seenRepos[key] {
				continue
			}
			seenRepos[key] = true
		}
		personalStars += r.Stargazers
		allRepos = append(allRepos, r)
	}

	orgReposCount, orgStars, orgAdded, err := c.collectOrgRepos(ctx, orgs, seenRepos)
	if err != nil {
		return Stats{}, nil, err
	}
	allRepos = append(allRepos, orgAdded...)

	extraStars, extraAdded, err := c.collectExtraStarRepos(ctx, extraStarRepos, seenRepos)
	if err != nil {
		return Stats{}, nil, err
	}
	allRepos = append(allRepos, extraAdded...)

	totalCommits, err := c.fetchTotalCommitCount(ctx, user)
	if err != nil {
		return Stats{}, nil, err
	}

	prCount, err := c.fetchPublicPRCount(ctx, user)
	if err != nil {
		return Stats{}, nil, err
	}

	issueCount, err := c.fetchPublicIssueCount(ctx, user)
	if err != nil {
		return Stats{}, nil, err
	}

	publicRepos := u.PublicRepos
	if publicRepos == 0 {
		publicRepos = len(userRepos)
	}

	stats := Stats{
		Username:      user,
		Followers:     u.Followers,
		PublicRepos:   publicRepos,
		PersonalStars: personalStars,
		OrgRepos:      orgReposCount,
		OrgStars:      orgStars,
		TotalStars:    personalStars + orgStars + extraStars,
		TotalCommits:  totalCommits,
		PublicPRs:     prCount,
		PublicIssues:  issueCount,
	}
	return stats, allRepos, nil
}

func repoDedupKey(fullName string) string {
	return strings.ToLower(strings.TrimSpace(fullName))
}

func (c *Client) collectOrgRepos(
	ctx context.Context,
	orgs []string,
	seenRepos map[string]bool,
) (int, int, []apiRepo, error) {
	orgReposCount := 0
	orgStars := 0
	var added []apiRepo
	seenOrgs := make(map[string]bool, len(orgs))
	for _, rawOrg := range orgs {
		org := strings.TrimSpace(rawOrg)
		if org == "" {
			continue
		}
		orgKey := strings.ToLower(org)
		if seenOrgs[orgKey] {
			continue
		}
		seenOrgs[orgKey] = true

		orgReposURL := fmt.Sprintf("%s/orgs/%s/repos?per_page=100", c.apiBaseURL(), url.PathEscape(org))
		orgRepos, err := c.listRepos(ctx, orgReposURL)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("fetch repos for org %s: %w", org, err)
		}
		for _, r := range orgRepos {
			if r.Private {
				continue
			}
			if key := repoDedupKey(r.FullName); key != "" {
				if seenRepos[key] {
					continue
				}
				seenRepos[key] = true
			}
			orgReposCount++
			orgStars += r.Stargazers
			added = append(added, r)
		}
	}
	return orgReposCount, orgStars, added, nil
}

func (c *Client) collectExtraStarRepos(
	ctx context.Context,
	extraRepos []string,
	seenRepos map[string]bool,
) (int, []apiRepo, error) {
	extraStars := 0
	var added []apiRepo
	for _, rawRepo := range extraRepos {
		trimmed := strings.TrimSpace(rawRepo)
		if trimmed == "" {
			continue
		}
		key := repoDedupKey(trimmed)
		if seenRepos[key] {
			continue
		}
		r, err := c.fetchAPIRepo(ctx, trimmed)
		if err != nil {
			if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidRepo) {
				continue
			}
			return 0, nil, err
		}
		seenRepos[key] = true
		if r.Private {
			continue
		}
		if strings.TrimSpace(r.FullName) == "" {
			r.FullName = trimmed
		}
		resolvedKey := repoDedupKey(r.FullName)
		if resolvedKey != "" && resolvedKey != key {
			if seenRepos[resolvedKey] {
				continue
			}
			seenRepos[resolvedKey] = true
		}
		extraStars += r.Stargazers
		added = append(added, r)
	}
	return extraStars, added, nil
}

func (c *Client) fetchTotalCommitCount(ctx context.Context, username string) (int, error) {
	q := url.Values{}
	q.Set("q", fmt.Sprintf("author:%s", username))
	q.Set("per_page", "1")
	endpoint := fmt.Sprintf("%s/search/commits?%s", c.apiBaseURL(), q.Encode())

	var res apiSearchCountResponse
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &res); err != nil {
		if isSearchDegradableError(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("fetch total commit count for %s: %w", username, err)
	}
	return res.TotalCount, nil
}

func (c *Client) fetchSearchIssuesCount(ctx context.Context, query, label string) (int, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("per_page", "1")
	endpoint := fmt.Sprintf("%s/search/issues?%s", c.apiBaseURL(), q.Encode())

	var res apiSearchCountResponse
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &res); err != nil {
		return 0, fmt.Errorf("fetch %s: %w", label, err)
	}
	return res.TotalCount, nil
}

func (c *Client) fetchPublicPRCount(ctx context.Context, username string) (int, error) {
	return c.fetchSearchIssuesCount(
		ctx,
		fmt.Sprintf("author:%s type:pr is:public", username),
		fmt.Sprintf("public PR count for %s", username),
	)
}

func (c *Client) fetchPublicIssueCount(ctx context.Context, username string) (int, error) {
	return c.fetchSearchIssuesCount(
		ctx,
		fmt.Sprintf("involves:%s type:issue is:public", username),
		fmt.Sprintf("public issue count for %s", username),
	)
}

// FetchRecentContributions reads public events for username within the past
// windowDays (defaulting to 180 when non-positive), skips any repository whose
// owner is in excludeOwners, tops up from public authored PRs and then public
// involved issues/PRs when fewer than limit repositories are found,
// deduplicates by repository keeping the newest timestamp, and enriches the
// top limit entries with repository descriptions.
func (c *Client) FetchRecentContributions(
	ctx context.Context,
	username string,
	excludeOwners []string,
	windowDays, limit int,
) ([]Contribution, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, errors.New("github: username must not be empty")
	}
	if limit <= 0 {
		return nil, nil
	}
	if windowDays <= 0 {
		windowDays = defaultContributionsWindowDays
	}
	cutoff := c.nowUTC().AddDate(0, 0, -windowDays)
	cutoffDate := cutoff.Format("2006-01-02")

	excludedSet, excludedList := buildExcludedOwners(excludeOwners)

	endpoint := fmt.Sprintf("%s/users/%s/events/public?per_page=100", c.apiBaseURL(), url.PathEscape(user))
	var events []apiEvent
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &events); err != nil {
		return nil, fmt.Errorf("fetch public events for %s: %w", user, err)
	}

	byRepo := collectEventContributions(events, excludedSet, cutoff, c.webBaseURL())

	if len(byRepo) < limit {
		prTerms := []string{
			fmt.Sprintf("author:%s", user),
			"type:pr",
			"is:public",
			"updated:>=" + cutoffDate,
		}
		if err := c.topUpContributionsFromSearch(
			ctx,
			prTerms,
			excludedSet,
			excludedList,
			cutoff,
			byRepo,
			fmt.Sprintf("fetch contribution pull requests for %s", user),
		); err != nil {
			return nil, err
		}
	}

	if len(byRepo) < limit {
		involvedTerms := []string{
			fmt.Sprintf("involves:%s", user),
			"is:public",
			"updated:>=" + cutoffDate,
		}
		if err := c.topUpContributionsFromSearch(
			ctx,
			involvedTerms,
			excludedSet,
			excludedList,
			cutoff,
			byRepo,
			fmt.Sprintf("fetch involved contributions for %s", user),
		); err != nil {
			return nil, err
		}
	}

	contributions := sortAndSliceContributions(byRepo, limit)
	c.enrichContributions(ctx, contributions)
	return contributions, nil
}

func collectEventContributions(
	events []apiEvent,
	excludedSet map[string]bool,
	cutoff time.Time,
	webBase string,
) map[string]Contribution {
	byRepo := make(map[string]Contribution)
	for _, ev := range events {
		if !contributionEventTypes[ev.Type] || ev.CreatedAt.Before(cutoff) {
			continue
		}
		repoName := strings.TrimSpace(ev.Repo.Name)
		if repoName == "" || isExcludedRepo(repoName, excludedSet) {
			continue
		}
		recordContribution(byRepo, repoName, webBase, ev.CreatedAt)
	}
	return byRepo
}

func sortAndSliceContributions(byRepo map[string]Contribution, limit int) []Contribution {
	contributions := make([]Contribution, 0, len(byRepo))
	for _, item := range byRepo {
		contributions = append(contributions, item)
	}
	slices.SortFunc(contributions, func(a, b Contribution) int {
		if !a.OccurredAt.Equal(b.OccurredAt) {
			if a.OccurredAt.After(b.OccurredAt) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.RepoFullName, b.RepoFullName)
	})
	if len(contributions) > limit {
		contributions = contributions[:limit]
	}
	return contributions
}

func (c *Client) enrichContributions(ctx context.Context, contributions []Contribution) {
	for i := range contributions {
		repo, err := c.fetchAPIRepo(ctx, contributions[i].RepoFullName)
		if err != nil {
			continue
		}
		contributions[i].Description = strings.TrimSpace(repo.Description)
		if repo.HTMLURL != "" {
			contributions[i].RepoURL = repo.HTMLURL
		}
	}
}

func buildExcludedOwners(excludeOwners []string) (map[string]bool, []string) {
	excluded := make(map[string]bool, len(excludeOwners))
	ordered := make([]string, 0, len(excludeOwners))
	for _, raw := range excludeOwners {
		owner := strings.ToLower(strings.TrimSpace(raw))
		if owner == "" || excluded[owner] {
			continue
		}
		excluded[owner] = true
		ordered = append(ordered, owner)
	}
	return excluded, ordered
}

func isExcludedRepo(repoFullName string, excluded map[string]bool) bool {
	if len(excluded) == 0 {
		return false
	}
	trimmed := strings.Trim(strings.TrimSpace(repoFullName), "/")
	owner, _, hasSlash := strings.Cut(trimmed, "/")
	if !hasSlash || owner == "" {
		return false
	}
	return excluded[strings.ToLower(owner)]
}

func isExcludedStarredRepo(repo apiRepo, excluded map[string]bool) bool {
	if len(excluded) == 0 {
		return false
	}
	if owner := strings.ToLower(strings.TrimSpace(repo.Owner.Login)); owner != "" && excluded[owner] {
		return true
	}
	return isExcludedRepo(repo.FullName, excluded)
}

func recordContribution(byRepo map[string]Contribution, repoName, webBase string, occurredAt time.Time) {
	key := strings.ToLower(repoName)
	existing, ok := byRepo[key]
	if !ok || occurredAt.After(existing.OccurredAt) {
		byRepo[key] = Contribution{
			RepoFullName: repoName,
			RepoURL:      fmt.Sprintf("%s/%s", webBase, repoName),
			OccurredAt:   occurredAt,
		}
	}
}

func (c *Client) topUpContributionsFromSearch(
	ctx context.Context,
	baseTerms []string,
	excludedSet map[string]bool,
	excludedList []string,
	cutoff time.Time,
	byRepo map[string]Contribution,
	errLabel string,
) error {
	terms := make([]string, 0, len(baseTerms)+len(excludedList))
	terms = append(terms, baseTerms...)
	for _, owner := range excludedList {
		terms = append(terms, fmt.Sprintf("-user:%s", owner))
	}

	q := url.Values{}
	q.Set("q", strings.Join(terms, " "))
	q.Set("sort", "updated")
	q.Set("order", "desc")
	q.Set("per_page", "30")
	endpoint := fmt.Sprintf("%s/search/issues?%s", c.apiBaseURL(), q.Encode())

	var res apiSearchIssuesResponse
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &res); err != nil {
		if ctx.Err() == nil && (isSearchDegradableError(err) || len(byRepo) > 0) {
			return nil
		}
		return fmt.Errorf("%s: %w", errLabel, err)
	}

	for _, item := range res.Items {
		repoName := repoFullNameFromURLs(item.RepositoryURL, item.HTMLURL)
		if repoName == "" || isExcludedRepo(repoName, excludedSet) {
			continue
		}
		occurredAt := item.UpdatedAt
		if occurredAt.IsZero() {
			occurredAt = item.CreatedAt
		}
		if occurredAt.Before(cutoff) {
			continue
		}
		recordContribution(byRepo, repoName, c.webBaseURL(), occurredAt)
	}
	return nil
}

// FetchRecentPullRequests queries the most recently created public pull
// requests authored by username.
func (c *Client) FetchRecentPullRequests(ctx context.Context, username string, limit int) ([]PullRequest, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, errors.New("github: username must not be empty")
	}
	if limit <= 0 {
		return nil, nil
	}

	perPage := min(limit, 100)
	q := url.Values{}
	q.Set("q", fmt.Sprintf("author:%s type:pr is:public", user))
	q.Set("sort", "created")
	q.Set("order", "desc")
	q.Set("per_page", strconv.Itoa(perPage))
	endpoint := fmt.Sprintf("%s/search/issues?%s", c.apiBaseURL(), q.Encode())

	var res apiSearchIssuesResponse
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &res); err != nil {
		return nil, fmt.Errorf("fetch pull requests for %s: %w", user, err)
	}

	items := res.Items
	if len(items) > limit {
		items = items[:limit]
	}

	prs := make([]PullRequest, 0, len(items))
	for _, item := range items {
		repoFull := repoFullNameFromURLs(item.RepositoryURL, item.HTMLURL)
		repoURL := ""
		if repoFull != "" {
			repoURL = fmt.Sprintf("%s/%s", c.webBaseURL(), repoFull)
		}
		state := item.State
		if item.PullRequest != nil && item.PullRequest.MergedAt != nil && !item.PullRequest.MergedAt.IsZero() {
			state = "merged"
		}
		prs = append(prs, PullRequest{
			Title:        item.Title,
			URL:          item.HTMLURL,
			RepoFullName: repoFull,
			RepoURL:      repoURL,
			State:        state,
			CreatedAt:    item.CreatedAt,
		})
	}
	return prs, nil
}

func repoFullNameFromURLs(repoAPIURL, prHTMLURL string) string {
	if u, err := url.Parse(strings.TrimSpace(repoAPIURL)); err == nil {
		trimmed := strings.Trim(u.Path, "/")
		if after, ok := strings.CutPrefix(trimmed, "repos/"); ok {
			parts := strings.Split(after, "/")
			if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
				return parts[0] + "/" + parts[1]
			}
		}
	}
	if u, err := url.Parse(strings.TrimSpace(prHTMLURL)); err == nil {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1]
		}
	}
	return ""
}

// FetchRecentStarred returns the repositories most recently starred by
// username, skipping any repository whose owner is in excludeOwners.
func (c *Client) FetchRecentStarred(
	ctx context.Context,
	username string,
	excludeOwners []string,
	limit int,
) ([]StarredRepo, error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, errors.New("github: username must not be empty")
	}
	if limit <= 0 {
		return nil, nil
	}

	excludedSet, _ := buildExcludedOwners(excludeOwners)

	perPage := min(max(limit*5, 50), 100)
	endpoint := fmt.Sprintf("%s/users/%s/starred?per_page=%d", c.apiBaseURL(), url.PathEscape(user), perPage)
	var items []apiStarredItem
	if _, err := c.getAPIJSON(ctx, endpoint, acceptStarJSON, &items); err != nil {
		return nil, fmt.Errorf("fetch starred repos for %s: %w", user, err)
	}

	starred := make([]StarredRepo, 0, min(len(items), limit))
	for _, item := range items {
		if isExcludedStarredRepo(item.Repo, excludedSet) {
			continue
		}
		repoURL := item.Repo.HTMLURL
		if repoURL == "" && item.Repo.FullName != "" {
			repoURL = fmt.Sprintf("%s/%s", c.webBaseURL(), item.Repo.FullName)
		}
		starred = append(starred, StarredRepo{
			RepoFullName: item.Repo.FullName,
			RepoURL:      repoURL,
			Description:  strings.TrimSpace(item.Repo.Description),
			Stars:        item.Repo.Stargazers,
			StarredAt:    item.StarredAt,
		})
		if len(starred) >= limit {
			break
		}
	}
	return starred, nil
}

func (c *Client) fetchAPIRepo(ctx context.Context, fullName string) (apiRepo, error) {
	owner, repo, err := splitRepoFullName(fullName)
	if err != nil {
		return apiRepo{}, err
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s", c.apiBaseURL(), url.PathEscape(owner), url.PathEscape(repo))
	var r apiRepo
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &r); err != nil {
		return apiRepo{}, fmt.Errorf("fetch repo %s: %w", fullName, err)
	}
	return r, nil
}

func (c *Client) fetchLatestRelease(ctx context.Context, fullName, repoURL, description string) (*Release, error) {
	owner, repo, err := splitRepoFullName(fullName)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.apiBaseURL(), url.PathEscape(owner), url.PathEscape(repo))
	var rel apiRelease
	if _, err := c.getAPIJSON(ctx, endpoint, acceptJSON, &rel); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch latest release for %s: %w", fullName, err)
	}
	published := rel.PublishedAt
	if published.IsZero() {
		published = rel.CreatedAt
	}
	if repoURL == "" {
		repoURL = fmt.Sprintf("%s/%s", c.webBaseURL(), fullName)
	}
	return &Release{
		RepoFullName: fullName,
		RepoURL:      repoURL,
		Description:  strings.TrimSpace(description),
		TagName:      rel.TagName,
		ReleaseURL:   rel.HTMLURL,
		PublishedAt:  published,
	}, nil
}

// FetchRepoMeta fetches repository metadata and its latest release. A missing
// release (HTTP 404) leaves LatestRelease nil without error.
func (c *Client) FetchRepoMeta(ctx context.Context, fullName string) (RepoMeta, error) {
	owner, repoName, err := splitRepoFullName(fullName)
	if err != nil {
		return RepoMeta{}, err
	}

	r, err := c.fetchAPIRepo(ctx, owner+"/"+repoName)
	if err != nil {
		return RepoMeta{}, err
	}

	resolvedFull := r.FullName
	if resolvedFull == "" {
		resolvedFull = owner + "/" + repoName
	}
	resolvedOwner := r.Owner.Login
	if resolvedOwner == "" {
		resolvedOwner = owner
	}
	resolvedName := r.Name
	if resolvedName == "" {
		resolvedName = repoName
	}
	htmlURL := r.HTMLURL
	if htmlURL == "" {
		htmlURL = fmt.Sprintf("%s/%s", c.webBaseURL(), resolvedFull)
	}

	latest, err := c.fetchLatestRelease(ctx, resolvedFull, htmlURL, r.Description)
	if err != nil {
		return RepoMeta{}, err
	}

	return RepoMeta{
		FullName:      resolvedFull,
		Owner:         resolvedOwner,
		Name:          resolvedName,
		Description:   strings.TrimSpace(r.Description),
		Homepage:      strings.TrimSpace(r.Homepage),
		HTMLURL:       htmlURL,
		DefaultBranch: r.DefaultBranch,
		Language:      r.Language,
		Stars:         r.Stargazers,
		LatestRelease: latest,
	}, nil
}

// FetchSponsors scrapes active and past public sponsors from GitHub's
// sponsors_partial endpoints. It uses fallback for active sponsors only when
// the filter=active HTTP request itself fails, never when HTTP 200 returns
// zero active sponsors.
func (c *Client) FetchSponsors(ctx context.Context, username string, fallback []Sponsor) (active, past []Sponsor, err error) {
	user := strings.TrimSpace(username)
	if user == "" {
		return nil, nil, errors.New("github: username must not be empty")
	}

	userKey := strings.ToLower(user)
	activeHTML, activeErr := c.fetchSponsorsPartial(ctx, user, "active")
	if activeErr != nil {
		if len(fallback) == 0 {
			return nil, nil, activeErr
		}
		active = cloneSponsors(fallback)
	} else {
		active = parseSponsorLogins(activeHTML, c.webBaseURL(), map[string]bool{userKey: true})
	}

	pastExclude := make(map[string]bool, len(active)+1)
	pastExclude[userKey] = true
	for _, s := range active {
		pastExclude[strings.ToLower(s.Login)] = true
	}

	inactiveHTML, inactiveErr := c.fetchSponsorsPartial(ctx, user, "inactive")
	if inactiveErr == nil {
		past = parseSponsorLogins(inactiveHTML, c.webBaseURL(), pastExclude)
	} else if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}

	return active, past, nil
}

func (c *Client) fetchSponsorsPartial(ctx context.Context, username, filter string) (string, error) {
	endpoint := fmt.Sprintf(
		"%s/sponsors/%s/sponsors_partial?filter=%s",
		c.webBaseURL(),
		url.PathEscape(username),
		url.QueryEscape(filter),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create sponsors request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", c.userAgent())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch sponsors partial %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sponsors partial %s returned HTTP %d", endpoint, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLBodyBytes))
	if err != nil {
		return "", fmt.Errorf("read sponsors partial %s: %w", endpoint, err)
	}
	return string(body), nil
}

func parseSponsorLogins(htmlBody, webBase string, exclude map[string]bool) []Sponsor {
	seen := make(map[string]bool, len(exclude))
	for k, v := range exclude {
		if v {
			seen[strings.ToLower(k)] = true
		}
	}
	var sponsors []Sponsor
	for _, match := range sponsorAltRe.FindAllStringSubmatch(htmlBody, -1) {
		if len(match) < 2 {
			continue
		}
		login := match[1]
		key := strings.ToLower(login)
		if seen[key] {
			continue
		}
		seen[key] = true
		sponsors = append(sponsors, Sponsor{
			Login: login,
			URL:   fmt.Sprintf("%s/%s", webBase, login),
		})
	}
	return sponsors
}

func cloneSponsors(in []Sponsor) []Sponsor {
	out := make([]Sponsor, len(in))
	copy(out, in)
	return out
}

// FetchRawFile downloads a file from <RawBaseURL>/<owner>/<repo>/<branch>/<filePath>,
// capping the body at 2 MiB and returning ErrNotFound on HTTP 404.
func (c *Client) FetchRawFile(ctx context.Context, owner, repo, branch, filePath string) ([]byte, error) {
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	branch = strings.TrimSpace(branch)
	cleanedPath := strings.TrimLeft(strings.TrimSpace(filePath), "/")
	if owner == "" || repo == "" || branch == "" || cleanedPath == "" {
		return nil, errors.New("github: owner, repo, branch, and filePath must be non-empty")
	}

	segments := strings.Split(cleanedPath, "/")
	escapedSegments := make([]string, 0, len(segments))
	for _, seg := range segments {
		if seg == "" || seg == "." || seg == ".." {
			return nil, fmt.Errorf("github: invalid raw file path %q", filePath)
		}
		escapedSegments = append(escapedSegments, url.PathEscape(seg))
	}

	endpoint := fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		c.rawBaseURL(),
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(branch),
		strings.Join(escapedSegments, "/"),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create raw file request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent())

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch raw file %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: %w", endpoint, ErrNotFound)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("raw file %s returned HTTP %d", endpoint, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRawFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read raw file %s: %w", endpoint, err)
	}
	if len(data) > maxRawFileBytes {
		return nil, fmt.Errorf("raw file %s exceeds %d bytes", endpoint, maxRawFileBytes)
	}
	return data, nil
}

// FetchProfile orchestrates all GitHub profile fetches and gathers recent
// releases sorted descending by PublishedAt.
func (c *Client) FetchProfile(
	ctx context.Context,
	username string,
	opts ProfileOptions,
) (*ProfileData, error) {
	maxItems := opts.MaxItems
	if maxItems <= 0 {
		maxItems = defaultProfileMaxItem
	}

	windowDays := opts.ContributionsWindowDays
	if windowDays <= 0 {
		windowDays = defaultContributionsWindowDays
	}

	stats, discoveredRepos, err := c.fetchUserStatsWithRepos(ctx, username, opts.Orgs, opts.ExtraReleaseRepos)
	if err != nil {
		return nil, err
	}

	contributions, err := c.FetchRecentContributions(ctx, username, opts.ExcludeOwners, windowDays, maxItems)
	if err != nil {
		return nil, err
	}

	prs, err := c.FetchRecentPullRequests(ctx, username, maxItems)
	if err != nil {
		return nil, err
	}

	starred, err := c.FetchRecentStarred(ctx, username, opts.ExcludeOwners, maxItems)
	if err != nil {
		return nil, err
	}

	activeSponsors, pastSponsors, err := c.FetchSponsors(ctx, username, opts.FallbackSponsors)
	if err != nil {
		return nil, err
	}

	releases, err := c.collectRecentReleases(ctx, discoveredRepos, opts.ExtraReleaseRepos, maxItems)
	if err != nil {
		return nil, err
	}

	return &ProfileData{
		Stats:         stats,
		Contributions: contributions,
		PullRequests:  prs,
		Releases:      releases,
		Starred:       starred,
		Sponsors:      activeSponsors,
		PastSponsors:  pastSponsors,
		SponsorCount:  len(activeSponsors),
	}, nil
}

func (c *Client) collectRecentReleases(
	ctx context.Context,
	discovered []apiRepo,
	extraRepos []string,
	limit int,
) ([]Release, error) {
	sortedDiscovered := make([]apiRepo, len(discovered))
	copy(sortedDiscovered, discovered)
	slices.SortFunc(sortedDiscovered, func(a, b apiRepo) int {
		if !a.PushedAt.Equal(b.PushedAt) {
			if a.PushedAt.After(b.PushedAt) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.FullName, b.FullName)
	})

	repoIndex := make(map[string]apiRepo, len(sortedDiscovered))
	for _, r := range sortedDiscovered {
		if r.FullName != "" {
			repoIndex[strings.ToLower(r.FullName)] = r
		}
	}

	seen := make(map[string]bool)
	var releases []Release

	appendRelease := func(rel *Release) {
		if rel == nil || rel.TagName == "" {
			return
		}
		releases = append(releases, *rel)
	}

	for _, fullName := range extraRepos {
		trimmed := strings.TrimSpace(fullName)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true

		if known, ok := repoIndex[key]; ok {
			rel, err := c.fetchLatestRelease(ctx, known.FullName, known.HTMLURL, known.Description)
			if err != nil {
				return nil, err
			}
			appendRelease(rel)
			continue
		}

		meta, err := c.FetchRepoMeta(ctx, trimmed)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		appendRelease(meta.LatestRelease)
	}

	candidateCap := max(limit*3, minReleaseCandidates)
	checked := 0
	for _, r := range sortedDiscovered {
		if checked >= candidateCap {
			break
		}
		if r.FullName == "" {
			continue
		}
		key := strings.ToLower(r.FullName)
		if seen[key] {
			continue
		}
		seen[key] = true
		checked++

		rel, err := c.fetchLatestRelease(ctx, r.FullName, r.HTMLURL, r.Description)
		if err != nil {
			return nil, err
		}
		appendRelease(rel)
	}

	slices.SortFunc(releases, func(a, b Release) int {
		if !a.PublishedAt.Equal(b.PublishedAt) {
			if a.PublishedAt.After(b.PublishedAt) {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.RepoFullName, b.RepoFullName)
	})

	if len(releases) > limit {
		releases = releases[:limit]
	}
	return releases, nil
}
