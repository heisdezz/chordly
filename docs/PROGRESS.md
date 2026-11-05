# Multi-Agent Progress & Coordination Board

This document coordinates concurrent agent contributions on **Chordly** without collisions, race conditions, or merge conflicts.

---

## 1. Multi-Agent Concurrency Protocol

To allow multiple autonomous agents to work concurrently on the codebase:

### Rule 1: Atomic Task Claiming

Before creating or editing any files, an agent **MUST**:

1. Inspect the **Task Matrix** below.
2. Select an unblocked task in status `TODO`.
3. Update the task row:
   - Change status to `IN_PROGRESS`.
   - Add your identifier (e.g. `Agent-Audio`, `Agent-UI`, or conversation ID).
   - Note the start timestamp.
4. Verify that no other task in `IN_PROGRESS` shares the same files in **File Scope**.

### Rule 2: Strict File Boundary Isolation

- Agents **ONLY** create and edit files listed in their task's **File Scope**.
- **Shared Extension Points** (e.g. `internal/state/state.go`, `appui/shell.go`, `router/router.go`):
  - Do NOT edit shared files directly unless your task explicitly owns the `INTEGRATION` phase.
  - Workstream modules must export standalone constructors or register via interfaces.
  - Shared registrations should be wired in designated lifecycle hooks.

### Rule 3: Interface-Driven Decoupling

- Subsystems communicate through interfaces defined in `internal/model` and `internal/ytm/interfaces.go`.
- Never write direct hard dependencies between two in-progress modules.

### Rule 4: Verification Gate Before Completion

Before marking a task `DONE`:

1. Run `go build ./...` (must compile with 0 errors).
2. Run `go test ./...` (all unit/integration tests must pass).
3. Update task status to `DONE` with git commit hash or brief summary.

---

## 2. Workstream File Ownership Boundaries

Each workstream is isolated to its own subdirectory:

| Workstream  | Package Path                       | Allowed File Scope                     | Primary Responsibility                          |
| ----------- | ---------------------------------- | -------------------------------------- | ----------------------------------------------- |
| **WS-CORE** | `internal/model`, `internal/state` | `internal/model/*`, `internal/state/*` | Shared data types & reactive signals            |
| **WS-YTM**  | `internal/ytm`                     | `internal/ytm/*`                       | InnerTube API, search, browse, streams          |
| **WS-AUD**  | `internal/audio`                   | `internal/audio/*`                     | Audio playback, decoding (Opus/AAC), buffer     |
| **WS-THM**  | `internal/theme`, `ui/theme`       | `internal/theme/*`, `appui/theme_*`    | Palettes, custom colors, dynamic album art LERP |
| **WS-PLG**  | `internal/plugin`                  | `internal/plugin/*`                    | Wazero WASM host, MPRIS, Discord RPC, hooks     |
| **WS-LYR**  | `internal/lyrics`                  | `internal/lyrics/*`                    | LRC parsing, LRCLIB API, synced text            |
| **WS-DB**   | `internal/cache`                   | `internal/cache/*`                     | SQLite persistence, audio disk cache, LRU       |
| **WS-UI**   | `appui`, `ui/views`                | `appui/*`                              | GoGPU widgets, shell, pages, vector graphics    |

---

## 3. Master Task Matrix

### Legend

- `TODO`: Available to be claimed.
- `IN_PROGRESS [Agent-ID]`: Claimed and actively being developed.
- `BLOCKED [Dependency-ID]`: Waiting for another task to complete.
- `DONE`: Completed, tested, and verified.

---

### Phase 0: Foundation & UI Alignment (Completed)

