package appui

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"chordly/internal/audio"
	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/ytm"
)

// The playback engine owns the real audio pipeline:
//
//	ytm stream URL -> HTTP body -> audio.StreamDecoder -> audio.Device -> CommandSink
//
// It reacts to CurrentTrack + PlaybackStatus and keeps Playhead/Duration in
// sync. Stream resolution (InnerTube or yt-dlp) is delegated to ytm.Service.

type playbackEngine struct {
	mu sync.Mutex

	activeID   string
	generation uint64
	cancel     context.CancelFunc

	device  *audio.Device
	sink    *audio.CommandSink
	decoder audio.StreamDecoder
	body    io.ReadCloser

	lastAdvanced string
}

var (
	playbackOnce      sync.Once
	playbackEngineCtl *playbackEngine
)

var playbackHTTPClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 20 * time.Second,
		DialContext: (&net.Dialer{
			Timeout: 15 * time.Second,
		}).DialContext,
	},
}

// StartPlaybackEngine wires the audio pipeline to the reactive playback state.
// It is safe to call multiple times; only the first call starts the engine.
func StartPlaybackEngine() {
	playbackOnce.Do(func() {
		e := &playbackEngine{}
		playbackEngineCtl = e

		state.CurrentTrack.SubscribeForever(func(model.Track) { e.onStateChange() })
		state.PlaybackStatus.SubscribeForever(func(model.PlaybackState) { e.onStateChange() })
		state.Volume.SubscribeForever(func(volume float32) {
			e.mu.Lock()
			device := e.device
			e.mu.Unlock()
			if device != nil {
				_ = device.SetVolume(volume)
			}
		})

		go e.tick()
	})
}

func (e *playbackEngine) onStateChange() {
	switch state.PlaybackStatus.Get() {
	case model.StatePaused:
		e.mu.Lock()
		device := e.device
		e.mu.Unlock()
		if device != nil {
			_ = device.Pause()
		}
		return
	case model.StatePlaying:
		// Continue below.
	default:
		return
	}

	track := state.CurrentTrack.Get()
	if track.ID == "" {
		return
	}

	e.mu.Lock()
	device := e.device
	sameTrack := track.ID == e.activeID
	e.mu.Unlock()

	if sameTrack {
		if device != nil && device.State() == audio.DevicePaused {
			_ = device.Play()
			return
		}
		if device != nil && device.State() == audio.DevicePlaying {
			return
		}
	}

	e.start(track)
}

func (e *playbackEngine) start(track model.Track) {
	e.mu.Lock()
	e.generation++
	generation := e.generation
	e.activeID = track.ID
	e.lastAdvanced = ""
	e.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.mu.Unlock()

	state.Playhead.Set(0)
	state.PlaybackStatus.Set(model.StateBuffering)

	go e.load(ctx, generation, track)
}

// stopLocked tears down the active pipeline. Caller must hold e.mu.
func (e *playbackEngine) stopLocked() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}

	device := e.device
	sink := e.sink
	decoder := e.decoder
	body := e.body
	e.device = nil
	e.sink = nil
	e.decoder = nil
	e.body = nil

	closePipeline(device, sink, decoder, body)
}

// closePipeline stops the pump and unblocks it before closing the decoder it
// may still be reading from.
func closePipeline(device *audio.Device, sink *audio.CommandSink, decoder audio.StreamDecoder, body io.ReadCloser) {
	if device != nil {
		_ = device.Stop()
	}
	if body != nil {
		_ = body.Close()
	}
	if sink != nil {
		_ = sink.Close()
	}
	if device != nil {
		device.Wait()
	}
	if decoder != nil {
		_ = decoder.Close()
	}
}

func (e *playbackEngine) isCurrent(generation uint64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.generation == generation
}

func (e *playbackEngine) load(ctx context.Context, generation uint64, track model.Track) {
	fail := func(err error) {
		e.mu.Lock()
		current := e.generation == generation
		if current {
			e.stopLocked()
		}
		e.mu.Unlock()
		if !current {
			return
		}
		log.Printf("[Audio] playback failed for %q: %v", track.Title, err)
		state.PlaybackStatus.Set(model.StateStopped)
		ShowError("Playback failed — " + err.Error())
	}

	if track.StreamURL == "" {
		resolveCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
		err := ytm.DefaultService.PrepareTrackStream(resolveCtx, &track)
		cancel()
		if err != nil {
			fail(err)
			return
		}
	}
	if !e.isCurrent(generation) {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, track.StreamURL, nil)
	if err != nil {
		fail(err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")

	resp, err := playbackHTTPClient.Do(req)
	if err != nil {
		fail(err)
		return
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		_ = resp.Body.Close()
		fail(playbackStatusError(resp.StatusCode))
		return
	}

	decoder, err := audio.NewStreamDecoder(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		_ = resp.Body.Close()
		fail(err)
		return
	}

	sink, err := audio.OpenCommandSink()
	if err != nil {
		_ = decoder.Close()
		_ = resp.Body.Close()
		fail(err)
		return
	}

	cfg := audio.DefaultDeviceConfig()
	cfg.Volume = state.Volume.Get()
	device, err := audio.NewDeviceWithReader(sink, decoder, cfg)
	if err != nil {
		_ = sink.Close()
		_ = decoder.Close()
		_ = resp.Body.Close()
		fail(err)
		return
	}

	if !e.isCurrent(generation) {
		closePipeline(device, sink, decoder, resp.Body)
		return
	}

	e.mu.Lock()
	if e.generation != generation {
		e.mu.Unlock()
		closePipeline(device, sink, decoder, resp.Body)
		return
	}
	e.device = device
	e.sink = sink
	e.decoder = decoder
	e.body = resp.Body
	e.mu.Unlock()

	if err := device.Play(); err != nil {
		fail(err)
		return
	}

	state.Playhead.Set(0)
	if track.Duration > 0 {
		state.CurrentTrack.Set(track)
	}
	state.PlaybackStatus.Set(model.StatePlaying)
	log.Printf("[Audio] playing %q via %s", track.Title, sink.Name())
}

func playbackStatusError(code int) error {
	return fmt.Errorf("audio stream returned HTTP %d", code)
}

func (e *playbackEngine) tick() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		e.mu.Lock()
		device := e.device
		activeID := e.activeID
		e.mu.Unlock()

		if device == nil {
			continue
		}

		if state.PlaybackStatus.Get() == model.StatePlaying {
			state.Playhead.Set(device.Position())
		}

		if device.State() == audio.DeviceStopped && state.PlaybackStatus.Get() == model.StatePlaying {
			e.advanceAfterEnd(activeID)
		}
	}
}

func (e *playbackEngine) advanceAfterEnd(endedID string) {
	e.mu.Lock()
	if e.lastAdvanced == endedID {
		e.mu.Unlock()
		return
	}
	e.lastAdvanced = endedID
	e.mu.Unlock()

	queue := state.Queue.Get()
	if len(queue) <= 1 {
		state.PlaybackStatus.Set(model.StateStopped)
		return
	}
	state.NextTrack()
}
