// Package theme implements Chordly's dynamic theming: color extraction from
// album artwork (this file) and Material You palette derivation / LERP
// transitions (dynamic.go, TSK-202).
package theme

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/gogpu/ui/widget"
	"golang.org/x/image/draw"
)

// ExtractTargetSize is the downsampled edge length (in pixels) used for
// quantization. 64x64 is the plan spec (plan-2 Module 7 §7.1): small enough
// for <3ms extraction, large enough to preserve the album's color regions.
const ExtractTargetSize = 64

// DefaultExtractColors is the K used when callers do not specify a count.
const DefaultExtractColors = 5

var (
	// ErrNilImage is returned when Extract / ExtractColors is called with a
	// nil image.
	ErrNilImage = errors.New("theme: nil image")
	// ErrEmptyImage is returned when the (non-nil) image has zero extent.
	ErrEmptyImage = errors.New("theme: empty image")
)

// Cluster is one dominant color from ExtractColors: the representative color
// (as a widget.Color for direct UI use), its sRGB components, the fraction of
// pixels that belong to it (Population), and its final Score.
type Cluster struct {
	// Color is the cluster centroid as a GoGPU widget color.
	Color widget.Color
	// RGBA holds the centroid in 8-bit sRGB.
	RGBA [3]uint8
	// Population is the share of the downsampled pixels (0..1) assigned to
	// this cluster.
	Population float64
	// Score is Saturation*0.6 + Population*0.4 (plan-2 §7.1).
	Score float64
	// Saturation is the HSV saturation of the centroid (0..1).
	Saturation float64
}

// DominantColor returns the highest-scoring cluster's color, or false when
// the image contains no usable (non near-black / near-white) color.
func DominantColor(clusters []Cluster) (widget.Color, bool) {
	if len(clusters) == 0 {
		return widget.ColorTransparent, false
	}
	best := clusters[0]
	for _, c := range clusters[1:] {
		if c.Score > best.Score {
			best = c
		}
	}
	return best.Color, true
}

// ExtractColors runs K-Means (K=k) over a 64x64 bilinear downsample of img
// and returns up to k clusters ranked by Score (descending). A non-positive k
// falls back to DefaultExtractColors. Clusters whose centroid luminance is
// near-black (L < 0.15) or near-white (L > 0.85) are excluded, per the plan
// spec; the returned slice may therefore be shorter than k, or empty if every
// cluster was filtered.
func ExtractColors(img image.Image, k int) ([]Cluster, error) {
	if img == nil {
		return nil, ErrNilImage
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil, ErrEmptyImage
	}
	if k <= 0 {
		k = DefaultExtractColors
	}

	pixels, err := downsample(img, ExtractTargetSize)
	if err != nil {
		return nil, fmt.Errorf("theme: downsample cover art: %w", err)
	}
	if len(pixels) == 0 {
		return nil, ErrEmptyImage
	}

	centroids, counts := kMeans(pixels, k)

	clusters := make([]Cluster, 0, len(centroids))
	for i, c := range centroids {
		count := counts[i]
		if count == 0 {
			continue
		}
		lum := luminance(int(c.r), int(c.g), int(c.b))
		if lum < 0.15 || lum > 0.85 {
			continue
		}
		pop := float64(count) / float64(len(pixels))
		sat := saturation(int(c.r), int(c.g), int(c.b))
		score := sat*0.6 + pop*0.4
		clusters = append(clusters, Cluster{
			Color:      widget.RGBA8(uint8(c.r), uint8(c.g), uint8(c.b), 0xff),
			RGBA:       [3]uint8{uint8(c.r), uint8(c.g), uint8(c.b)},
			Population: pop,
			Score:      score,
			Saturation: sat,
		})
	}

	sort.Slice(clusters, func(a, b int) bool { return clusters[a].Score > clusters[b].Score })
	return clusters, nil
}

// Extract is shorthand for ExtractColors with DefaultExtractColors.
func Extract(img image.Image) ([]Cluster, error) {
	return ExtractColors(img, DefaultExtractColors)
}

// ExtractDominant is shorthand for ExtractColors(img, 1).
func ExtractDominant(img image.Image) (Cluster, bool, error) {
	clusters, err := ExtractColors(img, 1)
	if err != nil {
		return Cluster{}, false, err
	}
	if len(clusters) == 0 {
		return Cluster{}, false, nil
	}
	return clusters[0], true, nil
}

