# Chordly Architecture & Technical Design

## 1. System Overview

Chordly is a modern, high-performance desktop YouTube Music player built in **Go** and rendered with **GoGPU UI** (WebGPU/wgpu-based native desktop UI). The interface and feature set are modeled after **Metrolist** (Material You aesthetic, fluid animations, persistent bottom mini-player, queue drawer, and lyrics overlay).

The system follows clean architecture principles, decoupling network ingestion (InnerTube API), audio decoding & playback, reactive state management (`github.com/gogpu/ui/state`), and GoGPU rendering.

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                             GoGPU App Shell                            │
 ├───────────────────┬────────────────────────────────────────────────────┤
 │  Metrolist Nav    │  Main Viewport (RouterView)                        │
 │  - Home           │  - Quick Picks / Trending Grid                     │
 │  - Search         │  - Search Results (Songs, Albums, Artists)         │
 │  - Library        │  - Album / Playlist Details                        │
 │  - Settings       │  - Fullscreen Player & Synced Lyrics               │
 ├───────────────────┴────────────────────────────────────────────────────┤
 │  Persistent Mini-Player Bar (Track info, Seekbar, Controls, Dynamic)   │
 └────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │              Reactive Core State (github.com/gogpu/ui/state)           │
 │   Signals: CurrentTrack | PlaybackState | Position | Theme | Queue     │
 └──────┬───────────────────────┬─────────────────────────┬───────────────┘
        ▼                       ▼                         ▼
 ┌───────────────┐      ┌───────────────┐         ┌───────────────┐
 │ Audio Engine  │      │ InnerTube YTM │         │ Plugin Engine │
 │ Stream Buffer │      │ Client        │         │ & Hooks       │
 │ Oto / Malgo   │      │ WEB_REMIX     │         │ WASM / RPC    │
 └───────────────┘      └───────────────┘         └───────────────┘
```

---

## 2. Core Layers & Responsibilities

### 2.1 Backend: InnerTube Service (`internal/ytm`)
- Interfaces directly with YouTube Music's private API endpoints using the `WEB_REMIX` client context.
- Responsible for:
  - Catalog search (songs, albums, artists, playlists).
  - Home feed generation (Quick picks, Listen again, Trending).
  - Audio stream URL resolution and cipher/n-parameter deciphering.
  - Radio/queue generation (`/next` endpoint).

### 2.2 Playback: Low-Latency Audio Engine (`internal/player`)
- Decodes and outputs audio streams to the host operating system's sound server (ALSA, PulseAudio, PipeWire, CoreAudio, WASAPI).
- Provides an asynchronous ring buffer (`RingReader`) for stream pre-buffering.
- Exposes playback controls: Play, Pause, Seek, Volume, and real-time position reporting.

### 2.3 State & Reactive Signals (`internal/state`)
- Built on `github.com/gogpu/ui/state` (`Signal[T]`).
- State mutations automatically invalidate dependent UI widgets, triggering targeted redraws in GoGPU.
- Core signals:
  - `SigCurrentTrack`: Metadata of currently playing song.
  - `SigPlaybackState`: Enum (`Stopped`, `Playing`, `Paused`, `Buffering`).
  - `SigPlayhead`: Current playback position (`time.Duration`).
  - `SigDuration`: Total duration (`time.Duration`).
  - `SigVolume`: Master volume level (`0.0` to `1.0`).
  - `SigQueue`: Current playlist / queue.
  - `SigActiveTheme`: Active color palette and styling tokens.

### 2.4 Presentation: GoGPU UI (`ui/`)
- Pure Go WebGPU declarative UI hierarchy.
- **Shell Layout**: Left navigation rail, top search bar, dynamic content area, and a persistent bottom transport bar.
- **Dynamic Theming**: Color extraction from album artwork generates an adaptive Material 3 color palette that smoothly tints the UI.

### 2.5 Extensibility: Plugin Host (`internal/plugin`)
- Sandboxed plugin runtime (WebAssembly via `wazero` or Go interfaces).
- Exposes event hooks for:
  - Track lifecycle events (`OnTrackStart`, `OnTrackEnd`, `OnPlaybackStateChange`).
  - Lyrics providers (Musixmatch, Genius, LRCLIB).
  - System integrations (Discord RPC, Linux MPRIS / D-Bus media keys, Last.fm scrobbler).

---

## 3. Directory Layout

```
chord/
├── cmd/
│   └── chordly/
│       └── main.go                 # Application entry point
├── internal/
│   ├── audio/                      # Audio decoding & hardware playback
│   │   ├── engine.go
│   │   ├── decoder.go
│   │   └── buffer.go
│   ├── cache/                      # Stream chunk cache & offline storage
│   │   ├── disk_cache.go
│   │   └── db.go                   # SQLite persistence (favorites, history)
│   ├── lyrics/                     # LRC parser & lyrics retrieval
│   │   ├── parser.go
│   │   └── provider.go
│   ├── plugin/                     # Plugin loader & sandbox host
│   │   ├── host.go
│   │   ├── hooks.go
│   │   └── wasm.go
│   ├── state/                      # Global reactive signals
│   │   ├── player_state.go
│   │   ├── queue_state.go
│   │   └── theme_state.go
│   └── ytm/                        # YouTube Music InnerTube client
│       ├── client.go
│       ├── search.go
│       ├── browse.go
│       └── stream.go
├── ui/
│   ├── appui/                      # Shell, layout, routing widgets
│   │   ├── shell.go
│   │   ├── routerview.go
│   │   └── bottom_player.go
│   ├── theme/                      # Theming, color palettes, dynamic extraction
│   │   ├── colors.go
│   │   ├── extract.go              # Dominant color & palette derivation
│   │   └── theme.go
│   └── views/                      # Feature views
│       ├── home.go
│       ├── search.go
│       ├── album.go
│       ├── queue.go
│       ├── lyrics.go
│       └── settings.go
├── docs/                           # Architectural specs & implementation plans
│   ├── README.md
│   ├── architecture.md
│   ├── plan-1-basic-capabilities.md
│   └── plan-2-advanced-extensibility.md
├── go.mod
└── go.sum
```
