# Plan 1: Basic Capabilities Implementation Plan

This implementation plan details the foundation needed for a fully functional desktop YouTube Music player in **GoGPU UI**, utilizing the **InnerTube** API and a native audio engine.

---

## Module 1: InnerTube & YouTube Music Backend (`internal/ytm`)

**Objective:** Query YouTube Music's catalog, parse structured metadata, resolve audio stream URLs, and handle YouTube player cipher unscrambling.

### Step 1.1: InnerTube Session & HTTP Client
* Implement a dedicated YouTube Music client targeting the `WEB_REMIX` client context (`clientVersion: "1.2024xxxx"`).
* Define endpoint configurations:
  * Base URL: `https://music.youtube.com/youtubei/v1`
  * Default payload context with `WEB_REMIX`, locale `en`, and country `US`.
* Structure core domain types:
  ```go
  type Track struct {
      ID          string        `json:"id"`
      Title       string        `json:"title"`
      Artist      string        `json:"artist"`
      ArtistID    string        `json:"artist_id"`
      Album       string        `json:"album"`
      AlbumID     string        `json:"album_id,omitempty"`
      Duration    time.Duration `json:"duration"`
      Thumbnail   string        `json:"thumbnail_url"`
      StreamURL   string        `json:"stream_url,omitempty"`
  }

  type SearchResults struct {
      Songs     []Track       `json:"songs"`
      Albums    []AlbumItem   `json:"albums"`
      Artists   []ArtistItem  `json:"artists"`
      Playlists []PlaylistItem `json:"playlists"`
  }
  ```

### Step 1.2: Catalog Search & Browse APIs
* Implement `Search(ctx context.Context, query string, filter SearchFilter) (*SearchResults, error)`:
  * Map search query to `/youtubei/v1/search`.
  * Support filters: `FilterAll`, `FilterSongs`, `FilterAlbums`, `FilterArtists`, `FilterPlaylists`.
  * Parse nested `musicResponsiveListItemRenderer` elements into normalized `Track` items.
* Implement `GetHomeFeed(ctx context.Context)`:
  * Query `/youtubei/v1/browse` with `browseId: "FEmusic_home"`.
  * Parse sections: "Quick picks", "Trending", "Recommended radios".
* Implement `GetAlbum(ctx context.Context, browseID string)`:
  * Retrieve full album details, release year, artist, and track list.

### Step 1.3: Audio Stream Extraction & Cipher Resolver
* Implement `ResolveStreamURL(ctx context.Context, videoID string) (string, error)`:
  * Call `/youtubei/v1/player` with client `ANDROID_MUSIC` or `WEB_REMIX`.
  * Inspect `streamingData.adaptiveFormats`.
  * Filter for audio formats:
    * Primary preference: `audio/webm; codecs="opus"` (itag 251, ~160kbps).
    * Fallback preference: `audio/mp4; codecs="mp4a.40.2"` (itag 140, ~128kbps).
  * If format requires signature decoding, extract player decipher script (`base.js`), compute signature decipher algorithm, and apply `n`-parameter transform.

### Step 1.4: Radio & Queue Generation
* Implement `GetNextRadio(ctx context.Context, videoID string) ([]Track, error)`:
  * Query `/youtubei/v1/next` with `videoId`.
  * Parse the `musicQueueRenderer` to retrieve automix recommendation tracks to dynamically extend the playback queue.

**Verification Criteria:**
* Integration test in `scratch/test_ytm.go` queries "Daft Punk", extracts top songs, resolves the stream URL for the first hit, and validates an HTTP `206 Partial Content` audio response.

---

## Module 2: Audio Decoding & Playback Engine (`internal/audio`)

**Objective:** Stream, buffer, decode, and output low-latency audio to the operating system's sound system.

### Step 2.1: Sound Hardware Integration
* Select pure-Go or CGO-free audio output (e.g. `github.com/ebitengine/oto/v3` or `github.com/gen2brain/malgo`).
* Initialize audio driver context:
  * Sample rate: 48,000 Hz.
  * Channels: 2 (stereo).
  * Format: 16-bit signed integer PCM (`int16` little-endian).

### Step 2.2: Stream Buffering Pipeline (`RingReader`)
* Build a non-blocking asynchronous streaming reader:
  ```go
  type StreamBuffer struct {
      mu       sync.Mutex
      reader   io.ReadCloser
      buffer   *ring.Buffer
      capacity int           // e.g. 5 MB pre-buffer
      eof      bool
  }
  ```
* Features:
  * Background worker continuously reads HTTP audio stream into ring buffer.
  * Eliminates audio stutter during transient network latency.
  * Supports seeking by issuing an HTTP `Range: bytes=<offset>-` request.

### Step 2.3: Audio Decoder
* Decode Opus/WebM audio stream into raw PCM frames:
  * Utilize pure-Go WebM demuxer and Opus decoder or AAC decoder for MP4 streams.
  * Expose an `io.Reader` that streams continuous PCM bytes to the Oto/Malgo audio player.

### Step 2.4: High-Level Player Controller
* Define the central player controller interface:
  ```go
  type PlayerState int
  const (
      StateStopped PlayerState = iota
      StatePlaying
      StatePaused
      StateBuffering
  )

  type Player interface {
      Load(ctx context.Context, track Track) error
      Play()
      Pause()
      TogglePlay()
      Seek(target time.Duration) error
      SetVolume(level float32) // 0.0 - 1.0
      Volume() float32
      Position() time.Duration
      Duration() time.Duration
      State() PlayerState
  }
  ```
* Implement a 30 Hz ticker goroutine that calculates current playhead position and updates playback signals.

**Verification Criteria:**
* Test playing an audio stream through the engine for 15 seconds, testing pause, resume, seek forward by 30 seconds, and volume adjustment.

