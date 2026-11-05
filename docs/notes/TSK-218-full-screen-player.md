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
