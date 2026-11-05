# InnerTune Services Architecture & Implementation Plan

This document details the architecture and implementation roadmap for the **InnerTune Services Layer** (`internal/ytm`), providing communication with YouTube's private **InnerTube API** (powering YouTube Music, InnerTune, and Metrolist).

---

## 1. Architecture & Service Overview

```
                     ┌──────────────────────────────────────────────┐
                     │               Chordly UI / Player            │
                     └──────────────────────┬───────────────────────┘
                                            │
                                            ▼
                     ┌──────────────────────────────────────────────┐
                     │         InnerTune Services Facade            │
                     │          (internal/ytm/service.go)           │
                     └──┬─────────┬─────────┬─────────┬──────────┬──┘
                        │         │         │         │          │
   ┌────────────────────┘         │         │         │          └────────────────────┐
   ▼                              ▼         ▼         ▼                               ▼
┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌───────────────┐ ┌─────────────┐
│ SearchService │ │ BrowseService │ │ PlayerService │ │ QueueService  │ │LyricsService│
│ - Text Search │ │ - Home Feed   │ │ - Stream URLs │ │ - Radio (/next│ │ - YTM Lyrics│
│ - Autocomplete│ │ - Albums      │ │ - Itag Select │ │ - Up Next     │ │ - LRCLIB    │
│ - Filter Chips│ │ - Artists     │ │ - Cipher/n-sig│ │ - Related Mix │ │   Fallback  │
└───────┬───────┘ └───────┬───────┘ └───────┬───────┘ └───────┬───────┘ └──────┬──────┘
        │                 │                 │                 │                │
        └─────────────────┴────────┬────────┴─────────────────┴────────────────┘
                                   ▼
        ┌─────────────────────────────────────────────────────────────┐
        │            InnerTube HTTP Client (Base Engine)              │
        │ - Context: WEB_REMIX & ANDROID_MUSIC (No-throttling)        │
        │ - Session & Visitor Data Token Manager                      │
        │ - Optional Account Auth (SAPISIDHASH / Cookies)             │
        └─────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
        ┌─────────────────────────────────────────────────────────────┐
        │               Local Disk Audio Cache Proxy                  │
        │  - Stream chunk persistence (~/.cache/chordly/audio/)       │
        │  - Range request forwarding & LRU eviction                  │
        └─────────────────────────────────────────────────────────────┘
```

---

## 2. Core Service Interfaces (`internal/ytm/interfaces.go`)

```go
package ytm

import (
    "context"
    "io"
    "time"
    "chordly/internal/model"
)

// InnerTuneService is the unified facade exposed to the player and UI.
type InnerTuneService interface {
    Search() SearchService
    Browse() BrowseService
    Player() PlayerService
    Queue() QueueService
    Lyrics() LyricsService
    Auth() AuthService
}

type SearchService interface {
    GetSuggestions(ctx context.Context, query string) ([]string, error)
    Search(ctx context.Context, query string, filter SearchFilter) (*model.SearchResults, error)
}

type BrowseService interface {
    GetHomeFeed(ctx context.Context) (*HomeFeed, error)
    GetAlbum(ctx context.Context, browseID string) (*AlbumDetails, error)
    GetArtist(ctx context.Context, browseID string) (*ArtistDetails, error)
    GetPlaylist(ctx context.Context, browseID string) (*PlaylistDetails, error)
}

type PlayerService interface {
    ResolveStream(ctx context.Context, videoID string) (*AudioStream, error)
    OpenStreamReader(ctx context.Context, stream *AudioStream, startOffset int64) (io.ReadCloser, error)
}

type QueueService interface {
    GetRadio(ctx context.Context, videoID string) ([]model.Track, error)
    GetContinuation(ctx context.Context, continuationToken string) ([]model.Track, string, error)
}

type LyricsService interface {
    GetLyrics(ctx context.Context, track model.Track) (*LyricsResult, error)
}

type AuthService interface {
    SetUserCookie(cookie string) error
    IsLoggedIn() bool
    GetUserLibrary(ctx context.Context) (*UserLibrary, error)
}
```

---

