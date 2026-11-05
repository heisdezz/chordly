package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/router"
)

// NewTopBar builds the top app bar containing the search field and utility buttons.
func NewTopBar(r *router.Router) widget.Widget {
	searchField := NewTextField(
		"Search songs, artists, albums...",
		TextFieldInitial(state.SearchQuery.Get()),
		TextFieldOnChange(func(value string) {
			state.SearchQuery.Set(value)
		}),
		TextFieldOnSubmit(func(query string) {
			if query == "" {
				return
			}
			RunSearch(query)
			r.Navigate("/search")
		}),
	)

	// Notification button (Vector Bell)
	bellBtn := NewVectorIconButton(IconBell, 40, 18, func() {
		// Toggle notifications
	})

	// Fullscreen button (Vector Expand)
	expandBtn := NewVectorIconButton(IconExpand, 40, 17, func() {
		state.IsFullscreen.Set(!state.IsFullscreen.Get())
	})

	actions := primitives.HBox(
		bellBtn,
		expandBtn,
	).Gap(10).CrossAlign(primitives.CrossAxisCenter)

	return primitives.HBox(
		searchField,
		primitives.Expanded(primitives.Box()),
		actions,
	).
		PaddingXY(20, 14).
		CrossAlign(primitives.CrossAxisCenter).
		Background(widget.ColorTransparent)
}
