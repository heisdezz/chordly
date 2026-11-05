# Task Specification & Architecture Note: TSK-216 (Library Shell)

**Workstream:** `WS-UI/DB`  
**Task ID:** `TSK-216` (also maps to `UI-PAGE-03` in `docs/ui/pages-plan.md`)  
**Parent Plan:** [`docs/ui/pages-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/pages-plan.md)  
**File Scope:**  
- [`appui/library_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/library_view.go)
- [`appui/library_tabs.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/library_tabs.go)
- [`appui/library_view_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/library_view_test.go)
- [`internal/cache/db.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/internal/cache/db.go) (read-only queries & singletons)
- [`internal/cache/disk_cache.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/internal/cache/disk_cache.go) (downloads enumeration)
- [`main.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/main.go) (route bindings)
**Status:** `IN_PROGRESS [Agent-Library-UI]`  
**Dependencies:** `TSK-214` (Sign-in & Auth UI), `TSK-107` (SQLite Storage Engine)

---

## 1. Executive Summary & Objective

The **Library Shell (`TSK-216`)** provides the user's primary catalog management center in Chordly. It unifies cloud-synchronized user content (Favorites, Playlists, Playback History) with local on-disk assets (Audio Cache / Downloads).

### Core Requirements
1. **Four Dedicated Subviews / Tabs:**
   - **Favorites (`/library/favorites`)**: Liked tracks, albums, and artists saved in local SQLite storage.
   - **Playlists (`/library/playlists`)**: Custom user playlists with track counts and last-updated timestamps.
   - **History (`/library/history`)**: Chronological log of played tracks with direct re-play actions.
   - **Downloads (`/library/downloads`)**: Local audio cache inspection, reporting cached file size, format (WebM/Opus), and offline availability.
2. **Guest Mode vs. Authenticated Gate:**
   - **Favorites**, **Playlists**, and **History** require YouTube Music authentication to sync. When unauthenticated (`state.UserAccount.Get() == ""` and `ActiveSession.Get() == nil`), these tabs display a clear **Guest Call-To-Action (CTA)** prompting the user to connect their YouTube Music session via `/login` (with `state.RouteReturn` set to `/library`).
   - **Downloads** is **100% offline and unauthenticated-safe**. Guests and signed-in users can browse, play, and inspect cached tracks without an account.
3. **Reactive GoGPU Layout:**
   - Standardized on `NewPageScaffold` with themed header, pill-style tab selector bar, and scrollable body.
   - Reactive re-renders upon authentication changes (`state.UserAccount`, `ActiveSession`) and tab switches.

---

## 2. Technical Design & Subsystems

### 2.1 Tab Navigation Architecture (`appui/library_tabs.go`)
- Defines `LibraryTab` enum:
  - `LibraryTabFavorites` (`"favorites"`)
  - `LibraryTabPlaylists` (`"playlists"`)
  - `LibraryTabHistory` (`"history"`)
  - `LibraryTabDownloads` (`"downloads"`)
- Global / view signal: `SigLibraryTab` tracks active selection.
- Pill selector widget renders tab items with active highlight, counts, and click handlers that update both the signal and the router location (e.g. `/library/favorites`).

### 2.2 Relational Data Integration (`internal/cache/db.go`)
- Utilizes `*cache.DB` queries:
  - `ListFavorites(ctx, itemType)`
  - `ListPlaylists(ctx)`
  - `GetRecentHistory(ctx, limit)`
  - `GetTrackMeta(ctx, trackID)`: Helper looking up track title, artist, duration, and thumbnail from history or favorites.
- Singleton accessors: `GetDefaultDB()` and `SetDefaultDB(db)` for lifecycle management and test mocking.

### 2.3 Audio Disk Cache Integration (`internal/cache/disk_cache.go`)
- Utilizes `*cache.DiskCache`:
  - `ListDownloaded()`: Enumerates completed `.webm` audio files on disk with video ID, byte size, and last modified timestamp.
  - Matches video IDs against SQLite track metadata to display song titles and artists instead of raw hashes when available.
  - Singleton accessors: `GetDefaultCache()` and `SetDefaultCache(dc)`.

### 2.4 Guest State CTA Card
- Clean card container with `theme.ColorSurfaceVariant` background and rounded corners.
- Message: "Sign in to sync your YouTube Music library"
- Description: "Connect your YouTube Music account via InnerTune session import to access your cloud playlists, liked songs, and synced history."
- Primary Action: "Connect YouTube Music" -> sets `RouteReturn.Set("/library")` and navigates to `/login`.
- Secondary Action: "Browse Offline Downloads" -> switches to `LibraryTabDownloads`.

---

## 3. Verification & Acceptance Criteria

- `go test ./appui -run TestLibrary` passes with 100% assertions covering guest states, authenticated lists, tab navigation, and empty states.
- `go test ./internal/cache -run Test` passes.
- `go build ./...` compiles cleanly with zero errors.
