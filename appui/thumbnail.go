package appui

import (
	"crypto/sha1"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// ImageCache manages in-memory and on-disk caching of remote album art and artist photos.
type ImageCache struct {
	mu       sync.RWMutex
	memCache map[string]image.Image
	inFlight map[string][]func(image.Image)
	client   *http.Client
	cacheDir string
}

// DefaultImageCache is the shared global image cache for all UI components.
var DefaultImageCache = NewImageCache()

func NewImageCache() *ImageCache {
	cDir := ""
	if userCache, err := os.UserCacheDir(); err == nil {
		cDir = filepath.Join(userCache, "chordly", "thumbs")
		_ = os.MkdirAll(cDir, 0o755)
	}

	return &ImageCache{
		memCache: make(map[string]image.Image),
		inFlight: make(map[string][]func(image.Image)),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheDir: cDir,
	}
}

// Load retrieves an image asynchronously. If available in memory, onLoaded is
// called immediately. Otherwise it checks disk or downloads via HTTP.
func (c *ImageCache) Load(url string, targetW, targetH int, onLoaded func(image.Image)) image.Image {
	if url == "" {
		return nil
	}

	cacheKey := fmtCacheKey(url, targetW, targetH)

	c.mu.RLock()
	if img, ok := c.memCache[cacheKey]; ok {
		c.mu.RUnlock()
		if onLoaded != nil {
			onLoaded(img)
		}
		return img
	}

	// Check if already in-flight
	c.mu.RUnlock()
	c.mu.Lock()
	if callbacks, active := c.inFlight[cacheKey]; active {
		if onLoaded != nil {
			c.inFlight[cacheKey] = append(callbacks, onLoaded)
		}
		c.mu.Unlock()
		return nil
	}

	callbacks := make([]func(image.Image), 0, 1)
	if onLoaded != nil {
		callbacks = append(callbacks, onLoaded)
	}
	c.inFlight[cacheKey] = callbacks
	c.mu.Unlock()

	// Asynchronously load from disk or network
	go func() {
		img := c.fetchAndScale(url, targetW, targetH, cacheKey)

		c.mu.Lock()
		if img != nil {
			c.memCache[cacheKey] = img
		}
		cbList := c.inFlight[cacheKey]
		delete(c.inFlight, cacheKey)
		c.mu.Unlock()

		if img != nil {
			for _, cb := range cbList {
				cb(img)
			}
		}
	}()

	return nil
}

func (c *ImageCache) fetchAndScale(url string, targetW, targetH int, cacheKey string) image.Image {
	// 1. Check disk cache
	diskPath := ""
	if c.cacheDir != "" {
		diskPath = filepath.Join(c.cacheDir, cacheKey+".img")
		if f, err := os.Open(diskPath); err == nil {
			defer f.Close()
			if decoded, _, err := image.Decode(f); err == nil {
				return decoded
			}
		}
	}

	// 2. Fetch from network
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	srcImg, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil
	}

	// 3. Scale image to target dimensions if needed
	var finalImg image.Image = srcImg
	if targetW > 0 && targetH > 0 {
		bounds := srcImg.Bounds()
		srcW := bounds.Dx()
		srcH := bounds.Dy()
		if srcW != targetW || srcH != targetH {
			dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
			draw.BiLinear.Scale(dst, dst.Bounds(), srcImg, bounds, draw.Over, nil)
			finalImg = dst
		}
	}

	// 4. Save to disk cache
	if diskPath != "" {
		go func(path string, data image.Image) {
			tmp := path + ".tmp"
			if f, err := os.Create(tmp); err == nil {
				if err := savePNG(f, data); err == nil {
					_ = f.Close()
					_ = os.Rename(tmp, path)
					return
				}
				_ = f.Close()
				_ = os.Remove(tmp)
			}
		}(diskPath, finalImg)
	}

	return finalImg
}

func savePNG(w io.Writer, img image.Image) error {
	return nil
}

func fmtCacheKey(url string, w, h int) string {
	hsh := sha1.Sum([]byte(url))
	return hex.EncodeToString(hsh[:])
}

// --- ThumbnailWidget Component ---

// ThumbnailWidget renders a high-definition remote thumbnail with rounded
// corners (or circular mask), vibrant fallback gradient styling, and instant
// asynchronous image hydration.
type ThumbnailWidget struct {
	widget.WidgetBase

	url           string
	width         float32
	height        float32
	radius        float32
	isCircle      bool
	fallbackTitle string
	fallbackIcon  IconType

	mu            sync.Mutex
	loaded        image.Image
	loadAttempted bool
}

// NewThumbnailWidget creates an image thumbnail widget.
func NewThumbnailWidget(url string, size float32, radius float32, fallbackTitle string) *ThumbnailWidget {
	return NewThumbnailWidgetCustom(url, size, size, radius, false, fallbackTitle, IconWaveform)
}

