package theme

import (
	"testing"

	"github.com/gogpu/ui/widget"
)

func TestApplyPresetUpdatesLegacyTokensAndSignals(t *testing.T) {
	t.Cleanup(func() { ApplyPreset(PresetEmerald) })

	ApplyPreset(PresetAMOLED)
	if got := SigActivePresetID.Get(); got != PresetAMOLED.ID {
		t.Fatalf("active preset = %q, want %q", got, PresetAMOLED.ID)
	}
	if got := ColorCanvas; got != PresetAMOLED.Palette.Canvas {
		t.Errorf("ColorCanvas = %v, want %v", got, PresetAMOLED.Palette.Canvas)
	}
	if got := ColorAccent; got != PresetAMOLED.Palette.Accent {
		t.Errorf("ColorAccent = %v, want %v", got, PresetAMOLED.Palette.Accent)
	}
	if !SigActiveIsDark.Get() {
		t.Error("AMOLED preset should report dark mode")
	}
	if got := SigActivePalette.Get(); got.Canvas != PresetAMOLED.Palette.Canvas {
		t.Errorf("active palette canvas = %v, want %v", got.Canvas, PresetAMOLED.Palette.Canvas)
	}

	ApplyPreset(PresetMaterialLight)
	if SigActiveIsDark.Get() {
		t.Error("Material Light preset should report light mode")
	}
	if got := ColorTextPrimary; got != PresetMaterialLight.Palette.TextPrimary {
		t.Errorf("ColorTextPrimary = %v, want %v", got, PresetMaterialLight.Palette.TextPrimary)
	}
}

func TestApplyCustomAccentDerivesPalette(t *testing.T) {
	t.Cleanup(func() { ApplyPreset(PresetEmerald) })

	before := ColorCanvas
	accent := widget.Hex(0x3B82F6)
	ApplyCustomAccent(accent, true)

	if got := SigActivePresetID.Get(); got != "custom" {
		t.Fatalf("active preset = %q, want custom", got)
	}
	if got := ColorAccent; got != accent {
		t.Errorf("ColorAccent = %v, want %v", got, accent)
	}
	if got := ColorCanvas; got == before {
		t.Error("custom palette did not change the canvas")
	}
}
