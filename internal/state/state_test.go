package state

import (
	"sync"
	"testing"
	"time"

	"chordly/internal/model"
)

func TestQueueOperations(t *testing.T) {
	tracks := []model.Track{
		{ID: "t-1", Title: "Track One", Artist: "Artist A", Duration: 180 * time.Second},
		{ID: "t-2", Title: "Track Two", Artist: "Artist B", Duration: 200 * time.Second},
		{ID: "t-3", Title: "Track Three", Artist: "Artist C", Duration: 220 * time.Second},
	}

	Queue.Set(tracks)
	QueueIndex.Set(0)

	// PlayQueueIndex
	PlayQueueIndex(1)
	if QueueIndex.Get() != 1 {
		t.Fatalf("expected QueueIndex 1, got %d", QueueIndex.Get())
	}
	if CurrentTrack.Get().ID != "t-2" {
		t.Fatalf("expected CurrentTrack t-2, got %s", CurrentTrack.Get().ID)
	}
	if PlaybackStatus.Get() != model.StatePlaying {
		t.Fatalf("expected StatePlaying, got %v", PlaybackStatus.Get())
	}

	// MoveQueueItemDown: move item 0 down to 1 (item 1 shifts to 0)
	// Currently playing item is at index 1 ("t-2"), so when item 0 moves to 1, "t-2" becomes index 0.
	MoveQueueItemDown(0)
	q := Queue.Get()
	if q[0].ID != "t-2" || q[1].ID != "t-1" {
		t.Fatalf("unexpected order after MoveQueueItemDown: 0=%s, 1=%s", q[0].ID, q[1].ID)
	}
	if QueueIndex.Get() != 0 {
		t.Fatalf("expected QueueIndex adjusted to 0, got %d", QueueIndex.Get())
	}

	// MoveQueueItemUp: move item 1 up to 0
	MoveQueueItemUp(1)
	q = Queue.Get()
	if q[0].ID != "t-1" || q[1].ID != "t-2" {
		t.Fatalf("unexpected order after MoveQueueItemUp: 0=%s, 1=%s", q[0].ID, q[1].ID)
	}
	if QueueIndex.Get() != 1 {
		t.Fatalf("expected QueueIndex adjusted back to 1, got %d", QueueIndex.Get())
	}

	// Move currently playing item (index 1) to index 2
	MoveQueueItem(1, 2)
	q = Queue.Get()
	if q[2].ID != "t-2" {
		t.Fatalf("expected t-2 at index 2, got %s", q[2].ID)
	}
	if QueueIndex.Get() != 2 {
		t.Fatalf("expected QueueIndex tracked to 2, got %d", QueueIndex.Get())
	}

	// Remove item before currently playing (remove index 0)
	RemoveQueueItem(0)
	q = Queue.Get()
	if len(q) != 2 || q[0].ID != "t-3" || q[1].ID != "t-2" {
		t.Fatalf("unexpected queue after removal: %+v", q)
	}
	if QueueIndex.Get() != 1 {
		t.Fatalf("expected QueueIndex adjusted to 1, got %d", QueueIndex.Get())
	}

	// AppendToQueue
	AppendToQueue(model.Track{ID: "t-4", Title: "Track Four", Artist: "Artist D"})
	q = Queue.Get()
	if len(q) != 3 || q[2].ID != "t-4" {
		t.Fatalf("expected 3 items with t-4 at end, got len=%d", len(q))
	}

	// ClearQueue
	ClearQueue()
	if len(Queue.Get()) != 0 || QueueIndex.Get() != -1 {
		t.Fatalf("expected empty queue and index -1, got len=%d idx=%d", len(Queue.Get()), QueueIndex.Get())
	}
}

func TestAppendToEmptyQueuePlaysFirst(t *testing.T) {
	ClearQueue()

	AppendToQueue(model.Track{ID: "solo", Title: "Solo Song"})
	if len(Queue.Get()) != 1 {
		t.Fatalf("expected len 1, got %d", len(Queue.Get()))
	}
	if QueueIndex.Get() != 0 || CurrentTrack.Get().ID != "solo" {
		t.Fatalf("expected playing first track, got idx=%d id=%s", QueueIndex.Get(), CurrentTrack.Get().ID)
	}
}

func TestContinuationTrigger(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	var lastPassed model.Track

	OnQueueNeedsContinuation = func(lastTrack model.Track) {
		lastPassed = lastTrack
		wg.Done()
	}
	defer func() {
		OnQueueNeedsContinuation = nil
	}()

	q := []model.Track{
		{ID: "q-1", Title: "One"},
		{ID: "q-2", Title: "Two"},
		{ID: "q-3", Title: "Three"},
	}
	Queue.Set(q)

	// Playing index 1 (which is >= len(q)-2 = 1) triggers continuation
	PlayQueueIndex(1)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if lastPassed.ID != "q-3" {
			t.Errorf("expected continuation called with last track q-3, got %s", lastPassed.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("continuation hook was not called within timeout")
	}
}

func TestTogglePlayLikeAndAliases(t *testing.T) {
	PlaybackStatus.Set(model.StateStopped)
	TogglePlay()
	if PlaybackStatus.Get() != model.StatePlaying {
		t.Errorf("expected StatePlaying, got %v", PlaybackStatus.Get())
	}
	TogglePlay()
	if PlaybackStatus.Get() != model.StatePaused {
		t.Errorf("expected StatePaused, got %v", PlaybackStatus.Get())
	}

	CurrentTrack.Set(model.Track{ID: "like-test", IsLiked: false})
	ToggleLike()
	if !CurrentTrack.Get().IsLiked {
		t.Errorf("expected IsLiked true")
	}
	ToggleLike()
	if CurrentTrack.Get().IsLiked {
		t.Errorf("expected IsLiked false")
	}
}
