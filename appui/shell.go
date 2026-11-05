package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
	"chordly/router"
)

// NewShell builds the complete Metrolist YouTube Music application layout.
// The shell rebuilds when the active theme palette changes.
func NewShell(r *router.Router) widget.Widget {
	return newReactiveView(
		func() widget.Widget { return buildShell(r) },
		WatchSignal(theme.SigActivePalette.AsReadonly()),
	)
}

func buildShell(r *router.Router) widget.Widget {
	// Left vertical navigation rail
	rail := NewNavRail(r)

	// Top search & action header
	topBar := NewTopBar(r)

	// Main middle content area: the router swaps pages in this viewport.
	pageView := newRouterView(r)

	// Horizontal viewport with center scrollable content and right "Up next" queue
	viewport := primitives.HBox(
		primitives.Expanded(pageView),
		primitives.Box().Width(16),
		NewUpNextPanel(),
	).CrossAlign(primitives.CrossAxisStart)

	// Bottom persistent transport & scrubber dock
	bottomPlayer := NewBottomPlayer(r)

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
