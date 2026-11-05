package plugin

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"chordly/internal/model"
)

// ScrobbleRecord represents an individual scrobble event.
type ScrobbleRecord struct {
	Track      model.Track `json:"track"`
	ListenedAt time.Time   `json:"listened_at"`
	Service    string      `json:"service"` // "listenbrainz", "lastfm", "all"
}

// ScrobbleConfig holds configuration keys and endpoints for scrobbler services.
type ScrobbleConfig struct {
	ListenBrainzToken string
	ListenBrainzURL   string // defaults to https://api.listenbrainz.org/1/submit-listens

	LastFmAPIKey     string
	LastFmSessionKey string
	LastFmSecret     string
	LastFmURL        string // defaults to https://ws.audioscrobbler.com/2.0/

	HTTPClient *http.Client
}

// ScrobbleThreshold calculates the required listen duration before a track qualifies for scrobbling.
// Formula: Min(duration/2, 4 minutes). Tracks shorter than 30s are ignored.
func ScrobbleThreshold(duration time.Duration) (time.Duration, bool) {
	if duration < 30*time.Second {
		return 0, false
	}
	half := duration / 2
	maxThreshold := 4 * time.Minute
	if half < maxThreshold {
		return half, true
	}
	return maxThreshold, true
}

// Scrobbler coordinates "Now Playing" broadcasts and scrobble submission across ListenBrainz and Last.fm.
type Scrobbler struct {
	cfg ScrobbleConfig

	mu              sync.Mutex
	currentTrack    model.Track
	accumulatedPlay time.Duration
	lastTickTime    time.Time
	scrobbled       bool
	nowPlayingSent  bool

	offlineQueue []ScrobbleRecord
}

// NewScrobbler creates a new scrobbler engine.
func NewScrobbler(cfg ScrobbleConfig) *Scrobbler {
	if cfg.ListenBrainzURL == "" {
		cfg.ListenBrainzURL = "https://api.listenbrainz.org/1/submit-listens"
	}
	if cfg.LastFmURL == "" {
		cfg.LastFmURL = "https://ws.audioscrobbler.com/2.0/"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 8 * time.Second}
	}
	return &Scrobbler{
		cfg:          cfg,
		offlineQueue: make([]ScrobbleRecord, 0),
	}
}

// TrackChanged notifies the scrobbler that a new track has been loaded.
func (s *Scrobbler) TrackChanged(t model.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentTrack = t
	s.accumulatedPlay = 0
	s.lastTickTime = time.Now()
	s.scrobbled = false
	s.nowPlayingSent = false
}

