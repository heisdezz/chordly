package appui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
)

// NewHeroBanner renders the "Trending" hero showcase matching the reference design.
func NewHeroBanner() widget.Widget {
	// Badge with Vector Waveform
	badge := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 13),
		primitives.Text("Trending").FontSize(12).Bold().Color(theme.ColorAccent),
	).PaddingXY(12, 6).
		Gap(6).
		Background(widget.RGBA8(92, 225, 182, 35)).
		Rounded(14).
		BorderStyle(1, widget.RGBA8(92, 225, 182, 70)).
		CrossAlign(primitives.CrossAxisCenter)

	// Title & Artist
	title := primitives.Text("Echoes of Tomorrow").FontSize(32).Bold().Color(theme.ColorTextPrimary)
	subtitle := primitives.Text("Rexha Mavrick • 12.5M listeners").FontSize(13).Color(theme.ColorTextSecondary)

	// Action buttons
	playBtn := NewPillPlayButton(func() {
		track := state.CurrentTrack.Get()
		state.PlaySpecificTrack(track)
	})

	heartBtn := NewVectorIconButton(IconHeart, 42, 18, func() {
		t := state.CurrentTrack.Get()
		t.IsLiked = !t.IsLiked
		state.CurrentTrack.Set(t)
	}, func(s *IconButtonStyle) {
		s.Background = widget.RGBA8(20, 32, 38, 160)
		s.HoverBg = widget.RGBA8(30, 46, 56, 200)
		s.BorderColor = widget.RGBA8(92, 225, 182, 60)
		s.Foreground = theme.ColorTextPrimary
	})

	moreBtn := NewVectorIconButton(IconMore, 42, 18, func() {
		// Context options
	}, func(s *IconButtonStyle) {
		s.Background = widget.RGBA8(20, 32, 38, 160)
		s.HoverBg = widget.RGBA8(30, 46, 56, 200)
		s.BorderColor = widget.RGBA8(92, 225, 182, 60)
		s.Foreground = theme.ColorTextPrimary
	})

	buttonRow := primitives.HBox(
		playBtn,
		heartBtn,
		moreBtn,
	).Gap(10).CrossAlign(primitives.CrossAxisCenter)

	// Pagination indicator dots
	dots := newPaginationDots()

	leftContent := primitives.VBox(
		badge,
		primitives.Box().Height(10),
		title,
		subtitle,
		primitives.Box().Height(16),
		buttonRow,
		primitives.Box().Height(20),
		dots,
	).Gap(4).CrossAlign(primitives.CrossAxisStart)

	// Right artwork visual
	rightArt := newHeroArtwork()

	return primitives.HBox(
		leftContent,
		primitives.Expanded(primitives.Box()),
		rightArt,
	).
		PaddingXY(28, 24).
		Background(widget.Hex(0x101D22)).
		Rounded(22).
		BorderStyle(1, widget.Hex(0x1C2F38)).
		CrossAlign(primitives.CrossAxisCenter)
}

// paginationDotsWidget renders the 3 pagination dots at the bottom of the hero banner.
type paginationDotsWidget struct {
	widget.WidgetBase
}

func newPaginationDots() *paginationDotsWidget {
	p := &paginationDotsWidget{}
	p.SetVisible(true)
	p.SetEnabled(true)
	return p
}

func (p *paginationDotsWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := geometry.Sz(50, 10)
	p.SetBounds(geometry.FromPointSize(p.Bounds().Min, size))
	return size
}

func (p *paginationDotsWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := p.Bounds()
	if b.IsEmpty() {
		return
	}

	y := b.Min.Y + 2

	// Active pill dot (mint)
	pillRect := geometry.Rect{
		Min: geometry.Pt(b.Min.X, y),
		Max: geometry.Pt(b.Min.X+16, y+6),
	}
	canvas.DrawRoundRect(pillRect, theme.ColorAccent, 3)

	// Dot 2 (muted)
	canvas.DrawCircle(geometry.Pt(b.Min.X+25, y+3), 3, widget.Hex(0x384E59))

	// Dot 3 (muted)
	canvas.DrawCircle(geometry.Pt(b.Min.X+36, y+3), 3, widget.Hex(0x384E59))
}

func (p *paginationDotsWidget) Event(ctx widget.Context, e event.Event) bool { return false }
func (p *paginationDotsWidget) Children() []widget.Widget                    { return nil }

// heroArtworkWidget draws the artistic futuristic metallic face silhouette on the right.
type heroArtworkWidget struct {
	widget.WidgetBase
}

func newHeroArtwork() *heroArtworkWidget {
	h := &heroArtworkWidget{}
	h.SetVisible(true)
	h.SetEnabled(true)
	return h
}

func (h *heroArtworkWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := geometry.Sz(280, 170)
	h.SetBounds(geometry.FromPointSize(h.Bounds().Min, size))
	return size
}

func (h *heroArtworkWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := h.Bounds()
	if b.IsEmpty() {
		return
	}

	// Stylized futuristic ambient cards
	center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
	canvas.DrawCircle(center, 75, widget.RGBA8(92, 225, 182, 25))

	// Metallic layered silhouettes
	for i := 0; i < 3; i++ {
		offsetX := float32(i) * 50
		headRect := geometry.Rect{
			Min: geometry.Pt(b.Min.X+40+offsetX, b.Min.Y+20),
			Max: geometry.Pt(b.Min.X+110+offsetX, b.Max.Y-20),
		}
		canvas.DrawRoundRect(headRect, widget.RGBA8(40, 80, 88, uint8(100+i*40)), 35)
		canvas.StrokeRoundRect(headRect, widget.RGBA8(92, 225, 182, uint8(60+i*30)), 35, 1.5)
	}

	canvas.DrawCircle(geometry.Pt(b.Max.X-30, b.Min.Y+25), 4, theme.ColorAccent)
}

func (h *heroArtworkWidget) Event(ctx widget.Context, e event.Event) bool { return false }
func (h *heroArtworkWidget) Children() []widget.Widget                    { return nil }
