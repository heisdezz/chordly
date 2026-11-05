# Playback Hardening: Seeking, AAC/MP4, and Session Persistence

**Tasks:** `TSK-110` (seeking + AAC/MP4), `UI-AUTH-04` (session persistence)
**Workstreams:** WS-AUD/WS-UI, WS-YTM/WS-UI
**Files:** `appui/playback.go`, `appui/scrubber.go`, `appui/lyrics_view.go`, `internal/audio/ffmpeg.go`, `internal/session/*`, `main.go`
**Status:** implemented; awaiting the concurrent `search_view.go` / `queue_view.go` rewrites before a full-suite run

---

## 1. Seeking

Before this change the scrubber only set `state.Playhead`; the 250 ms engine
ticker immediately overwrote it with the device clock, so the handle snapped
back and audio never moved.

Now two app-local signals (declared in `appui/playback.go` to avoid touching
`internal/state/state.go`, which TSK-220 owns) drive seeking:

| Signal | Purpose |
| --- | --- |
| `appui.SeekRequest` (`time.Duration`) | Set on scrubber release and lyric-line click. Every `Set` notifies, even for the same value. |
| `appui.IsScrubbing` (`bool`) | While true the engine ticker does not overwrite `Playhead`, so the handle follows the pointer. |

Flow:

1. Press/drag → `IsScrubbing.Set(true)` and preview `Playhead`.
2. Release → `IsScrubbing.Set(false)`, `SeekRequest.Set(target)`.
3. `playbackEngine.requestSeek` clamps to `[0, track.Duration]`, increments the
   pipeline generation, tears down the old pipeline, and rebuilds it at the
   target via `startAt`.
4. Seeking while paused prepares the pipeline and leaves it paused.

The device clock always starts at zero for a pipeline, so the engine keeps the
start offset and reports `offset + device.Position()`.

## 2. Seeking and AAC/MP4 via ffmpeg

`internal/audio/ffmpeg.go` adds `FFmpegDecoder`, which runs

```
ffmpeg -hide_banner -loglevel error -nostdin -i pipe:0 [-ss <sec>] \
  -vn -map 0:a:0 -f s16le -acodec pcm_s16le -ar 48000 -ac 2 pipe:1
```

It satisfies the same `StreamDecoder` contract as the Go decoders (48 kHz
stereo s16le PCM), so `Device` and `CommandSink` are untouched.

- **AAC/MP4**: `openStreamDecoder` sniffs the container first and routes MP4
  (or an `audio/mp4` content type) to ffmpeg.
- **Seeking**: any `offset > 0` goes through ffmpeg, because the pure-Go WebM
  and Ogg demuxers can only stream from the start. `-ss` is placed **after**
  `-i` (output seeking) so it works on non-seekable HTTP bodies.
- **Override**: `CHORDLY_FFMPEG` selects a specific binary; otherwise `PATH` is
  searched. When ffmpeg is missing, MP4 streams report
  `ErrUnsupportedFormat` and seek requests fail with a toast instead of
  silently hanging.

Important: `openStreamDecoder` wraps the body in `bufio.Reader` and calls
`Peek(32)` before choosing a backend. Never call `audio.NewStreamDecoder`
first and fall back to ffmpeg with the same reader — its internal `bufio`
consumes the peeked bytes on the failure path.

## 3. Session persistence

`internal/session/store.go` saves the imported session to
`$XDG_CONFIG_HOME/chordly/session.json`:

- Directory `0700`, file `0600`, atomic write (temp file + `rename`).
- Only `cookie`, `sapisid`, `visitor_data`, `account_name`, and `source` are
  stored. SID hashes, cookie maps, and `Authorization` are **derived at load
  time** with `ytm.ParseToken`, so the on-disk format does not embed secrets
  beyond the session cookie itself.
- `Load` returns `ErrNotFound` for a missing or empty file; corrupt JSON is a
  real error and is logged without echoing contents.
- `Clear` runs on sign-out (`ActiveSession` → `nil`).

`main.go` wires it before the UI starts:

1. Subscribe to `appui.ActiveSession`: apply to `ytm.DefaultService` **and**
   persist (`nil` clears the file).
2. Load a saved bundle, set `ActiveSession`, and restore `state.UserAccount`
   from `AccountName`.

The raw cookie is never logged; only `TokenBundle.Summary()` (masked SAPISID)
appears in the restore log line.

## 4. Verification

```
go test ./internal/session/ -count=1        # roundtrip, 0600/0700, clear, corrupt, no path
go test ./internal/audio/ -run TestFFmpeg -count=1   # decode + seek, skipped without ffmpeg
go test ./internal/ytm/ -count=1
```

`go build ./...` is currently blocked by other agents' in-flight
`appui/search_view.go` and `appui/queue_view.go` rewrites, not by these files.

## 5. Follow-ups

- Throttle/coalesce seek requests if dragging ever commits continuously
  (currently only release commits, which is one pipeline rebuild per user
  gesture).
- Optional: preload the next track to make transitions gapless.
- Session persistence assumes a single user/account; multi-account support would
  need a per-account filename.