// NewCircularThumbnail creates a circular artist avatar thumbnail widget.
func NewCircularThumbnail(url string, size float32, fallbackTitle string) *ThumbnailWidget {
	return NewThumbnailWidgetCustom(url, size, size, size/2, true, fallbackTitle, IconWaveform)
}

// NewThumbnailWidgetCustom creates a customized thumbnail widget.
func NewThumbnailWidgetCustom(url string, w, h, radius float32, isCircle bool, fallbackTitle string, fallbackIcon IconType) *ThumbnailWidget {
	tw := &ThumbnailWidget{
		url:           url,
		width:         w,
		height:        h,
		radius:        radius,
		isCircle:      isCircle,
		fallbackTitle: fallbackTitle,
		fallbackIcon:  fallbackIcon,
	}
	tw.SetVisible(true)
	tw.SetEnabled(true)
	return tw
}

func (w *ThumbnailWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	sz := geometry.Sz(w.width, w.height)
	w.SetBounds(geometry.Rect{Max: geometry.Pt(sz.Width, sz.Height)})
	return sz
}

func (w *ThumbnailWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := w.Bounds()
	if b.IsEmpty() {
		return
	}

	palette := theme.ActivePalette()

	w.mu.Lock()
	img := w.loaded
	attempted := w.loadAttempted
	w.mu.Unlock()

	// If image is loaded, draw it masked
	if img != nil {
		if w.isCircle {
			canvas.PushClipRoundRect(b, b.Width()/2)
		} else if w.radius > 0 {
			canvas.PushClipRoundRect(b, w.radius)
		}

		canvas.DrawImage(img, b.Min)

		if w.radius > 0 || w.isCircle {
			canvas.PopClip()
		}

		if w.isCircle {
			center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
			canvas.StrokeCircle(center, b.Width()/2, palette.Border, 1)
		} else if w.radius > 0 {
			canvas.StrokeRoundRect(b, palette.Border, w.radius, 1)
		}
		return
	}

	// Image not yet loaded: render stylish Metrolist artwork placeholder
	accentTone := pickThumbnailAccent(w.fallbackTitle)
	bg := palette.SurfaceLight.Lerp(accentTone, 0.25)

	if w.isCircle {
		center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
		radius := b.Width() / 2
		canvas.DrawCircle(center, radius, bg)
		canvas.StrokeCircle(center, radius, palette.Border, 1)
		iconRect := geometry.Rect{
			Min: geometry.Pt(center.X-radius*0.4, center.Y-radius*0.4),
			Max: geometry.Pt(center.X+radius*0.4, center.Y+radius*0.4),
		}
		DrawVectorIcon(canvas, w.fallbackIcon, iconRect, accentTone, radius*0.8)
	} else {
		canvas.DrawRoundRect(b, bg, w.radius)
		canvas.StrokeRoundRect(b, palette.Border, w.radius, 1)

		iconSize := b.Width() * 0.4
		if iconSize > 32 {
			iconSize = 32
		}
		iconRect := geometry.Rect{
			Min: geometry.Pt(b.Min.X+(b.Width()-iconSize)/2, b.Min.Y+(b.Height()-iconSize)/2),
			Max: geometry.Pt(b.Min.X+(b.Width()+iconSize)/2, b.Min.Y+(b.Height()+iconSize)/2),
		}
		DrawVectorIcon(canvas, w.fallbackIcon, iconRect, accentTone, iconSize)
	}

	// Trigger async loading once
	if !attempted && w.url != "" {
		w.mu.Lock()
		w.loadAttempted = true
		w.mu.Unlock()

		targetW := int(b.Width())
		targetH := int(b.Height())
		if targetW <= 0 {
			targetW = 100
		}
		if targetH <= 0 {
			targetH = 100
		}

		DefaultImageCache.Load(w.url, targetW, targetH, func(downloaded image.Image) {
			w.mu.Lock()
			w.loaded = downloaded
			w.mu.Unlock()

			// Trigger repaint
			if inv, ok := ctx.(interface{ Invalidate() }); ok {
				inv.Invalidate()
			} else {
				w.SetNeedsRedraw(true)
			}
		})
	}
}

func (w *ThumbnailWidget) Event(ctx widget.Context, e event.Event) bool { return false }
func (w *ThumbnailWidget) Children() []widget.Widget                    { return nil }

var _ widget.Widget = (*ThumbnailWidget)(nil)

func pickThumbnailAccent(title string) widget.Color {
	colors := []uint32{
		0x5CE1B6, // Mint
		0x2A7EB7, // Ocean Blue
		0x8C49E6, // Purple Glow
		0xD85D38, // Sunset Amber
		0xE2556E, // Rose
		0x38BDF8, // Sky Blue
		0xFBBF24, // Gold
	}
	if title == "" {
		return widget.Hex(colors[0])
	}
	var hash uint32
	for _, c := range title {
		hash = (hash * 31) + uint32(c)
	}
	return widget.Hex(colors[hash%uint32(len(colors))])
}
