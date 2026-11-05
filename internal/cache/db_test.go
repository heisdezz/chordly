package cache

import (
	"context"
	"testing"
	"time"

	"chordly/internal/model"
)

func TestDBFavorites(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Initial check
	fav, err := db.IsFavorite(ctx, "song-1")
	if err != nil {
		t.Fatalf("IsFavorite failed: %v", err)
	}
	if fav {
		t.Errorf("expected not favorite initially")
	}

	// Add favorite
	err = db.AddFavorite(ctx, FavoriteItem{
		ID:          "song-1",
		Type:        "track",
		Title:       "Starboy",
		Artist:      "The Weeknd",
		DurationSec: 230,
	})
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	// Verify is favorite
	fav, err = db.IsFavorite(ctx, "song-1")
	if err != nil || !fav {
		t.Errorf("expected is favorite true, got %v (err: %v)", fav, err)
	}

	// List favorites
	list, err := db.ListFavorites(ctx, "track")
	if err != nil {
		t.Fatalf("ListFavorites failed: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Starboy" {
		t.Errorf("unexpected list: %+v", list)
	}

	// Remove favorite
	err = db.RemoveFavorite(ctx, "song-1")
	if err != nil {
		t.Fatalf("RemoveFavorite failed: %v", err)
	}
	fav, err = db.IsFavorite(ctx, "song-1")
	if err != nil || fav {
		t.Errorf("expected not favorite after remove")
	}
}

func TestDBPlaylists(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Create playlist
	pl, err := db.CreatePlaylist(ctx, "Night Drive", "Vibes for evening drives")
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	if pl.ID == "" || pl.Name != "Night Drive" {
		t.Fatalf("unexpected playlist: %+v", pl)
	}

	// Add tracks
	t1 := model.Track{ID: "t-1", Title: "Blinding Lights", Artist: "The Weeknd", Duration: 200 * time.Second}
	t2 := model.Track{ID: "t-2", Title: "After Hours", Artist: "The Weeknd", Duration: 360 * time.Second}

	if err := db.AddTrackToPlaylist(ctx, pl.ID, t1); err != nil {
		t.Fatalf("AddTrackToPlaylist 1 failed: %v", err)
	}
	if err := db.AddTrackToPlaylist(ctx, pl.ID, t2); err != nil {
		t.Fatalf("AddTrackToPlaylist 2 failed: %v", err)
	}

	// Get tracks
	tracks, err := db.GetPlaylistTracks(ctx, pl.ID)
	if err != nil {
		t.Fatalf("GetPlaylistTracks failed: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(tracks))
	}
	if tracks[0].Title != "Blinding Lights" || tracks[1].Title != "After Hours" {
		t.Errorf("tracks mismatch: %+v", tracks)
	}

	// List playlists
	playlists, err := db.ListPlaylists(ctx)
	if err != nil {
		t.Fatalf("ListPlaylists failed: %v", err)
	}
	if len(playlists) != 1 || playlists[0].ItemCount != 2 {
		t.Errorf("expected 1 playlist with 2 items, got %+v", playlists)
	}

	// Delete playlist
	if err := db.DeletePlaylist(ctx, pl.ID); err != nil {
		t.Fatalf("DeletePlaylist failed: %v", err)
	}
	playlistsAfter, _ := db.ListPlaylists(ctx)
	if len(playlistsAfter) != 0 {
		t.Errorf("expected 0 playlists after delete")
	}
}

func TestDBHistoryAndSettings(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Settings
	val, err := db.GetSetting(ctx, "theme_preset", "emerald")
	if err != nil || val != "emerald" {
		t.Errorf("expected default value 'emerald', got %s", val)
	}

	if err := db.SetSetting(ctx, "theme_preset", "amoled"); err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	val, err = db.GetSetting(ctx, "theme_preset", "emerald")
	if err != nil || val != "amoled" {
		t.Errorf("expected updated value 'amoled', got %s", val)
	}

	// History
	track := model.Track{ID: "h-1", Title: "Instant Crush", Artist: "Daft Punk", Duration: 337 * time.Second}
	if err := db.RecordPlay(ctx, track); err != nil {
		t.Fatalf("RecordPlay failed: %v", err)
	}

	history, err := db.GetRecentHistory(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentHistory failed: %v", err)
	}
	if len(history) != 1 || history[0].Track.Title != "Instant Crush" {
		t.Errorf("unexpected history: %+v", history)
	}

	// Clear history
	if err := db.ClearHistory(ctx); err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}
	historyAfter, _ := db.GetRecentHistory(ctx, 10)
	if len(historyAfter) != 0 {
		t.Errorf("expected 0 history items after clear")
	}
}
