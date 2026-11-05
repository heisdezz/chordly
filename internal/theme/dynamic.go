package theme

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/gogpu/ui/widget"
)

// Palette represents a full dynamic Metrolist UI theme.
type Palette struct {
	Canvas        widget.Color
	Sidebar       widget.Color
	Surface       widget.Color
	SurfaceLight  widget.Color
	SurfaceSubtle widget.Color
	PlayerBar     widget.Color

	Accent      widget.Color
	AccentHover widget.Color
	AccentSoft  widget.Color
	OnAccent    widget.Color

	Border      widget.Color
	BorderMuted widget.Color

	TextPrimary   widget.Color
	TextSecondary widget.Color
	TextMuted     widget.Color
}

// DefaultPalette returns the default Metrolist slate & mint theme.
func DefaultPalette() Palette {
	return Palette{
		Canvas:        ColorCanvas,
		Sidebar:       ColorSidebar,
		Surface:       ColorSurface,
		SurfaceLight:  ColorSurfaceLight,
		SurfaceSubtle: ColorSurfaceSubtle,
		PlayerBar:     ColorPlayerBar,

		Accent:      ColorAccent,
		AccentHover: ColorAccentHover,
		AccentSoft:  ColorAccentSoft,
		OnAccent:    ColorOnAccent,

		Border:      ColorBorder,
		BorderMuted: ColorBorderMuted,

		TextPrimary:   ColorTextPrimary,
		TextSecondary: ColorTextSecondary,
		TextMuted:     ColorTextMuted,
	}
}

// LerpColor smoothly interpolates between two widget colors with factor t (0.0 to 1.0).
func LerpColor(from, to widget.Color, t float32) widget.Color {
	if t <= 0 {
		return from
	}
	if t >= 1 {
		return to
	}

	clamp := func(val float32) float32 {
		if val < 0 {
			return 0
		}
		if val > 1 {
			return 1
		}
		return val
	}

	return widget.Color{
		R: clamp(from.R + (to.R-from.R)*t),
		G: clamp(from.G + (to.G-from.G)*t),
		B: clamp(from.B + (to.B-from.B)*t),
		A: clamp(from.A + (to.A-from.A)*t),
	}
}

// LerpPalette interpolates all color tokens from palette a to palette b with factor t.
func LerpPalette(from, to Palette, t float32) Palette {
	if t <= 0 {
		return from
	}
	if t >= 1 {
		return to
	}

	return Palette{
		Canvas:        LerpColor(from.Canvas, to.Canvas, t),
		Sidebar:       LerpColor(from.Sidebar, to.Sidebar, t),
		Surface:       LerpColor(from.Surface, to.Surface, t),
		SurfaceLight:  LerpColor(from.SurfaceLight, to.SurfaceLight, t),
		SurfaceSubtle: LerpColor(from.SurfaceSubtle, to.SurfaceSubtle, t),
		PlayerBar:     LerpColor(from.PlayerBar, to.PlayerBar, t),

		Accent:      LerpColor(from.Accent, to.Accent, t),
		AccentHover: LerpColor(from.AccentHover, to.AccentHover, t),
		AccentSoft:  LerpColor(from.AccentSoft, to.AccentSoft, t),
		OnAccent:    LerpColor(from.OnAccent, to.OnAccent, t),

		Border:      LerpColor(from.Border, to.Border, t),
		BorderMuted: LerpColor(from.BorderMuted, to.BorderMuted, t),

		TextPrimary:   LerpColor(from.TextPrimary, to.TextPrimary, t),
		TextSecondary: LerpColor(from.TextSecondary, to.TextSecondary, t),
		TextMuted:     LerpColor(from.TextMuted, to.TextMuted, t),
	}
}

// Blend mixes base color with a tint color at the given tint ratio (0.0 to 1.0).
func Blend(base, tint widget.Color, tintRatio float32) widget.Color {
	return LerpColor(base, tint, tintRatio)
}

func colorToRGB8(c widget.Color) [3]uint8 {
	to8 := func(v float32) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 1 {
			return 255
		}
		return uint8(v*255.0 + 0.5)
	}
	return [3]uint8{to8(c.R), to8(c.G), to8(c.B)}
}

