# Task Specification & Resolution Note: TSK-218 (Full-Screen Player)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-218`  
**Parent Plan:** [`docs/ui/pages-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/pages-plan.md)  
**File Scope:**  
- [`appui/player_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/player_view.go)
- [`appui/player_view_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/player_view_test.go)
- [`appui/bottom_player.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/bottom_player.go)
- [`main.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/main.go)

---

## 1. Objective

Provide a dedicated full-screen now-playing view (`/player` route) expandable from the persistent bottom mini-player dock:
1. Large, prominent album artwork showcase with rounded geometry.
2. Full playback transport controls (Previous, Play/Pause, Next, Shuffle, Repeat).
3. Synced scrubber and timestamp indicators.
4. Volume slider widget and "Up next" peek indication.
5. Smooth navigation back to previous route or home.

---

## 2. Technical Design & Verification

- `PlayerPage`: Reactive container watching `state.CurrentTrack`.
- `PlayerArtwork`: Custom vector-drawn surface adhering to `widget.Widget` interface (`Layout`, `Draw`, `Event`, `Children`).
- Integrated into `main.go` route tree at `/player`.
- Unit tests in `appui/player_view_test.go` verifying layout and square aspect ratio constraints.

---

## 3. Status
- **Status:** `DONE`
- **Verification:** Unit tests passing (`go test ./appui/...`) and all packages building cleanly (`go build ./...`).

## 4. Mini-Player → Fullscreen Animation (Phase 1)

Implemented the expansion animation for the mini-player → full-screen player handoff:

- `appui/player_view.go` — new `playerScreen` wrapper around the full-screen panel:
  - Scrim fades in (opacity 0 → 0.55) over 200 ms.
  - Panel slides up from +28 px and fades in (opacity 0 → 1) over 250 ms, eased via `easeOutCubic`.
  - Animation clock stamped on first draw; frame loop via `widget.AnimationScheduler` / `ctx.Invalidate`.
  - Honors `motionReduced` (skip → instant final state).
- `appui/bottom_player.go` — new `miniPlayerExit` wrapper for the mini-player:
  - On `/player`, the mini-player surface is wrapped in `miniPlayerExit` (180 ms, fade + slide down).
  - Honors `motionReduced` (skip → immediate hide).
- Phase 2 (shared-element artwork morph + drag-to-dismiss) remains per `docs/ui/animations-plan.md` §4.

