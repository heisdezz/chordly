package appui

import (
	"context"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
)

// searchPillPainter paints the rounded search pill with vector magnifying glass.
type searchPillPainter struct{}

func (p searchPillPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	bg := theme.ColorSurfaceSubtle
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.04)
	}

	canvas.DrawRoundRect(s.Bounds, bg, s.Bounds.Height()/2)
	canvas.StrokeRoundRect(s.Bounds, theme.ColorBorder, s.Bounds.Height()/2, 1)

	// Vector Search Icon
	iconRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+14, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Min.X+32, s.Bounds.Max.Y),
	}
	DrawVectorIcon(canvas, IconSearch, iconRect, theme.ColorTextMuted, 16)

	// Search placeholder / text
	textRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+36, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Max.X-16, s.Bounds.Max.Y),
	}
	placeholder := "Search songs, artists, albums..."
	canvas.DrawText(placeholder, textRect, 13, theme.ColorTextMuted, false, widget.TextAlignLeft)
}

func (p searchPillPainter) ButtonHeight(button.Size) float32             { return 42 }
func (p searchPillPainter) ButtonPadding(button.Size) (float32, float32) { return 16, 0 }
func (p searchPillPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p searchPillPainter) ButtonRadius() float32                        { return 21 }

var _ button.Painter = searchPillPainter{}
var _ button.LayoutMetrics = searchPillPainter{}

// NewTopBar builds the top app bar containing the search pill and utility buttons.
func NewTopBar() widget.Widget {
	// Interactive Search Pill (Triggers live search against InnerTube)
	searchField := button.New(
		button.Text(""),
		button.OnClick(func() {
			ytm.DefaultService.LiveSearch(context.Background(), "Daft Punk")
		}),
		button.PainterOpt(searchPillPainter{}),
	).MinWidth(400).MaxWidth(420)

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
