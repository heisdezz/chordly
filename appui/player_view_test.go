package appui

import (
	"testing"

	"github.com/gogpu/ui/geometry"

	"chordly/router"
)

func TestPlayerPageBuilds(t *testing.T) {
	page := PlayerPage(router.RouteContext{})
	if page == nil {
		t.Fatal("PlayerPage returned nil")
	}
}

func TestPlayerArtworkLayoutIsSquare(t *testing.T) {
	art := NewPlayerArtwork()
	size := art.Layout(nil, geometry.Constraints{MaxWidth: 500, MaxHeight: 400})
	if size.Width != size.Height {
		t.Fatalf("artwork size = %v, want square", size)
	}
	if size.Width > 440 {
		t.Fatalf("artwork width = %v, want <= 440", size.Width)
	}
}
