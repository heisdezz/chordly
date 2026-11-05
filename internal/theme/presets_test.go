package theme

import (
	"testing"

	"github.com/gogpu/ui/widget"
)

func TestListAndGetPresets(t *testing.T) {
	presets := ListPresets()
	if len(presets) < 6 {
		t.Fatalf("expected at least 6 presets, got %d", len(presets))
	}

	// Case-insensitive lookup
	emerald, ok := GetPreset("EMERALD")
	if !ok {
		t.Fatalf("expected to find EMERALD preset")
	}
	if emerald.ID != "emerald" {
		t.Errorf("expected ID 'emerald', got '%s'", emerald.ID)
	}

	amoled, ok := GetPreset("amoled")
	if !ok || !amoled.IsDark {
		t.Errorf("expected dark AMOLED preset")
	}

	light, ok := GetPreset("material_light")
	if !ok || light.IsDark {
		t.Errorf("expected light Material Light preset")
	}

	_, ok = GetPreset("non_existent")
	if ok {
		t.Errorf("expected false for unknown preset")
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input       string
		expectedHex uint32
		wantErr     bool
	}{
		{"#5CE1B6", 0x5CE1B6, false},
		{"5CE1B6", 0x5CE1B6, false},
		{"#FFF", 0xFFFFFF, false},
		{"#000", 0x000000, false},
		{"invalid", 0, true},
		{"#12", 0, true},
		{"#12345", 0, true},
	}

	for _, tc := range tests {
		c, err := ParseHexColor(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("input %s: expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("input %s: unexpected error: %v", tc.input, err)
			continue
		}

		expected := widget.Hex(tc.expectedHex)
		if !closeColors(c, expected, 0.01) {
			t.Errorf("input %s: expected %+v, got %+v", tc.input, expected, c)
		}
	}

	// 8-digit hex test with alpha
	c8, err := ParseHexColor("#5CE1B680")
	if err != nil {
		t.Fatalf("unexpected error parsing 8-digit hex: %v", err)
	}
	if c8.A < 0.45 || c8.A > 0.55 {
		t.Errorf("expected ~50%% alpha, got %f", c8.A)
	}
}

func TestFormatHexColor(t *testing.T) {
	mint := widget.Hex(0x5CE1B6)
	formatted := FormatHexColor(mint)
	if formatted != "#5CE1B6" {
		t.Errorf("expected #5CE1B6, got %s", formatted)
	}
}

func TestNewCustomPalette(t *testing.T) {
	coral := widget.Hex(0xEF4444)

	darkPalette := NewCustomPalette(coral, true)
	if darkPalette.Canvas.R > 0.3 || darkPalette.Canvas.G > 0.3 || darkPalette.Canvas.B > 0.3 {
		t.Errorf("expected dark canvas in dark mode")
	}

	lightPalette := NewCustomPalette(coral, false)
	if lightPalette.Canvas.R < 0.8 || lightPalette.Canvas.G < 0.8 || lightPalette.Canvas.B < 0.8 {
		t.Errorf("expected light canvas in light mode")
	}
}
