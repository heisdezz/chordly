package appui

import (
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

// PlayerPage is the full-screen now-playing view reached from the mini-player.
func PlayerPage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget { return buildPlayerPage(c) },
		WatchSignal(state.CurrentTrack.AsReadonly()),
	)
}

func buildPlayerPage(c router.RouteContext) widget.Widget {
	track := state.CurrentTrack.Get()

	title := track.Title
	if title == "" {
		title = "Nothing playing"
	}
	artist := track.Artist
	if artist == "" {
		artist = "Choose a track to begin"
	}

	collapse := NewVectorIconButton(IconChevronDown, 38, 18, func() {
		state.IsFullscreen.Set(false)
		if !c.Router.Back() {
			c.Navigate("/")
		}
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextSecondary
	})

	queueBtn := NewVectorIconButton(IconQueue, 38, 17, func() {
		c.Navigate("/queue")
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	lyricsBtn := NewVectorIconButton(IconLyrics, 38, 17, func() {
		c.Navigate("/lyrics")
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	topBar := primitives.HBox(
		collapse,
		primitives.Box().Width(10),
		primitives.Text("NOW PLAYING").FontSize(10).Bold().Color(theme.ColorAccent),
		primitives.Expanded(primitives.Box()),
		lyricsBtn,
		queueBtn,
	).CrossAlign(primitives.CrossAxisCenter)

	likeButton := NewVectorIconButton(IconHeart, 42, 18, func() {
		t := state.CurrentTrack.Get()
		t.IsLiked = !t.IsLiked
		state.CurrentTrack.Set(t)
		if t.IsLiked {
			ShowSuccess("Added to Liked songs")
		} else {
			ShowInfo("Removed from Liked songs")
		}
	}, func(s *IconButtonStyle) {
		s.Background = theme.ColorSurfaceLight
		s.BorderColor = theme.ColorBorder
		s.Foreground = theme.ColorTextPrimary
		s.IsActive = func() bool { return state.CurrentTrack.Get().IsLiked }
		s.ActiveFg = theme.ColorHeartActive
	})

	moreButton := NewVectorIconButton(IconMore, 42, 18, func() {
		ShowInfo("Track options are coming with the detail pages")
	}, func(s *IconButtonStyle) {
		s.Background = theme.ColorSurfaceLight
		s.BorderColor = theme.ColorBorder
		s.Foreground = theme.ColorTextPrimary
	})

	details := primitives.VBox(
		primitives.Text(title).FontSize(30).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(artist).FontSize(15).Color(theme.ColorTextSecondary),
		primitives.Box().Height(8),
		primitives.HBox(likeButton, moreButton).Gap(10),
		primitives.Box().Height(18),
		NewScrubberWidget(),
		primitives.Box().Height(6),
		buildPlayerTransport(),
		primitives.Box().Height(16),
		buildPlayerUtilities(),
	).
		Gap(4).
		CrossAlign(primitives.CrossAxisStretch)

	scrim := primitives.Box().Background(theme.ColorCanvas.WithAlpha(0.55))

	return newPlayerScreen(
		primitives.VBox(
			topBar,
			primitives.Box().Height(18),
			primitives.Expanded(
				primitives.HBox(
					primitives.Expanded(NewPlayerArtwork()),
					primitives.Box().Width(36),
					primitives.Expanded(details),
				).CrossAlign(primitives.CrossAxisCenter),
			),
		).
			Padding(28).
			CrossAlign(primitives.CrossAxisStretch).
			Background(theme.ColorCanvas),
		scrim,
		300*time.Millisecond,
		200*time.Millisecond,
	)
}

// playerScreen animates the full-screen player surface in: the scrim fades
// in over 200ms and the panel slides up from +28px while fading in over 300ms
// (M3EmphasizedDecelerate), respecting the reduced-motion preference.
type playerScreen struct {
	widget.WidgetBase
	panel    widget.Widget
	scrim    widget.Widget
	start    time.Time
	duration time.Duration
	scrimDur time.Duration
	done     bool
}

func newPlayerScreen(panel widget.Widget, scrim widget.Widget, duration, scrimDur time.Duration) *playerScreen {
	s := &playerScreen{panel: panel, scrim: scrim, duration: duration, scrimDur: scrimDur}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

func (s *playerScreen) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := widget.LayoutChild(s.panel, ctx, constraints)
	if setter, ok := s.panel.(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
	}
	s.SetBounds(geometry.FromPointSize(s.Position(), size))
	return size
}

func (s *playerScreen) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := s.Bounds()
	if b.IsEmpty() {
		return
	}

	progress := float32(1)
	scrimProgress := float32(1)
	if !s.done {
		if motionReduced() {
			s.done = true
		} else {
			elapsed := float64(ctx.Now().Sub(s.start).Milliseconds())
			t := elapsed / float64(s.duration)
			if t >= 1 {
				s.done = true
			} else {
				progress = easeOutCubic(float32(t))
			}
			st := elapsed / float64(s.scrimDur)
			if st < 1 {
				scrimProgress = easeOutCubic(float32(st))
			}
			if !s.done {
				if sched, ok := ctx.(widget.AnimationScheduler); ok {
					sched.ScheduleAnimationFrame()
				} else {
					ctx.Invalidate()
				}
				s.SetNeedsRedraw(true)
			}
		}
	}

	pushToastPaint(canvas, float32(scrimProgress), func() {
		canvas.DrawRoundRect(b, theme.ColorCanvas.WithAlpha(0.55), 24)
	})

	canvas.PushTransform(geometry.Pt(0, (1-progress)*28))
	pushToastPaint(canvas, progress, func() {
		widget.StampScreenOrigin(s.panel, canvas)
		s.panel.Draw(ctx, canvas)
	})
	canvas.PopTransform()
}

func (s *playerScreen) Event(ctx widget.Context, e event.Event) bool { return false }

func (s *playerScreen) Children() []widget.Widget { return nil }

var _ widget.Widget = (*playerScreen)(nil)

func buildPlayerTransport() widget.Widget {
	shuffle := NewVectorIconButton(IconShuffle, 38, 17, func() {
		state.IsShuffle.Set(!state.IsShuffle.Get())
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
		s.IsActive = func() bool { return state.IsShuffle.Get() }
		s.ActiveFg = theme.ColorAccent
	})

	previous := NewVectorIconButton(IconPrev, 40, 18, func() {
		state.PreviousTrack()
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextPrimary
	})

	next := NewVectorIconButton(IconNext, 40, 18, func() {
		state.NextTrack()
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextPrimary
	})

	repeat := NewVectorIconButton(IconRepeat, 38, 17, func() {
		state.Repeat.Set((state.Repeat.Get() + 1) % 3)
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
		s.IsActive = func() bool { return state.Repeat.Get() != model.RepeatOff }
		s.ActiveFg = theme.ColorAccent
	})

	return primitives.HBox(
		shuffle,
		primitives.Box().Width(10),
		previous,
		primitives.Box().Width(10),
		newPlayPauseButton(),
		primitives.Box().Width(10),
		next,
		primitives.Box().Width(10),
		repeat,
	).CrossAlign(primitives.CrossAxisCenter)
}

func buildPlayerUtilities() widget.Widget {
	queue := state.Queue.Get()
	current := state.QueueIndex.Get()

	nextTitle := ""
	if current+1 >= 0 && current+1 < len(queue) {
		nextTitle = queue[current+1].Title
	}

	utilities := primitives.HBox(
		NewVolumeSliderWidget(),
		primitives.Expanded(primitives.Box()),
	)
	if nextTitle != "" {
		utilities = primitives.HBox(
			NewVolumeSliderWidget(),
			primitives.Expanded(primitives.Box()),
			primitives.Text("Up next • "+nextTitle).
				FontSize(11).
				Color(theme.ColorTextMuted),
		)
	}
	return utilities.CrossAlign(primitives.CrossAxisCenter)
}

// PlayerArtwork is the large now-playing artwork placeholder.
type PlayerArtwork struct {
	widget.WidgetBase
}

// NewPlayerArtwork creates the large album-art surface.
func NewPlayerArtwork() *PlayerArtwork {
	a := &PlayerArtwork{}
	a.SetVisible(true)
	a.SetEnabled(true)
	return a
}

func (a *PlayerArtwork) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	side := constraints.MaxWidth
	if constraints.MaxHeight > 0 && constraints.MaxHeight < side {
		side = constraints.MaxHeight
	}
	if side <= 0 {
		side = 320
	}
	if side > 440 {
		side = 440
	}
	size := constraints.Constrain(geometry.Sz(side, side))
	a.SetBounds(geometry.FromPointSize(a.Bounds().Min, size))
	return size
}

func (a *PlayerArtwork) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := a.Bounds()
	if bounds.IsEmpty() {
		return
	}

	side := bounds.Width()
	if bounds.Height() < side {
		side = bounds.Height()
	}
	center := geometry.Pt(bounds.Min.X+bounds.Width()/2, bounds.Min.Y+bounds.Height()/2)
	b := geometry.Rect{
		Min: geometry.Pt(center.X-side/2, center.Y-side/2),
		Max: geometry.Pt(center.X+side/2, center.Y+side/2),
	}

	const radius = 24
	canvas.DrawRoundRect(b, theme.ColorSurface, radius)

	canvas.DrawCircle(center, side*0.36, theme.ColorAccent.WithAlpha(0.10))
	canvas.DrawCircle(center, side*0.24, theme.ColorAccent.WithAlpha(0.16))

	DrawVectorIcon(canvas, IconWaveform, b, theme.ColorAccent, side*0.28)

	canvas.StrokeRoundRect(b, theme.ColorBorder, radius, 1)
}

func (a *PlayerArtwork) Children() []widget.Widget { return nil }

func (a *PlayerArtwork) Event(ctx widget.Context, e event.Event) bool { return false }

var _ widget.Widget = (*PlayerArtwork)(nil)
