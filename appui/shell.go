package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
	"chordly/router"
)

// NewShell builds the complete Metrolist YouTube Music application layout.
func NewShell(r *router.Router) widget.Widget {
	// Left vertical navigation rail
	rail := NewNavRail()

	// Top search & action header
	topBar := NewTopBar()

	// Main middle content area:
	// - Hero showcase (Trending)
	// - "Made for you" curated mixes
	// - "New releases" track list
	centerContent := primitives.VBox(
		NewHeroBanner(),
		primitives.Box().Height(18),
		NewMadeForYouSection(),
		primitives.Box().Height(18),
		NewNewReleasesSection(),
	).CrossAlign(primitives.CrossAxisStretch)

	// Horizontal viewport with center scrollable content and right "Up next" queue
	viewport := primitives.HBox(
		primitives.Expanded(centerContent),
		primitives.Box().Width(16),
		NewUpNextPanel(),
	).CrossAlign(primitives.CrossAxisStart)

	// Bottom persistent transport & scrubber dock
	bottomPlayer := NewBottomPlayer()

	// Assembled right-hand main stage
	mainStage := primitives.VBox(
		topBar,
		primitives.Box().Height(6),
		primitives.Expanded(viewport),
		primitives.Box().Height(12),
		bottomPlayer,
	).
		PaddingXY(18, 12).
		CrossAlign(primitives.CrossAxisStretch)

	return primitives.HBox(
		rail,
		primitives.Expanded(mainStage),
	).Background(theme.ColorCanvas)
}
