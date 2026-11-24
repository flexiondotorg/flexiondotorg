package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
}

func TestFetchUserStatsPaginationAndAuth(t *testing.T) {
	t.Parallel()

	var srvURL string
	var gotAuth string
	extraFetches := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/users/flexiondotorg":
			writeJSON(t, w, map[string]any{
				"login":        "flexiondotorg",
				"followers":    1234,
				"public_repos": 42,
			})
		case "/users/flexiondotorg/repos":
			if r.URL.Query().Get("page") == "2" {
				writeJSON(t, w, []map[string]any{
					{"full_name": "flexiondotorg/b", "stargazers_count": 30},
					{"full_name": "flexiondotorg/secret", "stargazers_count": 999, "private": true},
					{"full_name": "flexiondotorg/forked", "stargazers_count": 500, "fork": true},
				})
				return
			}
			w.Header().Set("Link", fmt.Sprintf(`<%s/users/flexiondotorg/repos?type=owner&per_page=100&page=2>; rel="next"`, srvURL))
			writeJSON(t, w, []map[string]any{
				{"full_name": "flexiondotorg/a", "stargazers_count": 20},
			})
		case "/orgs/wimpysworld/repos":
			writeJSON(t, w, []map[string]any{
				{"full_name": "wimpysworld/tailor", "stargazers_count": 100},
				{"full_name": "wimpysworld/sidra", "stargazers_count": 50},
				{"full_name": "flexiondotorg/a", "stargazers_count": 20},
			})
		case "/repos/linuxmatters/jive-vocals":
			extraFetches++
			writeJSON(t, w, map[string]any{
				"full_name":        "linuxmatters/jive-vocals",
				"stargazers_count": 40,
			})
		case "/repos/linuxmatters/missing":
			http.NotFound(w, r)
		case "/search/commits":
			if r.URL.Query().Get("q") != "author:flexiondotorg" {
				http.Error(w, "unexpected commits query", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, map[string]any{"total_count": 4321})
		case "/search/issues":
			q := r.URL.Query().Get("q")
			switch q {
			case "author:flexiondotorg type:pr is:public":
				writeJSON(t, w, map[string]any{"total_count": 88, "items": []any{}})
			case "involves:flexiondotorg type:issue is:public":
				writeJSON(t, w, map[string]any{"total_count": 64, "items": []any{}})
			default:
				http.Error(w, "unexpected query: "+q, http.StatusBadRequest)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	client := &Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		Token:      "test-token",
	}

	stats, err := client.FetchUserStats(
		context.Background(),
		"flexiondotorg",
		[]string{"wimpysworld", "WimpysWorld"},
		[]string{"WimpysWorld/Tailor", "linuxmatters/jive-vocals", "LinuxMatters/jive-vocals", "linuxmatters/missing"},
	)
	if err != nil {
		t.Fatalf("FetchUserStats returned error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Bearer token header, got %q", gotAuth)
	}
	if extraFetches != 1 {
		t.Fatalf("expected extra repo to be fetched once, got %d", extraFetches)
	}
	if stats.Followers != 1234 || stats.PublicRepos != 42 {
		t.Fatalf("unexpected user counts: %+v", stats)
	}
	if stats.PersonalStars != 50 || stats.OrgRepos != 2 || stats.OrgStars != 150 || stats.TotalStars != 240 {
		t.Fatalf("unexpected star/org totals: %+v", stats)
	}
	if stats.TotalCommits != 4321 || stats.PublicPRs != 88 || stats.PublicIssues != 64 {
		t.Fatalf("unexpected activity totals: %+v", stats)
	}
}

func TestFetchUserStatsCommitSearchDegradesGracefully(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusUnprocessableEntity, http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(fmt.Sprintf("HTTP_%d", status), func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/users/flexiondotorg":
					writeJSON(t, w, map[string]any{"login": "flexiondotorg", "followers": 10, "public_repos": 1})
				case "/users/flexiondotorg/repos":
					writeJSON(t, w, []map[string]any{
						{"full_name": "flexiondotorg/a", "stargazers_count": 7},
					})
				case "/search/commits":
					http.Error(w, "search unavailable", status)
				case "/search/issues":
					if strings.Contains(r.URL.Query().Get("q"), "type:pr") {
						writeJSON(t, w, map[string]any{"total_count": 12})
						return
					}
					writeJSON(t, w, map[string]any{"total_count": 5})
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			client := &Client{HTTPClient: srv.Client(), APIBaseURL: srv.URL}
			stats, err := client.FetchUserStats(context.Background(), "flexiondotorg", nil, nil)
			if err != nil {
				t.Fatalf("expected commit search HTTP %d to degrade gracefully, got error: %v", status, err)
			}
			if stats.TotalCommits != 0 || stats.TotalStars != 7 || stats.PublicPRs != 12 || stats.PublicIssues != 5 {
				t.Fatalf("unexpected stats on degraded commit search: %+v", stats)
			}
		})
	}
}