---

## Module 3: Reactive State & Queue Management (`internal/state`)

**Objective:** Orchestrate application-wide reactive state using `github.com/gogpu/ui/state`.

### Step 3.1: Reactive State Signals
* Define state signals:
  ```go
  package state

  import (
      "time"
      "github.com/gogpu/ui/state"
      "chordly/internal/ytm"
  )

  var (
      // Playback Signals
      SigCurrentTrack  = state.NewSignal[ytm.Track](ytm.Track{})
      SigPlaybackState = state.NewSignal[PlayerState](StateStopped)
      SigPosition      = state.NewSignal[time.Duration](0)
      SigDuration      = state.NewSignal[time.Duration](0)
      SigVolume        = state.NewSignal[float32](0.8)

      // Queue Signals
      SigQueue         = state.NewSignal[[]ytm.Track](nil)
      SigQueueIndex    = state.NewSignal[int](-1)
      SigShuffle       = state.NewSignal[bool](false)
      SigRepeatMode    = state.NewSignal[RepeatMode](RepeatOff)
  )

  type RepeatMode int
  const (
      RepeatOff RepeatMode = iota
      RepeatAll
      RepeatOne
  )
  ```

### Step 3.2: Queue Orchestrator
* Implement queue manipulation:
  * `PlayTrack(track ytm.Track)`: Sets queue to single track and starts playback.
  * `Enqueue(tracks ...ytm.Track)`: Appends tracks to current queue.
  * `PlayNext(track ytm.Track)`: Inserts track immediately after current index.
  * `Next()`: Advances to next index (or loops if `RepeatAll`, re-plays if `RepeatOne`).
  * `Previous()`: If `Position() > 3s`, seek to 0; otherwise go to previous index.
  * Auto-fetch: If `QueueIndex >= len(Queue) - 2`, trigger `ytm.GetNextRadio` to append similar tracks automatically.

**Verification Criteria:**
* Queue unit tests validating:
  * Sequential playback.
  * Repeat mode behaviors (Off, All, One).
  * Shuffle order retention and un-shuffling.

---

## Module 4: GoGPU Metrolist Navigation Shell (`ui/appui`)

**Objective:** Build the modern Metrolist-styled desktop application frame with responsive navigation and a sticky bottom player.

### Step 4.1: Layout Geometry & Hierarchy
* Layout structure:
  ```
  +-------------------------------------------------------------+
  |  Sidebar (240px)   |  Top Header (Search + Window controls) |
  |  - Brand / Logo    |----------------------------------------|
  |  - Home            |  Content Viewport (RouterView)         |
  |  - Explore         |  (Scrollable Page Content)             |
  |  - Library         |                                        |
  |  - Queue           |                                        |
  |  - Settings        |                                        |
  +--------------------+----------------------------------------+
  |  Persistent Mini-Player Bar (Height: 76px, Full Width)     |
  +-------------------------------------------------------------+
  ```
* Implement responsive collapsible sidebar:
  * Full width: 240px (icon + label).
  * Compact mode: 72px (icon only) when window width < 900px.

### Step 4.2: Client-side Routing Map
* Define routes in `chordly/router`:
  * `/`: Home (Quick picks, New releases)
  * `/explore`: Trending, Top charts, Moods & Genres
  * `/search`: Search input, filters, and result lists
  * `/album/:id`: Album details & tracklist
  * `/artist/:id`: Artist profile, top songs, discography
  * `/queue`: Active playback queue and upcoming tracks
  * `/settings`: Audio configuration, appearance, plugins

**Verification Criteria:**
* Navigate through each route; verify page swaps without layout thrashing or memory leaks.

---

## Module 5: Core Views & Bottom Player Widget (`ui/views`)

**Objective:** Build user-facing screens adhering to Metrolist's clean typography and Material aesthetic.

### Step 5.1: Persistent Bottom Mini-Player (`bottom_player.go`)
* Subscribes to `SigCurrentTrack`, `SigPlaybackState`, and `SigPosition`.
* Components:
  1. **Left section (Now Playing metadata):**
     * Rounded album cover thumbnail (52x52).
     * Track title (bold, 14pt, truncated).
     * Artist name (muted, 12pt).
     * Favorite heart button.
  2. **Center section (Playback Controls & Scrubber):**
     * Shuffle, Previous, Play/Pause toggle, Next, Repeat.
     * Seekbar slider showing elapsed time (`mm:ss`) / total time (`mm:ss`).
     * Real-time drag support for seeking.
  3. **Right section (Utilities):**
     * Queue toggle button.
     * Lyrics toggle button.
     * Volume icon with mini volume slider.

### Step 5.2: Home View (`home.go`)
* Displays personalized recommendations:
  * "Quick Picks" grid: 4 columns x 3 rows of track cards.
  * "Listen Again" horizontal scrolling carousel.
  * "Top Charts" section.
* Clicking any track initiates playback and populates the auto-radio queue.

### Step 5.3: Search View (`search.go`)
* Search input bar with debounce (300ms) to prevent rate limits.
* Filter pills row: `All`, `Songs`, `Albums`, `Artists`, `Playlists`.
* Track List widget:
  * Index / Play button on hover.
  * Thumbnail, Title, Artist, Album, Duration.
  * Context menu: "Play Next", "Add to Queue", "Go to Artist", "Go to Album".

### Step 5.4: Queue View (`queue.go`)
* Shows currently playing track at top with animated soundwave indicator.
* Drag-and-drop or reorderable list of upcoming tracks.
* "Clear Queue" and "Save as Playlist" buttons.

**Verification Criteria:**
* User can open app -> search for a song -> click to play -> audio starts -> mini-player updates -> queue populated -> scrubbing works smoothly.
