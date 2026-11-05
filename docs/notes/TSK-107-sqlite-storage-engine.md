# Task Specification & Architecture Note: TSK-107 (SQLite Storage Engine)

**Workstream:** `WS-DB`  
**File Scope:** `internal/cache/db.go`, `internal/cache/db_test.go`  
**Dependencies:** None  
**Claimed Status:** `IN_PROGRESS [Agent-DB-SQL]`  
**Target Package:** `package cache` (`chordly/internal/cache`)  

---

## 1. Executive Summary & Objective

The **SQLite Storage Engine (`TSK-107`)** provides Chordly's persistent local relational storage for user library data, playback history, custom playlists, and application configuration. While `TSK-108` (`disk_cache.go`) manages raw audio binary chunks on disk (`~/.cache/chordly/audio/{videoID}.webm`), `TSK-107` delivers the structured data layer required for offline catalog browsing, favorites management, track metadata persistence, and playback resume.

---

## 2. Storage Location & Environment Strategy

Chordly follows the **XDG Base Directory Specification** on Linux with fallbacks across platforms:

| Data Type | Primary Path | Fallback Path |
| :--- | :--- | :--- |
| **Relational Database** | `$XDG_DATA_HOME/chordly/chordly.db` | `~/.local/share/chordly/chordly.db` |
| **Database WAL/SHM** | Same directory as `chordly.db` | Directory created with mode `0755` |
| **Audio Chunk Cache** | `$XDG_CACHE_HOME/chordly/audio/` | `~/.cache/chordly/audio/` |
| **In-Memory Test DB** | `file::memory:?cache=shared` | Clean per unit-test execution |

---

## 3. Pure-Go Engine Selection & Pragmas

To adhere to Chordly's zero-CGO requirement for cross-compilation across Linux, macOS, and Windows:
- **Driver:** `modernc.org/sqlite` (Pure Go translated C-SQLite, no GCC/Clang or CGO required).
- **Driver Name in Go:** `"sqlite"`
- **Essential PRAGMAs (executed on every database connection open):**
  ```sql
  PRAGMA journal_mode = WAL;          -- Write-Ahead Logging for high concurrency
  PRAGMA synchronous = NORMAL;        -- Safe and performant with WAL
  PRAGMA foreign_keys = ON;           -- Enforce relational integrity
  PRAGMA busy_timeout = 5000;         -- Wait up to 5s on lock contention before busy error
  PRAGMA cache_size = -20000;         -- 20 MB memory cache
  PRAGMA temp_store = MEMORY;         -- In-memory temporary tables and indexes
  ```

---

## 4. Database Schema (DDL)

### 4.1. Versioning & Migrations (`schema_migrations`)

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 4.2. Favorites Table (`favorites`)

Tracks, albums, or artists marked as favorites / liked by the user.

```sql
CREATE TABLE IF NOT EXISTS favorites (
    id            TEXT PRIMARY KEY,              -- video_id, browse_id, or composite
    item_type     TEXT NOT NULL DEFAULT 'track', -- 'track', 'album', 'artist', 'playlist'
    title         TEXT NOT NULL,
    artist        TEXT NOT NULL,
    artist_id     TEXT DEFAULT '',
    album         TEXT DEFAULT '',
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    thumbnail_url TEXT DEFAULT '',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata_json TEXT DEFAULT '{}'              -- Serialized model.Track for instant reconstruction
);

CREATE INDEX IF NOT EXISTS idx_favorites_item_type ON favorites(item_type);
CREATE INDEX IF NOT EXISTS idx_favorites_created_at ON favorites(created_at DESC);
```

### 4.3. Custom User Playlists (`playlists` & `playlist_tracks`)

```sql
CREATE TABLE IF NOT EXISTS playlists (
    id            TEXT PRIMARY KEY,              -- UUID or custom slug
    title         TEXT NOT NULL,
    description   TEXT DEFAULT '',
    thumbnail_url TEXT DEFAULT '',
    track_count   INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    playlist_id   TEXT NOT NULL,
    track_id      TEXT NOT NULL,
    position      INTEGER NOT NULL,              -- 0-indexed ordering for drag-and-drop / reordering
    title         TEXT NOT NULL,
    artist        TEXT NOT NULL,
    album         TEXT DEFAULT '',
    duration_ms   INTEGER NOT NULL DEFAULT 0,
    thumbnail_url TEXT DEFAULT '',
    added_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata_json TEXT DEFAULT '{}',
    FOREIGN KEY(playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_playlist_position ON playlist_tracks(playlist_id, position);
CREATE INDEX IF NOT EXISTS idx_playlist_track_id ON playlist_tracks(track_id);
```

### 4.4. Playback History Table (`playback_history`)

Maintains chronological listening logs with completion metrics for analytics and scrobbling.

