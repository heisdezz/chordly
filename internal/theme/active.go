package theme

import (
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// Reactive theme state. Widgets that are built from the package-level Color*
// tokens can subscribe to SigActivePalette (through appui's reactiveView) and
// rebuild after ApplyPalette.
var (
	// SigActivePalette holds the currently applied palette.
	SigActivePalette = state.NewSignal(DefaultPalette())

	// SigActivePresetID identifies the active preset, or "custom" after a
	// custom accent is applied.
	SigActivePresetID = state.NewSignal(PresetEmerald.ID)

	// SigActiveIsDark reports whether the active palette uses a dark canvas.
	SigActiveIsDark = state.NewSignal(true)
)

// ApplyPalette copies p into the legacy package-level Color* tokens and notifies
// SigActivePalette subscribers. It must be called on the UI thread; existing
// widgets read the tokens while being built or painted.
func ApplyPalette(p Palette) {
	ColorCanvas = p.Canvas
	ColorSidebar = p.Sidebar
	ColorSurface = p.Surface
	ColorSurfaceLight = p.SurfaceLight
	ColorSurfaceSubtle = p.SurfaceSubtle
	ColorPlayerBar = p.PlayerBar

	ColorAccent = p.Accent
	ColorAccentHover = p.AccentHover
	ColorAccentSoft = p.AccentSoft
	ColorOnAccent = p.OnAccent

	ColorBorder = p.Border
	ColorBorderMuted = p.BorderMuted

	ColorTextPrimary = p.TextPrimary
	ColorTextSecondary = p.TextSecondary
	ColorTextMuted = p.TextMuted

	// Tokens that are not part of Palette but must follow its contrast level.
	ColorScrubberTrack = p.BorderMuted
	ColorCardOverlay = p.SurfaceLight

	SigActiveIsDark.Set(IsDarkPalette(p))
	SigActivePalette.Set(p)
}

// ApplyPreset applies a built-in preset and records it as the active selection.
func ApplyPreset(p Preset) {
	SigActivePresetID.Set(p.ID)
	ApplyPalette(p.Palette)
}

// ApplyCustomAccent derives a palette from a custom accent and records the
// selection as "custom".
func ApplyCustomAccent(accent widget.Color, isDark bool) {
	SigActivePresetID.Set("custom")
	ApplyPalette(NewCustomPalette(accent, isDark))
}

// ActivePalette returns the currently applied palette.
func ActivePalette() Palette {
	return SigActivePalette.Get()
}

// IsDarkPalette reports whether a palette's canvas is visually dark.
func IsDarkPalette(p Palette) bool {
	return Luminance(colorToRGB8(p.Canvas)) < 0.5
}
