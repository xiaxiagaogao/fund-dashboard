// Package binance is a tiny HMAC-signed read-only client for the Binance USD-M
// futures REST API. Scope is intentionally narrow: only the endpoints the
// dashboard needs (account equity + asset transfer history).
//
// Why not a full SDK? The dashboard uses two endpoints. A 200-line client is
// easier to audit for the read-only-ness invariant than auditing a third-party
// SDK that supports order placement.
package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// Default endpoint — USD-M futures API.
const DefaultBaseURL = "https://fapi.binance.com"

// Client is a read-only Binance API client. Construct via New and call methods
// in package equity / transfers.
type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client

	// recvWindow tells Binance how stale the request timestamp is allowed to be.
	recvWindow time.Duration
}

func New(apiKey, apiSecret string) *Client {
	return &Client{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		baseURL:    DefaultBaseURL,
		http:       &http.Client{Timeout: 15 * time.Second},
		recvWindow: 5 * time.Second,
	}
}

// SetBaseURL is for testing against the testnet or a local mock.
func (c *Client) SetBaseURL(u string) { c.baseURL = u }

// signedGET issues an HMAC-SHA256 signed GET. params do not need to include
// timestamp/signature — those are appended here.
func (c *Client) signedGET(ctx context.Context, path string, params url.Values, out any) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	params.Set("recvWindow", strconv.FormatInt(c.recvWindow.Milliseconds(), 10))
	q := canonicalEncode(params)
	sig := sign(c.apiSecret, q)

	endpoint := fmt.Sprintf("%s%s?%s&signature=%s", c.baseURL, path, q, sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance %s %d: %s", path, resp.StatusCode, string(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode %s: %w (body=%s)", path, err, string(body))
		}
	}
	return nil
}

// publicGET issues an unsigned GET against a public market-data endpoint
// (klines etc.) — no timestamp, no signature, no API key required. Works even
// when the client was constructed without keys.
func (c *Client) publicGET(ctx context.Context, path string, params url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("binance %s %d: %s", path, resp.StatusCode, string(body))
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode %s: %w (body=%s)", path, err, string(body))
		}
	}
	return nil
}

// canonicalEncode produces a deterministic query string with sorted keys —
// HMAC must hash the exact bytes that go on the wire.
func canonicalEncode(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b []byte
	for i, k := range keys {
		if i > 0 {
			b = append(b, '&')
		}
		b = append(b, url.QueryEscape(k)...)
		b = append(b, '=')
		b = append(b, url.QueryEscape(v.Get(k))...)
	}
	return string(b)
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
