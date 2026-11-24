package site

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flexiondotorg/flexiondotorg/internal/config"
)

func TestSocialFetcherMastodonGoToSocialAndBluesky(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/accounts/lookup":
			switch r.URL.Query().Get("acct") {
			case "masto":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"followers_count": 1200,
					"statuses_count":  450,
				})
			default:
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}
		case "/@gts":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><body>
				<div class="stats-item"><dt class="postsdt">Posts</dt><dd class="postsdd text-cutoff">327</dd></div>
				<div class="stats-item"><dt class="followeddt">Followed by</dt><dd class="followeddd text-cutoff">3,282</dd></div>
			</body></html>`))
		case "/xrpc/app.bsky.actor.getProfile":
			if r.URL.Query().Get("actor") == "wimpys.world" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"handle":         "wimpys.world",
					"followersCount": 821,
					"postsCount":     144,
				})
				return
			}
			http.NotFound(w, r)
		case "/twitch/followcount/WimpysWorld":
			_, _ = w.Write([]byte("1533"))
		case "/WimpysWorld":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><script>var ytInitialData = {"header":{"pageHeaderRenderer":{"content":{"pageHeaderViewModel":{"metadata":{"contentMetadataViewModel":{"metadataRows":[{"metadataParts":[{"text":{"content":"@WimpysWorld"}}]},{"metadataParts":[{"text":{"content":"4.48k subscribers"}},{"text":{"content":"144 videos"}}]}]}}}}}}};</script></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	fetcher := &SocialFetcher{
		HTTPClient:        srv.Client(),
		BlueskyAPIBaseURL: srv.URL,
		DecapiBaseURL:     srv.URL,
	}

	links := []config.SiteLink{
		{Title: "Mastodon API", Slug: "masto-api", Icon: "mastodon", URL: srv.URL + "/@masto"},
		{Title: "GoToSocial", Slug: "mastodon", Icon: "mastodon", URL: srv.URL + "/@gts"},
		{Title: "Bluesky", Slug: "bluesky", Icon: "bluesky", URL: "https://bsky.app/profile/wimpys.world"},
		{Title: "Twitch", Slug: "twitch", Icon: "twitch", URL: "https://twitch.tv/WimpysWorld"},
		{Title: "YouTube", Slug: "youtube", Icon: "youtube", URL: srv.URL + "/WimpysWorld"},
		{Title: "Broken Bluesky", Slug: "broken-bsky", Icon: "bluesky", URL: "https://bsky.app/profile/missing.invalid"},
		{Title: "Static Link", Slug: "oval-tutu", Icon: "globe", URL: "https://oval-tutu.com"},
	}

	got, errs := fetcher.FetchAll(context.Background(), links)
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1 (%v)", len(errs), errs)
	}
	if got["masto-api"] != (SocialProfile{Followers: 1200, Posts: 450}) {
		t.Errorf("masto-api = %+v, want {1200 450}", got["masto-api"])
	}
	if got["mastodon"] != (SocialProfile{Followers: 3282, Posts: 327}) {
		t.Errorf("mastodon (GoToSocial fallback) = %+v, want {3282 327}", got["mastodon"])
	}
	if got["bluesky"] != (SocialProfile{Followers: 821, Posts: 144}) {
		t.Errorf("bluesky = %+v, want {821 144}", got["bluesky"])
	}
	if got["twitch"] != (SocialProfile{Followers: 1533}) {
		t.Errorf("twitch = %+v, want {Followers:1533}", got["twitch"])
	}
	if got["youtube"] != (SocialProfile{Subscribers: "4.48K", Videos: "144"}) {
		t.Errorf("youtube = %+v, want {Subscribers:4.48K Videos:144}", got["youtube"])
	}
	if _, exists := got["oval-tutu"]; exists {
		t.Errorf("unexpected social stats for non-social link: %+v", got["oval-tutu"])
	}
}
