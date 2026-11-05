# Chordly Project Notes & Multi-Agent Task Knowledge Base

This directory (`docs/notes/`) holds implementation blueprints, architectural decisions, schema specifications, and technical reference notes for tasks claimed from `docs/PROGRESS.md`.

---

## 1. Master Task Implementation Matrix

| Task ID      | Workstream | Task Title                                              | Documentation Note                                                                   | Status |
| :----------- | :--------- | :------------------------------------------------------ | :----------------------------------------------------------------------------------- | :----- |
| `TSK-101`    | **WS-AUD** | Low-Latency Audio Output Driver                         | [`TSK-101-audio-output-driver.md`](TSK-101-audio-output-driver.md)                   | `DONE` |
| `TSK-102`    | **WS-AUD** | Opus/AAC Stream Demuxer & Decoder                       | [`TSK-102-stream-decoder.md`](TSK-102-stream-decoder.md)                             | `DONE` |
| `TSK-103`    | **WS-AUD** | Asynchronous Ring Buffer (`RingReader`)                 | Reference: `internal/audio/buffer.go`                                                | `DONE` |
| `TSK-104`    | **WS-YTM** | Stream URL & Signature Deciphering                      | Reference: `internal/ytm/player.go`                                                  | `DONE` |
| `TSK-105`    | **WS-YTM** | Endless Radio Queue (`/next`)                           | Reference: `internal/ytm/queue.go`                                                   | `DONE` |
| `TSK-106`    | **WS-LYR** | Synchronized LRC Parser & LRCLIB Ingestion              | Reference: `internal/lyrics/lrc.go`                                                  | `DONE` |
| `TSK-107`    | **WS-DB**  | SQLite Storage Engine                                   | [`TSK-107-sqlite-storage-engine.md`](TSK-107-sqlite-storage-engine.md)               | `DONE` |
| `TSK-108`    | **WS-DB**  | Audio Chunk Disk Cache                                  | Reference: `internal/cache/disk_cache.go`                                            | `DONE` |
| `TSK-201`    | **WS-THM** | Album Art Color Quantizer                               | Reference: `internal/theme/extract.go`                                               | `DONE` |
| `TSK-202`    | **WS-THM** | Material You LERP Engine                                | Reference: `internal/theme/dynamic.go`                                               | `DONE` |
| `TSK-203`    | **WS-THM** | Custom Theme Presets & Color Picker                     | Reference: `internal/theme/presets.go`                                               | `DONE` |
| `TSK-204`    | **WS-UI**  | Karaoke Synced Lyrics                                   | Reference: `appui/lyrics_view.go`                                                    | `DONE` |
| `TSK-205`    | **WS-UI**  | Search Results View with Filter Pills                   | Reference: `appui/search_view.go`                                                    | `DONE` |
| `TSK-213`    | **WS-UI**  | GoGPU Core Component Adoption Plan                      | [`UI-INF-01-gogpu-core-adoption.md`](UI-INF-01-gogpu-core-adoption.md)               | `DONE` |
| `TSK-214`    | **WS-UI**  | Sign-In Page & InnerTune Import Flow                    | [`TSK-214-signin-page-innertune-import.md`](TSK-214-signin-page-innertune-import.md) | `DONE` |
| `TSK-215`    | **WS-UI**  | Account Identity & Session Health Page (`/account`)     | [`TSK-215-account-profile-page.md`](TSK-215-account-profile-page.md)                 | `DONE` |
| `TSK-218`    | **WS-UI**  | Full-Screen Player View (`/player`)                     | [`TSK-218-full-screen-player.md`](TSK-218-full-screen-player.md)                     | `DONE` |
| `TSK-219`    | **WS-UI**  | Motion & Loading Kit (Spinners, Skeletons, Transitions) | [`TSK-219-motion-loading-kit.md`](TSK-219-motion-loading-kit.md)                     | `DONE` |
| `TSK-221`    | **WS-UI**  | Toast Notification System                               | Reference: `appui/toast.go`                                                          | `DONE` |
| `TSK-301`    | **WS-PLG** | Wazero WASM Host                                        | Reference: `internal/plugin/host.go`                                                 | `DONE` |
| `TSK-302`    | **WS-PLG** | Linux MPRIS & Media Keys                                | [`TSK-302-mpris-linux-integration.md`](TSK-302-mpris-linux-integration.md)           | `DONE` |
| `TSK-303`    | **WS-PLG** | Discord Rich Presence                                   | Reference: `internal/plugin/discord.go`                                              | `DONE` |
| `TSK-304`    | **WS-PLG** | Last.fm / ListenBrainz Scrobbler                        | Reference: `internal/plugin/scrobbler.go`                                            | `DONE` |
| `UI-AUTH-01` | **WS-YTM** | Google Sign-In & InnerTube Token Bundle Parser          | [`UI-AUTH-01-google-signin-token.md`](UI-AUTH-01-google-signin-token.md)             | `DONE` |

---

## 2. Multi-Agent Concurrency Rules Summary

1. **Atomic Claiming:** Always update `docs/PROGRESS.md` to `IN_PROGRESS [Agent-ID]` with start timestamp before creating code files.
2. **File Scope Isolation:** Respect directory boundaries per workstream table in `PROGRESS.md`.
3. **Decoupled Interfaces:** Interact only via public domain models (`internal/model`) or interfaces (`internal/ytm/interfaces.go`).
4. **Verification Gate:** Verify with `go test ./...` and `go build ./...` before switching status to `DONE`.
