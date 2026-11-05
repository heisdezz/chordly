package appui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// IconButtonStyle configures the appearance of an icon button.
type IconButtonStyle struct {
	Icon        IconType
	IconSize    float32
	Size        float32
	FontSize    float32
	Radius      float32
	Background  widget.Color
	Foreground  widget.Color
	HoverBg     widget.Color
	PressedBg   widget.Color
	BorderColor widget.Color
	BorderWidth float32
	IsActive    func() bool
	ActiveBg    widget.Color
	ActiveFg    widget.Color
}

type iconButtonPainter struct {
	style IconButtonStyle
}

func (p iconButtonPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	active := p.style.IsActive != nil && p.style.IsActive()

	bg := p.style.Background
	fg := p.style.Foreground

	switch {
	case active && !p.style.ActiveBg.IsTransparent():
		bg = p.style.ActiveBg
		fg = p.style.ActiveFg
	case s.Pressed && !p.style.PressedBg.IsTransparent():
		bg = p.style.PressedBg
	case s.Hovered && !p.style.HoverBg.IsTransparent():
		bg = p.style.HoverBg
	}

	if !bg.IsTransparent() {
		if p.style.Radius >= p.style.Size/2 {
			center := geometry.Pt(s.Bounds.Min.X+s.Bounds.Width()/2, s.Bounds.Min.Y+s.Bounds.Height()/2)
			canvas.DrawCircle(center, s.Bounds.Width()/2, bg)
		} else {
			canvas.DrawRoundRect(s.Bounds, bg, p.style.Radius)
		}
	}

	if p.style.BorderWidth > 0 && !p.style.BorderColor.IsTransparent() {
		if p.style.Radius >= p.style.Size/2 {
			center := geometry.Pt(s.Bounds.Min.X+s.Bounds.Width()/2, s.Bounds.Min.Y+s.Bounds.Height()/2)
			canvas.StrokeCircle(center, s.Bounds.Width()/2-p.style.BorderWidth/2, p.style.BorderColor, p.style.BorderWidth)
		} else {
			canvas.StrokeRoundRect(s.Bounds, p.style.BorderColor, p.style.Radius, p.style.BorderWidth)
		}
	}

	if p.style.Icon != IconNone {
		iconSz := p.style.IconSize
		if iconSz <= 0 {
			iconSz = p.style.Size * 0.52
		}
		DrawVectorIcon(canvas, p.style.Icon, s.Bounds, fg, iconSz)
	} else if s.Text != "" {
		canvas.DrawText(s.Text, s.Bounds, p.style.FontSize, fg, true, widget.TextAlignCenter)
	}
}

func (p iconButtonPainter) ButtonHeight(button.Size) float32 {
	return p.style.Size
}

func (p iconButtonPainter) ButtonPadding(button.Size) (float32, float32) {
	return 0, 0
}

func (p iconButtonPainter) ButtonFontSize(button.Size) float32 {
	return p.style.FontSize
}

func (p iconButtonPainter) ButtonRadius() float32 {
	return p.style.Radius
}

var _ button.Painter = iconButtonPainter{}
var _ button.LayoutMetrics = iconButtonPainter{}

// NewVectorIconButton creates a button using crisp vector drawing.
func NewVectorIconButton(icon IconType, size, iconSize float32, onClick func(), opts ...func(*IconButtonStyle)) *button.Widget {
	s := IconButtonStyle{
		Icon:        icon,
		IconSize:    iconSize,
		Size:        size,
		FontSize:    iconSize,
		Radius:      size / 2, // Default circle
		Background:  theme.ColorSurfaceLight,
		Foreground:  theme.ColorTextPrimary,
		HoverBg:     theme.ColorSurfaceLight.Lerp(widget.ColorWhite, 0.1),
		PressedBg:   theme.ColorSurfaceLight.Lerp(widget.ColorBlack, 0.15),
		BorderColor: theme.ColorBorder,
		BorderWidth: 1.0,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return button.New(
		button.Text(""),
		button.OnClick(onClick),
		button.PainterOpt(iconButtonPainter{style: s}),
	).MinWidth(size).MaxWidth(size)
}

// NewIconButton creates an icon button with fallback glyph support.
func NewIconButton(glyph string, size, fontSize float32, onClick func(), opts ...func(*IconButtonStyle)) *button.Widget {
	s := IconButtonStyle{
		Size:        size,
		FontSize:    fontSize,
		Radius:      size / 2,
		Background:  theme.ColorSurfaceLight,
		Foreground:  theme.ColorTextPrimary,
		HoverBg:     theme.ColorSurfaceLight.Lerp(widget.ColorWhite, 0.1),
		PressedBg:   theme.ColorSurfaceLight.Lerp(widget.ColorBlack, 0.15),
		BorderColor: theme.ColorBorder,
		BorderWidth: 1.0,
	}

	for _, opt := range opts {
		opt(&s)
	}

	return button.New(
		button.Text(glyph),
		button.OnClick(onClick),
		button.PainterOpt(iconButtonPainter{style: s}),
	).MinWidth(size).MaxWidth(size)
}

// playPillButtonPainter draws the mint Play Now button with vector play icon.
type playPillButtonPainter struct{}

func (p playPillButtonPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	bg := theme.ColorAccent
	if s.Hovered {
		bg = theme.ColorAccentHover
	}
	if s.Pressed {
		bg = theme.ColorAccent.Lerp(widget.ColorBlack, 0.15)
	}

	canvas.DrawRoundRect(s.Bounds, bg, s.Bounds.Height()/2)

	// Draw vector play triangle
	iconRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+16, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Min.X+32, s.Bounds.Max.Y),
	}
	DrawVectorIcon(canvas, IconPlay, iconRect, theme.ColorOnAccent, 12)

	// Draw text label
	textRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+32, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Max.X-16, s.Bounds.Max.Y),
	}
	canvas.DrawText("Play Now", textRect, 13, theme.ColorOnAccent, true, widget.TextAlignLeft)
}

func (p playPillButtonPainter) ButtonHeight(button.Size) float32             { return 42 }
func (p playPillButtonPainter) ButtonPadding(button.Size) (float32, float32) { return 18, 0 }
func (p playPillButtonPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p playPillButtonPainter) ButtonRadius() float32                        { return 21 }

// NewPillPlayButton creates the distinctive mint "Play Now" pill button from the reference design.
func NewPillPlayButton(onClick func()) *button.Widget {
	return button.New(
		button.Text("Play Now"),
		button.OnClick(onClick),
		button.PainterOpt(playPillButtonPainter{}),
	).MinWidth(130).MaxWidth(145)
}

// VectorIconWidget is a pure widget that renders a vector icon.
type VectorIconWidget struct {
	widget.WidgetBase
	Icon  IconType
	Color widget.Color
	Size  float32
}

var _ widget.Widget = (*VectorIconWidget)(nil)

func NewVectorIcon(icon IconType, color widget.Color, size float32) *VectorIconWidget {
	v := &VectorIconWidget{Icon: icon, Color: color, Size: size}
	v.SetVisible(true)
	v.SetEnabled(true)
	return v
}

func (v *VectorIconWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := geometry.Sz(v.Size, v.Size)
	v.SetBounds(geometry.FromPointSize(v.Bounds().Min, size))
	return size
}

func (v *VectorIconWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	DrawVectorIcon(canvas, v.Icon, v.Bounds(), v.Color, v.Size)
}

func (v *VectorIconWidget) Event(ctx widget.Context, e event.Event) bool {
	return false
}

func (v *VectorIconWidget) Children() []widget.Widget {
	return nil
}
