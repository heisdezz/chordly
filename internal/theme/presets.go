package theme

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gogpu/ui/widget"
)

// Preset identifies a pre-configured theme.
type Preset struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	AccentHex   uint32  `json:"accent_hex"`
	IsDark      bool    `json:"is_dark"`
	Palette     Palette `json:"-"`
}

var (
	// PresetEmerald is the default Metrolist slate & mint theme.
	PresetEmerald = Preset{
		ID:          "emerald",
		Name:        "Emerald Slate",
		Description: "Signature Metrolist deep dark slate with glowing mint accents",
		AccentHex:   0x5CE1B6,
		IsDark:      true,
		Palette:     DefaultPalette(),
	}

	// PresetAMOLED is a battery-saving pure-black theme for OLED displays.
	PresetAMOLED = Preset{
		ID:          "amoled",
		Name:        "Pure AMOLED",
		Description: "Absolute pitch black surfaces with vibrant neon mint accents",
		AccentHex:   0x5CE1B6,
		IsDark:      true,
		Palette: Palette{
			Canvas:        widget.Hex(0x000000),
			Sidebar:       widget.Hex(0x000000),
			Surface:       widget.Hex(0x0A0A0A),
			SurfaceLight:  widget.Hex(0x161616),
			SurfaceSubtle: widget.Hex(0x0E0E0E),
			PlayerBar:     widget.Hex(0x050505),

			Accent:      widget.Hex(0x5CE1B6),
			AccentHover: widget.Hex(0x4FD4A9),
			AccentSoft:  widget.Hex(0x003322),
			OnAccent:    widget.Hex(0x000000),

			Border:      widget.Hex(0x1F1F1F),
			BorderMuted: widget.Hex(0x141414),

			TextPrimary:   widget.Hex(0xFFFFFF),
			TextSecondary: widget.Hex(0xA0A0A0),
			TextMuted:     widget.Hex(0x666666),
		},
	}

	// PresetMaterialLight is a clean, modern daylight theme.
	PresetMaterialLight = Preset{
		ID:          "material_light",
		Name:        "Material Light",
		Description: "Crisp light surfaces with deep teal and emerald accents",
		AccentHex:   0x10B981,
		IsDark:      false,
		Palette: Palette{
			Canvas:        widget.Hex(0xF8FAFC),
			Sidebar:       widget.Hex(0xF1F5F9),
			Surface:       widget.Hex(0xFFFFFF),
			SurfaceLight:  widget.Hex(0xF1F5F9),
			SurfaceSubtle: widget.Hex(0xE2E8F0),
			PlayerBar:     widget.Hex(0xFFFFFF),

			Accent:      widget.Hex(0x0D9488),
			AccentHover: widget.Hex(0x0F766E),
			AccentSoft:  widget.Hex(0xCCFBF1),
			OnAccent:    widget.Hex(0xFFFFFF),

			Border:      widget.Hex(0xE2E8F0),
			BorderMuted: widget.Hex(0xCBD5E1),

			TextPrimary:   widget.Hex(0x0F172A),
			TextSecondary: widget.Hex(0x475569),
			TextMuted:     widget.Hex(0x94A3B8),
		},
	}

	// PresetOceanBlue is a cool, calm oceanic midnight palette.
	PresetOceanBlue = Preset{
		ID:          "ocean_blue",
		Name:        "Deep Ocean",
		Description: "Midnight maritime indigo with crisp electric blue accents",
		AccentHex:   0x38BDF8,
		IsDark:      true,
		Palette: Palette{
			Canvas:        widget.Hex(0x080E18),
			Sidebar:       widget.Hex(0x0B1320),
			Surface:       widget.Hex(0x0F1A2A),
			SurfaceLight:  widget.Hex(0x16253C),
			SurfaceSubtle: widget.Hex(0x111C2E),
			PlayerBar:     widget.Hex(0x0C1624),

			Accent:      widget.Hex(0x38BDF8),
			AccentHover: widget.Hex(0x0EA5E9),
			AccentSoft:  widget.Hex(0x0C2D48),
			OnAccent:    widget.Hex(0x04131E),

			Border:      widget.Hex(0x1C2D46),
			BorderMuted: widget.Hex(0x132034),

			TextPrimary:   widget.Hex(0xF0F6FC),
			TextSecondary: widget.Hex(0x94A9C4),
			TextMuted:     widget.Hex(0x5A7394),
		},
	}

	// PresetPurpleGlow is a vibrant cyberpunk synthwave palette.
	PresetPurpleGlow = Preset{
		ID:          "purple_glow",
		Name:        "Cyber Purple",
		Description: "Deep neon violet with glowing lavender accents",
		AccentHex:   0xC084FC,
		IsDark:      true,
		Palette: Palette{
			Canvas:        widget.Hex(0x0F0918),
			Sidebar:       widget.Hex(0x140C20),
			Surface:       widget.Hex(0x1B102C),
			SurfaceLight:  widget.Hex(0x26173E),
			SurfaceSubtle: widget.Hex(0x1E1231),
			PlayerBar:     widget.Hex(0x150D23),

			Accent:      widget.Hex(0xC084FC),
			AccentHover: widget.Hex(0xA855F7),
			AccentSoft:  widget.Hex(0x3B175E),
			OnAccent:    widget.Hex(0x1A0830),

			Border:      widget.Hex(0x2F1C4D),
			BorderMuted: widget.Hex(0x211337),

			TextPrimary:   widget.Hex(0xF7F2FA),
			TextSecondary: widget.Hex(0xB8A3CB),
			TextMuted:     widget.Hex(0x7C6590),
		},
	}

	// PresetSunsetAmber is a rich, warm sunset palette.
	PresetSunsetAmber = Preset{
		ID:          "sunset_amber",
		Name:        "Sunset Amber",
		Description: "Warm terracotta and roasted amber with fiery accents",
		AccentHex:   0xFB923C,
		IsDark:      true,
		Palette: Palette{
			Canvas:        widget.Hex(0x140B07),
			Sidebar:       widget.Hex(0x1B0E09),
			Surface:       widget.Hex(0x24140D),
			SurfaceLight:  widget.Hex(0x321C13),
			SurfaceSubtle: widget.Hex(0x28160E),
			PlayerBar:     widget.Hex(0x1C0F0A),

			Accent:      widget.Hex(0xFB923C),
			AccentHover: widget.Hex(0xF97316),
			AccentSoft:  widget.Hex(0x4D2411),
			OnAccent:    widget.Hex(0x210B03),

			Border:      widget.Hex(0x3E2317),
			BorderMuted: widget.Hex(0x2C1810),

			TextPrimary:   widget.Hex(0xFAF4F0),
			TextSecondary: widget.Hex(0xCCAAA0),
			TextMuted:     widget.Hex(0x8D6F64),
		},
	}
)