// downsample reduces img to size x size RGBA pixels using bilinear filtering.
// It returns one entry per output pixel in row-major order.
func downsample(img image.Image, size int) ([]rgb, error) {
	bounds := img.Bounds()
	if size <= 0 {
		size = ExtractTargetSize
	}
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Src, nil)

	pixels := make([]rgb, 0, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := dst.RGBAAt(x, y)
			// Skip fully transparent pixels (alpha 0) — they carry no color.
			if c.A == 0 {
				continue
			}
			// Un-premultiply so stored RGBAAt values are true sRGB.
			if c.A != 0xff {
				c.R = uint8(min(255, (int(c.R)*255+int(c.A)/2)/int(c.A)))
				c.G = uint8(min(255, (int(c.G)*255+int(c.A)/2)/int(c.A)))
				c.B = uint8(min(255, (int(c.B)*255+int(c.A)/2)/int(c.A)))
			}
			pixels = append(pixels, rgb{r: int(c.R), g: int(c.G), b: int(c.B)})
		}
	}
	if len(pixels) == 0 {
		return nil, ErrEmptyImage
	}
	return pixels, nil
}

// rgb is a 0..255 sRGB triple used by kMeans.
type rgb struct {
	r, g, b int
}

// centroid is a floating-point cluster center.
type centroid struct {
	r, g, b float64
}

// kMeans runs Lloyd's algorithm with deterministic seeding: K centroids are
// initialised at the K pixels spaced evenly through the pixel list. It
// iterates until assignments stop changing or a fixed cap is hit. It returns
// the centroids and their population counts.
//
// Determinism matters for reproducibility and for stable test fixtures; no
// randomness is used.
func kMeans(pixels []rgb, k int) ([]centroid, []int) {
	if k > len(pixels) {
		k = len(pixels)
	}

	// Deterministic seed: evenly spaced pixels.
	centroids := make([]centroid, k)
	for i := 0; i < k; i++ {
		idx := i * len(pixels) / k
		centroids[i] = centroid{
			r: float64(pixels[idx].r),
			g: float64(pixels[idx].g),
			b: float64(pixels[idx].b),
		}
	}

	assign := make([]int, len(pixels))
	counts := make([]int, k)

	// Iterate at most 64 times; stop early when assignments stabilise.
	for iter := 0; iter < 64; iter++ {
		for i := range counts {
			counts[i] = 0
		}
		for i, p := range pixels {
			best := 0
			bestDist := math.MaxFloat64
			for c := 0; c < k; c++ {
				dx := float64(p.r) - centroids[c].r
				dy := float64(p.g) - centroids[c].g
				dz := float64(p.b) - centroids[c].b
				d := dx*dx + dy*dy + dz*dz
				if d < bestDist {
					bestDist, best = d, c
				}
			}
			assign[i] = best
			counts[best]++
		}

		// Recompute means; empty clusters keep their previous centroid.
		sums := make([]centroid, k)
		for i, p := range pixels {
			c := assign[i]
			sums[c].r += float64(p.r)
			sums[c].g += float64(p.g)
			sums[c].b += float64(p.b)
		}
		moved := 0.0
		for c := 0; c < k; c++ {
			if counts[c] == 0 {
				continue
			}
			next := centroid{
				r: sums[c].r / float64(counts[c]),
				g: sums[c].g / float64(counts[c]),
				b: sums[c].b / float64(counts[c]),
			}
			moved += math.Abs(next.r-centroids[c].r) + math.Abs(next.g-centroids[c].g) + math.Abs(next.b-centroids[c].b)
			centroids[c] = next
		}
		if moved < 1e-9 {
			break
		}
	}

	return centroids, counts
}

// luminance returns the relative luminance L = 0.2126R + 0.7152G + 0.0722B
// of an sRGB color, with components normalised to 0..1. This is the formula
// used by plan-2 §6.3 and §7.1, matching M3/WCAG contrast calculations.
func luminance(r, g, b int) float64 {
	return 0.2126*float64(r)/255 + 0.7152*float64(g)/255 + 0.0722*float64(b)/255
}

// saturation returns the HSV saturation of an sRGB color (0..1).
func saturation(r, g, b int) float64 {
	max := math.Max(float64(r), math.Max(float64(g), float64(b)))
	min := math.Min(float64(r), math.Min(float64(g), float64(b)))
	if max == 0 {
		return 0
	}
	return (max - min) / max
}

// Luminance is exported for TSK-202 (Material You tone derivation) which
// needs the same metric to harmonise tone and contrast.
func Luminance(rgb [3]uint8) float64 {
	return luminance(int(rgb[0]), int(rgb[1]), int(rgb[2]))
}

// Saturation is exported for TSK-202.
func Saturation(rgb [3]uint8) float64 {
	return saturation(int(rgb[0]), int(rgb[1]), int(rgb[2]))
}

var _ = color.RGBA64{}
