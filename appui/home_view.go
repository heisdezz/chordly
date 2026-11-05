package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// NewHomeContent assembles the Metrolist home feed from the independently
// owned sections (hero, mixes, releases). It is the body of the "/" route and
// is intentionally separate from NewShell so routing and layout stay decoupled.
func NewHomeContent() widget.Widget {
	return primitives.VBox(
		NewHeroBanner(),
		primitives.Box().Height(18),
		NewMadeForYouSection(),
		primitives.Box().Height(18),
		NewNewReleasesSection(),
	).CrossAlign(primitives.CrossAxisStretch)
}