func TestFetchRecentContributionsDedupAndEnrich(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	tooOld := now.AddDate(0, 0, -181)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/flexiondotorg/events/public":
			writeJSON(t, w, []map[string]any{
				{"type": "WatchEvent", "repo": map[string]string{"name": "ignored/star"}, "created_at": t2.Add(time.Hour)},
				{"type": "PushEvent", "repo": map[string]string{"name": "stale/repo"}, "created_at": tooOld},
				{"type": "PushEvent", "repo": map[string]string{"name": "wimpysworld/tailor"}, "created_at": t1},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "wimpysworld/tailor"}, "created_at": t2},
				{"type": "ReleaseEvent", "repo": map[string]string{"name": "wimpysworld/sidra"}, "created_at": t3},
				{"type": "IssuesEvent", "repo": map[string]string{"name": "wimpysworld/nyala"}, "created_at": t1.Add(-time.Hour)},
			})
		case "/repos/wimpysworld/tailor":
			writeJSON(t, w, map[string]any{
				"full_name":   "wimpysworld/tailor",
				"html_url":    "https://github.com/wimpysworld/tailor",
				"description": "Tailor repository manager",
			})
		case "/repos/wimpysworld/sidra":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		now:        func() time.Time { return now },
	}
	got, err := client.FetchRecentContributions(context.Background(), "flexiondotorg", nil, 0, 2)
	if err != nil {
		t.Fatalf("FetchRecentContributions error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 contributions, got %d", len(got))
	}
	if got[0].RepoFullName != "wimpysworld/tailor" || !got[0].OccurredAt.Equal(t2) || got[0].Description != "Tailor repository manager" {
		t.Fatalf("unexpected first contribution: %+v", got[0])
	}
	if got[1].RepoFullName != "wimpysworld/sidra" || got[1].RepoURL != "https://github.com/wimpysworld/sidra" {
		t.Fatalf("unexpected second contribution: %+v", got[1])
	}
}

