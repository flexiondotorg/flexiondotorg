// Package discord fetches live Discord server online and member counts without API secrets.
package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultInviteBaseURL is the default Discord public invite API endpoint.
	DefaultInviteBaseURL = "https://discord.com/api/v9/invites"
	// DefaultWidgetBaseURL is the default Discord server widget JSON endpoint.
	DefaultWidgetBaseURL = "https://discord.com/api/guilds"
	// DefaultShieldsBaseURL is the default Shields.io Discord JSON endpoint.
	DefaultShieldsBaseURL = "https://img.shields.io/discord"

	defaultHTTPTimeout = 10 * time.Second
	maxBodyBytes       = 1 << 20 // 1 MiB
	userAgent          = "vitrine"
	sourceInvite       = "invite"
	sourceWidget       = "widget"
	sourceShields      = "shields"
)

// Status describes the live online and member counts of a Discord server.
type Status struct {
	Name    string
	Online  int
	Members int
	URL     string
	Source  string
}

// Client fetches Discord server counts using public endpoints without API secrets.
type Client struct {
	HTTPClient     *http.Client
	InviteBaseURL  string
	WidgetBaseURL  string
	ShieldsBaseURL string
}

type invitePayload struct {
	Guild struct {
		Name string `json:"name"`
	} `json:"guild"`
	ApproximateMemberCount   *int `json:"approximate_member_count"`
	ApproximatePresenceCount *int `json:"approximate_presence_count"`
}

type widgetPayload struct {
	Name          string `json:"name"`
	InstantInvite string `json:"instant_invite"`
	PresenceCount *int   `json:"presence_count"`
}

type shieldsPayload struct {
	Value   string `json:"value"`
	Message string `json:"message"`
}

// NewClient creates a Client with default invite, widget, and Shields endpoints.
func NewClient(httpClient *http.Client) *Client {
	return &Client{
		HTTPClient:     httpClient,
		InviteBaseURL:  DefaultInviteBaseURL,
		WidgetBaseURL:  DefaultWidgetBaseURL,
		ShieldsBaseURL: DefaultShieldsBaseURL,
	}
}

// Fetch returns live Discord server counts using the invite API with widget and Shields fallbacks.
func (c *Client) Fetch(ctx context.Context, inviteCode, guildID, joinURL string) (Status, error) {
	code := strings.TrimSpace(inviteCode)
	guild := strings.TrimSpace(guildID)
	resolvedURL := strings.TrimSpace(joinURL)
	if code == "" && guild == "" {
		return Status{}, errors.New("discord invite code or guild ID must not be empty")
	}
	if resolvedURL == "" && code != "" {
		resolvedURL = "https://discord.gg/" + url.PathEscape(code)
	}

	var errs []error
	if code != "" {
		status, err := c.fetchInvite(ctx, code, resolvedURL)
		if err == nil {
			return status, nil
		}
		errs = append(errs, fmt.Errorf("invite: %w", err))
	}

	if guild != "" {
		status, err := c.fetchWidget(ctx, guild, resolvedURL)
		if err == nil {
			return status, nil
		}
		errs = append(errs, fmt.Errorf("widget: %w", err))

		status, err = c.fetchShields(ctx, guild, resolvedURL)
		if err == nil {
			return status, nil
		}
		errs = append(errs, fmt.Errorf("shields: %w", err))
	}

	return Status{}, fmt.Errorf("fetch discord status failed: %w", errors.Join(errs...))
}

func (c *Client) fetchInvite(ctx context.Context, inviteCode, joinURL string) (Status, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.InviteBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultInviteBaseURL
	}
	endpoint := baseURL + "/" + url.PathEscape(inviteCode) + "?with_counts=true"

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Status{}, err
	}

	var payload invitePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Status{}, fmt.Errorf("decode invite json: %w", err)
	}
	if payload.ApproximatePresenceCount == nil && payload.ApproximateMemberCount == nil {
		return Status{}, errors.New("invite json missing counts")
	}

	var online, members int
	if payload.ApproximatePresenceCount != nil {
		online = *payload.ApproximatePresenceCount
	}
	if payload.ApproximateMemberCount != nil {
		members = *payload.ApproximateMemberCount
	}
	if online < 0 || members < 0 {
		return Status{}, errors.New("invite json contains negative counts")
	}

	return Status{
		Name:    strings.TrimSpace(payload.Guild.Name),
		Online:  online,
		Members: members,
		URL:     joinURL,
		Source:  sourceInvite,
	}, nil
}

func (c *Client) fetchWidget(ctx context.Context, guildID, joinURL string) (Status, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.WidgetBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultWidgetBaseURL
	}
	endpoint := baseURL + "/" + url.PathEscape(guildID) + "/widget.json"

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Status{}, err
	}

	var payload widgetPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Status{}, fmt.Errorf("decode widget json: %w", err)
	}
	if payload.PresenceCount == nil {
		return Status{}, errors.New("widget json missing presence_count")
	}
	if *payload.PresenceCount < 0 {
		return Status{}, errors.New("widget json contains negative presence_count")
	}

	resolvedURL := joinURL
	if resolvedURL == "" {
		resolvedURL = strings.TrimSpace(payload.InstantInvite)
	}

	return Status{
		Name:   strings.TrimSpace(payload.Name),
		Online: *payload.PresenceCount,
		URL:    resolvedURL,
		Source: sourceWidget,
	}, nil
}

func (c *Client) fetchShields(ctx context.Context, guildID, joinURL string) (Status, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.ShieldsBaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultShieldsBaseURL
	}
	endpoint := baseURL + "/" + url.PathEscape(guildID) + ".json"

	body, err := c.doGet(ctx, endpoint)
	if err != nil {
		return Status{}, err
	}

	var payload shieldsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return Status{}, fmt.Errorf("decode shields json: %w", err)
	}

	online, err := parseShieldsOnline(payload)
	if err != nil {
		return Status{}, err
	}

	return Status{
		Online: online,
		URL:    joinURL,
		Source: sourceShields,
	}, nil
}

func parseShieldsOnline(payload shieldsPayload) (int, error) {
	for _, raw := range []string{payload.Value, payload.Message} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if n, err := parseLeadingInt(raw); err == nil {
			return n, nil
		}
	}
	return 0, errors.New("shields json missing leading integer in value and message")
}

func parseLeadingInt(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, errors.New("empty count text")
	}

	var digits strings.Builder
	for i := range len(trimmed) {
		ch := trimmed[i]
		if ch >= '0' && ch <= '9' {
			digits.WriteByte(ch)
			continue
		}
		if ch == ',' && digits.Len() > 0 && i+1 < len(trimmed) && trimmed[i+1] >= '0' && trimmed[i+1] <= '9' {
			continue
		}
		break
	}

	if digits.Len() == 0 {
		return 0, fmt.Errorf("no leading integer in %q", raw)
	}

	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return 0, fmt.Errorf("parse integer from %q: %w", raw, err)
	}
	return n, nil
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
	req.Header.Set("Accept", "application/json")

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
