package appui

import (
	"strings"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/widget"

	"chordly/router"
)

// Palette used by the demo shell.
var (
	colorCanvas     = widget.RGBA8(244, 243, 247, 255)
	colorSurface    = widget.ColorWhite
	colorSidebar    = widget.RGBA8(250, 249, 252, 255)
	colorInk        = widget.RGBA8(32, 30, 38, 255)
	colorMuted      = widget.RGBA8(110, 106, 120, 255)
	colorBorder     = widget.RGBA8(226, 222, 232, 255)
	colorAccent     = widget.Hex(0x6750A4)
	colorAccentSoft = widget.Hex(0xEADDFF)
	colorHover      = widget.Hex(0xF1ECF8)
	colorPressed    = widget.Hex(0xE4DCF1)
	colorOnAccent   = widget.ColorWhite
)

// textButtonPainter paints left-aligned, flat buttons used for navigation,
// breadcrumbs, and actions. It implements both button.Painter and
// button.LayoutMetrics so the button's measured size matches the painted size.
type textButtonPainter struct {
	active func() bool

	background        widget.Color
	hoverBackground   widget.Color
	pressedBackground widget.Color
	activeBackground  widget.Color

	foreground       widget.Color
	activeForeground widget.Color

	paddingX float32
	height   float32
	fontSize float32
	radius   float32
	bold     bool
	align    widget.TextAlign
}

// PaintButton implements button.Painter.
func (p textButtonPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	if state.Bounds.IsEmpty() {
		return
	}

	active := p.active != nil && p.active()

	background := p.background
	switch {
	case active:
		background = p.activeBackground
	case state.Pressed:
		background = p.pressedBackground
	case state.Hovered:
		background = p.hoverBackground
	}
	if state.Disabled {
		background = background.WithAlpha(0.4)
	}
	if !background.IsTransparent() {
		canvas.DrawRoundRect(state.Bounds, background, p.radius)
	}

	foreground := p.foreground
	if active {
		foreground = p.activeForeground
	}
	if state.Disabled {
		foreground = foreground.WithAlpha(0.45)
	}

	textBounds := state.Bounds
	textBounds.Min.X += p.paddingX
	textBounds.Max.X -= p.paddingX
	canvas.DrawText(state.Text, textBounds, p.fontSize, foreground, p.bold, p.align)

	if state.Focused && !state.Disabled {
		canvas.StrokeRoundRect(state.Bounds.Expand(1), colorAccent.WithAlpha(0.55), p.radius+1, 1.5)
	}
}

// ButtonHeight implements button.LayoutMetrics.
func (p textButtonPainter) ButtonHeight(button.Size) float32 {
	return p.height
}

// ButtonPadding implements button.LayoutMetrics.
func (p textButtonPainter) ButtonPadding(button.Size) (paddingX, paddingY float32) {
	return p.paddingX, 0
}

// ButtonFontSize implements button.LayoutMetrics.
func (p textButtonPainter) ButtonFontSize(button.Size) float32 {
	return p.fontSize
}

// ButtonRadius implements button.LayoutMetrics.
func (p textButtonPainter) ButtonRadius() float32 {
	return p.radius
}

// --- Button constructors ---

func navButton(r *router.Router, label, prefix string) *button.Widget {
	painter := textButtonPainter{
		active:            activePrefix(r, prefix),
		background:        widget.ColorTransparent,
		hoverBackground:   colorHover,
		pressedBackground: colorPressed,
		activeBackground:  colorAccentSoft,
		foreground:        colorMuted,
		activeForeground:  colorAccent,
		paddingX:          14,
		height:            40,
		fontSize:          14,
		radius:            10,
		align:             widget.TextAlignLeft,
	}

	return button.New(
		button.Text(label),
		button.OnClick(func() { r.Navigate(prefix) }),
		button.PainterOpt(painter),
	).MinWidth(sidebarWidth - 2*sidebarPadding).MaxWidth(sidebarWidth - 2*sidebarPadding)
}

type buttonTone uint8

const (
	toneFilled buttonTone = iota
	toneTonal
	toneText
)

func actionButton(label string, tone buttonTone, onClick func()) *button.Widget {
	painter := textButtonPainter{
		background:        widget.ColorTransparent,
		hoverBackground:   colorHover,
		pressedBackground: colorPressed,
		activeBackground:  widget.ColorTransparent,
		foreground:        colorAccent,
		activeForeground:  colorAccent,
		paddingX:          14,
		height:            36,
		fontSize:          13,
		radius:            9,
		align:             widget.TextAlignCenter,
	}

	switch tone {
	case toneFilled:
		painter.background = colorAccent
		painter.hoverBackground = colorAccent.Lerp(widget.ColorWhite, 0.12)
		painter.pressedBackground = colorAccent.Lerp(widget.ColorBlack, 0.15)
		painter.foreground = colorOnAccent
		painter.bold = true
	case toneTonal:
		painter.background = colorAccentSoft
		painter.foreground = colorAccent
		painter.bold = true
	case toneText:
		// Defaults are already the text-only style.
	}

	return button.New(
		button.Text(label),
		button.OnClick(onClick),
		button.PainterOpt(painter),
	)
}

func crumbButton(r *router.Router, label, target string) *button.Widget {
	painter := textButtonPainter{
		background:        widget.ColorTransparent,
		hoverBackground:   colorHover,
		pressedBackground: colorPressed,
		activeBackground:  widget.ColorTransparent,
		foreground:        colorMuted,
		activeForeground:  colorMuted,
		paddingX:          7,
		height:            26,
		fontSize:          12,
		radius:            7,
		align:             widget.TextAlignCenter,
	}

	return button.New(
		button.Text(label),
		button.OnClick(func() { r.Navigate(target) }),
		button.PainterOpt(painter),
	)
}

func activePrefix(r *router.Router, prefix string) func() bool {
	return func() bool {
		path := r.Match().Get().Path
		if prefix == "/" {
			return path == "/"
		}
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
}

var _ button.Painter = textButtonPainter{}
var _ button.LayoutMetrics = textButtonPainter{}