func TestFetchRecentContributionsExternalOnlyAndPRTopUp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	tEvent := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	tPRNew := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	tPROld := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	tooOld := now.AddDate(0, 0, -181)

	var searchQueries []string
	var gotSearchSort string
	var gotSearchPerPage string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/flexiondotorg/events/public":
			writeJSON(t, w, []map[string]any{
				{"type": "PushEvent", "repo": map[string]string{"name": "flexiondotorg/dotfiles"}, "created_at": tPRNew.Add(time.Hour)},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "WimpysWorld/tailor"}, "created_at": tPRNew},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "stale/event"}, "created_at": tooOld},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "upstream/one"}, "created_at": tEvent},
			})
		case "/search/issues":
			searchQueries = append(searchQueries, r.URL.Query().Get("q"))
			gotSearchSort = r.URL.Query().Get("sort")
			gotSearchPerPage = r.URL.Query().Get("per_page")
			writeJSON(t, w, map[string]any{
				"total_count": 5,
				"items": []map[string]any{
					{
						"title":          "Sneaky org PR",
						"html_url":       "https://github.com/WimpysWorld/sneaky/pull/1",
						"repository_url": "https://api.github.com/repos/WimpysWorld/sneaky",
						"updated_at":     tPRNew.Add(2 * time.Hour),
					},
					{
						"title":          "Stale PR beyond window",
						"html_url":       "https://github.com/external/stale/pull/3",
						"repository_url": "https://api.github.com/repos/external/stale",
						"updated_at":     tooOld,
					},
					{
						"title":          "External two",
						"html_url":       "https://github.com/external/two/pull/5",
						"repository_url": "https://api.github.com/repos/external/two",
						"updated_at":     tPRNew,
					},
					{
						"title":          "Older duplicate of upstream/one",
						"html_url":       "https://github.com/upstream/one/pull/2",
						"repository_url": "https://api.github.com/repos/upstream/one",
						"updated_at":     tPROld,
					},
					{
						"title":          "External three",
						"html_url":       "https://github.com/external/three/pull/9",
						"repository_url": "https://api.github.com/repos/external/three",
						"created_at":     tPROld,
					},
				},
			})
		case "/repos/external/two":
			writeJSON(t, w, map[string]any{
				"full_name":   "external/two",
				"html_url":    "https://github.com/external/two",
				"description": "External two description",
			})
		case "/repos/upstream/one":
			writeJSON(t, w, map[string]any{
				"full_name":   "upstream/one",
				"html_url":    "https://github.com/upstream/one",
				"description": "Upstream one description",
			})
		case "/repos/external/three":
			writeJSON(t, w, map[string]any{
				"full_name":   "external/three",
				"html_url":    "https://github.com/external/three",
				"description": "External three description",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		now:        func() time.Time { return now },
	}
	got, err := client.FetchRecentContributions(
		context.Background(),
		"flexiondotorg",
		[]string{"flexiondotorg", "wimpysworld", " ", "WimpysWorld"},
		180,
		3,
	)
	if err != nil {
		t.Fatalf("FetchRecentContributions error: %v", err)
	}
	if gotSearchSort != "updated" || gotSearchPerPage != "30" {
		t.Fatalf("unexpected search params: sort=%q per_page=%q", gotSearchSort, gotSearchPerPage)
	}
	expectedQuery := "author:flexiondotorg type:pr is:public updated:>=2026-05-28 -user:flexiondotorg -user:wimpysworld"
	if len(searchQueries) != 1 || searchQueries[0] != expectedQuery {
		t.Fatalf("expected single PR search query %q, got %v", expectedQuery, searchQueries)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 external contributions, got %d: %+v", len(got), got)
	}
	if got[0].RepoFullName != "external/two" || got[0].Description != "External two description" {
		t.Fatalf("unexpected first contribution: %+v", got[0])
	}
	if got[1].RepoFullName != "upstream/one" || !got[1].OccurredAt.Equal(tEvent) || got[1].Description != "Upstream one description" {
		t.Fatalf("unexpected second contribution: %+v", got[1])
	}
	if got[2].RepoFullName != "external/three" || !got[2].OccurredAt.Equal(tPROld) || got[2].Description != "External three description" {
		t.Fatalf("unexpected third contribution: %+v", got[2])
	}
}

