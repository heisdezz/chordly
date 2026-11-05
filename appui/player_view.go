package appui

import (
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

	return primitives.VBox(
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
		Background(theme.ColorCanvas)
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
	canvas.DrawRoundRect(b, theme.ColorSurface, radius)

	canvas.DrawCircle(center, side*0.36, theme.ColorAccent.WithAlpha(0.10))
	canvas.DrawCircle(center, side*0.24, theme.ColorAccent.WithAlpha(0.16))

	DrawVectorIcon(canvas, IconWaveform, b, theme.ColorAccent, side*0.28)

	canvas.StrokeRoundRect(b, theme.ColorBorder, radius, 1)
}

func (a *PlayerArtwork) Children() []widget.Widget { return nil }

func (a *PlayerArtwork) Event(ctx widget.Context, e event.Event) bool { return false }

var _ widget.Widget = (*PlayerArtwork)(nil)
