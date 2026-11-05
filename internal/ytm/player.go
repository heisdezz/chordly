package ytm

import (
	"context"
	"errors"
	"fmt"
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
type PlayerService struct {
	mu    sync.RWMutex
	cache map[string]*StreamInfo
}

var DefaultPlayerService = NewPlayerService()

func NewPlayerService() *PlayerService {
	return &PlayerService{
		cache: make(map[string]*StreamInfo),
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

	// 2. Resolve via fast extractor
	streamURL, err := resolveAudioStream(ctx, videoID)
	if err != nil {
		return "", fmt.Errorf("resolve audio stream for %s failed: %w", videoID, err)
	}

	// 3. Store in cache (URLs typically valid for 6 hours; expire at 4 hours for safety)
	p.mu.Lock()
	p.cache[videoID] = &StreamInfo{
		VideoID:   videoID,
		URL:       streamURL,
		ExpiresAt: time.Now().Add(4 * time.Hour),
	}
	p.mu.Unlock()

	return streamURL, nil
}

// resolveAudioStream executes direct audio stream resolution.
func resolveAudioStream(ctx context.Context, videoID string) (string, error) {
	targetURL := fmt.Sprintf("https://music.youtube.com/watch?v=%s", videoID)

	cmd := exec.CommandContext(ctx, "yt-dlp",
		"-g",
		"-f", "ba[ext=m4a]/ba[ext=webm]/ba",
		"--no-warnings",
		"--no-playlist",
		targetURL,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	streamURL := strings.TrimSpace(string(out))
	if !strings.HasPrefix(streamURL, "http") {
		return "", fmt.Errorf("invalid stream url returned: %q", streamURL)
	}

	// If multiple URLs returned (separated by newline), take the first one
	if lines := strings.Split(streamURL, "\n"); len(lines) > 0 {
		streamURL = strings.TrimSpace(lines[0])
	}

	return streamURL, nil
}
