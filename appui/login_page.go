package appui

import (
	"github.com/gogpu/ui/widget"

	"chordly/router"
)

// LoginPage renders the Google Sign-In & InnerTune session import view.
func LoginPage(c router.RouteContext) widget.Widget {
	return SignInPage(c)
}
