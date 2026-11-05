package appui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
)

// NewBottomPlayer builds the bottom player dock matching the reference UI.
func NewBottomPlayer() widget.Widget {
	// Left: Album thumbnail & Metadata
	track := state.CurrentTrack.Get()

	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 18),
	).
		PaddingXY(14, 12).
		Background(widget.Hex(0x182C29)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(track.Artist).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	leftSection := primitives.HBox(
		thumb,
		primitives.Box().Width(12),
		meta,
	).CrossAlign(primitives.CrossAxisCenter).Width(240)

	// Center: Controls & Scrubber (Vector Icons)
	shuffleBtn := NewVectorIconButton(IconShuffle, 34, 16, func() {
		state.IsShuffle.Set(!state.IsShuffle.Get())
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
		s.IsActive = func() bool { return state.IsShuffle.Get() }
		s.ActiveFg = theme.ColorAccent
	})

	prevBtn := NewVectorIconButton(IconPrev, 36, 17, func() {
		state.PreviousTrack()
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextPrimary
	})

	// Large circular Mint Play/Pause button
	playPauseBtn := newPlayPauseButton()

	nextBtn := NewVectorIconButton(IconNext, 36, 17, func() {
		state.NextTrack()
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextPrimary
	})

	repeatBtn := NewVectorIconButton(IconRepeat, 34, 16, func() {
		cur := state.Repeat.Get()
		state.Repeat.Set((cur + 1) % 3)
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
		s.IsActive = func() bool { return state.Repeat.Get() != model.RepeatOff }
		s.ActiveFg = theme.ColorAccent
	})

	transportControls := primitives.HBox(
		shuffleBtn,
		prevBtn,
		playPauseBtn,
		nextBtn,
		repeatBtn,
	).Gap(14).CrossAlign(primitives.CrossAxisCenter)

	scrubber := NewScrubberWidget()

	centerSection := primitives.VBox(
		transportControls,
		primitives.Box().Height(4),
		scrubber,
	).CrossAlign(primitives.CrossAxisCenter)

	// Right: Volume & Utility Toggles
	volumeSlider := NewVolumeSliderWidget()

	queueBtn := NewVectorIconButton(IconQueue, 34, 16, func() {
		state.IsQueueOpen.Set(!state.IsQueueOpen.Get())
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	expandBtn := NewVectorIconButton(IconExpand, 34, 16, func() {
		state.IsFullscreen.Set(!state.IsFullscreen.Get())
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	rightSection := primitives.HBox(
		volumeSlider,
		primitives.Box().Width(12),
		queueBtn,
		expandBtn,
	).CrossAlign(primitives.CrossAxisCenter).Width(240)

	return primitives.HBox(
		leftSection,
		primitives.Expanded(centerSection),
		rightSection,
	).
		PaddingXY(20, 10).
		Height(78).
		Background(theme.ColorPlayerBar).
		Rounded(20).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}

// playPauseButtonPainter draws the circular mint button with vector play/pause icon.
type playPauseButtonPainter struct{}

func (p playPauseButtonPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
	radius := b.Width() / 2

	bg := theme.ColorAccent
	if s.Hovered {
		bg = theme.ColorAccentHover
	}
	if s.Pressed {
		bg = theme.ColorAccent.Lerp(widget.ColorBlack, 0.15)
	}

	canvas.DrawCircle(center, radius, bg)

	icon := IconPause
	if state.PlaybackStatus.Get() != model.StatePlaying {
		icon = IconPlay
	}
	DrawVectorIcon(canvas, icon, b, theme.ColorOnAccent, 18)
}

func (p playPauseButtonPainter) ButtonHeight(button.Size) float32             { return 42 }
func (p playPauseButtonPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p playPauseButtonPainter) ButtonFontSize(button.Size) float32           { return 16 }
func (p playPauseButtonPainter) ButtonRadius() float32                        { return 21 }

var _ button.Painter = playPauseButtonPainter{}
var _ button.LayoutMetrics = playPauseButtonPainter{}

func newPlayPauseButton() *button.Widget {
	return button.New(
		button.Text(""),
		button.OnClick(func() {
			state.TogglePlayPause()
		}),
		button.PainterOpt(playPauseButtonPainter{}),
	).MinWidth(42).MaxWidth(42)
}
