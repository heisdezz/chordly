# Plan 2: Advanced Capabilities & Extensibility Plan

This implementation plan details the advanced capabilities mimicking **Metrolist**: custom color theming, dynamic album-art theming, a sandboxed plugin engine, synchronized lyrics, and offline caching.

---

## Module 6: Custom Theming Engine (`ui/theme`)

**Objective:** Provide a reactive, tokenized styling system supporting Material 3 color roles, dark/light modes, pure AMOLED black, and custom hex color overrides.

### Step 6.1: Semantic Color Tokens & Theme Struct
* Replace hardcoded styling values with semantic tokens:
  ```go
  type ColorScheme struct {
      Name             string
      IsDark           bool
      Canvas           widget.Color // Window background
      Surface          widget.Color // Card & panel backgrounds
      SurfaceVariant   widget.Color // Secondary card background
      Sidebar          widget.Color // Navigation sidebar
      Border           widget.Color // Dividers and borders
      Primary          widget.Color // Main accent (buttons, active tabs)
      OnPrimary        widget.Color // Text/icons on top of Primary
      Secondary        widget.Color // Accent soft / secondary pill
      OnSecondary      widget.Color
      TextPrimary      widget.Color // Main headings and labels
      TextMuted        widget.Color // Secondary/subtitle text
      Hover            widget.Color // Hover overlay
      Pressed          widget.Color // Click overlay
  }

  var SigActiveTheme = state.NewSignal[*ColorScheme](ThemeMetrolistDark)
  ```

### Step 6.2: Built-in Palette Presets
* Implement presets:
  1. **Metrolist Dark (Default):** Deep slate `#121316`, surface `#1E1F24`, accent `#6750A4`.
  2. **AMOLED Black:** Pure `#000000` canvas, dark charcoal cards, vibrant neon accents.
  3. **Material Light:** Clean `#F8F9FA` canvas, crisp cards, deep violet accents.
  4. **Nordic Blue:** Deep twilight `#1A1D24`, arctic blue accent `#88C0D0`.
  5. **Emerald Velvet:** Forest dark `#0F1715`, emerald accent `#10B981`.

### Step 6.3: Custom Color Overrides & Settings UI
* Provide custom hex color customization in the Settings page:
  * Primary Accent Picker.
  * Dark / Light / AMOLED toggle.
  * Contrast checking: automatically compute `OnPrimary` (black or white) based on relative luminance ($L = 0.2126R + 0.7152G + 0.0722B$) to preserve WCAG accessibility standards.

**Verification Criteria:**
* Switching presets in Settings triggers instant UI repaint with zero stutter.

---

## Module 7: Dynamic Album Art Theming (`ui/theme/dynamic`)

**Objective:** Extract the dominant color palette from the currently playing track's artwork and dynamically adapt the entire UI theme in real time.

```
 [Track Starts] ──► Fetch Album Art Bytes ──► Downsample to 64x64
                           │
                           ▼
               Quantization (K-Means / MMCQ)
                           │
                           ▼
                 Dominant Color Chosen
                           │
                           ▼
          Material Tone & Contrast Harmonization
                           │
                           ▼
             Smooth Color LERP (250ms Ticker)
                           │
                           ▼
               SigActiveTheme.Set(NewTheme)
```

### Step 7.1: Image Decoding & Color Quantization
* Implement high-speed color extraction:
  * Download track thumbnail bytes into memory.
  * Downsample image to `64x64` using `golang.org/x/image/draw.BiLinear` (executes in < 3ms).
  * Run **Modified Median Cut Quantization (MMCQ)** or **K-Means clustering** ($K = 5$) to obtain the top 5 dominant colors.
  * Filter out near-black ($L < 0.15$) and near-white ($L > 0.85$) clusters.
  * Rank candidates by a scoring formula:
    $$\text{Score} = \text{Saturation} \times 0.6 + \text{PopulationWeight} \times 0.4$$

### Step 7.2: Adaptive Palette Derivation
* Given the winning dominant color $C$:
  * **Primary Accent:** Tone-adjusted $C$ ensuring saturation $\ge 50\%$.
  * **Canvas & Surface:** Subtle desaturated tint of $C$ blended with dark slate (e.g. 92% dark slate + 8% dominant color).
  * **Mini-Player Bar:** Gradient or tinted background matching the album vibe.
  * **Text Colors:** High-contrast calculations for readability.

### Step 7.3: Smooth Animated Theme Transition
* To avoid jarring color jumps between songs:
  * Implement a 250ms color transition worker running at 60 FPS (15 frames).
  * Linearly interpolate (`Lerp`) each color token from `CurrentTheme` to `TargetTheme`.
  * Update `SigActiveTheme` per frame.

**Verification Criteria:**
* Skip between tracks with diverse album art (e.g. bright yellow album vs. dark red album). Verify palette transitions smoothly within 250ms without frame drops.