| Task ID   | Workstream | Description                                                                        | File Scope                                                                                              | Status | Owner |
| --------- | ---------- | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ------ | ----- |
| `TSK-001` | WS-CORE    | Domain Models (`Track`, `MixCard`, `PlaybackState`)                                | `internal/model/types.go`                                                                               | `DONE` | Core  |
| `TSK-002` | WS-THM     | Reference Theme Palette (Slate `#0C1216`, Mint `#5CE1B6`)                          | `internal/theme/theme.go`                                                                               | `DONE` | Core  |
| `TSK-003` | WS-CORE    | Reactive Playback & Queue State Signals                                            | `internal/state/state.go`                                                                               | `DONE` | Core  |
| `TSK-004` | WS-UI      | Metrolist Shell Layout & Navigation Rail                                           | `appui/sidebar.go`, `appui/shell.go`                                                                    | `DONE` | Core  |
| `TSK-005` | WS-UI      | Top Search Bar & Action Buttons                                                    | `appui/topbar.go`                                                                                       | `DONE` | Core  |
| `TSK-006` | WS-UI      | Hero Banner (Trending showcase & Art)                                              | `appui/hero.go`                                                                                         | `DONE` | Core  |
| `TSK-007` | WS-UI      | "Made for you" & "New releases" Sections                                           | `appui/made_for_you.go`, `appui/new_releases.go`                                                        | `DONE` | Core  |
| `TSK-008` | WS-UI      | "Up next" Queue Panel & Mini-Player Dock                                           | `appui/up_next.go`, `appui/bottom_player.go`, `appui/scrubber.go`                                       | `DONE` | Core  |
| `TSK-009` | WS-UI      | **Vector Icons Engine** (replaces missing Unicode emojis with 2D GPU vectors)      | `appui/icons.go`, `appui/components.go`                                                                 | `DONE` | Core  |
| `TSK-010` | WS-YTM     | **InnerTube Client & Search** (Live `WEB_REMIX` search & `FEmusic_home` feed sync) | `internal/ytm/client.go`, `internal/ytm/search.go`, `internal/ytm/browse.go`, `internal/ytm/service.go` | `DONE` | Core  |

---

### Phase 1: Parallel Backend & Engine Streams (Active)

| Task ID   | Workstream | Description                                                                                               | File Scope                                                                           | Dependencies | Status | Owner        |
| --------- | ---------- | --------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ | ------------ | ------ | ------------ |
| `TSK-101` | **WS-AUD** | **Low-Latency Audio Output Driver**: Initialize sound server device (`oto/v3` or `malgo` at 48kHz stereo) | `internal/audio/device.go`, `internal/audio/device_test.go`                          | None         | `TODO` | -            |
| `TSK-102` | **WS-AUD** | **Opus/AAC Stream Demuxer & Decoder**: Pure Go audio decoder pipeline yielding PCM frames                 | `internal/audio/decoder.go`, `internal/audio/decoder_test.go`                        | `TSK-101`    | `TODO` | -            |
| `TSK-103` | **WS-AUD** | **Asynchronous Ring Buffer (`RingReader`)**: Pre-buffers HTTP chunks to eliminate playback jitter         | `internal/audio/buffer.go`, `internal/audio/buffer_test.go`                          | None         | `DONE` | Agent-Audio  |
| `TSK-104` | **WS-YTM** | **Stream URL & Signature Deciphering**: Resolve unthrottled audio stream URLs via InnerTube / yt-dlp      | `internal/ytm/player.go`, `internal/ytm/player_test.go`                              | None         | `DONE` | Agent-Stream |
| `TSK-105` | **WS-YTM** | **Endless Radio Queue (`/next`)**: Song seeds and continuation automix generator                          | `internal/ytm/queue.go`, `internal/ytm/queue_test.go`                                | None         | `DONE` | Agent-Queue  |
| `TSK-106` | **WS-LYR** | **Synchronized LRC Parser & LRCLIB Ingestion**: Parse timestamped lyrics into searchable slices           | `internal/lyrics/lrc.go`, `internal/lyrics/lrclib.go`, `internal/lyrics/lrc_test.go` | None         | `DONE` | Agent-Lyrics |
| `TSK-107` | **WS-DB**  | **SQLite Storage Engine**: Database schema for favorites, playlists, and play history                     | `internal/cache/db.go`, `internal/cache/db_test.go`                                  | None         | `TODO` | -            |
| `TSK-108` | **WS-DB**  | **Audio Chunk Disk Cache**: Transparent cache storing streamed `.webm` files to `~/.cache/chordly/audio/` | `internal/cache/disk_cache.go`, `internal/cache/disk_cache_test.go`                  | None         | `TODO` | -            |

