package theme

import (
	"image"
	"image/color"
	"testing"
)

func createSolidImage(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func createQuadrantImage(w, h int, c1, c2, c3, c4 color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	midX, midY := w/2, h/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch {
			case x < midX && y < midY:
				img.Set(x, y, c1)
			case x >= midX && y < midY:
				img.Set(x, y, c2)
			case x < midX && y >= midY:
				img.Set(x, y, c3)
			default:
				img.Set(x, y, c4)
			}
		}
	}
	return img
}

func TestExtractColorsNilOrEmpty(t *testing.T) {
	if _, err := ExtractColors(nil, 5); err != ErrNilImage {
		t.Errorf("expected ErrNilImage, got %v", err)
	}

	emptyImg := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if _, err := ExtractColors(emptyImg, 5); err != ErrEmptyImage {
		t.Errorf("expected ErrEmptyImage, got %v", err)
	}
}

func TestExtractColorsSolid(t *testing.T) {
	// A saturated mint color (R: 92, G: 225, B: 182)
	mint := color.RGBA{R: 92, G: 225, B: 182, A: 255}
	img := createSolidImage(128, 128, mint)

	clusters, err := ExtractColors(img, 3)
	if err != nil {
		t.Fatalf("ExtractColors failed: %v", err)
	}
	if len(clusters) == 0 {
		t.Fatalf("expected at least 1 cluster, got 0")
	}

	best := clusters[0]
	// Should be close to (92, 225, 182)
	if best.RGBA[0] < 85 || best.RGBA[0] > 100 ||
		best.RGBA[1] < 215 || best.RGBA[1] > 235 ||
		best.RGBA[2] < 175 || best.RGBA[2] > 190 {
		t.Errorf("unexpected extracted color: %+v", best.RGBA)
	}
	if best.Saturation < 0.5 {
		t.Errorf("expected high saturation, got %f", best.Saturation)
	}
}

func TestExtractColorsLuminanceFiltering(t *testing.T) {
	// Pure black (lum = 0 < 0.15) and pure white (lum = 1.0 > 0.85) should both be filtered out.
	blackImg := createSolidImage(64, 64, color.Black)
	clusters, err := ExtractColors(blackImg, 5)
	if err != nil {
		t.Fatalf("ExtractColors failed: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("expected near-black to be filtered out, got %d clusters", len(clusters))
	}

	whiteImg := createSolidImage(64, 64, color.White)
	clusters, err = ExtractColors(whiteImg, 5)
	if err != nil {
		t.Fatalf("ExtractColors failed: %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("expected near-white to be filtered out, got %d clusters", len(clusters))
	}
}

func TestExtractQuadrants(t *testing.T) {
	red := color.RGBA{R: 220, G: 20, B: 20, A: 255}
	green := color.RGBA{R: 20, G: 220, B: 20, A: 255}
	blue := color.RGBA{R: 20, G: 20, B: 220, A: 255}
	nearBlack := color.RGBA{R: 10, G: 10, B: 10, A: 255}

	img := createQuadrantImage(128, 128, red, green, blue, nearBlack)

	clusters, err := ExtractColors(img, 4)
	if err != nil {
		t.Fatalf("ExtractColors failed: %v", err)
	}

	// nearBlack should be filtered out, leaving 3 saturated quadrants
	if len(clusters) < 3 {
		t.Errorf("expected at least 3 clusters, got %d", len(clusters))
	}

	dom, ok := DominantColor(clusters)
	if !ok {
		t.Fatalf("expected DominantColor to return true")
	}
	if dom.A == 0 {
		t.Errorf("expected opaque dominant color")
	}
}

func BenchmarkExtract(b *testing.B) {
	mint := color.RGBA{R: 92, G: 225, B: 182, A: 255}
	navy := color.RGBA{R: 20, G: 40, B: 80, A: 255}
	crimson := color.RGBA{R: 200, G: 30, B: 50, A: 255}
	amber := color.RGBA{R: 230, G: 180, B: 40, A: 255}
	img := createQuadrantImage(256, 256, mint, navy, crimson, amber)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Extract(img)
	}
}
