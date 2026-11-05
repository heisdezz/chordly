package appui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/primitives"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
	"chordly/router"
)

// LibraryTab defines the active subview inside the Library shell.
type LibraryTab string

const (
	LibraryTabFavorites LibraryTab = "favorites"
	LibraryTabPlaylists LibraryTab = "playlists"
	LibraryTabHistory   LibraryTab = "history"
	LibraryTabDownloads LibraryTab = "downloads"
)

var (
	// SigLibraryTab tracks the currently selected library tab.
	SigLibraryTab = gpstate.NewSignal(LibraryTabFavorites)
)

// SetLibraryTab updates the active library tab signal.
func SetLibraryTab(tab LibraryTab) {
	SigLibraryTab.Set(tab)
}

type libraryTabDef struct {
	Tab   LibraryTab
	Label string
	Icon  IconType
}

var libraryTabDefs = []libraryTabDef{
	{Tab: LibraryTabFavorites, Label: "Favorites", Icon: IconHeart},
	{Tab: LibraryTabPlaylists, Label: "Playlists", Icon: IconQueue},
	{Tab: LibraryTabHistory, Label: "History", Icon: IconWaveform},
	{Tab: LibraryTabDownloads, Label: "Downloads", Icon: IconLibrary},
}

type libraryTabPillPainter struct {
	tab LibraryTab
}

func (p libraryTabPillPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	active := SigLibraryTab.Get() == p.tab

	bg := theme.ColorSurfaceLight
	fg := theme.ColorTextSecondary
	if active {
		bg = theme.ColorAccentSoft
		fg = theme.ColorAccent
	}
	switch {
	case s.Pressed:
		bg = bg.Lerp(widget.ColorBlack, 0.15)
	case s.Hovered:
		bg = bg.Lerp(widget.ColorWhite, 0.08)
	}

	canvas.DrawRoundRect(s.Bounds, bg, s.Bounds.Height()/2)
	if !active {
		canvas.StrokeRoundRect(s.Bounds, theme.ColorBorder, s.Bounds.Height()/2, 1)
	}

	textRect := s.Bounds
	textRect.Min.X += 16
	textRect.Max.X -= 16
	canvas.DrawText(s.Text, textRect, 13, fg, active, widget.TextAlignCenter)
}

func (p libraryTabPillPainter) ButtonHeight(button.Size) float32             { return 34 }
func (p libraryTabPillPainter) ButtonPadding(button.Size) (float32, float32) { return 18, 0 }
func (p libraryTabPillPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p libraryTabPillPainter) ButtonRadius() float32                        { return 17 }

func buildLibraryTabBar(r *router.Router) widget.Widget {
	pills := make([]widget.Widget, 0, len(libraryTabDefs))
	for _, def := range libraryTabDefs {
		tab := def.Tab
		label := def.Label
		pill := button.New(
			button.Text(label),
			button.OnClick(func() {
				SigLibraryTab.Set(tab)
				if r != nil {
					r.Navigate("/library/" + string(tab))
				}
			}),
			button.PainterOpt(libraryTabPillPainter{tab: tab}),
		)
		pills = append(pills, pill)
	}
	return primitives.HBox(pills...).Gap(10)
}
