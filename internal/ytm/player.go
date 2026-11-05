package ytm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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

	// ytDlpPath is an optional fallback extractor. It is only set for the
	// production service; injected test clients leave it empty.
	ytDlpPath string
}

var DefaultPlayerService = NewPlayerService()

func NewPlayerService() *PlayerService {
	return &PlayerService{
		client:    NewClient(),
		cache:     make(map[string]*StreamInfo),
		ytDlpPath: findYTDLP(),
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

// SetAuth applies an imported session to the player's InnerTube client.
// Player requests otherwise go out anonymously and gated tracks return
// LOGIN_REQUIRED even when the rest of the app is signed in.
func (p *PlayerService) SetAuth(bundle *TokenBundle) {
	if p.client != nil {
		p.client.SetAuth(bundle)
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
		if p.ytDlpPath != "" && shouldFallbackToYTDLP(err) {
			fallbackURL, fallbackMime, fallbackErr := p.resolveWithYTDLP(ctx, videoID)
			if fallbackErr == nil {
				log.Printf("[YTM] InnerTube failed, using yt-dlp stream fallback for %s", videoID)
				streamURL, mimeType, err = fallbackURL, fallbackMime, nil
			} else {
				log.Printf("[YTM] yt-dlp fallback failed for %s: %v", videoID, fallbackErr)
			}
		}
		if err != nil {
			return "", fmt.Errorf("resolve audio stream for %s failed: %w", videoID, err)
		}
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
		Formats         []playerFormat `json:"formats"`
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

// findYTDLP locates an optional yt-dlp binary. CHORDLY_YTDLP overrides PATH.
func findYTDLP() string {
	if configured := os.Getenv("CHORDLY_YTDLP"); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured
		}
	}
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
	}
	return ""
}

// shouldFallbackToYTDLP reports whether an InnerTube failure is likely to be
// solved by the external extractor (sign-in/HTTP gating rather than a bad ID).
func shouldFallbackToYTDLP(err error) bool {
	if err == nil {
		return false
	}
	var playability *PlayabilityError
	if errors.As(err, &playability) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "status 400") ||
		strings.Contains(msg, "status 401") ||
		strings.Contains(msg, "status 403")
}

// resolveWithYTDLP asks yt-dlp for a direct audio URL. No cookie file is
// passed: yt-dlp uses its own client/PO-token handling, and fallback only runs
// after InnerTube has failed.
func (p *PlayerService) resolveWithYTDLP(ctx context.Context, videoID string) (string, string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, p.ytDlpPath,
		"--no-warnings",
		"--no-playlist",
		"--no-progress",
		"-f", "bestaudio[acodec=opus]/bestaudio",
		"-g",
		"--", "https://music.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			return "", "", fmt.Errorf("yt-dlp: %w", err)
		}
		return "", "", fmt.Errorf("yt-dlp: %w: %s", err, detail)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", "", errors.New("yt-dlp returned no stream URL")
	}

	streamURL := strings.TrimSpace(lines[0])
	mimeType := "audio/webm"
	if strings.Contains(streamURL, "mime=audio%2Fmp4") || strings.Contains(streamURL, "mime=audio/mp4") {
		mimeType = "audio/mp4"
	}
	return streamURL, mimeType, nil
}
