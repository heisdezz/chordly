package appui

import (
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"chordly/internal/lyrics"
	"chordly/internal/model"
	appstate "chordly/internal/state"
)

func TestLyricsViewWidgetLayout(t *testing.T) {
	w := NewLyricsViewWidget()
	constraints := geometry.Constraints{
		MinWidth:  200,
		MaxWidth:  500,
		MinHeight: 300,
		MaxHeight: 600,
	}

	size := w.Layout(nil, constraints)
	if size.Width != 500 {
		t.Errorf("expected width 500, got %f", size.Width)
	}
	if size.Height != 600 {
		t.Errorf("expected height 600, got %f", size.Height)
	}
}

func TestLyricsViewWidgetSeekOnClick(t *testing.T) {
	w := NewLyricsViewWidget()
	w.SetBounds(geometry.Rect{
		Min: geometry.Pt(0, 0),
		Max: geometry.Pt(400, 500),
	})

	doc := &lyrics.LyricsDocument{
		Title:    "Test Song",
		Artist:   "Test Artist",
		IsSynced: true,
		Lines: []lyrics.LyricLine{
			{Timestamp: 10 * time.Second, Text: "Line 0"},
			{Timestamp: 20 * time.Second, Text: "Line 1"},
			{Timestamp: 30 * time.Second, Text: "Line 2"},
			{Timestamp: 40 * time.Second, Text: "Line 3"},
		},
	}
	ActiveLyricsDoc.Set(doc)
	appstate.Playhead.Set(0)

	// Line height is 52px. Lines start at bounds.Min.Y + 20 (so Y = 20 to 72 is Line 0, 72 to 124 is Line 1, 124 to 176 is Line 2)
	// Click inside Line 2 at Y = 140
	clickPos := geometry.Pt(50, 140)
	clickEvent := event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		0,
		clickPos,
		clickPos,
		0,
	)

	handled := w.Event(nil, clickEvent)
	if !handled {
		t.Fatalf("expected click to be handled")
	}

	// Verify playhead was set to 30s (timestamp of Line 2)
	currentPlayhead := appstate.Playhead.Get()
	if currentPlayhead != 30*time.Second {
		t.Errorf("expected playhead to seek to 30s, got %v", currentPlayhead)
	}
}

func TestLyricsViewWidgetWheelScroll(t *testing.T) {
	w := NewLyricsViewWidget()
	w.SetBounds(geometry.Rect{
		Min: geometry.Pt(0, 0),
		Max: geometry.Pt(400, 500),
	})

	wheelPos := geometry.Pt(100, 100)
	wheelEvent := event.NewWheelEvent(
		geometry.Pt(0, 2), // scroll down
		wheelPos,
		wheelPos,
		0,
	)

	handled := w.Event(nil, wheelEvent)
	if !handled {
		t.Fatalf("expected wheel event to be handled")
	}

	if w.scrollY <= 0 {
		t.Errorf("expected scrollY > 0 after wheel scroll, got %f", w.scrollY)
	}
}

func TestNewLyricsOverlay(t *testing.T) {
	overlay := NewLyricsOverlay()
	if overlay == nil {
		t.Fatal("expected NewLyricsOverlay to return non-nil widget")
	}
}

func TestLyricsViewWidgetMountSubscriptions(t *testing.T) {
	old := appstate.CurrentTrack.Get()
	appstate.CurrentTrack.Set(model.Track{})
	t.Cleanup(func() { appstate.CurrentTrack.Set(old) })

	w := NewLyricsViewWidget()
	w.Mount(widget.NewContext())

	w.mu.Lock()
	mounted := len(w.unsubs)
	w.mu.Unlock()
	if mounted != 4 {
		t.Fatalf("subscriptions after Mount = %d, want 4", mounted)
	}

	w.Unmount()
	w.mu.Lock()
	remaining := len(w.unsubs)
	w.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("subscriptions after Unmount = %d, want 0", remaining)
	}
}
