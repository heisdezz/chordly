package appui

import (
	"sync"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
	"chordly/router"
)

var (
	appRouterMu sync.RWMutex
	appRouter   *router.Router
)

// SetAppRouter sets the active global router for component navigation.
func SetAppRouter(r *router.Router) {
	appRouterMu.Lock()
	appRouter = r
	appRouterMu.Unlock()
}

// AppRouter returns the active application router if one has been registered.
func AppRouter() *router.Router {
	appRouterMu.RLock()
	defer appRouterMu.RUnlock()
	return appRouter
}

// Navigate routes to the specified path if an active router is set.
func Navigate(path string) {
	if r := AppRouter(); r != nil {
		r.Navigate(path)
	}
}

// NewShell builds the complete Metrolist YouTube Music application layout.
// The shell rebuilds when the active theme palette changes; the toast layer
// wraps it so notifications survive those rebuilds.
func NewShell(r *router.Router) widget.Widget {
	SetAppRouter(r)
	return NewToastHost(newReactiveView(
		func() widget.Widget { return buildShell(r) },
		WatchSignal(theme.SigActivePalette.AsReadonly()),
	))
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