// PlaybackTick updates elapsed listening time and triggers Now Playing or Scrobble dispatch.
func (s *Scrobbler) PlaybackTick(ctx context.Context, isPlaying bool) error {
	s.mu.Lock()
	track := s.currentTrack
	if track.Title == "" || track.Artist == "" {
		s.mu.Unlock()
		return nil
	}

	now := time.Now()
	if isPlaying {
		delta := now.Sub(s.lastTickTime)
		if delta > 0 && delta < 5*time.Second {
			s.accumulatedPlay += delta
		}
	}
	s.lastTickTime = now

	// 1. Send "Now Playing" after first ~2s of playback
	shouldSendNowPlaying := isPlaying && !s.nowPlayingSent && s.accumulatedPlay >= 2*time.Second
	if shouldSendNowPlaying {
		s.nowPlayingSent = true
	}

	// 2. Check if eligible for scrobbling (50% or 4 minutes)
	threshold, eligible := ScrobbleThreshold(track.Duration)
	shouldScrobble := isPlaying && eligible && !s.scrobbled && s.accumulatedPlay >= threshold
	if shouldScrobble {
		s.scrobbled = true
	}

	s.mu.Unlock()

	var errs []error
	if shouldSendNowPlaying {
		if err := s.sendNowPlaying(ctx, track); err != nil {
			errs = append(errs, err)
		}
	}

	if shouldScrobble {
		if err := s.sendScrobble(ctx, track, now); err != nil {
			// Enqueue for offline retry
			s.mu.Lock()
			s.offlineQueue = append(s.offlineQueue, ScrobbleRecord{
				Track:      track,
				ListenedAt: now,
				Service:    "all",
			})
			s.mu.Unlock()
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// sendNowPlaying dispatches "playing_now" to ListenBrainz and Last.fm.
func (s *Scrobbler) sendNowPlaying(ctx context.Context, track model.Track) error {
	var errs []error

	if s.cfg.ListenBrainzToken != "" {
		if err := s.submitListenBrainz(ctx, "playing_now", track, time.Now()); err != nil {
			errs = append(errs, fmt.Errorf("listenbrainz now_playing: %w", err))
		}
	}

	if s.cfg.LastFmSessionKey != "" {
		if err := s.submitLastFmNowPlaying(ctx, track); err != nil {
			errs = append(errs, fmt.Errorf("lastfm now_playing: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// sendScrobble dispatches permanent scrobbles to ListenBrainz and Last.fm.
func (s *Scrobbler) sendScrobble(ctx context.Context, track model.Track, timestamp time.Time) error {
	var errs []error

	if s.cfg.ListenBrainzToken != "" {
		if err := s.submitListenBrainz(ctx, "single", track, timestamp); err != nil {
			errs = append(errs, fmt.Errorf("listenbrainz scrobble: %w", err))
		}
	}

	if s.cfg.LastFmSessionKey != "" {
		if err := s.submitLastFmScrobble(ctx, track, timestamp); err != nil {
			errs = append(errs, fmt.Errorf("lastfm scrobble: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// submitListenBrainz submits a single listen or now-playing to the ListenBrainz API.
func (s *Scrobbler) submitListenBrainz(ctx context.Context, listenType string, track model.Track, t time.Time) error {
	type trackMeta struct {
		ArtistName  string `json:"artist_name"`
		TrackName   string `json:"track_name"`
		ReleaseName string `json:"release_name,omitempty"`
	}
	type listenPayload struct {
		ListenedAt    *int64    `json:"listened_at,omitempty"`
		TrackMetadata trackMeta `json:"track_metadata"`
	}
	type lbRequest struct {
		ListenType string          `json:"listen_type"`
		Payload    []listenPayload `json:"payload"`
	}

	payloadItem := listenPayload{
		TrackMetadata: trackMeta{
			ArtistName: track.Artist,
			TrackName:  track.Title,
		},
	}
	if listenType == "single" {
		ts := t.Unix()
		payloadItem.ListenedAt = &ts
	}

	body, err := json.Marshal(lbRequest{
		ListenType: listenType,
		Payload:    []listenPayload{payloadItem},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.ListenBrainzURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+s.cfg.ListenBrainzToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("listenbrainz api returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// submitLastFmNowPlaying sends track.updateNowPlaying to the Last.fm Audioscrobbler 2.0 API.
func (s *Scrobbler) submitLastFmNowPlaying(ctx context.Context, track model.Track) error {
	params := map[string]string{
		"method": "track.updateNowPlaying",
		"artist": track.Artist,
		"track":  track.Title,
		"sk":     s.cfg.LastFmSessionKey,
	}
	return s.callLastFm(ctx, params)
}

// submitLastFmScrobble sends track.scrobble to the Last.fm Audioscrobbler 2.0 API.
func (s *Scrobbler) submitLastFmScrobble(ctx context.Context, track model.Track, timestamp time.Time) error {
	params := map[string]string{
		"method":    "track.scrobble",
		"artist":    track.Artist,
		"track":     track.Title,
		"timestamp": strconv.FormatInt(timestamp.Unix(), 10),
		"sk":        s.cfg.LastFmSessionKey,
	}
	return s.callLastFm(ctx, params)
}

// callLastFm prepares, signs, and posts a request to Last.fm.
func (s *Scrobbler) callLastFm(ctx context.Context, params map[string]string) error {
	params["api_key"] = s.cfg.LastFmAPIKey
	params["format"] = "json"

	// Sign request according to Last.fm spec
	sig := SignLastFm(params, s.cfg.LastFmSecret)
	params["api_sig"] = sig

	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.LastFmURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("lastfm api returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SignLastFm creates an MD5 signature of alphabetically sorted key-value pairs appended with secret.
// Excludes "format" parameter per Last.fm spec.
func SignLastFm(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "format" || k == "api_sig" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString(params[k])
	}
	sb.WriteString(secret)

	hasher := md5.New()
	hasher.Write([]byte(sb.String()))
	return hex.EncodeToString(hasher.Sum(nil))
}

// QueueLength returns the number of pending offline scrobbles.
func (s *Scrobbler) QueueLength() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.offlineQueue)
}

// FlushQueue retries submitting all pending offline scrobbles.
func (s *Scrobbler) FlushQueue(ctx context.Context) error {
	s.mu.Lock()
	if len(s.offlineQueue) == 0 {
		s.mu.Unlock()
		return nil
	}
	items := s.offlineQueue
	s.offlineQueue = make([]ScrobbleRecord, 0)
	s.mu.Unlock()

	var remaining []ScrobbleRecord
	for _, item := range items {
		if err := s.sendScrobble(ctx, item.Track, item.ListenedAt); err != nil {
			remaining = append(remaining, item)
		}
	}

	if len(remaining) > 0 {
		s.mu.Lock()
		s.offlineQueue = append(remaining, s.offlineQueue...)
		s.mu.Unlock()
		return fmt.Errorf("failed to flush %d scrobbles", len(remaining))
	}

	return nil
}