```sql
CREATE TABLE IF NOT EXISTS playback_history (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id           TEXT NOT NULL,
    title              TEXT NOT NULL,
    artist             TEXT NOT NULL,
    album              TEXT DEFAULT '',
    duration_ms        INTEGER NOT NULL DEFAULT 0,
    listened_ms        INTEGER NOT NULL DEFAULT 0,
    completed          BOOLEAN NOT NULL DEFAULT 0,
    played_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata_json      TEXT DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_history_played_at ON playback_history(played_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_track_id ON playback_history(track_id);
```

### 4.5. Key-Value Settings Table (`settings`)

Persistent application preferences, player state, audio volume, and selected theme preset.

```sql
CREATE TABLE IF NOT EXISTS settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 5. Go API Interface Contracts (`internal/cache/db.go`)

```go
package cache

import (
    "context"
    "database/sql"
    "time"

    "chordly/internal/model"
)

type DB struct {
    sqlDB *sql.DB
    path  string
}

// OpenDB opens (and initializes schemas for) the SQLite database.
func OpenDB(path string) (*DB, error)

// Close releases the underlying SQLite database pool.
func (d *DB) Close() error

// Favorites API
func (d *DB) AddFavorite(ctx context.Context, track model.Track) error
func (d *DB) RemoveFavorite(ctx context.Context, trackID string) error
func (d *DB) IsFavorite(ctx context.Context, trackID string) (bool, error)
func (d *DB) ListFavorites(ctx context.Context, limit, offset int) ([]model.Track, error)
func (d *DB) CountFavorites(ctx context.Context) (int, error)

// Playlists API
type Playlist struct {
    ID           string        `json:"id"`
    Title        string        `json:"title"`
    Description  string        `json:"description"`
    ThumbnailURL string        `json:"thumbnail_url"`
    TrackCount   int           `json:"track_count"`
    CreatedAt    time.Time     `json:"created_at"`
    UpdatedAt    time.Time     `json:"updated_at"`
    Tracks       []model.Track `json:"tracks,omitempty"`
}

func (d *DB) CreatePlaylist(ctx context.Context, title, description string) (*Playlist, error)
func (d *DB) DeletePlaylist(ctx context.Context, playlistID string) error
func (d *DB) ListPlaylists(ctx context.Context) ([]Playlist, error)
func (d *DB) GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)
func (d *DB) AddTrackToPlaylist(ctx context.Context, playlistID string, track model.Track) error
func (d *DB) RemoveTrackFromPlaylist(ctx context.Context, playlistID string, position int) error
func (d *DB) ReorderPlaylistTrack(ctx context.Context, playlistID string, oldPos, newPos int) error

// Playback History API
type HistoryEntry struct {
    ID          int64         `json:"id"`
    Track       model.Track   `json:"track"`
    Listened    time.Duration `json:"listened"`
    Completed   bool          `json:"completed"`
    PlayedAt    time.Time     `json:"played_at"`
}

func (d *DB) RecordHistory(ctx context.Context, track model.Track, listened time.Duration, completed bool) error
func (d *DB) ListHistory(ctx context.Context, limit, offset int) ([]HistoryEntry, error)
func (d *DB) ClearHistory(ctx context.Context) error

// Settings API
func (d *DB) SetSetting(ctx context.Context, key, value string) error
func (d *DB) GetSetting(ctx context.Context, key, fallback string) (string, error)
func (d *DB) DeleteSetting(ctx context.Context, key string) error
```

---

## 6. Concurrency, Connection Pooling & Thread Safety

1. **Connection Pool Limits:**
   - SQLite handles multi-threaded reading well in WAL mode, but only **one** writer at any time.
   - Set `db.SetMaxOpenConns(max(4, runtime.NumCPU()))`.
   - Set `db.SetMaxIdleConns(2)`.
   - Set `db.SetConnMaxLifetime(time.Hour)`.
2. **Atomic Reordering in Playlists:**
   - Drag-and-drop playlist reordering must be wrapped in `db.BeginTx(ctx, &sql.TxOptions{})`.
   - Shifting positions `position >= newPos` uses temporary negative offsetting to prevent index collisions under unique constraint `idx_playlist_position`.
3. **Graceful Fallback:**
   - If SQLite database fails to open or is locked, `OpenDB` safely reports a detailed wrapped error without causing panics in upstream services.

---

## 7. Verification Gate Checklist

Before marking `TSK-107` as `DONE`:
- [ ] Run `go test -v -race ./internal/cache/...` with in-memory and temporary file databases.
- [ ] Verify foreign key cascade deletions (`DeletePlaylist` cleans up associated `playlist_tracks`).
- [ ] Verify thread safety with 20 parallel goroutines writing favorites and history simultaneously.
- [ ] Verify `go build ./...` compiles with 0 errors.
- [ ] Update `docs/PROGRESS.md` status from `IN_PROGRESS` to `DONE`.