var defaultPresets = []Preset{
	PresetEmerald,
	PresetAMOLED,
	PresetMaterialLight,
	PresetOceanBlue,
	PresetPurpleGlow,
	PresetSunsetAmber,
}

// ListPresets returns all available built-in theme presets.
func ListPresets() []Preset {
	out := make([]Preset, len(defaultPresets))
	copy(out, defaultPresets)
	return out
}

// GetPreset looks up a theme preset by its unique ID.
func GetPreset(id string) (Preset, bool) {
	for _, p := range defaultPresets {
		if strings.EqualFold(p.ID, id) {
			return p, true
		}
	}
	return Preset{}, false
}

// ParseHexColor parses a color hex string like "#5CE1B6" or "5CE1B6" into a widget.Color.
func ParseHexColor(hex string) (widget.Color, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(clean) == 3 {
		// Expand short form #RGB -> #RRGGBB
		clean = string([]byte{clean[0], clean[0], clean[1], clean[1], clean[2], clean[2]})
	}
	if len(clean) != 6 && len(clean) != 8 {
		return widget.ColorTransparent, fmt.Errorf("invalid hex color string: %s", hex)
	}

	val, err := strconv.ParseUint(clean, 16, 32)
	if err != nil {
		return widget.ColorTransparent, fmt.Errorf("parse hex failed: %w", err)
	}

	if len(clean) == 6 {
		return widget.Hex(uint32(val)), nil
	}

	// 8 chars: RRGGBBAA
	r := uint8((val >> 24) & 0xFF)
	g := uint8((val >> 16) & 0xFF)
	b := uint8((val >> 8) & 0xFF)
	a := uint8(val & 0xFF)
	return widget.RGBA8(r, g, b, a), nil
}

// FormatHexColor formats a widget.Color as a 6-digit hex string "#RRGGBB".
func FormatHexColor(c widget.Color) string {
	to8 := func(v float32) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 1 {
			return 255
		}
		return uint8(v*255.0 + 0.5)
	}
	return fmt.Sprintf("#%02X%02X%02X", to8(c.R), to8(c.G), to8(c.B))
}

// NewCustomPalette creates a fully styled palette from a custom hex accent color and dark/light flag.
func NewCustomPalette(accent widget.Color, isDark bool) Palette {
	if isDark {
		return DerivePalette(accent)
	}

	// Light mode derivation
	accentRGB := colorToRGB8(accent)
	lum := Luminance(accentRGB)
	onAccent := widget.Hex(0xFFFFFF)
	if lum > 0.60 {
		onAccent = widget.Hex(0x0F172A)
	}

	return Palette{
		Canvas:        Blend(widget.Hex(0xF8FAFC), accent, 0.04),
		Sidebar:       Blend(widget.Hex(0xF1F5F9), accent, 0.05),
		Surface:       widget.Hex(0xFFFFFF),
		SurfaceLight:  Blend(widget.Hex(0xF1F5F9), accent, 0.06),
		SurfaceSubtle: Blend(widget.Hex(0xE2E8F0), accent, 0.05),
		PlayerBar:     widget.Hex(0xFFFFFF),

		Accent:      accent,
		AccentHover: LerpColor(accent, widget.Hex(0x000000), 0.12),
		AccentSoft:  Blend(widget.Hex(0xFFFFFF), accent, 0.20),
		OnAccent:    onAccent,

		Border:      Blend(widget.Hex(0xE2E8F0), accent, 0.08),
		BorderMuted: Blend(widget.Hex(0xCBD5E1), accent, 0.06),

		TextPrimary:   widget.Hex(0x0F172A),
		TextSecondary: widget.Hex(0x475569),
		TextMuted:     widget.Hex(0x94A3B8),
	}
}