## 3. Module-by-Module Specification

### Module 1: InnerTube Core Client Engine (`internal/ytm/client`)
**Objective:** Provide a resilient HTTP transport that injects the required client contexts, handles visitor tracking, and avoids bot checks.

* **Multi-Client Contexts:**
  YouTube Music protects or throttles certain endpoints depending on the client identifier. We implement dual-client context rotation:
  1. `WEB_REMIX`:
     * Primary client for `browse`, `search`, `next`, and metadata.
     * `clientName: "WEB_REMIX"`, `clientVersion: "1.20241030.01.00"`, `hl: "en"`, `gl: "US"`.
  2. `ANDROID_MUSIC`:
     * Primary client for `player` stream resolution (audio streams served directly without JavaScript signature scrambling).
     * `clientName: "ANDROID_MUSIC"`, `clientVersion: "6.45.52"`, `androidSdkVersion: 34`.
* **Visitor Data Tracking:**
  * Read the `X-Goog-Visitor-Id` response header from the initial ping and persist it in memory.
  * Injects visitor ID into subsequent request payloads to avoid recurring CAPTCHA challenges.
* **Request Pipeline:**
  * Build a generic `Do(ctx, endpoint, payload, out)` helper supporting `gzip`/`brotli` decompression and JSON unmarshaling.

---

### Module 2: Search & Auto-complete Service (`internal/ytm/search`)
**Objective:** Live search, instant suggestions, and categorized filtering.

* **Search Suggestions / Autocomplete:**
  * Endpoint: `/youtubei/v1/music/get_search_suggestions`.
  * Input: `{ input: "query" }`.
  * Output: Slice of suggestion strings rendered in the search dropdown.
* **Filtered Catalog Search:**
  * Endpoint: `/youtubei/v1/search`.
  * Filter params:
    * `Songs`: `EgWKAQIIAWoKEAkQBRAKEAMQBA%3D%3D`
    * `Albums`: `EgWKAQIBAWoKEAkQBRAKEAMQBA%3D%3D`
    * `Artists`: `EgWKAQIgAWoKEAkQBRAKEAMQBA%3D%3D`
    * `Playlists`: `EgWKAQIoAWoKEAkQBRAKEAMQBA%3D%3D`
* **Parser Pipeline:**
  * Traverse `contents.tabbedSearchResultsRenderer.tabs[0].tabRenderer.content.sectionListRenderer.contents`.
  * Extract `musicResponsiveListItemRenderer` elements $\rightarrow$ convert into normalized `model.Track`, `model.Album`, `model.Artist`.

---

### Module 3: Browse Service (Home, Albums, Artists) (`internal/ytm/browse`)
**Objective:** Deliver personalized feeds, Trending showcases, and discographies.

* **Home Feed Parsing (`FEmusic_home`):**
  * Endpoint: `/youtubei/v1/browse` with `browseId: "FEmusic_home"`.
  * Extract rows:
    * **"Trending / Hero Banner"**: First large item with subtitle listener count.
    * **"Made for you"**: Quick picks and curated mix playlists (`musicCarouselShelfRenderer`).
    * **"New releases"**: Recent singles and albums.
* **Album & Playlist Tracklist (`MPREb_...` / `VL...`):**
  * Endpoint: `/youtubei/v1/browse` with `browseId: albumID`.
  * Extract: High-res cover art, release year, artist link, total track count, and ordered track items.
* **Artist Profile & Top Tracks:**
  * Endpoint: `/youtubei/v1/browse` with `browseId: artistID`.
  * Extract: Header banner, subscriber count, "Top Songs" section, "Albums", and "Singles".

---

### Module 4: Audio Stream Resolution & Deciphering (`internal/ytm/player`)
**Objective:** Extract the highest-fidelity audio stream with minimal latency.

* **Adaptive Format Selection:**
  * Endpoint: `/youtubei/v1/player`.
  * Prioritize audio-only formats:
    1. **Opus (itag 251):** `audio/webm; codecs="opus"`, sample rate 48kHz, average bitrate ~160kbps (best audio fidelity).
    2. **AAC (itag 140):** `audio/mp4; codecs="mp4a.40.2"`, sample rate 44.1kHz, bitrate ~128kbps (universal fallback).
