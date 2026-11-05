package appui

import (
	"context"
	"testing"
	"time"

	"chordly/internal/cache"
	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/router"
)

func TestQueuePageBuilds(t *testing.T) {
	r := router.New(&router.Route{Segment: "queue", Name: "queue"})
	ctx := router.RouteContext{
		Router: r,
		Match:  router.Match{Path: "/queue"},
	}

	// 1. Populated queue
	state.Queue.Set([]model.Track{
		{ID: "t-1", Title: "Song One", Artist: "Artist One", Duration: 210 * time.Second, DurationStr: "3:30"},
		{ID: "t-2", Title: "Song Two", Artist: "Artist Two", Duration: 180 * time.Second, DurationStr: "3:00"},
	})
	state.QueueIndex.Set(0)

	wPopulated := QueuePage(ctx)
	if wPopulated == nil {
		t.Fatal("QueuePage returned nil for populated queue")
	}

	// 2. Empty queue
	state.ClearQueue()
	wEmpty := QueuePage(ctx)
	if wEmpty == nil {
		t.Fatal("QueuePage returned nil for empty queue")
	}
}

func TestQueueReorderAndRemoveUI(t *testing.T) {
	tracks := []model.Track{
		{ID: "track-1", Title: "Track 1", Artist: "Artist 1"},
		{ID: "track-2", Title: "Track 2", Artist: "Artist 2"},
		{ID: "track-3", Title: "Track 3", Artist: "Artist 3"},
	}
	state.Queue.Set(tracks)
	state.QueueIndex.Set(0)

	// Move down
	state.MoveQueueItemDown(0)
	if state.Queue.Get()[0].ID != "track-2" || state.Queue.Get()[1].ID != "track-1" {
		t.Errorf("MoveQueueItemDown failed: %+v", state.Queue.Get())
	}

	// Move up
	state.MoveQueueItemUp(1)
	if state.Queue.Get()[0].ID != "track-1" || state.Queue.Get()[1].ID != "track-2" {
		t.Errorf("MoveQueueItemUp failed: %+v", state.Queue.Get())
	}

	// Remove item
	state.RemoveQueueItem(0)
	if len(state.Queue.Get()) != 2 || state.Queue.Get()[0].ID != "track-2" {
		t.Errorf("RemoveQueueItem failed: %+v", state.Queue.Get())
	}
}

func TestSaveQueueAsPlaylist(t *testing.T) {
	db, err := cache.OpenDB(":memory:")
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()
	cache.SetDefaultDB(db)

	tracks := []model.Track{
		{ID: "q-save-1", Title: "Midnight Drive", Artist: "The Weeknd", Duration: 220 * time.Second},
		{ID: "q-save-2", Title: "Starlight", Artist: "Dua Lipa", Duration: 200 * time.Second},
	}
	state.Queue.Set(tracks)
	state.QueueIndex.Set(0)

	saveQueueAsPlaylist()

	// Verify in database
	ctx := context.Background()
	playlists, err := db.ListPlaylists(ctx)
	if err != nil {
		t.Fatalf("ListPlaylists failed: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1 playlist created, got %d", len(playlists))
	}
	if playlists[0].ItemCount != 2 {
		t.Errorf("expected playlist item count 2, got %d", playlists[0].ItemCount)
	}

	plTracks, err := db.GetPlaylistTracks(ctx, playlists[0].ID)
	if err != nil {
		t.Fatalf("GetPlaylistTracks failed: %v", err)
	}
	if len(plTracks) != 2 || plTracks[0].ID != "q-save-1" || plTracks[1].ID != "q-save-2" {
		t.Errorf("unexpected tracks saved in playlist: %+v", plTracks)
	}
}

func TestUpNextPanel(t *testing.T) {
	state.Queue.Set([]model.Track{
		{ID: "up-1", Title: "First Up", Artist: "Artist Up", DurationStr: "2:45"},
		{ID: "up-2", Title: "Second Up", Artist: "Artist Up 2", DurationStr: "3:15"},
	})
	state.QueueIndex.Set(0)

	w := NewUpNextPanel()
	if w == nil {
		t.Fatal("NewUpNextPanel returned nil")
	}
}
