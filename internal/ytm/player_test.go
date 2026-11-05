package ytm

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestResolveStreamURL(t *testing.T) {
	svc := NewPlayerService()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	videoID := "qPRNIHxLhmc" // The Weeknd - I Feel It Coming (feat. Daft Punk)
	streamURL, err := svc.ResolveStreamURL(ctx, videoID)
	if err != nil {
		t.Fatalf("ResolveStreamURL failed: %v", err)
	}

	if !strings.HasPrefix(streamURL, "https://") {
		t.Fatalf("Expected valid https URL, got: %s", streamURL)
	}

	t.Logf("Resolved stream URL: %s... (len=%d)", streamURL[:60], len(streamURL))

	// Verify URL responds with partial or complete audio stream
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, streamURL, nil)
	if err != nil {
		t.Fatalf("Failed to create HEAD request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HEAD request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("Expected 200, 302, or 206 status, got: %d", resp.StatusCode)
	}

	// Verify caching
	cachedURL, err := svc.ResolveStreamURL(ctx, videoID)
	if err != nil {
		t.Fatalf("Cached lookup failed: %v", err)
	}
	if cachedURL != streamURL {
		t.Errorf("Expected identical cached URL, got different")
	}
}
