package theme

import (
	"errors"
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/gogpu/ui/widget"
)

// solid renders a w×h image of a single color.
func solid(w, h int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// split renders a horizontal gradient between two colors (left/right halves).
func split(w, h int, left, right color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.SetRGBA(x, y, left)
			} else {
				img.SetRGBA(x, y, right)
			}
		}
	}
	return img
}

func TestExtractNilAndEmpty(t *testing.T) {
	if _, err := Extract(nil); !errors.Is(err, ErrNilImage) {
		t.Errorf("Extract(nil) = %v, want ErrNilImage", err)
	}
	empty := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if _, err := Extract(empty); !errors.Is(err, ErrEmptyImage) {
		t.Errorf("Extract(empty) = %v, want ErrEmptyImage", err)
	}
	// Fully transparent: all pixels alpha 0 → nothing to quantise.
	blank := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if _, err := Extract(blank); !errors.Is(err, ErrEmptyImage) {
		t.Errorf("Extract(transparent) = %v, want ErrEmptyImage", err)
	}
}

func TestExtractSolidColor(t *testing.T) {
	// Pure green (0,255,0): L=0.7152, saturation 1.
	img := solid(64, 64, color.RGBA{0, 255, 0, 0xff})
	clusters, err := Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1", len(clusters))
	}
	c := clusters[0]
	if c.RGBA != [3]uint8{0, 255, 0} {
		t.Errorf("RGBA = %v, want [0 255 0]", c.RGBA)
	}
	if math.Abs(c.Population-1) > 1e-9 {
		t.Errorf("Population = %v, want 1", c.Population)
	}
	if math.Abs(c.Saturation-1) > 1e-9 {
		t.Errorf("Saturation = %v, want 1", c.Saturation)
	}
	// Score = 1*0.6 + 1*0.4 = 1
	if math.Abs(c.Score-1) > 1e-9 {
		t.Errorf("Score = %v, want 1", c.Score)
	}
	// widget.Color round-trip via RGBA8()
	r, g, b, a := c.Color.RGBA8()
	if r != 0 || g != 255 || b != 0 || a != 255 {
		t.Errorf("widget.Color = (%d,%d,%d,%d), want (0,255,0,255)", r, g, b, a)
	}
}

func TestExtractDownsamplesLargeImage(t *testing.T) {
	// 512x512 image downsampled to 64x64.
	img := solid(512, 512, color.RGBA{30, 144, 255, 0xff}) // dodger blue
	clusters, err := ExtractColors(img, 3)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1 (solid input)", len(clusters))
	}
	c := clusters[0]
	// Centroid may drift by ~1 due to bilinear rounding at edges.
	for i, want := range [3]int{30, 144, 255} {
		if math.Abs(float64(int(c.RGBA[i])-want)) > 2 {
			t.Errorf("RGBA[%d] = %d, want %d", i, c.RGBA[i], want)
		}
	}
}

func TestExtractTwoRegions(t *testing.T) {
	// Left 50% orange, right 50% blue — two luminance-filtered clusters.
	// Pure blue has L=0.0722 < 0.15 (near-black by the plan metric) and is
	// filtered, so we use orange (L≈0.68) + bright green (L≈0.72) instead.
	img := split(64, 64, color.RGBA{255, 165, 0, 0xff}, color.RGBA{0, 255, 0, 0xff})
	clusters, err := ExtractColors(img, 2)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("clusters = %d, want 2", len(clusters))
	}
	totalPop := clusters[0].Population + clusters[1].Population
	if math.Abs(totalPop-1) > 1e-9 {
		t.Errorf("Population sum = %v, want 1", totalPop)
	}
	for _, c := range clusters {
		if c.Population < 0.3 || c.Population > 0.7 {
			t.Errorf("cluster %v Population = %v, want ~0.5", c.RGBA, c.Population)
		}
	}
}

func TestExtractFiltersNearBlackAndWhite(t *testing.T) {
	// All near-black (L<0.15): (0,10,0) → L≈0.028. Result must be empty.
	img := solid(32, 32, color.RGBA{0, 10, 0, 0xff})
	clusters, err := Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("clusters = %d, want 0 (near-black filtered)", len(clusters))
	}
	if _, ok := DominantColor(clusters); ok {
		t.Error("DominantColor returned true for empty clusters")
	}

	// All near-white (L>0.85): (250,250,250) → L≈0.98.
	img = solid(32, 32, color.RGBA{250, 250, 250, 0xff})
	clusters, err = Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("clusters = %d, want 0 (near-white filtered)", len(clusters))
	}
}

func TestExtractScoresSaturationOverBlack(t *testing.T) {
	// 75% vivid red, 25% near-black. Vivid wins on saturation despite
	// the black cluster's larger population being excluded anyway.
	red := color.RGBA{255, 0, 0, 0xff}
	black := color.RGBA{5, 5, 5, 0xff}
	h := 64
	w := 64
	blackRows := h / 4
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if y < blackRows {
				img.SetRGBA(x, y, black)
			} else {
				img.SetRGBA(x, y, red)
			}
		}
	}
	clusters, err := ExtractColors(img, 2)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1 (black filtered)", len(clusters))
	}
	if clusters[0].RGBA != [3]uint8{255, 0, 0} {
		t.Errorf("winner = %v, want red", clusters[0].RGBA)
	}
	if clusters[0].Population < 0.6 {
		t.Errorf("Population = %v, want ≥0.6", clusters[0].Population)
	}
}

