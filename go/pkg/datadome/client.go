// Package datadome provides a simple SDK for DataDome fingerprint submission.
package datadome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CircuitSavage/datadome-solver/internal/builder"
	ddcrypto "github.com/CircuitSavage/datadome-solver/internal/crypto"
)

const clientVersion = "5.6.6"

// Client solves DataDome challenges for a target site.
type Client struct {
	SiteURL  string
	ProxyURL string
	DDJSKey  string
	CID      string
	Profile  string
	HTTP     *http.Client
}

// Result holds the API response from tags.js.
type Result struct {
	Status int    `json:"status"`
	Cookie string `json:"cookie"`
	Raw    map[string]any
}

// New creates a client for the given protected origin.
// siteURL must include a scheme and host. Set DDJSKey via WithDDJSKey (required).
func New(siteURL string, opts ...Option) (*Client, error) {
	if siteURL == "" {
		return nil, fmt.Errorf("datadome: site URL is required")
	}
	if !strings.HasPrefix(siteURL, "http://") && !strings.HasPrefix(siteURL, "https://") {
		siteURL = "https://" + siteURL
	}
	u, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("datadome: invalid site URL: %w", err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("datadome: site URL must include a host")
	}

	c := &Client{
		SiteURL: strings.TrimSuffix(siteURL, "/") + "/",
		Profile: "chrome_win10",
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.DDJSKey == "" {
		return nil, fmt.Errorf("datadome: DDJSKey is required (use WithDDJSKey)")
	}
	if c.HTTP == nil {
		c.HTTP, err = newHTTPClient(c.ProxyURL)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Option configures the client.
type Option func(*Client)

// WithProxy sets an HTTP/SOCKS proxy URL.
func WithProxy(proxyURL string) Option {
	return func(c *Client) { c.ProxyURL = proxyURL }
}

// WithDDJSKey sets the ddjskey (ddk) value.
func WithDDJSKey(key string) Option {
	return func(c *Client) { c.DDJSKey = key }
}

// WithCID sets the session CID cookie value if known.
func WithCID(cid string) Option {
	return func(c *Client) { c.CID = cid }
}

// WithProfile sets the browser profile name (e.g. chrome_win10).
func WithProfile(profile string) Option {
	return func(c *Client) { c.Profile = profile }
}

// WithHTTPClient uses a custom http.Client (proxy can be set on its Transport).
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.HTTP = client }
}

// BuildPayload generates ordered fingerprint signals without submitting.
func (c *Client) BuildPayload(serverHash *string, bpc int) []ddcrypto.Signal {
	return builder.BuildPayload(builder.Options{
		Profile:    c.Profile,
		URL:        c.SiteURL,
		ServerHash: serverHash,
		BPC:        bpc,
	})
}

// EncryptJSPL encrypts signals into a jspl string.
func (c *Client) EncryptJSPL(signals []ddcrypto.Signal) (string, error) {
	return ddcrypto.Encrypt(signals, c.DDJSKey, c.CID, nil)
}

// Solve builds a fingerprint, posts to tags.js, and returns the result.
func (c *Client) Solve(ctx context.Context) (*Result, error) {
	signals := c.BuildPayload(nil, 1)
	jspl, err := c.EncryptJSPL(signals)
	if err != nil {
		return nil, fmt.Errorf("datadome: encrypt: %w", err)
	}

	endpoint, origin, referer, err := c.endpoints()
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("jspl", jspl)
	form.Set("eventCounters", "[]")
	form.Set("jsType", "ch")
	form.Set("cid", c.CID)
	form.Set("ddk", c.DDJSKey)
	form.Set("Referer", url.QueryEscape(c.SiteURL))
	form.Set("request", "%2F")
	form.Set("responsePage", "origin")
	form.Set("ddv", clientVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("downlink", "10")
	req.Header.Set("dpr", "1")
	req.Header.Set("ect", "4g")
	req.Header.Set("origin", origin)
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", referer)
	req.Header.Set("rtt", "0")
	req.Header.Set("sec-ch-dpr", "1")
	req.Header.Set("sec-ch-ua", `"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`)
	req.Header.Set("sec-ch-ua-arch", `"x86"`)
	req.Header.Set("sec-ch-ua-bitness", `"64"`)
	req.Header.Set("sec-ch-ua-full-version-list", `"Chromium";v="148.0.7778.179", "Google Chrome";v="148.0.7778.179", "Not/A)Brand";v="99.0.0.0"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-ch-ua-platform-version", `"14.0.0"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("datadome: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("datadome: invalid JSON response (%d): %s", resp.StatusCode, truncate(string(body), 200))
	}

	result := &Result{Raw: raw}
	if st, ok := raw["status"].(float64); ok {
		result.Status = int(st)
	}
	if cookie, ok := raw["cookie"].(string); ok {
		result.Cookie = cookie
	}

	if result.Status != 200 {
		return result, fmt.Errorf("datadome: solve failed with status %d", result.Status)
	}
	return result, nil
}

func (c *Client) endpoints() (tagsEndpoint, origin, referer string, err error) {
	u, err := url.Parse(c.SiteURL)
	if err != nil {
		return "", "", "", err
	}
	origin = u.Scheme + "://" + u.Host
	referer = c.SiteURL
	tagsEndpoint = origin + "/include/tags.js"
	return tagsEndpoint, origin, referer, nil
}

func newHTTPClient(proxyURL string) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	if proxyURL != "" {
		pu, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("datadome: invalid proxy URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(pu)
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
