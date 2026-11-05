package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"chordly/internal/model"
)

func TestScrobbleThreshold(t *testing.T) {
	// < 30s: ignored
	if _, eligible := ScrobbleThreshold(20 * time.Second); eligible {
		t.Errorf("expected < 30s to be ineligible")
	}

	// 2 minutes (120s): threshold 60s
	th, eligible := ScrobbleThreshold(120 * time.Second)
	if !eligible || th != 60*time.Second {
		t.Errorf("expected 60s threshold, got %v", th)
	}

	// 6 minutes (360s): threshold 180s
	th, eligible = ScrobbleThreshold(360 * time.Second)
	if !eligible || th != 180*time.Second {
		t.Errorf("expected 180s threshold, got %v", th)
	}

	// 10 minutes (600s): capped at 4 minutes (240s)
	th, eligible = ScrobbleThreshold(600 * time.Second)
	if !eligible || th != 4*time.Minute {
		t.Errorf("expected 4m threshold cap, got %v", th)
	}
}

func TestSignLastFm(t *testing.T) {
	params := map[string]string{
		"method": "track.scrobble",
		"artist": "Daft Punk",
		"track":  "Get Lucky",
		"format": "json", // should be ignored by signature
	}
	secret := "secret123"

	sig := SignLastFm(params, secret)
	if len(sig) != 32 {
		t.Errorf("expected 32-char hex MD5, got %s", sig)
	}

	// Signature must be deterministic
	sig2 := SignLastFm(params, secret)
	if sig != sig2 {
		t.Errorf("expected identical signatures")
	}
}

func TestScrobblerFlow(t *testing.T) {
	var mu sync.Mutex
	requests := make([]string, 0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		listenType, _ := body["listen_type"].(string)
		requests = append(requests, listenType)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := ScrobbleConfig{
		ListenBrainzToken: "fake-token",
		ListenBrainzURL:   server.URL,
		HTTPClient:        server.Client(),
	}

	scrobbler := NewScrobbler(cfg)
	ctx := context.Background()

	track := model.Track{
		ID:       "song-1",
		Title:    "Harder Better Faster Stronger",
		Artist:   "Daft Punk",
		Duration: 220 * time.Second, // 50% = 110s
	}

	scrobbler.TrackChanged(track)

	// Tick 1: 0s played -> no requests
	_ = scrobbler.PlaybackTick(ctx, true)

	// Simulate accumulated 2.5s played -> should trigger "playing_now"
	scrobbler.mu.Lock()
	scrobbler.accumulatedPlay = 3 * time.Second
	scrobbler.mu.Unlock()

	_ = scrobbler.PlaybackTick(ctx, true)

	mu.Lock()
	if len(requests) != 1 || requests[0] != "playing_now" {
		t.Fatalf("expected 1 playing_now request, got %+v", requests)
	}
	mu.Unlock()

	// Simulate accumulated 50s played (under 110s threshold) -> no scrobble yet
	scrobbler.mu.Lock()
	scrobbler.accumulatedPlay = 50 * time.Second
	scrobbler.mu.Unlock()

	_ = scrobbler.PlaybackTick(ctx, true)

	mu.Lock()
	if len(requests) != 1 {
		t.Fatalf("expected still only 1 request, got %d", len(requests))
	}
	mu.Unlock()

	// Simulate reaching 115s played (past 110s threshold) -> should trigger "single" scrobble
	scrobbler.mu.Lock()
	scrobbler.accumulatedPlay = 115 * time.Second
	scrobbler.mu.Unlock()

	_ = scrobbler.PlaybackTick(ctx, true)

	mu.Lock()
	if len(requests) != 2 || requests[1] != "single" {
		t.Fatalf("expected [playing_now, single], got %+v", requests)
	}
	mu.Unlock()

	// Subsequent ticks must not double-scrobble
	_ = scrobbler.PlaybackTick(ctx, true)
	mu.Lock()
	if len(requests) != 2 {
		t.Errorf("expected no duplicate scrobble, got %d requests", len(requests))
	}
	mu.Unlock()
}

func TestScrobblerOfflineQueueAndFlush(t *testing.T) {
	shouldFail := true
	var mu sync.Mutex
	submitted := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if shouldFail {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		submitted++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := ScrobbleConfig{
		ListenBrainzToken: "fake-token",
		ListenBrainzURL:   server.URL,
		HTTPClient:        server.Client(),
	}

	scrobbler := NewScrobbler(cfg)
	ctx := context.Background()

	track := model.Track{
		ID:       "song-2",
		Title:    "Voyager",
		Artist:   "Daft Punk",
		Duration: 200 * time.Second,
	}

	scrobbler.TrackChanged(track)
	scrobbler.mu.Lock()
	scrobbler.accumulatedPlay = 110 * time.Second // triggers scrobble
	scrobbler.nowPlayingSent = true
	scrobbler.mu.Unlock()

	// Tick should encounter 500 error and enqueue into offline queue
	_ = scrobbler.PlaybackTick(ctx, true)

	if scrobbler.QueueLength() != 1 {
		t.Fatalf("expected 1 item in offline queue, got %d", scrobbler.QueueLength())
	}

	// Server recovers
	mu.Lock()
	shouldFail = false
	mu.Unlock()

	// Flush queue
	if err := scrobbler.FlushQueue(ctx); err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	if scrobbler.QueueLength() != 0 {
		t.Errorf("expected empty queue after flush, got %d", scrobbler.QueueLength())
	}
	if submitted != 1 {
		t.Errorf("expected 1 successful submission, got %d", submitted)
	}
}
