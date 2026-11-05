package ytm

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseEndpoint       = "https://music.youtube.com/youtubei/v1"
	webRemixClient     = "WEB_REMIX"
	webRemixVer        = "1.20241030.01.00"
	androidMusicClient = "ANDROID_MUSIC"
	androidMusicVer    = "6.45.52"
	ytmOrigin          = "https://music.youtube.com"
)

// Client handles low-level InnerTube API communication.
type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
	visitorID  string
	auth       *TokenBundle
	now        func() time.Time
}

// NewClient creates a new InnerTube client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
		now: time.Now,
	}
}

// SetAuth applies an imported YouTube Music session to every subsequent
// request. Passing nil clears authentication.
func (c *Client) SetAuth(bundle *TokenBundle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if bundle == nil {
		c.auth = nil
		return
	}
	copy := *bundle
	c.auth = &copy
}

// ClearAuth removes the active session.
func (c *Client) ClearAuth() {
	c.SetAuth(nil)
}

// HasAuth reports whether a session is attached.
func (c *Client) HasAuth() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.auth != nil
}

func (c *Client) requestState() (visitor string, auth *TokenBundle) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	visitor = c.visitorID
	if c.auth != nil {
		copy := *c.auth
		auth = &copy
		if auth.VisitorData != "" {
			visitor = auth.VisitorData
		}
	}
	return visitor, auth
}

func (c *Client) applyAuth(req *http.Request, auth *TokenBundle, visitor string) {
	if visitor != "" {
		req.Header.Set("X-Goog-Visitor-Id", visitor)
	}
	if auth == nil {
		return
	}
	if auth.Cookie != "" {
		req.Header.Set("Cookie", auth.Cookie)
	}

	authorization := ""
	if auth.SAPISID != "" {
		now := time.Now
		if c.now != nil {
			now = c.now
		}
		authorization = sapisidHash(auth.SAPISID, now().Unix())
	} else {
		// Fall back to a supplied header when the bundle has no SAPISID.
		authorization = auth.Authorization
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	req.Header.Set("X-Goog-AuthUser", "0")
	req.Header.Set("Origin", ytmOrigin)
	req.Header.Set("X-Origin", ytmOrigin)
}

// sapisidHash builds the Authorization header used by YouTube Music web
// sessions: SAPISIDHASH <ts>_<SHA1(ts + " " + SAPISID + " " + origin)>.
func sapisidHash(sapisid string, timestamp int64) string {
	payload := fmt.Sprintf("%d %s %s", timestamp, sapisid, ytmOrigin)
	sum := sha1.Sum([]byte(payload))
	return fmt.Sprintf("SAPISIDHASH %d_%x", timestamp, sum)
}

// RequestPayload represents the standard YouTube InnerTube request body.
type RequestPayload struct {
	Context      RequestContext `json:"context"`
	Query        string         `json:"query,omitempty"`
	BrowseID     string         `json:"browseId,omitempty"`
	VideoID      string         `json:"videoId,omitempty"`
	PlaylistID   string         `json:"playlistId,omitempty"`
	Continuation string         `json:"continuation,omitempty"`
	Params       string         `json:"params,omitempty"`
	IsAudioOnly  bool           `json:"isAudioOnly,omitempty"`
}

type RequestContext struct {
	Client ContextClient `json:"client"`
}

type ContextClient struct {
	ClientName    string `json:"clientName"`
	ClientVersion string `json:"clientVersion"`
	HL            string `json:"hl"`
	GL            string `json:"gl"`
	VisitorData   string `json:"visitorData,omitempty"`
}

// Do executes a POST request against a YouTube Music InnerTube endpoint.
func (c *Client) Do(ctx context.Context, endpoint string, payload RequestPayload, out interface{}) error {
	visitor, auth := c.requestState()

	payload.Context = RequestContext{
		Client: ContextClient{
			ClientName:    webRemixClient,
			ClientVersion: webRemixVer,
			HL:            "en",
			GL:            "US",
			VisitorData:   visitor,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s", baseEndpoint, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.youtube.com/")
	c.applyAuth(req, auth, visitor)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("endpoint %s failed with status %d: %s", endpoint, resp.StatusCode, string(respBytes))
	}

	c.updateVisitor(resp.Header)
	return jsonResponse(resp, out)
}

// doPlayer resolves a `/player` request. It stamps the ANDROID_MUSIC client
// context (WEB_REMIX is unsuitable for stream resolution: fewer direct URLs,
// more ciphering) and shares visitor-id bookkeeping with Do.
func (c *Client) doPlayer(ctx context.Context, payload RequestPayload, out interface{}) error {
	visitor, auth := c.requestState()

	payload.Context = RequestContext{
		Client: ContextClient{
			ClientName:    androidMusicClient,
			ClientVersion: androidMusicVer,
			HL:            "en",
			GL:            "US",
			VisitorData:   visitor,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}

	reqURL := fmt.Sprintf("%s/player", baseEndpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://music.youtube.com/")
	c.applyAuth(req, auth, visitor)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("endpoint player failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	c.updateVisitor(resp.Header)
	return jsonResponse(resp, out)
}

// updateVisitor persists the X-Goog-Visitor-Id header for subsequent requests.
func (c *Client) updateVisitor(h http.Header) {
	if v := h.Get("X-Goog-Visitor-Id"); v != "" {
		c.mu.Lock()
		c.visitorID = v
		c.mu.Unlock()
	}
}

func jsonResponse(resp *http.Response, out interface{}) error {
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error: %w", err)
	}
	if out != nil {
		if err := json.Unmarshal(respData, out); err != nil {
			return fmt.Errorf("unmarshal response error: %w", err)
		}
	}
	return nil
}