---

### Phase 2: Theming & Visual Extensibility (Parallelizable)

| Task ID   | Workstream | Description                                                                                                   | File Scope                                                    | Dependencies | Status                          | Owner           |
| --------- | ---------- | ------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------- | ------------ | ------------------------------- | --------------- |
| `TSK-201` | **WS-THM** | **Album Art Color Quantizer**: K-Means / MMCQ extraction from 64x64 album art thumbnail                       | `internal/theme/extract.go`, `internal/theme/extract_test.go` | None         | `DONE`                          | Agent-THM       |
| `TSK-202` | **WS-THM** | **Material You Tone Derivation & LERP Engine**: Smooth 250ms color transition ticker                          | `internal/theme/dynamic.go`, `internal/theme/dynamic_test.go` | `TSK-201`    | `TODO`                          | -               |
| `TSK-203` | **WS-THM** | **Custom Theme Presets & Color Picker**: AMOLED, Material Light, Emerald, and Hex picker                      | `internal/theme/presets.go`                                   | None         | `TODO`                          | -               |
| `TSK-204` | **WS-UI**  | **Karaoke Synced Lyrics Overlay View**: Auto-scrolling lyrics with active line highlighting and seek-on-click | `appui/lyrics_view.go`                                        | `TSK-106`    | `IN_PROGRESS [Agent-UI-Lyrics]` | Agent-UI-Lyrics |
| `TSK-205` | **WS-UI**  | **Search Results Page with Filter Pills**: Categorized results (`Songs`, `Albums`, `Artists`, `Playlists`)    | `appui/search_view.go`                                        | None         | `DONE`                          | Agent-UI        |

---

### Phase 3: Plugins & System Integration

| Task ID   | Workstream | Description                                                                                       | File Scope                                                | Dependencies | Status | Owner |
| --------- | ---------- | ------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | ------------ | ------ | ----- |
| `TSK-301` | **WS-PLG** | **Wazero WASM Plugin Host**: Sandboxed runtime with host bindings (`log`, `fetch`, `track_event`) | `internal/plugin/host.go`, `internal/plugin/host_test.go` | None         | `TODO` | -     |
| `TSK-302` | **WS-PLG** | **Linux MPRIS & Media Keys Integration**: D-Bus interface `org.mpris.MediaPlayer2.chordly`        | `internal/plugin/mpris_linux.go`                          | None         | `TODO` | -     |
| `TSK-303` | **WS-PLG** | **Discord Rich Presence Integration**: Discord IPC socket bridge for track and time display       | `internal/plugin/discord.go`                              | None         | `TODO` | -     |
| `TSK-304` | **WS-PLG** | **Last.fm / ListenBrainz Scrobbler**: 50% elapsed playback scrobble dispatcher                    | `internal/plugin/scrobbler.go`                            | None         | `TODO` | -     |

---

## 4. Integration & Coordination Checkpoints

When an agent completes a task:

1. Ensure all tests in that task pass: `go test -v ./<package>/...`.
2. Commit changes with a message mentioning the Task ID: `git commit -m "[TSK-201] Implement album art K-Means color quantizer"`.
3. Update this file (`docs/PROGRESS.md`):
   - Move the task to `DONE`.
   - Unblock any dependent tasks (change `BLOCKED` to `TODO`).

---

## 5. Conflict Resolution Guidelines

- If two agents inadvertently modify the same file, the agent holding the official task claim in the matrix takes precedence.
- Configuration constants or global signals must be introduced as additive declarations rather than mutating existing signatures.