func TestFetchRecentContributionsInvolvesTopUpAndCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	tEvent := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	tPR := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	tInvolvedNewerDup := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	tInvolvedIssue := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	tooOld := now.AddDate(0, 0, -91)

	var searchQueries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/flexiondotorg/events/public":
			writeJSON(t, w, []map[string]any{
				{"type": "PushEvent", "repo": map[string]string{"name": "stale/event"}, "created_at": tooOld},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "upstream/one"}, "created_at": tEvent},
			})
		case "/search/issues":
			q := r.URL.Query().Get("q")
			searchQueries = append(searchQueries, q)
			switch {
			case strings.HasPrefix(q, "author:flexiondotorg type:pr"):
				writeJSON(t, w, map[string]any{
					"total_count": 2,
					"items": []map[string]any{
						{
							"title":          "Stale PR",
							"html_url":       "https://github.com/stale/pr/pull/1",
							"repository_url": "https://api.github.com/repos/stale/pr",
							"updated_at":     tooOld,
						},
						{
							"title":          "Recent PR",
							"html_url":       "https://github.com/external/pr-repo/pull/2",
							"repository_url": "https://api.github.com/repos/external/pr-repo",
							"updated_at":     tPR,
						},
					},
				})
			case strings.HasPrefix(q, "involves:flexiondotorg is:public"):
				writeJSON(t, w, map[string]any{
					"total_count": 4,
					"items": []map[string]any{
						{
							"title":          "Excluded org issue",
							"html_url":       "https://github.com/WimpysWorld/owned/issues/1",
							"repository_url": "https://api.github.com/repos/WimpysWorld/owned",
							"updated_at":     tInvolvedNewerDup.Add(time.Hour),
						},
						{
							"title":          "Stale involved issue",
							"html_url":       "https://github.com/stale/issue/issues/2",
							"repository_url": "https://api.github.com/repos/stale/issue",
							"updated_at":     tooOld,
						},
						{
							"title":          "Newer activity on upstream/one",
							"html_url":       "https://github.com/Upstream/One/issues/9",
							"repository_url": "https://api.github.com/repos/Upstream/One",
							"updated_at":     tInvolvedNewerDup,
						},
						{
							"title":          "Community discussion",
							"html_url":       "https://github.com/community/issue-repo/issues/4",
							"repository_url": "https://api.github.com/repos/community/issue-repo",
							"updated_at":     tInvolvedIssue,
						},
					},
				})
			default:
				http.Error(w, "unexpected search query: "+q, http.StatusBadRequest)
			}
		case "/repos/Upstream/One":
			writeJSON(t, w, map[string]any{
				"full_name":   "Upstream/One",
				"html_url":    "https://github.com/Upstream/One",
				"description": "Upstream one description",
			})
		case "/repos/external/pr-repo":
			writeJSON(t, w, map[string]any{
				"full_name":   "external/pr-repo",
				"html_url":    "https://github.com/external/pr-repo",
				"description": "External PR repo",
			})
		case "/repos/community/issue-repo":
			writeJSON(t, w, map[string]any{
				"full_name":   "community/issue-repo",
				"html_url":    "https://github.com/community/issue-repo",
				"description": "Community issue repo",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		now:        func() time.Time { return now },
	}
	got, err := client.FetchRecentContributions(
		context.Background(),
		"flexiondotorg",
		[]string{"flexiondotorg", "wimpysworld"},
		90,
		3,
	)
	if err != nil {
		t.Fatalf("FetchRecentContributions error: %v", err)
	}

	wantPRQuery := "author:flexiondotorg type:pr is:public updated:>=2026-08-26 -user:flexiondotorg -user:wimpysworld"
	wantInvolvesQuery := "involves:flexiondotorg is:public updated:>=2026-08-26 -user:flexiondotorg -user:wimpysworld"
	if len(searchQueries) != 2 || searchQueries[0] != wantPRQuery || searchQueries[1] != wantInvolvesQuery {
		t.Fatalf("unexpected search queries:\ngot:  %v\nwant: [%q %q]", searchQueries, wantPRQuery, wantInvolvesQuery)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 contributions, got %d: %+v", len(got), got)
	}
	if got[0].RepoFullName != "Upstream/One" || !got[0].OccurredAt.Equal(tInvolvedNewerDup) || got[0].Description != "Upstream one description" {
		t.Fatalf("unexpected first contribution: %+v", got[0])
	}
	if got[1].RepoFullName != "external/pr-repo" || !got[1].OccurredAt.Equal(tPR) || got[1].Description != "External PR repo" {
		t.Fatalf("unexpected second contribution: %+v", got[1])
	}
	if got[2].RepoFullName != "community/issue-repo" || !got[2].OccurredAt.Equal(tInvolvedIssue) || got[2].Description != "Community issue repo" {
		t.Fatalf("unexpected third contribution: %+v", got[2])
	}
}

