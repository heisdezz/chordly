package appui

import (
	"math"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// Spinner is a paint-only indeterminate progress indicator.
type Spinner struct {
	widget.WidgetBase

	Size   float32
	Stroke float32
	Color  widget.Color
}

// NewSpinner creates a spinner of the given diameter. Pass 0 for the default
// 20px size.
func NewSpinner(size float32) *Spinner {
	if size <= 0 {
		size = 20
	}
	s := &Spinner{
		Size:   size,
		Stroke: 2.5,
		Color:  theme.ColorAccent,
	}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

func (s *Spinner) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(s.Size, s.Size))
	s.SetBounds(geometry.FromPointSize(s.Bounds().Min, size))
	return size
}

func (s *Spinner) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := s.Bounds()
	if b.IsEmpty() {
		return
	}

	center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
	radius := b.Width()/2 - s.Stroke
	if radius < 1 {
		radius = 1
	}

	canvas.StrokeCircle(center, radius, theme.ColorBorder, s.Stroke)

	start := float64(-math.Pi / 2)
	sweep := float64(math.Pi * 1.4)

	if !motionReduced() && ctx != nil {
		now := ctx.Now()
		phase := float64(now.UnixNano()%int64(time.Second)) / float64(time.Second)
		start += phase * 2 * math.Pi
		sweep = 1.4 + 0.4*math.Sin(phase*2*math.Pi)

		if sched, ok := ctx.(widget.AnimationScheduler); ok {
			sched.ScheduleAnimationFrame()
		} else {
			ctx.Invalidate()
		}
		s.SetNeedsRedraw(true)
	}

	canvas.StrokeArc(center, radius, start, sweep, s.Color, s.Stroke)
}

func (s *Spinner) Event(ctx widget.Context, e event.Event) bool { return false }

func (s *Spinner) Children() []widget.Widget { return nil }

// Skeleton is a pulsing placeholder block. A zero width fills the available
// constraint; a zero height uses 12px.
type Skeleton struct {
	widget.WidgetBase

	Width  float32
	Height float32
	Radius float32
}

// NewSkeleton creates a placeholder block.
func NewSkeleton(width, height, radius float32) *Skeleton {
	if height <= 0 {
		height = 12
	}
	s := &Skeleton{Width: width, Height: height, Radius: radius}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

func (s *Skeleton) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	width := s.Width
	if width <= 0 {
		width = constraints.MaxWidth
	}
	height := s.Height
	if height <= 0 {
		height = 12
	}
	size := constraints.Constrain(geometry.Sz(width, height))
	s.SetBounds(geometry.FromPointSize(s.Bounds().Min, size))
	return size
}

func (s *Skeleton) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := s.Bounds()
	if b.IsEmpty() {
		return
	}

	alpha := float32(0.55)
	if !motionReduced() && ctx != nil {
		now := ctx.Now()
		phase := float64(now.UnixNano()%int64(1200*time.Millisecond)) / float64(1200*time.Millisecond)
		wave := (math.Sin(phase*2*math.Pi) + 1) / 2
		alpha = 0.35 + 0.4*float32(wave)

		if sched, ok := ctx.(widget.AnimationScheduler); ok {
			sched.ScheduleAnimationFrame()
		} else {
			ctx.Invalidate()
		}
		s.SetNeedsRedraw(true)
	}

	canvas.DrawRoundRect(b, theme.ColorSurfaceLight.WithAlpha(alpha), s.Radius)
}

func (s *Skeleton) Event(ctx widget.Context, e event.Event) bool { return false }

func (s *Skeleton) Children() []widget.Widget { return nil }

// NewSkeletonTrackRows builds placeholder rows for track lists.
func NewSkeletonTrackRows(count int) widget.Widget {
	if count <= 0 {
		count = 4
	}
	rows := make([]widget.Widget, 0, count)
	for i := 0; i < count; i++ {
		row := primitives.HBox(
			NewSkeleton(40, 40, 10),
			primitives.Box().Width(12),
			primitives.Expanded(
				primitives.VBox(
					NewSkeleton(0, 12, 6),
					NewSkeleton(0, 10, 5),
				).Gap(7).CrossAlign(primitives.CrossAxisStretch),
			),
			primitives.Box().Width(10),
			NewSkeleton(36, 10, 5),
		).
			PaddingXY(10, 8).
			CrossAlign(primitives.CrossAxisCenter)
		rows = append(rows, row)
	}
	return primitives.VBox(rows...).Gap(6)
}

// NewSkeletonCardRow builds placeholder poster cards.
func NewSkeletonCardRow(count int) widget.Widget {
	if count <= 0 {
		count = 4
	}
	cells := make([]widget.Widget, 0, count*2)
	for i := 0; i < count; i++ {
		if i > 0 {
			cells = append(cells, primitives.Box().Width(12))
		}
		cells = append(cells, primitives.Expanded(
			primitives.VBox(
				NewSkeleton(0, 120, 14),
				NewSkeleton(0, 12, 6),
				NewSkeleton(0, 10, 5),
			).Gap(8).CrossAlign(primitives.CrossAxisStretch),
		))
	}
	return primitives.HBox(cells...)
}

// NewSkeletonHero builds a placeholder hero banner.
func NewSkeletonHero() widget.Widget {
	return NewSkeleton(0, 180, 22)
}

// NewSearchLoading builds the search loading state with a spinner and rows.
func NewSearchLoading(query string) widget.Widget {
	message := "Searching YouTube Music…"
	if query != "" {
		message = "Searching YouTube Music for " + `"` + query + `"` + "…"
	}

	header := primitives.HBox(
		NewSpinner(18),
		primitives.Box().Width(10),
		primitives.Text(message).FontSize(13).Color(theme.ColorTextSecondary),
	).CrossAlign(primitives.CrossAxisCenter)

	return primitives.VBox(
		header,
		primitives.Box().Height(14),
		NewSkeletonTrackRows(5),
	).Gap(0).CrossAlign(primitives.CrossAxisStretch)
}
