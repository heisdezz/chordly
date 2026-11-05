package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/router"
)

// NewTopBar builds the top app bar containing the search field and utility buttons.
func NewTopBar(r *router.Router) widget.Widget {
	searchField := NewSearchField(
		"Search songs, artists, albums...",
		state.SearchQuery.Get(),
		func(value string) {
			state.SearchQuery.Set(value)
		},
		func(query string) {
			if query == "" {
				return
			}
			RunSearch(query)
			r.Navigate("/search")
		},
	)

	// Notification button (Vector Bell)
	bellBtn := NewVectorIconButton(IconBell, 38, 17, func() {
		ShowInfo("No new notifications")
	})

	// Fullscreen button (Vector Expand)
	expandBtn := NewVectorIconButton(IconExpand, 38, 16, func() {
		state.IsFullscreen.Set(!state.IsFullscreen.Get())
	})

	actions := primitives.HBox(
		bellBtn,
		expandBtn,
	).Gap(8).CrossAlign(primitives.CrossAxisCenter)

	return primitives.HBox(
		searchField,
		primitives.Expanded(primitives.Box()),
		actions,
	).
		PaddingXY(20, 14).
		CrossAlign(primitives.CrossAxisCenter).
		Background(widget.ColorTransparent)
}
