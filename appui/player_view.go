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
		250*time.Millisecond,
		200*time.Millisecond,
		miniPlayerArtSource,
		NewPlayerArtwork(),
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

	source     widget.Widget
	sourceRect geometry.Rect
	targetRect geometry.Rect
	morphStart time.Time
	morphDur   time.Duration
	morphDone  bool
	target     widget.Widget
}

func newPlayerScreen(panel widget.Widget, scrim widget.Widget, duration, scrimDur time.Duration, source, target widget.Widget) *playerScreen {
	s := &playerScreen{panel: panel, scrim: scrim, duration: duration, scrimDur: scrimDur, source: source, target: target}
	if s.morphDur <= 0 {
		s.morphDur = 400 * time.Millisecond
	}
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
	scrim := float32(0.55)
	if !s.done {
		if s.start.IsZero() {
			s.start = ctx.Now()
		}
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
			scrim = float32(0.55) * easeOutCubic(float32(st))
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

	// Shared-element artwork morph: while the full-screen player is entering,
	// crossfade the mini-player artwork into the player artwork rect over
	// morphDur (M3Emphasized), then hand off to the panel draw.
	if s.source != nil && !s.morphDone && !s.done {
		mp := s.morphProgress(ctx)
		if mp < 1 {
			if sched, ok := ctx.(widget.AnimationScheduler); ok {
				sched.ScheduleAnimationFrame()
			} else {
				ctx.Invalidate()
			}
			s.SetNeedsRedraw(true)

			src := s.sourceRect
			dst := s.targetRect
			if r, ok := s.source.(interface{ Bounds() geometry.Rect }); ok && !r.Bounds().IsEmpty() {
				src = r.Bounds()
			}
			if dst.IsZero() && !b.IsEmpty() {
				dst = playerArtworkRect(b)
			}
			if !src.IsEmpty() && !dst.IsEmpty() && src != dst {
				canvas.PushClip(src)
				canvas.PushTransform(geometry.Pt(src.Min.X, src.Min.Y))
				pushToastPaint(canvas, 0.5, func() {
					widget.StampScreenOrigin(s.source, canvas)
					s.source.Draw(ctx, canvas)
				})
				canvas.PopTransform()
				canvas.PushTransform(geometry.Pt(dst.Min.X, dst.Min.Y))
				pushToastPaint(canvas, 0.5, func() {
					widget.StampScreenOrigin(s.target, canvas)
					s.target.Draw(ctx, canvas)
				})
				canvas.PopTransform()
				canvas.PopClip()
			}
			return
		}
		s.morphDone = true
	}

	pushToastPaint(canvas, scrim, func() {
		canvas.DrawRoundRect(b, theme.ColorCanvas, 24)
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

// morphProgress returns the eased shared-element morph progress in [0, 1].
func (s *playerScreen) morphProgress(ctx widget.Context) float32 {
	if s.morphDur <= 0 {
		return 1
	}
	if s.morphStart.IsZero() {
		s.morphStart = ctx.Now()
	}
	elapsed := float64(ctx.Now().Sub(s.morphStart).Milliseconds())
	t := elapsed / float64(s.morphDur)
	if t >= 1 {
		return 1
	}
	return easeOutCubic(float32(t))
}

// playerArtworkRect returns the centered square artwork rect within bounds.
func playerArtworkRect(b geometry.Rect) geometry.Rect {
	side := b.Width()
	if b.Height() < side {
		side = b.Height()
	}
	if side > 440 {
		side = 440
	}
	if side <= 0 {
		side = 320
	}
	x := b.Min.X + (b.Width()-side)/2
	y := b.Min.Y + (b.Height()-side)/2
	return geometry.Rect{
		Min: geometry.Pt(x, y),
		Max: geometry.Pt(x+side, y+side),
	}
}

// lerpRect interpolates rect a -> b by t.
func lerpRect(a, b geometry.Rect, t float32) geometry.Rect {
	return geometry.Rect{
		Min: geometry.Pt(a.Min.X+(b.Min.X-a.Min.X)*t, a.Min.Y+(b.Min.Y-a.Min.Y)*t),
		Max: geometry.Pt(a.Max.X+(b.Max.X-a.Max.X)*t, a.Max.Y+(b.Max.Y-a.Max.Y)*t),
	}
}

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
	track := state.CurrentTrack.Get()
	// Try to use thumbnail widget if available
	tw := NewThumbnailWidget(track.Thumbnail, side, radius, track.Title)
	// Layout it to match bounds
	tw.Layout(ctx, geometry.Constraints{MaxWidth: side, MaxHeight: side})
	if setter, ok := interface{}(tw).(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(b)
	}
	// Draw the thumbnail widget
	tw.Draw(ctx, canvas)
	canvas.StrokeRoundRect(b, theme.ColorBorder, radius, 1)
}

func (a *PlayerArtwork) Children() []widget.Widget { return nil }

func (a *PlayerArtwork) Event(ctx widget.Context, e event.Event) bool { return false }

var _ widget.Widget = (*PlayerArtwork)(nil)
