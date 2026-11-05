package ytm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseEndpoint   = "https://music.youtube.com/youtubei/v1"
	webRemixClient = "WEB_REMIX"
	webRemixVer    = "1.20241030.01.00"
)

// Client handles low-level InnerTube API communication.
type Client struct {
	httpClient *http.Client
	mu         sync.RWMutex
	visitorID  string
}

// NewClient creates a new InnerTube client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

// RequestPayload represents the standard YouTube InnerTube request body.
type RequestPayload struct {
	Context  RequestContext `json:"context"`
	Query    string         `json:"query,omitempty"`
	BrowseID string         `json:"browseId,omitempty"`
	VideoID  string         `json:"videoId,omitempty"`
	Params   string         `json:"params,omitempty"`
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
	c.mu.RLock()
	visitor := c.visitorID
	c.mu.RUnlock()

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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("endpoint %s failed with status %d: %s", endpoint, resp.StatusCode, string(respBytes))
	}

	// Capture visitor data header if updated
	if v := resp.Header.Get("X-Goog-Visitor-Id"); v != "" {
		c.mu.Lock()
		c.visitorID = v
		c.mu.Unlock()
	}

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