// DerivePalette constructs a harmonized Material You palette from a dominant artwork color.
func DerivePalette(dominant widget.Color) Palette {
	// 1. Boost saturation of accent if below 50%
	domRGB := colorToRGB8(dominant)
	sat := Saturation(domRGB)
	accent := dominant
	if sat < 0.50 {
		accent = boostSaturation(dominant, 0.55)
	}

	// 2. Compute OnAccent readability contrast
	accentRGB := colorToRGB8(accent)
	accentLum := Luminance(accentRGB)
	onAccent := widget.Hex(0x091D16) // Deep slate text
	if accentLum < 0.40 {
		onAccent = widget.Hex(0xFFFFFF) // Pure white text
	}

	// 3. Derive accent variations
	accentHover := LerpColor(accent, widget.Hex(0xFFFFFF), 0.15)
	accentSoft := LerpColor(ColorSurface, accent, 0.18)

	// 4. Subtle tinted surfaces (8% to 15% tint blended with base dark slate)
	canvas := Blend(ColorCanvas, dominant, 0.08)
	sidebar := Blend(ColorSidebar, dominant, 0.06)
	surface := Blend(ColorSurface, dominant, 0.12)
	surfaceLight := Blend(ColorSurfaceLight, dominant, 0.15)
	surfaceSubtle := Blend(ColorSurfaceSubtle, dominant, 0.09)
	playerBar := Blend(ColorPlayerBar, dominant, 0.14)

	border := Blend(ColorBorder, dominant, 0.15)
	borderMuted := Blend(ColorBorderMuted, dominant, 0.10)

	return Palette{
		Canvas:        canvas,
		Sidebar:       sidebar,
		Surface:       surface,
		SurfaceLight:  surfaceLight,
		SurfaceSubtle: surfaceSubtle,
		PlayerBar:     playerBar,

		Accent:      accent,
		AccentHover: accentHover,
		AccentSoft:  accentSoft,
		OnAccent:    onAccent,

		Border:      border,
		BorderMuted: borderMuted,

		TextPrimary:   ColorTextPrimary,
		TextSecondary: ColorTextSecondary,
		TextMuted:     ColorTextMuted,
	}
}

// boostSaturation scales RGB components to increase chroma while preserving perceptual lightness.
func boostSaturation(c widget.Color, targetSat float32) widget.Color {
	r := c.R
	g := c.G
	b := c.B

	gray := 0.2126*r + 0.7152*g + 0.0722*b
	denom := float32(math.Max(0.01, float64(1.0-gray)))
	factor := targetSat / denom

	boost := func(val float32) float32 {
		adjusted := gray + (val-gray)*(1.0+factor)
		if adjusted < 0 {
			return 0
		}
		if adjusted > 1 {
			return 1
		}
		return adjusted
	}

	return widget.Color{
		R: boost(r),
		G: boost(g),
		B: boost(b),
		A: c.A,
	}
}

// TransitionEngine manages 250ms 60FPS animated color transitions between palettes.
type TransitionEngine struct {
	mu           sync.RWMutex
	current      Palette
	target       Palette
	cancelActive context.CancelFunc

	onUpdate func(Palette)
}

// NewTransitionEngine creates a new transition engine initialized with default palette.
func NewTransitionEngine(initial Palette, onUpdate func(Palette)) *TransitionEngine {
	return &TransitionEngine{
		current:  initial,
		target:   initial,
		onUpdate: onUpdate,
	}
}

// Current returns the current palette.
func (e *TransitionEngine) Current() Palette {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current
}

// TransitionTo smoothly transitions to target palette over the specified duration (typically 250ms).
func (e *TransitionEngine) TransitionTo(target Palette, duration time.Duration) {
	e.mu.Lock()
	if e.cancelActive != nil {
		e.cancelActive()
	}

	if duration <= 0 {
		e.current = target
		e.target = target
		e.cancelActive = nil
		onUpdate := e.onUpdate
		e.mu.Unlock()
		if onUpdate != nil {
			onUpdate(target)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancelActive = cancel
	from := e.current
	e.target = target
	onUpdate := e.onUpdate
	e.mu.Unlock()

	go func() {
		defer cancel()

		startTime := time.Now()
		ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				elapsed := now.Sub(startTime)
				t := float32(elapsed.Seconds() / duration.Seconds())
				if t >= 1.0 {
					t = 1.0
				}

				// Cubic ease-out curve: t = 1 - (1-t)^3
				inv := 1.0 - t
				easedT := 1.0 - inv*inv*inv

				interpolated := LerpPalette(from, target, easedT)

				e.mu.Lock()
				e.current = interpolated
				e.mu.Unlock()

				if onUpdate != nil {
					onUpdate(interpolated)
				}

				if t >= 1.0 {
					return
				}
			}
		}
	}()
}
