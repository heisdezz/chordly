package ytm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// StreamInfo caches resolved audio stream metadata.
type StreamInfo struct {
	VideoID   string
	URL       string
	MimeType  string
	ExpiresAt time.Time
}

// PlayerService handles resolving playable audio stream URLs.
//
// Resolution strategy (per InnerTune services plan Module 4):
//  1. Query `/youtubei/v1/player` with the ANDROID_MUSIC client, which serves
//     direct `url` fields on `streamingData.adaptiveFormats` without requiring
//     signature deciphering.
//  2. Select the highest-fidelity audio-only format: Opus/webm (itag 251)
//     first, AAC/mp4 (itag 140) as fallback.
//  3. Cache the resolved URL in memory; URLs typically stay valid for ~6h, so
//     a 4h expiry is used for safety.
type PlayerService struct {
	client *Client
	mu     sync.RWMutex
	cache  map[string]*StreamInfo
}

var DefaultPlayerService = NewPlayerService()

func NewPlayerService() *PlayerService {
	return &PlayerService{
		client: NewClient(),
		cache:  make(map[string]*StreamInfo),
	}
}

// NewPlayerServiceWithClient allows tests and the Service facade to inject a
// client (e.g. pointed at a stub transport).
func NewPlayerServiceWithClient(c *Client) *PlayerService {
	return &PlayerService{
		client: c,
		cache:  make(map[string]*StreamInfo),
	}
}

// ResolveStreamURL retrieves a direct playable audio stream URL for a YouTube video ID.
func (p *PlayerService) ResolveStreamURL(ctx context.Context, videoID string) (string, error) {
	if videoID == "" {
		return "", errors.New("empty videoID")
	}

	// 1. Check in-memory cache
	p.mu.RLock()
	cached, ok := p.cache[videoID]
	p.mu.RUnlock()
	if ok && time.Now().Before(cached.ExpiresAt) {
		return cached.URL, nil
	}

	// 2. Resolve via InnerTube player endpoint
	streamURL, mimeType, err := p.resolveStream(ctx, videoID)
	if err != nil {
		return "", fmt.Errorf("resolve audio stream for %s failed: %w", videoID, err)
	}

	// 3. Store in cache (see header comment for the 4h safety expiry)
	p.mu.Lock()
	p.cache[videoID] = &StreamInfo{
		VideoID:   videoID,
		URL:       streamURL,
		MimeType:  mimeType,
		ExpiresAt: time.Now().Add(4 * time.Hour),
	}
	p.mu.Unlock()

	return streamURL, nil
}

// playerResponse models the subset of `/youtubei/v1/player` needed to select
// an audio stream.
type playerResponse struct {
	StreamingData struct {
		AdaptiveFormats []playerFormat `json:"adaptiveFormats"`
		Formats        []playerFormat `json:"formats"`
	} `json:"streamingData"`
	PlayabilityStatus struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"playabilityStatus"`
}

type playerFormat struct {
	Itag         int    `json:"itag"`
	URL          string `json:"url"`
	MimeType     string `json:"mimeType"`
	AudioQuality string `json:"audioQuality"`
}

// resolveStream queries the player endpoint and returns the chosen URL plus
// its mime type.
func (p *PlayerService) resolveStream(ctx context.Context, videoID string) (string, string, error) {
	client := p.client
	if client == nil {
		client = NewClient()
	}

	payload := RequestPayload{VideoID: videoID}

	// The player endpoint must be queried with the ANDROID_MUSIC client
	// context (WEB_REMIX is unsuitable for stream resolution: fewer direct
	// URLs, more ciphering). Client.Do always stamps WEB_REMIX into the
	// payload, so this call bypasses Do's context injection and reuses the
	// client's transport instead.
	var resp playerResponse
	if err := client.doPlayer(ctx, payload, &resp); err != nil {
		return "", "", err
	}

	if status := resp.PlayabilityStatus.Status; status != "" && status != "OK" {
		reason := resp.PlayabilityStatus.Reason
		if reason == "" {
			reason = "unknown"
		}
		return "", "", NewPlayabilityError(status, reason)
	}

	format := SelectAudioFormat(resp.StreamingData.AdaptiveFormats)
	if format == nil {
		// Some videos only expose progressive `formats` (older uploads).
		format = SelectAudioFormat(resp.StreamingData.Formats)
	}
	if format == nil {
		return "", "", errors.New("no playable audio format in player response")
	}
	if format.URL == "" {
		return "", "", fmt.Errorf("audio format itag %d has no direct url (signature deciphering required)", format.Itag)
	}

	return format.URL, format.MimeType, nil
}

// SelectAudioFormat picks the best audio-only format from a list of adaptive
// formats. Preference order:
//  1. Opus in WebM (itag 251) — ~160kbps, 48kHz.
//  2. AAC in MP4 (itag 140) — ~128kbps, 44.1kHz.
//  3. Lowest itag audio format (as a last resort).
//
// Non-audio formats (video) are skipped.
func SelectAudioFormat(formats []playerFormat) *playerFormat {
	var opus, aac, fallback *playerFormat
	for i := range formats {
		f := &formats[i]
		if !isAudioMime(f.MimeType) {
			continue
		}
		switch {
		case f.Itag == 251 || (strings.HasPrefix(f.MimeType, "audio/webm") && (f.Itag == 0 || f.Itag == 251)):
			if opus == nil {
				opus = f
			}
		case f.Itag == 140 || strings.HasPrefix(f.MimeType, "audio/mp4"):
			if aac == nil {
				aac = f
			}
		default:
			if fallback == nil {
				fallback = f
			}
		}
	}
	if opus != nil {
		return opus
	}
	if aac != nil {
		return aac
	}
	return fallback
}

func isAudioMime(mime string) bool {
	return strings.HasPrefix(mime, "audio/")
}

// PlayabilityError reports a video that InnerTube refuses to serve audio for
// (age-gated, region-blocked, removed, etc.).
type PlayabilityError struct {
	Status string
	Reason string
}

func (e *PlayabilityError) Error() string {
	return fmt.Sprintf("playability status %s: %s", e.Status, e.Reason)
}

// NewPlayabilityError wraps a non-OK status into a *PlayabilityError.
func NewPlayabilityError(status, reason string) error {
	return &PlayabilityError{Status: status, Reason: reason}
}