func TestExtractScoreFormula(t *testing.T) {
	// Verify Score = Saturation*0.6 + Population*0.4 for a single cluster.
	img := solid(16, 16, color.RGBA{200, 100, 50, 0xff})
	clusters, err := Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1", len(clusters))
	}
	c := clusters[0]
	want := c.Saturation*0.6 + c.Population*0.4
	if math.Abs(c.Score-want) > 1e-9 {
		t.Errorf("Score = %v, want %v", c.Score, want)
	}
	// Saturation of (200,100,50): (200-50)/200 = 0.75
	if math.Abs(c.Saturation-0.75) > 1e-9 {
		t.Errorf("Saturation = %v, want 0.75", c.Saturation)
	}
}

func TestExtractDominantShorthand(t *testing.T) {
	img := solid(32, 32, color.RGBA{120, 40, 180, 0xff})
	c, ok, err := ExtractDominant(img)
	if err != nil {
		t.Fatalf("ExtractDominant: %v", err)
	}
	if !ok {
		t.Fatal("ExtractDominant returned ok=false for solid color")
	}
	// L = 0.2126*120/255 + 0.7152*40/255 + 0.0722*180/255 ≈ 0.263, in range.
	for i, want := range [3]uint8{120, 40, 180} {
		if c.RGBA[i] != want {
			t.Errorf("RGBA[%d] = %d, want %d", i, c.RGBA[i], want)
		}
	}
}

func TestExtractDeterministic(t *testing.T) {
	// Same image must produce identical clusters (no RNG in kMeans).
	img := solid(64, 64, color.RGBA{90, 180, 60, 0xff})
	a, err := Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	b, err := Extract(img)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("len mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("clusters differ: %v vs %v", a[i], b[i])
		}
	}
}

func TestExtractPerformance(t *testing.T) {
	// Extraction of a 64x64 image must stay well under the plan's 3ms
	// budget in practice (generous threshold here for CI headroom).
	img := solid(64, 64, color.RGBA{200, 30, 30, 0xff})
	start := time.Now()
	for i := 0; i < 100; i++ {
		if _, err := Extract(img); err != nil {
			t.Fatalf("Extract: %v", err)
		}
	}
	elapsed := time.Since(start)
	// -race inflates timings; keep a generous ceiling while still catching
	// pathological regressions (full 512x512 without downsample would be
	// orders of magnitude slower).
	if elapsed > 5*time.Second {
		t.Errorf("100 extractions took %v, want < 5s", elapsed)
	}
}

func TestDominantColorPicksHighestScore(t *testing.T) {
	clusters := []Cluster{
		{Color: widget.RGBA8(255, 0, 0, 0xff), RGBA: [3]uint8{255, 0, 0}, Score: 0.5},
		{Color: widget.RGBA8(0, 255, 0, 0xff), RGBA: [3]uint8{0, 255, 0}, Score: 0.9},
	}
	c, ok := DominantColor(clusters)
	if !ok {
		t.Fatal("DominantColor ok=false")
	}
	r, g, _, _ := c.RGBA8()
	if r != 0 || g != 255 {
		t.Errorf("dominant = (%d,%d), want green", r, g)
	}
}

func TestLuminanceAndSaturation(t *testing.T) {
	// White → L=1, black → L=0.
	if got := luminance(255, 255, 255); math.Abs(got-1) > 1e-9 {
		t.Errorf("luminance(white) = %v, want 1", got)
	}
	if got := luminance(0, 0, 0); got != 0 {
		t.Errorf("luminance(black) = %v, want 0", got)
	}
	// Gray → saturation 0; red → saturation 1.
	if got := saturation(128, 128, 128); got != 0 {
		t.Errorf("saturation(gray) = %v, want 0", got)
	}
	if got := saturation(255, 0, 0); math.Abs(got-1) > 1e-9 {
		t.Errorf("saturation(red) = %v, want 1", got)
	}
	// Public wrappers must agree with internal helpers.
	if math.Abs(Luminance([3]uint8{255, 255, 255})-1) > 1e-9 {
		t.Error("Luminance(white) != 1")
	}
	if math.Abs(Saturation([3]uint8{255, 0, 0})-1) > 1e-9 {
		t.Error("Saturation(red) != 1")
	}
}

func TestExtractNonPositiveKUsesDefault(t *testing.T) {
	img := solid(32, 32, color.RGBA{50, 200, 100, 0xff})
	clusters, err := ExtractColors(img, 0)
	if err != nil {
		t.Fatalf("ExtractColors(0): %v", err)
	}
	if len(clusters) != 1 {
		t.Errorf("clusters = %d, want 1", len(clusters))
	}
}

func TestExtractKClampedToPixels(t *testing.T) {
	// K > pixel count must not panic; clamped internally.
	img := solid(4, 4, color.RGBA{10, 200, 10, 0xff})
	clusters, err := ExtractColors(img, 1000)
	if err != nil {
		t.Fatalf("ExtractColors(1000): %v", err)
	}
	if len(clusters) == 0 || len(clusters) > 16 {
		t.Errorf("clusters = %d, want ≤16", len(clusters))
	}
}
