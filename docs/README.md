# Chordly - YouTube Music Desktop Player

A desktop YouTube Music client built in **Go** and rendered with **GoGPU UI**, inspired by the design and feature set of **Metrolist** and **InnerTune** (Material You aesthetics, persistent bottom player, queue drawer, and lyrics overlay).

---

## Documentation Index

1. [**Multi-Agent Progress & Coordination Board**](PROGRESS.md)
   - Real-time task board with atomic task claims and file lock scopes.
   - Concurrency protocol preventing multi-agent collisions and merge conflicts.
   - Clear workstream separation (`WS-AUD`, `WS-YTM`, `WS-THM`, `WS-PLG`, `WS-LYR`, `WS-DB`, `WS-UI`).

2. [**System Architecture & Technical Design**](architecture.md)
   - High-level layered architecture.
   - Component communication flow.
   - State management model (`github.com/gogpu/ui/state`).
   - Repository directory structure.

3. [**InnerTune Services Architecture & Implementation Plan**](innertune-services-plan.md)
   - **Core Engine:** Multi-client context (`WEB_REMIX` + `ANDROID_MUSIC`), visitor ID persistence.
   - **Search Service:** Instant autocomplete, categorized filter tokens (`Songs`, `Albums`, `Artists`, `Playlists`).
   - **Browse Service:** `FEmusic_home` feed, Hero Trending banner, Quick picks, Album & Artist pages.
   - **Player Service:** Adaptive audio format extraction (Opus 48kHz itag 251, AAC itag 140), cipher/n-sig handler.
   - **Queue & Radio Service:** Automix continuations, endless recommendations.
   - **Lyrics Service:** InnerTube native lyrics + synced LRCLIB fallback.
   - **Cache Service:** Tiered disk chunk cache, range request forwarding, LRU eviction.

4. [**Plan 1: Basic Capabilities Implementation Plan**](plan-1-basic-capabilities.md)
   - **Module 1: InnerTube & YouTube Music Backend:** API requests, search, browse, audio stream extraction, cipher resolver.
   - **Module 2: Audio Decoding & Playback Engine:** Low-latency audio streaming, ring-buffered playback, seek & volume controls.
   - **Module 3: Reactive State & Queue Management:** Reactive signals, auto-radio queue, repeat and shuffle logic.
   - **Module 4: Metrolist Navigation Shell:** Responsive sidebar, top search header, router configuration.
   - **Module 5: Core Views & Bottom Player:** Quick Picks Home grid, Search with filter pills, and persistent mini-player.

5. [**InnerTune Services Architecture & Implementation Plan**](innertune-services-plan.md)
   - **Core Engine:** Multi-client context (`WEB_REMIX` + `ANDROID_MUSIC`), visitor ID persistence.
   - **Search Service:** Instant autocomplete, categorized filter tokens (`Songs`, `Albums`, `Artists`, `Playlists`).
   - **Browse Service:** `FEmusic_home` feed, Hero Trending banner, Quick picks, Album & Artist pages.
   - **Player Service:** Adaptive audio format extraction (Opus 48kHz itag 251, AAC itag 140), cipher/n-sig handler.
   - **Queue & Radio Service:** Automix continuations, endless recommendations.
   - **Lyrics Service:** InnerTube native lyrics + synced LRCLIB fallback.
   - **Cache Service:** Tiered disk chunk cache, range request forwarding, LRU eviction.

6. [**Plan 2: Advanced Capabilities & Extensibility Plan**](plan-2-advanced-extensibility.md)
   - **Module 6: Custom Theming Engine:** Semantic color tokens, dark/light/AMOLED presets, custom hex color picker.
   - **Module 7: Dynamic Album Art Theming:** Fast color quantization, Material You palette derivation, smooth animated color transitions.
   - **Module 8: Plugin Architecture & Extensibility:** WASM runtime via `wazero`, event hooks, Linux MPRIS, Discord RPC, scrobbler, SponsorBlock.
   - **Module 9: Synchronized Karaoke Lyrics:** LRC parser, synchronized auto-scrolling lyrics view, seek-on-click.
   - **Module 10: Local Caching, Offline Mode & Database:** SQLite library storage, disk audio caching, LRU eviction.
     5

---

## Development Roadmap & Execution Phases

```
 Phase 1: Core Foundation && InnerTune ServicesInnerTune Services
 ├── 1. InnerTube Base HTTPBase HTTP Client && VisitorVisitor ManagerManager[DONE]
 ├── 2. Search & Browse Services (Live YTM Data) [DONE]
 ├── 3. Audio Player Stream ResolverStream Resolver (OpusOpus // AACAAC) [READY TO CLAIM]
 └── 44. Reactive State (Playback,Playback, QueueQueue & HistoryHistory) [DONE]

 Phase 2: User Interface (Metrolist Design)Design)
 ├── 5. Metrolist Navigation Rail & Top Bar [DONE]
 ├── 6. 2D Vector Icons Engine (Tofu/Emoji Fix) [DONE]
 ├── 7. Hero Banner, Made For You & New Releases Views [DONE]
 └── 8. Persistent Bottom Mini-Player Bar & QueueQueue DrawerDrawer [DONE]

 Phase 3: Theming & Visual Delight
 ├── 89. CustomTheming & Palette Palette Presets [READY TO CLAIM]
 └── 910. Dynamic Album Art Color Extraction (Real-time LERP) [READY TO CLAIM]

 Phase 4: Advanced Extensibility & Offline
 ├── 1011. Synchronized Karaoke Karaoke Lyrics View [READY TO CLAIM]
 ├── 1112. WebAssembly Plugin Host (Discord RPC, MPRIS, Scrobbler) [READY TO CLAIM]
 └── 1213. SQLite Local Library & Disk Audio Cache [READY TO CLAIM]
```
