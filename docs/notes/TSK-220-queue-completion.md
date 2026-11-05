# Task Specification & Architecture Note: TSK-220 (Queue Completion)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-220`  
**Parent Plan:** [`docs/ui/pages-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/pages-plan.md)  
**File Scope:**  
- [`appui/queue_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/queue_view.go)
- [`appui/up_next.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/up_next.go)
- [`appui/queue_view_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/queue_view_test.go)
- [`internal/state/state.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/internal/state/state.go)
- [`internal/state/state_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/internal/state/state_test.go)
**Status:** `IN_PROGRESS [Agent-Queue-UI]`  
**Dependencies:** `TSK-217` (Detail pages & browse endpoints)

---

## 1. Executive Summary & Objective

The **Queue Completion (`TSK-220`)** deliverable transforms Chordly's playback queue into a fully dynamic, user-interactive playlist staging engine.

### Key Capabilities
1. **Queue Mutation & Reordering:**
   - Move tracks up and down in the queue with atomic preservation of the currently playing index.
   - Remove individual tracks from the queue.
   - Jump to any track index with immediate transport playback.
   - Clear entire queue with state reset.
2. **Save Queue as Playlist:**
   - Action in `/queue` header to persist the entire active queue into local SQLite relational storage (`*cache.DB`), calling `CreatePlaylist` and `AddTrackToPlaylist`.
   - Fires toast feedback (`ShowSuccess`) on completion.
3. **Automix Continuation Trigger:**
   - Detects when playback advances near the end of the queue (`index >= len(queue) - 2`).
   - Dispatches `OnQueueNeedsContinuation` hook to fetch endless radio automix continuation from InnerTube without interrupting playback.
4. **Active-Row Playing Indicator:**
   - Distinctive visual styling on active row in both `/queue` page and the `Up Next` sidebar panel:
     - Playing badge / active waveform vector icon.
     - Tinted accent background with subtle border.
     - Live duration and title accent color highlighting.
5. **Page Scaffold & Responsive Layout:**
   - Implemented in `appui/queue_view.go` using `NewPageScaffold`.
   - Synchronized with `NewUpNextPanel()` in `appui/up_next.go`.

---

## 2. State Operations (`internal/state/state.go`)

- `MoveQueueItem(from, to int)`: Swaps/shifts slice elements, updating `QueueIndex` if the currently playing song moved.
- `MoveQueueItemUp(index int)`: Convenience helper for moving item towards head.
- `MoveQueueItemDown(index int)`: Convenience helper for moving item towards tail.
- `RemoveQueueItem(index int)`: Removes item at index, adjusting `QueueIndex` as appropriate.
- `PlayQueueIndex(index int)`: Sets active index, sets `CurrentTrack`, resets playhead, and starts playback.
- `ClearQueue()`: Resets `Queue` to empty slice and `QueueIndex` to -1.
- `AppendToQueue(tracks ...model.Track)`: Extends the queue with new items.
- `OnQueueNeedsContinuation`: Global callback invoked when nearing end of queue.

---

## 3. UI Components

### 3.1 Full Queue View (`appui/queue_view.go`)
- **Header:**
  - Title: "Play Queue"
  - Summary: Total track count and cumulative queue duration.
  - Controls: "Save as Playlist" (primary tonal button) and "Clear Queue" (danger text button).
- **Track Rows:**
  - Numeric position label.
  - Active status indicator (play icon / waveform).
  - Track title, artist, and duration.
  - Action buttons: Move Up, Move Down, Remove, Play.

### 3.2 Up Next Panel (`appui/up_next.go`)
- Sidebar widget showing upcoming tracks with instant click-to-play, active-row glow, and "Open Full Queue" navigation button.

---

## 4. Verification Gate

- Unit tests in `internal/state/state_test.go` covering all reorder, remove, append, and index adjustment operations.
- Unit tests in `appui/queue_view_test.go` covering UI rendering, empty queue, active track highlight, and save-as-playlist action.
- Full verification with `go test ./...` and `go test -race ./...`.