* **Decipher & n-Parameter Handler:**
  * If the audio format provides `signatureCipher` instead of direct `url`:
    * Fetch YouTube's HTML5 player script (`base.js`).
    * Cache and execute the regex-extracted reverse/swap/slice transform function.
    * Compute the `n`-parameter transform to bypass bandwidth throttling.
  * *(Optimization: Querying with the `ANDROID_MUSIC` client bypasses `signatureCipher` entirely for 98% of music tracks).*

---

### Module 5: Queue & Auto-Mix Radio Service (`internal/ytm/queue`)
**Objective:** Keep music playing indefinitely using YouTube Music's algorithmic radio seeds.

* **Song Radio Seed (`/youtubei/v1/next`):**
  * Payload: `{ videoId: currentTrack.ID, isAudioOnly: true }`.
  * Parse `musicQueueRenderer`:
    * Current track context.
    * Immediate upcoming list (typically 25 tracks).
    * `continuation`: Token for endless scrolling.
* **Continuation Fetcher:**
  * When remaining queue $< 3$ tracks, trigger background continuation call with `continuationToken` to fetch the next batch of 25 tracks without interrupting playback.

---

### Module 6: Synchronized Lyrics Service (`internal/ytm/lyrics`)
**Objective:** Deliver real-time karaoke lyrics matching Metrolist's full-screen mode.

* **InnerTube Native Lyrics:**
  * Call `/youtubei/v1/next` $\rightarrow$ inspect `tabs` for `browseId` with lyrics tab.
  * Call `/youtubei/v1/browse` with that `browseId` $\rightarrow$ parse `musicDescriptionShelfRenderer`.
* **Synced LRCLIB Fallback:**
  * If InnerTube only provides static/unsynced text:
    * Query `https://lrclib.net/api/get?track_name={title}&artist_name={artist}&duration={duration_sec}`.
    * Parse returned synced `.lrc` timestamp data (`[01:24.50] lyric line`).
    * Return parsed `[]model.LyricLine` to the UI scrubber.

---

### Module 7: Local Audio Caching & Proxy Service (`internal/ytm/cache`)
**Objective:** Prevent re-downloading previously played tracks and enable offline mode.

* **Tiered File Cache:**
  * Cache location: `~/.cache/chordly/audio/{videoID}.webm`.
  * Implements `http.RoundTripper` or custom `io.ReadSeeker`:
    * While the player streams chunks from the web, stream chunks are simultaneously tee'd into the disk cache file.
    * Subsequent plays open the local file directly (0 network traffic, instant playback).
* **Quota Management (LRU):**
  * Configurable cache limit in settings (e.g. 2 GB).
  * Worker checks disk size on app startup and purges least-recently-accessed tracks if quota is exceeded.

---

## 4. Implementation Steps & Milestones

| Step | Task | Output File | Verification |
|---|---|---|---|
| **1** | Multi-client Base HTTP Client | `internal/ytm/client.go` | Successful ping to `/youtubei/v1/browse` returning HTTP 200 |
| **2** | Search & Auto-complete Service | `internal/ytm/search.go` | Querying "Rexha Mavrick" returns track list & suggestions |
| **3** | Browse Home Feed & Albums | `internal/ytm/browse.go` | Parsed "Trending" banner, Quick Picks, and New Releases |
| **4** | Player Stream Resolver | `internal/ytm/player.go` | Resolves playable Opus audio URL for a YouTube video ID |
| **5** | Queue / Next Radio Service | `internal/ytm/queue.go` | Returns 25 similar tracks for "Echoes of Tomorrow" |
| **6** | Lyrics Service (YTM + LRCLIB) | `internal/ytm/lyrics.go` | Returns synced timestamps for popular tracks |
| **7** | Local Disk Audio Cache | `internal/ytm/cache.go` | Second play reads 100% from disk without network fetch |
| **8** | Connect to GoGPU Reactive Signals | `internal/state/` | Clicking any song in UI triggers live stream resolution & playback |