func TestFetchRecentPullRequestsAndStarred(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC)
	merged := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	starredAt := time.Date(2026, 11, 22, 8, 30, 0, 0, time.UTC)
	var gotStarAccept string
	var gotStarPerPages []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			writeJSON(t, w, map[string]any{
				"total_count": 2,
				"items": []map[string]any{
					{
						"title":          "Add vitrine",
						"html_url":       "https://github.com/wimpysworld/tailor/pull/12",
						"repository_url": "https://api.github.com/repos/wimpysworld/tailor",
						"state":          "closed",
						"created_at":     created,
						"pull_request":   map[string]any{"merged_at": merged},
					},
					{
						"title":          "Open PR",
						"html_url":       "https://github.com/wimpysworld/nyala/pull/4",
						"repository_url": "",
						"state":          "open",
						"created_at":     created.Add(-time.Hour),
					},
				},
			})
		case "/users/flexiondotorg/starred":
			gotStarAccept = r.Header.Get("Accept")
			gotStarPerPages = append(gotStarPerPages, r.URL.Query().Get("per_page"))
			writeJSON(t, w, []map[string]any{
				{
					"starred_at": starredAt.Add(2 * time.Hour),
					"repo": map[string]any{
						"full_name":        "flexiondotorg/dotfiles",
						"html_url":         "https://github.com/flexiondotorg/dotfiles",
						"stargazers_count": 42,
					},
				},
				{
					"starred_at": starredAt.Add(time.Hour),
					"repo": map[string]any{
						"full_name":        "WimpysWorld/tailor",
						"owner":            map[string]string{"login": "WimpysWorld"},
						"html_url":         "https://github.com/WimpysWorld/tailor",
						"stargazers_count": 99,
					},
				},
				{
					"starred_at": starredAt,
					"repo": map[string]any{
						"full_name":        "golang/go",
						"owner":            map[string]string{"login": "golang"},
						"html_url":         "https://github.com/golang/go",
						"description":      "The Go programming language",
						"stargazers_count": 125000,
					},
				},
				{
					"starred_at": starredAt.Add(-time.Hour),
					"repo": map[string]any{
						"full_name":        "charmbracelet/bubbletea",
						"owner":            map[string]string{"login": "charmbracelet"},
						"html_url":         "https://github.com/charmbracelet/bubbletea",
						"description":      "Go TUI framework",
						"stargazers_count": 30000,
					},
				},
				{
					"starred_at": starredAt.Add(-2 * time.Hour),
					"repo": map[string]any{
						"full_name":        "neovim/neovim",
						"owner":            map[string]string{"login": "neovim"},
						"html_url":         "https://github.com/neovim/neovim",
						"description":      "Vim-fork",
						"stargazers_count": 80000,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{HTTPClient: srv.Client(), APIBaseURL: srv.URL}

	prs, err := client.FetchRecentPullRequests(context.Background(), "flexiondotorg", 5)
	if err != nil {
		t.Fatalf("FetchRecentPullRequests error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].State != "merged" || prs[0].RepoFullName != "wimpysworld/tailor" {
		t.Fatalf("unexpected first PR: %+v", prs[0])
	}
	if prs[1].State != "open" || prs[1].RepoFullName != "wimpysworld/nyala" {
		t.Fatalf("unexpected second PR: %+v", prs[1])
	}

	starred, err := client.FetchRecentStarred(
		context.Background(),
		"flexiondotorg",
		[]string{"flexiondotorg", "wimpysworld"},
		2,
	)
	if err != nil {
		t.Fatalf("FetchRecentStarred error: %v", err)
	}
	if gotStarAccept != acceptStarJSON {
		t.Fatalf("expected star accept header %q, got %q", acceptStarJSON, gotStarAccept)
	}
	if len(starred) != 2 || starred[0].RepoFullName != "golang/go" || !starred[0].StarredAt.Equal(starredAt) || starred[1].RepoFullName != "charmbracelet/bubbletea" {
		t.Fatalf("unexpected starred repos: %+v", starred)
	}

	// Verify per_page scales with limit*5 and caps at 100.
	if _, err := client.FetchRecentStarred(context.Background(), "flexiondotorg", nil, 25); err != nil {
		t.Fatalf("FetchRecentStarred large limit error: %v", err)
	}
	if len(gotStarPerPages) != 2 || gotStarPerPages[0] != "50" || gotStarPerPages[1] != "100" {
		t.Fatalf("expected starred per_page [50 100], got %v", gotStarPerPages)
	}
}

func TestFetchRepoMetaWithAndWithoutRelease(t *testing.T) {
	t.Parallel()

	relTime := time.Date(2026, 11, 22, 15, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/wimpysworld/tailor":
			writeJSON(t, w, map[string]any{
				"full_name":        "wimpysworld/tailor",
				"name":             "tailor",
				"owner":            map[string]string{"login": "wimpysworld"},
				"description":      "Repo health tool",
				"homepage":         "https://tailor.wimpys.world/",
				"html_url":         "https://github.com/wimpysworld/tailor",
				"default_branch":   "main",
				"language":         "Go",
				"stargazers_count": 77,
			})
		case "/repos/wimpysworld/tailor/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name":     "v0.6.6",
				"html_url":     "https://github.com/wimpysworld/tailor/releases/tag/v0.6.6",
				"published_at": relTime,
			})
		case "/repos/wimpysworld/unreleased":
			writeJSON(t, w, map[string]any{
				"full_name":        "wimpysworld/unreleased",
				"name":             "unreleased",
				"owner":            map[string]string{"login": "wimpysworld"},
				"default_branch":   "main",
				"stargazers_count": 3,
			})
		case "/repos/wimpysworld/unreleased/releases/latest":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{HTTPClient: srv.Client(), APIBaseURL: srv.URL}

	meta, err := client.FetchRepoMeta(context.Background(), "wimpysworld/tailor")
	if err != nil {
		t.Fatalf("FetchRepoMeta error: %v", err)
	}
	if meta.Stars != 77 || meta.Language != "Go" || meta.LatestRelease == nil || meta.LatestRelease.TagName != "v0.6.6" {
		t.Fatalf("unexpected repo meta: %+v", meta)
	}

	unreleased, err := client.FetchRepoMeta(context.Background(), "wimpysworld/unreleased")
	if err != nil {
		t.Fatalf("expected 404 release to be non-fatal, got error: %v", err)
	}
	if unreleased.LatestRelease != nil {
		t.Fatalf("expected nil LatestRelease, got %+v", unreleased.LatestRelease)
	}
}

func TestFetchSponsorsPartialActivePastAndFallback(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			http.Error(w, "missing X-Requested-With header", http.StatusBadRequest)
			return
		}
		filter := r.URL.Query().Get("filter")
		switch r.URL.Path {
		case "/sponsors/flexiondotorg/sponsors_partial":
			switch filter {
			case "active":
				_, _ = w.Write([]byte(`
					<div>
						<img alt="@flexiondotorg" />
						<img alt="@alice" />
						<img alt="@bob-42" />
						<img alt="@Alice" />
					</div>
				`))
			case "inactive":
				_, _ = w.Write([]byte(`
					<div>
						<img alt="@flexiondotorg" />
						<img alt="@alice" />
						<img alt="@charlie" />
						<img alt="@dana" />
						<img alt="@Charlie" />
					</div>
				`))
			default:
				http.NotFound(w, r)
			}
		case "/sponsors/emptyuser/sponsors_partial":
			switch filter {
			case "active":
				_, _ = w.Write([]byte(`<div></div>`))
			case "inactive":
				_, _ = w.Write([]byte(`<img alt="@former-supporter" />`))
			default:
				http.NotFound(w, r)
			}
		case "/sponsors/broken/sponsors_partial":
			switch filter {
			case "active":
				http.Error(w, "upstream failure", http.StatusBadGateway)
			case "inactive":
				_, _ = w.Write([]byte(`<img alt="@fallback-user" /><img alt="@past-only" />`))
			default:
				http.NotFound(w, r)
			}
		case "/sponsors/inactivebroken/sponsors_partial":
			switch filter {
			case "active":
				_, _ = w.Write([]byte(`<img alt="@active-only" />`))
			case "inactive":
				http.Error(w, "upstream failure", http.StatusBadGateway)
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{HTTPClient: srv.Client(), WebBaseURL: srv.URL}
	fallback := []Sponsor{{Login: "fallback-user", URL: "https://github.com/fallback-user"}}

	active, past, err := client.FetchSponsors(context.Background(), "flexiondotorg", fallback)
	if err != nil {
		t.Fatalf("FetchSponsors error: %v", err)
	}
	if len(active) != 2 || active[0].Login != "alice" || active[1].Login != "bob-42" {
		t.Fatalf("unexpected active sponsors: %+v", active)
	}
	if len(past) != 2 || past[0].Login != "charlie" || past[1].Login != "dana" {
		t.Fatalf("unexpected past sponsors: %+v", past)
	}

	emptyActive, emptyPast, err := client.FetchSponsors(context.Background(), "emptyuser", fallback)
	if err != nil {
		t.Fatalf("expected HTTP 200 empty active sponsors to succeed, got error: %v", err)
	}
	if len(emptyActive) != 0 {
		t.Fatalf("expected 0 active sponsors (no fallback on HTTP 200), got %+v", emptyActive)
	}
	if len(emptyPast) != 1 || emptyPast[0].Login != "former-supporter" {
		t.Fatalf("unexpected past sponsors for emptyuser: %+v", emptyPast)
	}

	errActive, errPast, err := client.FetchSponsors(context.Background(), "broken", fallback)
	if err != nil {
		t.Fatalf("expected fallback on HTTP 502, got error: %v", err)
	}
	if len(errActive) != 1 || errActive[0].Login != "fallback-user" {
		t.Fatalf("unexpected fallback active sponsors on error: %+v", errActive)
	}
	if len(errPast) != 1 || errPast[0].Login != "past-only" {
		t.Fatalf("unexpected past sponsors with fallback active dedup: %+v", errPast)
	}

	if _, _, err := client.FetchSponsors(context.Background(), "broken", nil); err == nil {
		t.Fatal("expected error on HTTP 502 with no fallback")
	}

	okActive, nilPast, err := client.FetchSponsors(context.Background(), "inactivebroken", fallback)
	if err != nil {
		t.Fatalf("expected inactive failure to degrade gracefully, got error: %v", err)
	}
	if len(okActive) != 1 || okActive[0].Login != "active-only" || len(nilPast) != 0 {
		t.Fatalf("unexpected results when inactive partial fails: active=%+v past=%+v", okActive, nilPast)
	}
}

func TestFetchRawFileNotFoundAndSizeCap(t *testing.T) {
	t.Parallel()

	oversize := bytes.Repeat([]byte("a"), maxRawFileBytes+10)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wimpysworld/tailor/main/pages/icon.svg":
			_, _ = w.Write([]byte(`<svg viewBox="0 0 10 10"></svg>`))
		case "/wimpysworld/tailor/main/huge.bin":
			_, _ = w.Write(oversize)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{HTTPClient: srv.Client(), RawBaseURL: srv.URL}

	data, err := client.FetchRawFile(context.Background(), "wimpysworld", "tailor", "main", "pages/icon.svg")
	if err != nil {
		t.Fatalf("FetchRawFile error: %v", err)
	}
	if string(data) != `<svg viewBox="0 0 10 10"></svg>` {
		t.Fatalf("unexpected raw file contents: %q", string(data))
	}

	_, err = client.FetchRawFile(context.Background(), "wimpysworld", "tailor", "main", "missing.svg")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	_, err = client.FetchRawFile(context.Background(), "wimpysworld", "tailor", "main", "huge.bin")
	if err == nil {
		t.Fatal("expected size limit error for huge file")
	}

	_, err = client.FetchRawFile(context.Background(), "wimpysworld", "tailor", "main", "../secret")
	if err == nil {
		t.Fatal("expected error for path traversal segment")
	}
}

func TestValidationAndLinkOriginGuard(t *testing.T) {
	t.Parallel()

	c := NewClient("tok")
	if c.APIBaseURL != DefaultAPIBaseURL || c.RawBaseURL != DefaultRawBaseURL || c.WebBaseURL != DefaultWebBaseURL {
		t.Fatalf("unexpected defaults from NewClient: %+v", c)
	}

	if _, err := c.FetchRepoMeta(context.Background(), "not-a-repo"); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("expected ErrInvalidRepo, got %v", err)
	}

	if next := parseNextLink(`<https://evil.example/repos?page=2>; rel="next"`, DefaultAPIBaseURL); next != "" {
		t.Fatalf("expected off-origin next link to be rejected, got %q", next)
	}
	if next := parseNextLink(`<https://api.github.com/users/x/repos?page=2>; rel="next"`, DefaultAPIBaseURL); next != "https://api.github.com/users/x/repos?page=2" {
		t.Fatalf("expected same-origin next link, got %q", next)
	}
}

func TestFetchProfileOrchestration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	older := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/flexiondotorg":
			writeJSON(t, w, map[string]any{"login": "flexiondotorg", "followers": 10, "public_repos": 1})
		case "/users/flexiondotorg/repos":
			writeJSON(t, w, []map[string]any{
				{
					"full_name":        "flexiondotorg/alpha",
					"html_url":         "https://github.com/flexiondotorg/alpha",
					"description":      "Alpha",
					"stargazers_count": 5,
					"pushed_at":        older,
				},
			})
		case "/orgs/wimpysworld/repos":
			writeJSON(t, w, []map[string]any{
				{
					"full_name":        "wimpysworld/tailor",
					"html_url":         "https://github.com/wimpysworld/tailor",
					"description":      "Tailor",
					"stargazers_count": 15,
					"pushed_at":        newer,
				},
			})
		case "/search/commits":
			writeJSON(t, w, map[string]any{"total_count": 250})
		case "/search/issues":
			q := r.URL.Query().Get("q")
			switch {
			case strings.Contains(q, "type:issue"):
				writeJSON(t, w, map[string]any{"total_count": 4, "items": []any{}})
			case strings.Contains(q, "-user:"):
				writeJSON(t, w, map[string]any{
					"total_count": 1,
					"items": []map[string]any{
						{
							"title":          "Community fix",
							"html_url":       "https://github.com/community/project/pull/7",
							"repository_url": "https://api.github.com/repos/community/project",
							"updated_at":     older,
						},
					},
				})
			default:
				writeJSON(t, w, map[string]any{
					"total_count": 9,
					"items": []map[string]any{
						{
							"title":          "Ship it",
							"html_url":       "https://github.com/wimpysworld/tailor/pull/1",
							"repository_url": "https://api.github.com/repos/wimpysworld/tailor",
							"state":          "open",
							"created_at":     newer,
						},
					},
				})
			}
		case "/users/flexiondotorg/events/public":
			writeJSON(t, w, []map[string]any{
				{"type": "PushEvent", "repo": map[string]string{"name": "wimpysworld/tailor"}, "created_at": newer},
				{"type": "PullRequestEvent", "repo": map[string]string{"name": "external/tool"}, "created_at": newer},
			})
		case "/repos/external/tool":
			writeJSON(t, w, map[string]any{
				"full_name":   "external/tool",
				"html_url":    "https://github.com/external/tool",
				"description": "External tool",
			})
		case "/repos/community/project":
			writeJSON(t, w, map[string]any{
				"full_name":   "community/project",
				"html_url":    "https://github.com/community/project",
				"description": "Community project",
			})
		case "/users/flexiondotorg/starred":
			writeJSON(t, w, []map[string]any{
				{
					"starred_at": newer,
					"repo": map[string]any{
						"full_name":        "wimpysworld/nyala",
						"html_url":         "https://github.com/wimpysworld/nyala",
						"stargazers_count": 9,
					},
				},
				{
					"starred_at": older,
					"repo": map[string]any{
						"full_name":        "golang/go",
						"html_url":         "https://github.com/golang/go",
						"stargazers_count": 125000,
					},
				},
			})
		case "/sponsors/flexiondotorg/sponsors_partial":
			if r.URL.Query().Get("filter") == "inactive" {
				_, _ = w.Write([]byte(`<img alt="@pastfriend" />`))
				return
			}
			_, _ = w.Write([]byte(`<img alt="@friend" />`))
		case "/repos/flexiondotorg/alpha/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name":     "v0.1.0",
				"html_url":     "https://github.com/flexiondotorg/alpha/releases/tag/v0.1.0",
				"published_at": older,
			})
		case "/repos/wimpysworld/tailor/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name":     "v0.6.6",
				"html_url":     "https://github.com/wimpysworld/tailor/releases/tag/v0.6.6",
				"published_at": newer,
			})
		case "/repos/linuxmatters/jive-vocals":
			writeJSON(t, w, map[string]any{
				"full_name":        "linuxmatters/jive-vocals",
				"html_url":         "https://github.com/linuxmatters/jive-vocals",
				"description":      "Vocal separation",
				"stargazers_count": 12,
			})
		case "/repos/linuxmatters/jive-vocals/releases/latest":
			writeJSON(t, w, map[string]any{
				"tag_name":     "v0.7.1",
				"html_url":     "https://github.com/linuxmatters/jive-vocals/releases/tag/v0.7.1",
				"published_at": older.Add(24 * time.Hour),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := &Client{
		HTTPClient: srv.Client(),
		APIBaseURL: srv.URL,
		WebBaseURL: srv.URL,
		now:        func() time.Time { return now },
	}

	profile, err := client.FetchProfile(
		context.Background(),
		"flexiondotorg",
		ProfileOptions{
			Orgs:                    []string{"wimpysworld"},
			MaxItems:                2,
			ContributionsWindowDays: 180,
			ExtraReleaseRepos:       []string{"linuxmatters/jive-vocals", "wimpysworld/tailor"},
			ExcludeOwners:           []string{"flexiondotorg", "wimpysworld"},
		},
	)
	if err != nil {
		t.Fatalf("FetchProfile error: %v", err)
	}
	if profile.Stats.TotalStars != 32 || profile.Stats.TotalCommits != 250 || profile.Stats.PublicPRs != 9 || profile.Stats.PublicIssues != 4 || profile.Stats.Followers != 10 {
		t.Fatalf("unexpected 5-metric stats: %+v", profile.Stats)
	}
	if profile.SponsorCount != 1 || len(profile.Sponsors) != 1 || profile.Sponsors[0].Login != "friend" {
		t.Fatalf("unexpected active sponsors: sponsors=%+v count=%d", profile.Sponsors, profile.SponsorCount)
	}
	if len(profile.PastSponsors) != 1 || profile.PastSponsors[0].Login != "pastfriend" {
		t.Fatalf("unexpected past sponsors: %+v", profile.PastSponsors)
	}
	if len(profile.Contributions) != 2 || profile.Contributions[0].RepoFullName != "external/tool" || profile.Contributions[1].RepoFullName != "community/project" {
		t.Fatalf("unexpected external contributions: %+v", profile.Contributions)
	}
	if len(profile.Starred) != 1 || profile.Starred[0].RepoFullName != "golang/go" {
		t.Fatalf("unexpected owner-filtered starred repos: %+v", profile.Starred)
	}
	if len(profile.Releases) != 2 {
		t.Fatalf("expected 2 releases (capped at maxItems), got %d", len(profile.Releases))
	}
	if profile.Releases[0].RepoFullName != "wimpysworld/tailor" || profile.Releases[1].RepoFullName != "linuxmatters/jive-vocals" {
		t.Fatalf("expected releases sorted descending by PublishedAt, got %+v", profile.Releases)
	}
}
