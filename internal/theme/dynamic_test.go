package theme

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/gogpu/ui/widget"
)

func closeColors(c1, c2 widget.Color, eps float32) bool {
	return math.Abs(float64(c1.R-c2.R)) < float64(eps) &&
		math.Abs(float64(c1.G-c2.G)) < float64(eps) &&
		math.Abs(float64(c1.B-c2.B)) < float64(eps) &&
		math.Abs(float64(c1.A-c2.A)) < float64(eps)
}

func TestLerpColor(t *testing.T) {
	c1 := widget.RGBA8(0, 100, 200, 255)
	c2 := widget.RGBA8(100, 200, 100, 255)

	// t = 0
	if got := LerpColor(c1, c2, 0); got != c1 {
		t.Errorf("expected c1, got %+v", got)
	}

	// t = 1
	if got := LerpColor(c1, c2, 1); got != c2 {
		t.Errorf("expected c2, got %+v", got)
	}

	// t = 0.5
	mid := LerpColor(c1, c2, 0.5)
	expected := widget.RGBA8(50, 150, 150, 255)
	if !closeColors(mid, expected, 0.01) {
		t.Errorf("expected %+v, got %+v", expected, mid)
	}
}

func TestDerivePalette(t *testing.T) {
	// Sunny Yellow (high luminance)
	sunny := widget.RGBA8(250, 220, 50, 255)
	paletteLight := DerivePalette(sunny)

	// On sunny yellow, OnAccent must be dark text for readability
	if paletteLight.OnAccent != widget.Hex(0x091D16) {
		t.Errorf("expected dark onAccent for sunny yellow, got %+v", paletteLight.OnAccent)
	}

	// Deep Indigo (low luminance)
	indigo := widget.RGBA8(30, 20, 80, 255)
	paletteDark := DerivePalette(indigo)

	// On deep indigo, OnAccent must be white text
	if paletteDark.OnAccent != widget.Hex(0xFFFFFF) {
		t.Errorf("expected white onAccent for dark indigo, got %+v", paletteDark.OnAccent)
	}

	// Surfaces should remain dark and tinted (luminance < 0.25)
	if paletteLight.Canvas.R > 0.25 || paletteLight.Canvas.G > 0.25 || paletteLight.Canvas.B > 0.25 {
		t.Errorf("canvas should remain deep dark background, got %+v", paletteLight.Canvas)
	}
}

func TestTransitionEngine(t *testing.T) {
	initial := DefaultPalette()
	var mu sync.Mutex
	updateCount := 0

	engine := NewTransitionEngine(initial, func(p Palette) {
		mu.Lock()
		updateCount++
		mu.Unlock()
	})

	if engine.Current().Accent != initial.Accent {
		t.Errorf("initial accent mismatch")
	}

	// Instant transition
	target := DerivePalette(widget.RGBA8(220, 50, 80, 255))
	engine.TransitionTo(target, 0)

	if engine.Current().Accent != target.Accent {
		t.Errorf("expected instant transition to target accent")
	}

	// Animated 100ms transition
	nextTarget := DerivePalette(widget.RGBA8(50, 120, 240, 255))
	engine.TransitionTo(nextTarget, 100*time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	count := updateCount
	mu.Unlock()

	if count < 3 {
		t.Errorf("expected multiple animation frame updates, got %d", count)
	}

	current := engine.Current()
	if !closeColors(current.Accent, nextTarget.Accent, 0.05) {
		t.Errorf("expected current accent close to nextTarget, got %+v vs %+v", current.Accent, nextTarget.Accent)
	}
}

func TestTransitionEngineInterruption(t *testing.T) {
	initial := DefaultPalette()
	engine := NewTransitionEngine(initial, nil)

	p1 := DerivePalette(widget.RGBA8(200, 20, 20, 255))
	p2 := DerivePalette(widget.RGBA8(20, 200, 20, 255))

	// Start transition to p1, then immediately interrupt with p2
	engine.TransitionTo(p1, 200*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	engine.TransitionTo(p2, 100*time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	if !closeColors(engine.Current().Accent, p2.Accent, 0.05) {
		t.Errorf("expected final accent to reach interrupted target p2, got %+v vs %+v", engine.Current().Accent, p2.Accent)
	}
}
