package appui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

const (
	railWidth float32 = 74
)

// NewNavRail renders the left vertical navigation rail matching the reference UI.
func NewNavRail(r *router.Router) widget.Widget {
	// Top Brand Icon (Waveform)
	brandIcon := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 24),
	).PaddingXY(14, 12).
		Background(widget.ColorTransparent).
		Rounded(16)

	// Navigation Item Buttons (Vector Drawn Icons)
	homeBtn := NewVectorIconButton(IconHome, 44, 20, func() {
		state.ActiveNavTab.Set("home")
		r.Navigate("/")
	}, func(s *IconButtonStyle) {
		s.Radius = 14
		s.IsActive = func() bool { return navActive(r, "/") }
		s.ActiveBg = theme.ColorAccent
		s.ActiveFg = theme.ColorOnAccent
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextSecondary
		s.BorderWidth = 0
	})

	exploreBtn := NewVectorIconButton(IconExplore, 44, 20, func() {
		state.ActiveNavTab.Set("explore")
		r.Navigate("/explore")
	}, func(s *IconButtonStyle) {
		s.Radius = 14
		s.IsActive = func() bool { return navActive(r, "/explore") }
		s.ActiveBg = theme.ColorAccent
		s.ActiveFg = theme.ColorOnAccent
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextMuted
		s.BorderWidth = 0
	})

	libraryBtn := NewVectorIconButton(IconLibrary, 44, 20, func() {
		state.ActiveNavTab.Set("library")
		r.Navigate("/library")
	}, func(s *IconButtonStyle) {
		s.Radius = 14
		s.IsActive = func() bool { return navActive(r, "/library") }
		s.ActiveBg = theme.ColorAccent
		s.ActiveFg = theme.ColorOnAccent
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextMuted
		s.BorderWidth = 0
	})

	navGroup := primitives.VBox(
		homeBtn,
		exploreBtn,
		libraryBtn,
	).Gap(12).CrossAlign(primitives.CrossAxisCenter)

	// Settings & Profile at bottom
	settingsBtn := NewVectorIconButton(IconSettings, 42, 19, func() {
		state.ActiveNavTab.Set("settings")
		r.Navigate("/settings")
	}, func(s *IconButtonStyle) {
		s.Radius = 14
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextMuted
		s.BorderWidth = 0
		s.IsActive = func() bool { return navActive(r, "/settings") }
		s.ActiveFg = theme.ColorAccent
	})

	// Profile Avatar with green online dot
	avatarWidget := newProfileAvatar()

	bottomGroup := primitives.VBox(
		settingsBtn,
		primitives.Box().Height(6),
		avatarWidget,
	).Gap(8).CrossAlign(primitives.CrossAxisCenter)

	return primitives.VBox(
		brandIcon,
		primitives.Box().Height(16),
		navGroup,
		primitives.Expanded(primitives.Box()),
		bottomGroup,
	).
		PaddingXY(12, 18).
		Width(railWidth).
		Background(theme.ColorSidebar).
		BorderStyle(1, theme.ColorBorderMuted).
		CrossAlign(primitives.CrossAxisCenter)
}

// navActive reports whether the current route is the given section root.
func navActive(r *router.Router, prefix string) bool {
	path := r.Match().Get().Path
	if prefix == "/" {
		return path == "/"
	}
	return path == prefix || len(path) > len(prefix) && path[:len(prefix)+1] == prefix+"/"
}

// profileAvatarPainter draws the circular user avatar with vector profile shape.
type profileAvatarPainter struct{}

func (p profileAvatarPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	center := geometry.Pt(s.Bounds.Min.X+s.Bounds.Width()/2, s.Bounds.Min.Y+s.Bounds.Height()/2)
	radius := s.Bounds.Width() / 2

	// Avatar Circle
	canvas.DrawCircle(center, radius, widget.Hex(0x23313C))
	canvas.StrokeCircle(center, radius-1, theme.ColorBorder, 1.5)

	// Vector Head
	headCenter := geometry.Pt(center.X, center.Y-4)
	canvas.DrawCircle(headCenter, 6, theme.ColorTextSecondary)

	// Vector Shoulders
	shoulderRect := geometry.Rect{
		Min: geometry.Pt(center.X-10, center.Y+4),
		Max: geometry.Pt(center.X+10, center.Y+12),
	}
	canvas.DrawRoundRect(shoulderRect, theme.ColorTextSecondary, 5)

	// Online dot (bottom right)
	dotCenter := geometry.Pt(s.Bounds.Max.X-5, s.Bounds.Max.Y-5)
	canvas.DrawCircle(dotCenter, 4.5, theme.ColorAccent)
	canvas.StrokeCircle(dotCenter, 4.5, theme.ColorSidebar, 1.5)
}

func (p profileAvatarPainter) ButtonHeight(button.Size) float32             { return 38 }
func (p profileAvatarPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p profileAvatarPainter) ButtonFontSize(button.Size) float32           { return 14 }
func (p profileAvatarPainter) ButtonRadius() float32                        { return 19 }

var _ button.Painter = profileAvatarPainter{}
var _ button.LayoutMetrics = profileAvatarPainter{}

func newProfileAvatar() *button.Widget {
	return button.New(
		button.Text(""),
		button.PainterOpt(profileAvatarPainter{}),
	).MinWidth(38).MaxWidth(38)
}
