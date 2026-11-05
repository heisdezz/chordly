package appui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	gpstate "github.com/gogpu/ui/state"

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

	// offset is the timestamp the current pipeline started from. The device
	// counts from zero, so reported playhead positions add this back.
	offset time.Duration

	lastAdvanced string
}

var (
	// SeekRequest carries seek commands from the scrubber and lyrics view to the
	// engine. Every Set notifies, even when the target is unchanged.
	SeekRequest = gpstate.NewSignal(time.Duration(0))

	// IsScrubbing suppresses playhead updates while the user drags the scrubber,
	// so the handle follows the pointer instead of the audio clock.
	IsScrubbing = gpstate.NewSignal(false)
)

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
		SeekRequest.SubscribeForever(func(target time.Duration) { e.requestSeek(target) })
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
	e.startAt(track, 0, false)
}

// startAt (re)builds the pipeline at offset. When paused is true the pipeline is
// prepared and immediately paused, so seeking while paused keeps the transport
// in the paused state.
func (e *playbackEngine) startAt(track model.Track, offset time.Duration, paused bool) {
	e.mu.Lock()
	e.generation++
	generation := e.generation
	e.activeID = track.ID
	e.offset = offset
	e.lastAdvanced = ""
	e.stopLocked()

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.mu.Unlock()

	state.Playhead.Set(offset)
	state.PlaybackStatus.Set(model.StateBuffering)

	go e.load(ctx, generation, track, offset, paused)
}

// requestSeek reopens the stream at target. The stream URL is reused; only the
// decoder starts from a later timestamp.
func (e *playbackEngine) requestSeek(target time.Duration) {
	track := state.CurrentTrack.Get()
	if track.ID == "" {
		return
	}
	if target < 0 {
		target = 0
	}
	if track.Duration > 0 && target > track.Duration {
		target = track.Duration
	}

	e.mu.Lock()
	// activeID implies a pipeline for this track, including the brief window
	// where a previous seek is still buffering (device is nil but the track is
	// already active).
	active := e.activeID == track.ID
	e.mu.Unlock()
	if !active {
		// Nothing is loaded: accept the position so a later Play starts there.
		state.Playhead.Set(target)
		return
	}

	e.startAt(track, target, state.PlaybackStatus.Get() == model.StatePaused)
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

func (e *playbackEngine) load(ctx context.Context, generation uint64, track model.Track, offset time.Duration, paused bool) {
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

	decoder, err := openStreamDecoder(resp.Body, resp.Header.Get("Content-Type"), offset, track.Duration)
	if err != nil {
		_ = resp.Body.Close()
		fail(err)
		return
	}

	sink, err := audio.OpenCommandSink()
	if err != nil {
		_ = resp.Body.Close()
		_ = decoder.Close()
		fail(err)
		return
	}

	cfg := audio.DefaultDeviceConfig()
	cfg.Volume = state.Volume.Get()
	device, err := audio.NewDeviceWithReader(sink, decoder, cfg)
	if err != nil {
		closePipeline(nil, sink, decoder, resp.Body)
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

	state.Playhead.Set(offset)
	if track.Duration > 0 {
		state.CurrentTrack.Set(track)
	}
	if paused {
		_ = device.Pause()
		state.PlaybackStatus.Set(model.StatePaused)
	} else {
		state.PlaybackStatus.Set(model.StatePlaying)
	}
	log.Printf("[Audio] playing %q via %s at %s", track.Title, sink.Name(), offset.Round(time.Millisecond))
}

func playbackStatusError(code int) error {
	return fmt.Errorf("audio stream returned HTTP %d", code)
}

// openStreamDecoder picks a decoder for the response body. AAC/MP4 and every
// seek (offset > 0) go through ffmpeg, because the pure-Go demuxers can only
// stream Opus/WAV from the beginning. The reader is wrapped in a bufio.Reader
// first so the container peek does not consume bytes either backend needs.
func openStreamDecoder(body io.Reader, contentType string, offset, duration time.Duration) (audio.StreamDecoder, error) {
	br, ok := body.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(body)
	}

	head, _ := br.Peek(32)
	container := audio.SniffContainer(head)

	needsFFmpeg := offset > 0 || container == "mp4" || isMP4Mime(contentType)
	if needsFFmpeg && audio.FFmpegAvailable() {
		return audio.NewFFmpegDecoder(br, offset, duration)
	}
	return audio.NewStreamDecoder(br, contentType)
}

func isMP4Mime(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "mp4") ||
		strings.Contains(contentType, "m4a") ||
		strings.Contains(contentType, "aac")
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

		if state.PlaybackStatus.Get() == model.StatePlaying && !IsScrubbing.Get() {
			e.mu.Lock()
			offset := e.offset
			e.mu.Unlock()
			state.Playhead.Set(offset + device.Position())
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
