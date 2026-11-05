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

// NewMadeForYouSection renders the "Made for you" mix cards row. It rebuilds
// when the live home feed replaces the mixes.
func NewMadeForYouSection() widget.Widget {
	return newReactiveView(
		buildMadeForYouContent,
		WatchSignal(state.MadeForYouMixes.AsReadonly()),
	)
}

func buildMadeForYouContent() widget.Widget {
	header := primitives.HBox(
		primitives.Text("Made for you").FontSize(20).Bold().Color(theme.ColorTextPrimary),
		primitives.Expanded(primitives.Box()),
		primitives.Text("See all →").FontSize(13).Color(theme.ColorTextMuted),
	).CrossAlign(primitives.CrossAxisCenter)

	mixes := state.MadeForYouMixes.Get()
	cards := make([]widget.Widget, 0, len(mixes))
	for _, mix := range mixes {
		cards = append(cards, primitives.Expanded(newMixCard(mix)))
	}

	cardRow := primitives.HBox(cards...).Gap(12)

	return primitives.VBox(
		header,
		primitives.Box().Height(12),
		cardRow,
	)
}

// mixCardPainter renders a vibrant gradient card with a bottom white play button.
type mixCardPainter struct {
	mix model.MixCard
}

func (p mixCardPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	baseColor := widget.Hex(p.mix.ColorHex)
	if s.Hovered {
		baseColor = baseColor.Lerp(widget.ColorWhite, 0.08)
	}

	// Card background with rounded corners
	canvas.DrawRoundRect(b, baseColor, 18)
	canvas.StrokeRoundRect(b, theme.ColorBorder, 18, 1)

	// Sheen overlay
	sheenRect := geometry.Rect{
		Min: b.Min,
		Max: geometry.Pt(b.Max.X, b.Min.Y+b.Height()*0.6),
	}
	canvas.DrawRoundRect(sheenRect, widget.RGBA8(255, 255, 255, 20), 18)

	// Card Title (Bottom Left), fitted so it never runs into the play button.
	titleRect := geometry.Rect{
		Min: geometry.Pt(b.Min.X+12, b.Max.Y-40),
		Max: geometry.Pt(b.Max.X-44, b.Max.Y-10),
	}
	title := fitText(canvas, p.mix.Title, 13, true, titleRect.Width())
	canvas.DrawText(title, titleRect, 13, widget.ColorWhite, true, widget.TextAlignLeft)

	// Play button circle (Bottom Right)
	playCenter := geometry.Pt(b.Max.X-24, b.Max.Y-24)
	playRadius := float32(14)
	canvas.DrawCircle(playCenter, playRadius, widget.ColorWhite)
	playRect := geometry.Rect{
		Min: geometry.Pt(playCenter.X-playRadius, playCenter.Y-playRadius),
		Max: geometry.Pt(playCenter.X+playRadius, playCenter.Y+playRadius),
	}
	DrawVectorIcon(canvas, IconPlay, playRect, widget.ColorBlack, 11)
}

func (p mixCardPainter) ButtonHeight(button.Size) float32             { return 130 }
func (p mixCardPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p mixCardPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p mixCardPainter) ButtonRadius() float32                        { return 18 }

var _ button.Painter = mixCardPainter{}
var _ button.LayoutMetrics = mixCardPainter{}

func newMixCard(mix model.MixCard) *button.Widget {
	return button.New(
		button.Text(mix.Title),
		button.OnClick(func() {
			// Trigger play of mix
		}),
		button.PainterOpt(mixCardPainter{mix: mix}),
	)
}
