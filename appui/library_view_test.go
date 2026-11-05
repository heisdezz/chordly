package appui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"chordly/internal/cache"
	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/ytm"
	"chordly/router"
)

func TestLibraryPageBuilds(t *testing.T) {
	r := router.New(&router.Route{Segment: "library", Name: "library"})

	routesToTest := []string{
		"/library",
		"/library/favorites",
		"/library/playlists",
		"/library/history",
		"/library/downloads",
	}

	for _, path := range routesToTest {
		ctx := router.RouteContext{
			Router: r,
			Match:  router.Match{Path: path},
		}
		w := LibraryPage(ctx)
		if w == nil {
			t.Fatalf("LibraryPage(%q) returned nil", path)
		}
	}
}

func TestLibraryGuestCTA(t *testing.T) {
	// Ensure guest mode (no account, no session)
	state.UserAccount.Set("")
	ActiveSession.Set(nil)

	r := router.New(&router.Route{Segment: "library", Name: "library"})
	db, err := cache.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()
	cache.SetDefaultDB(db)

	favWidget := renderLibraryFavorites(r, db)
	if favWidget == nil {
		t.Fatal("renderLibraryFavorites returned nil in guest mode")
	}

	playlistWidget := renderLibraryPlaylists(r, db)
	if playlistWidget == nil {
		t.Fatal("renderLibraryPlaylists returned nil in guest mode")
	}

	historyWidget := renderLibraryHistory(r, db)
	if historyWidget == nil {
		t.Fatal("renderLibraryHistory returned nil in guest mode")
	}
}

func TestLibraryDownloadsGuestAllowed(t *testing.T) {
	// Guest mode should still allow inspecting and rendering downloads
	state.UserAccount.Set("")
	ActiveSession.Set(nil)

	r := router.New(&router.Route{Segment: "library", Name: "library"})
	tempDir := t.TempDir()
	dc, err := cache.NewAt(filepath.Join(tempDir, "audio"), 0)
	if err != nil {
		t.Fatalf("NewAt failed: %v", err)
	}
	cache.SetDefaultCache(dc)

	db, err := cache.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()
	cache.SetDefaultDB(db)

	// Empty downloads
	w := renderLibraryDownloads(r, dc, db)
	if w == nil {
		t.Fatal("renderLibraryDownloads returned nil for empty downloads")
	}

	// Add a dummy cached file
	filePath := filepath.Join(dc.Dir(), "testVid12345.webm")
	if err := os.WriteFile(filePath, []byte("fake webm audio bytes"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	wWithFiles := renderLibraryDownloads(r, dc, db)
	if wWithFiles == nil {
		t.Fatal("renderLibraryDownloads returned nil with cached files")
	}
}

func TestLibraryAuthenticatedViews(t *testing.T) {
	// Authenticated mode
	state.UserAccount.Set("AudioLover")
	ActiveSession.Set(&ytm.TokenBundle{
		SAPISID: "test_sapisid_token",
	})
	defer func() {
		state.UserAccount.Set("")
		ActiveSession.Set(nil)
	}()

	r := router.New(&router.Route{Segment: "library", Name: "library"})
	db, err := cache.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()
	cache.SetDefaultDB(db)

	ctx := context.Background()

	// 1. Empty state tests
	wFavEmpty := renderLibraryFavorites(r, db)
	if wFavEmpty == nil {
		t.Fatal("renderLibraryFavorites empty returned nil")
	}

	wPlEmpty := renderLibraryPlaylists(r, db)
	if wPlEmpty == nil {
		t.Fatal("renderLibraryPlaylists empty returned nil")
	}

	wHistEmpty := renderLibraryHistory(r, db)
	if wHistEmpty == nil {
		t.Fatal("renderLibraryHistory empty returned nil")
	}

	// 2. Populate data
	err = db.AddFavorite(ctx, cache.FavoriteItem{
		ID:          "fav-song-1",
		Title:       "Blinding Lights",
		Artist:      "The Weeknd",
		Album:       "After Hours",
		DurationSec: 200,
	})
	if err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}

	pl, err := db.CreatePlaylist(ctx, "Synthwave Mix", "Retro 80s vibes")
	if err != nil {
		t.Fatalf("CreatePlaylist failed: %v", err)
	}
	_ = db.AddTrackToPlaylist(ctx, pl.ID, model.Track{
		ID:          "fav-song-1",
		Title:       "Blinding Lights",
		Artist:      "The Weeknd",
		Duration:    200 * time.Second,
		DurationStr: "3:20",
	})

	err = db.RecordPlay(ctx, model.Track{
		ID:          "fav-song-1",
		Title:       "Blinding Lights",
		Artist:      "The Weeknd",
		Duration:    200 * time.Second,
		DurationStr: "3:20",
	})
	if err != nil {
		t.Fatalf("RecordPlay failed: %v", err)
	}

	// 3. Render populated lists
	wFavPopulated := renderLibraryFavorites(r, db)
	if wFavPopulated == nil {
		t.Fatal("renderLibraryFavorites populated returned nil")
	}

	wPlPopulated := renderLibraryPlaylists(r, db)
	if wPlPopulated == nil {
		t.Fatal("renderLibraryPlaylists populated returned nil")
	}

	wHistPopulated := renderLibraryHistory(r, db)
	if wHistPopulated == nil {
		t.Fatal("renderLibraryHistory populated returned nil")
	}
}

func TestLibraryTabPills(t *testing.T) {
	r := router.New(&router.Route{Segment: "library", Name: "library"})
	bar := buildLibraryTabBar(r)
	if bar == nil {
		t.Fatal("buildLibraryTabBar returned nil")
	}

	SetLibraryTab(LibraryTabFavorites)
	if SigLibraryTab.Get() != LibraryTabFavorites {
		t.Errorf("expected tab favorites, got %v", SigLibraryTab.Get())
	}

	SetLibraryTab(LibraryTabDownloads)
	if SigLibraryTab.Get() != LibraryTabDownloads {
		t.Errorf("expected tab downloads, got %v", SigLibraryTab.Get())
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024 * 5, "5.0 MB"},
		{1024 * 1024 * 1024 * 2, "2.0 GB"},
	}

	for _, tt := range tests {
		got := formatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
