// Package twitch checks live and offline Twitch channel status without API secrets.
package twitch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultPrimaryBaseURL is the default decapi.me uptime endpoint.
	DefaultPrimaryBaseURL = "https://decapi.me/twitch/uptime"
	// DefaultFallbackBaseURL is the default Shields JSON status endpoint.
	DefaultFallbackBaseURL = "https://img.shields.io/twitch/status"

	defaultHTTPTimeout = 10 * time.Second
	maxBodyBytes       = 64 << 10 // 64 KiB
	userAgent          = "vitrine/1.0"
	sourcePrimary      = "decapi"
	sourceFallback     = "shields"
)

var decapiErrorSubstrings = []string{
	"error",
	"no user",
	"could not",
	"not found",
}

// Status describes the live or offline state of a Twitch channel.
type Status struct {
	Channel string
	Live    bool
	Uptime  string
	URL     string
	Source  string
}

// Client checks Twitch channel status using decapi.me with a Shields JSON fallback.
type Client struct {
	HTTPClient      *http.Client
	PrimaryBaseURL  string
	FallbackBaseURL string
}

type shieldsPayload struct {
	Value   string `json:"value"`
	Message string `json:"message"`
}

// NewClient creates a Client with default primary and fallback endpoints.
func NewClient(httpClient *http.Client) *Client {
	return &Client{
		HTTPClient:      httpClient,
		PrimaryBaseURL:  DefaultPrimaryBaseURL,
		FallbackBaseURL: DefaultFallbackBaseURL,
	}
}

// Fetch returns the current live or offline Status for channel.
func (c *Client) Fetch(ctx context.Context, channel string) (Status, error) {
	ch := strings.TrimSpace(channel)
	if ch == "" {
		return Status{}, errors.New("twitch channel must not be empty")
	}

	status, primaryErr := c.fetchPrimary(ctx, ch)
	if primaryErr == nil {
		return status, nil
	}

	fallbackStatus, fallbackErr := c.fetchFallback(ctx, ch)
	if fallbackErr == nil {
		return fallbackStatus, nil
	}

	return Status{}, fmt.Errorf("fetch twitch status for %q failed: primary: %w; fallback: %w", ch, primaryErr, fallbackErr)
}

func (c *Client) fetchPrimary(ctx context.Context, channel string) (Status, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.PrimaryBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultPrimaryBaseURL
	}
	endpoint := baseURL + "/" + url.PathEscape(channel)

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Status{}, err
	}

	text := strings.TrimSpace(string(body))
	if text == "" {
		return Status{}, errors.New("empty response from primary endpoint")
	}

	lower := strings.ToLower(text)
	for _, needle := range decapiErrorSubstrings {
		if strings.Contains(lower, needle) {
			return Status{}, fmt.Errorf("primary endpoint error response: %q", text)
		}
	}

	channelURL := "https://www.twitch.tv/" + url.PathEscape(channel)
	if strings.Contains(lower, "offline") {
		return Status{
			Channel: channel,
			Live:    false,
			URL:     channelURL,
			Source:  sourcePrimary,
		}, nil
	}

	return Status{
		Channel: channel,
		Live:    true,
		Uptime:  text,
		URL:     channelURL,
		Source:  sourcePrimary,
	}, nil
}

func (c *Client) fetchFallback(ctx context.Context, channel string) (Status, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.FallbackBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultFallbackBaseURL
	}
	endpoint := baseURL + "/" + url.PathEscape(channel) + ".json"

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Status{}, err
	}

	var payload shieldsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Status{}, fmt.Errorf("decode shields json: %w", err)
	}

	val := strings.ToLower(strings.TrimSpace(payload.Value))
	msg := strings.ToLower(strings.TrimSpace(payload.Message))
	if val == "" && msg == "" {
		return Status{}, errors.New("shields json missing value and message")
	}

	live := val == "live" || msg == "live"
	return Status{
		Channel: channel,
		Live:    live,
		URL:     "https://www.twitch.tv/" + url.PathEscape(channel),
		Source:  sourceFallback,
	}, nil
}

func (c *Client) doGet(ctx context.Context, endpoint string) ([]byte, error) {
	if err := validateHTTPURL(endpoint); err != nil {
		return nil, err
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	// #nosec G704 -- endpoint URL is validated to http/https from configured base URL.
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBodyBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxBodyBytes)
	}
	return body, nil
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return errors.New("url host must not be empty")
	}
	return nil
}
