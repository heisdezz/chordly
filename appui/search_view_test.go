package appui

import (
	"testing"

	"github.com/gogpu/ui/geometry"

	"chordly/internal/model"
)

func TestSearchViewWidget(t *testing.T) {
	w := NewSearchView()
	if w == nil {
		t.Fatal("expected non-nil widget from NewSearchView")
	}

	// Test layout
	constraints := geometry.Constraints{MaxWidth: 800, MaxHeight: 600}
	size := w.Layout(nil, constraints)
	if size.Width < 0 || size.Height < 0 {
		t.Errorf("unexpected size from SearchView layout: %v", size)
	}

	children := w.Children()
	if len(children) == 0 {
		t.Fatal("expected SearchView to have children")
	}
}

func TestSearchCardBuilders(t *testing.T) {
	testTracks := []model.Track{
		{
			ID:          "trk1",
			Title:       "Midnight City",
			Artist:      "M83",
			Album:       "Hurry Up, We're Dreaming",
			DurationStr: "4:03",
			Thumbnail:   "https://lh3.googleusercontent.com/test1",
		},
		{
			ID:          "trk2",
			Title:       "Wait",
			Artist:      "M83",
			Album:       "Hurry Up, We're Dreaming",
			DurationStr: "5:43",
			Thumbnail:   "https://lh3.googleusercontent.com/test2",
		},
		{
			ID:          "trk3",
			Title:       "Starboy",
			Artist:      "The Weeknd",
			Album:       "Starboy",
			DurationStr: "3:50",
			Thumbnail:   "https://lh3.googleusercontent.com/test3",
		},
	}

	// 1. Top result card
	topResult := buildTopResultCard(testTracks[0], func() {})
	if topResult == nil {
		t.Fatal("expected non-nil top result card")
	}

	// 2. Track row
	row := buildSearchTrackRow(0, testTracks[0], func() {})
	if row == nil {
		t.Fatal("expected non-nil search track row")
	}

	// 3. Artist card
	artistCard := buildArtistCard("M83", 2, "https://lh3.googleusercontent.com/test1", func() {})
	if artistCard == nil {
		t.Fatal("expected non-nil artist card")
	}

	// 4. Album card
	albumCard := buildAlbumCard("Starboy", []model.Track{testTracks[2]}, 1, func() {})
	if albumCard == nil {
		t.Fatal("expected non-nil album card")
	}

	// 5. Combined results
	combined := buildCombinedSearchResults(testTracks)
	if len(combined) == 0 {
		t.Fatal("expected non-empty combined search results")
	}
}