---

## Module 8: Plugin Architecture & Extensibility (`internal/plugin`)

**Objective:** A sandboxed, safe extension system allowing community plugins for lyrics, scrobbling, desktop integration, and audio enhancements.

### Step 8.1: Plugin Runtime: WebAssembly (WASM via `wazero`)
* Use `github.com/tetratelabs/wazero` (pure-Go WebAssembly runtime, zero CGO).
* Benefits:
  * Sandboxed memory safety (plugins cannot crash the host player or access files outside designated folders).
  * Multi-language support (plugins can be authored in Go, Rust, TypeScript, or C).

### Step 8.2: Plugin Manifest & Host API
* `plugin.json` manifest:
  ```json
  {
    "id": "com.chordly.discord-rpc",
    "name": "Discord Rich Presence",
    "version": "1.0.0",
    "permissions": ["network", "events:playback"]
  }
  ```
* Host API bindings exposed to plugins:
  * `host_log(level, msg)`
  * `host_http_fetch(url, options)`
  * `host_get_current_track()`
  * `host_register_lyrics_provider()`

### Step 8.3: First-Party Built-In Plugins
1. **Linux MPRIS & Media Keys:**
   * Exposes DBus service `org.mpris.MediaPlayer2.chordly`.
   * Enables system tray controls, keyboard media keys (Play/Pause/Next/Prev), and lock screen widgets on GNOME, KDE, and system bars.
2. **Discord Rich Presence:**
   * Shows currently playing song, artist, album art, and progress bar in Discord status.
3. **Last.fm & ListenBrainz Scrobbler:**
   * Updates "Now Playing" immediately; submits scrobble once 50% or 4 minutes of track has elapsed.
4. **SponsorBlock / Non-Music Silence Skipping:**
   * Fetches segment timestamps from SponsorBlock API and skips intro/outro silences.

**Verification Criteria:**
* Run plugin test suite: verify a failing/panicking plugin is quarantined without interrupting active audio playback.

---

## Module 9: Synchronized Karaoke Lyrics (`ui/views/lyrics`)

**Objective:** Real-time synchronized lyrics display with smooth scrolling and interactive timestamp seeking.

### Step 9.1: Lyrics Ingestion Pipeline
* Multi-source lyrics fallback chain:
  1. InnerTube Lyrics endpoint (`/browse` with lyrics tab).
  2. Public LRCLIB API (`https://lrclib.net/api/get` by track title, artist, duration).
  3. Active plugin providers (e.g. Musixmatch).

### Step 9.2: LRC Parser & Model
* Parse synchronized `.lrc` format:
  ```go
  type LyricLine struct {
      Timestamp time.Duration
      Text      string
  }

  type LyricsDocument struct {
      Lines    []LyricLine
      IsSynced bool
  }
  ```
* Implement binary search (`sort.Search`) to pinpoint the active line matching `SigPosition.Get()`.

### Step 9.3: GoGPU Animated Lyrics View
* Render full-screen or side-drawer lyrics:
  * Active line: Large font (22pt), bold, colored with `Primary` accent, full opacity ($1.0$).
  * Neighboring lines: Medium font (16pt), muted color, graduated opacity ($0.4$).
  * Auto-scroll: Smoothly center the active lyric in the viewport.
  * Click to seek: Clicking any lyric line calls `Player.Seek(line.Timestamp)`.

**Verification Criteria:**
* Test with sample LRC file: lyrics advance synchronously with audio time; clicking line 12 immediately jumps audio to that timestamp.

---

## Module 10: Local Caching, Offline Mode & SQLite Storage (`internal/cache`)

**Objective:** Offline playback, bandwidth conservation, and local library persistence.

### Step 10.1: SQLite Database (`internal/cache/db.go`)
* Use modern pure-Go SQLite (`modernc.org/sqlite`):
  * Tables:
    * `favorites`: Bookmarked tracks and albums.
    * `playlists` & `playlist_items`: Custom user playlists.
    * `playback_history`: Recently played tracks with timestamps.
    * `settings`: Key-value persistent preferences.

### Step 10.2: Audio Stream Disk Cache (`internal/cache/disk_cache.go`)
* Tiered audio cache:
  * When streaming audio over HTTP, concurrently write received chunks to cache directory:
    `~/.cache/chordly/audio/{video_id}.webm`
  * When `Load(track)` is invoked, check if cache file exists and is complete:
    * If present: play directly from local disk (instant start, zero network traffic).
    * If partial: resume download from byte offset.
* LRU Cache Eviction:
  * Configurable cache quota in Settings (e.g. 2 GB, 5 GB, 10 GB, Unlimited).
  * Oldest unplayed tracks automatically pruned when quota is exceeded.

**Verification Criteria:**
* Play track while online -> Verify cache file created -> Disconnect network -> Restart app -> Play track -> Plays smoothly offline.
