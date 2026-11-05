package appui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/lyrics"
	"chordly/internal/model"
	appstate "chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

var (
	// IsLyricsOpen controls whether the lyrics overlay view is visible.
	IsLyricsOpen = state.NewSignal(false)

	// ActiveLyricsDoc holds the currently loaded lyrics document.
	ActiveLyricsDoc = state.NewSignal[*lyrics.LyricsDocument](nil)

	// LyricsLoading tracks whether lyrics are being fetched.
	LyricsLoading = state.NewSignal(false)
)

// LyricsFetcher is the client used by the UI to fetch lyrics from LRCLIB.
var (
	lyricsClientOnce sync.Once
	lyricsClient     *lyrics.Client

	lyricsFetchMu  sync.Mutex
	lyricsFetchID  uint64
	lyricsFetchKey string
)

func getLyricsClient() *lyrics.Client {
	lyricsClientOnce.Do(func() {
		lyricsClient = lyrics.NewClient()
	})
	return lyricsClient
}

// FetchLyricsForCurrentTrack triggers background retrieval of lyrics for the currently playing track.
// Duplicate requests for the same title/artist are ignored.
func FetchLyricsForCurrentTrack() {
	track := appstate.CurrentTrack.Get()
	if strings.TrimSpace(track.Title) == "" {
		lyricsFetchMu.Lock()
		lyricsFetchID++
		lyricsFetchKey = ""
		lyricsFetchMu.Unlock()
		ActiveLyricsDoc.Set(nil)
		return
	}

	key := track.Title + "\x00" + track.Artist

	lyricsFetchMu.Lock()
	if lyricsFetchKey == key {
		lyricsFetchMu.Unlock()
		return
	}
	lyricsFetchKey = key
	lyricsFetchID++
	id := lyricsFetchID
	lyricsFetchMu.Unlock()

	LyricsLoading.Set(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		durationSec := int(track.Duration.Seconds())
		doc, err := getLyricsClient().FetchLyrics(ctx, track.Title, track.Artist, "", durationSec)

		lyricsFetchMu.Lock()
		if lyricsFetchID != id {
			lyricsFetchMu.Unlock()
			return
		}
		if err != nil {
			// Allow a retry on the next track notification.
			lyricsFetchKey = ""
		}
		lyricsFetchMu.Unlock()

		if err != nil {
			// No lyrics found or network issue: set document with fallback title
			ActiveLyricsDoc.Set(&lyrics.LyricsDocument{
				Title:    track.Title,
				Artist:   track.Artist,
				IsSynced: false,
			})
		} else {
			ActiveLyricsDoc.Set(doc)
		}
		LyricsLoading.Set(false)
	}()
}

// LyricsViewWidget is a high-performance 2D canvas widget that renders
// real-time synchronized karaoke lyrics with auto-centering and seek-on-click.
type LyricsViewWidget struct {
	widget.WidgetBase

	scrollY       float32
	targetScrollY float32
	lastActiveIdx int
	userScrolling bool
	lastWheelTime time.Time

	lineHeight float32

	mu     sync.Mutex
	unsubs []func()
}

// NewLyricsViewWidget constructs a new LyricsViewWidget.
func NewLyricsViewWidget() *LyricsViewWidget {
	w := &LyricsViewWidget{
		lineHeight:    52.0,
		lastActiveIdx: -1,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

// Mount subscribes to playback and lyrics state. Repaints are paint-only: the
// widget is never rebuilt, so karaoke auto-scroll stays smooth at the player's
// playhead update rate.
func (w *LyricsViewWidget) Mount(ctx widget.Context) {
	repaint := func() {
		w.SetNeedsRedraw(true)
		ctx.Invalidate()
	}

	unsubs := []func(){
		appstate.Playhead.SubscribeForever(func(time.Duration) { repaint() }),
		ActiveLyricsDoc.SubscribeForever(func(*lyrics.LyricsDocument) { repaint() }),
		LyricsLoading.SubscribeForever(func(bool) { repaint() }),
		appstate.CurrentTrack.SubscribeForever(func(model.Track) {
			repaint()
			FetchLyricsForCurrentTrack()
		}),
	}

	w.mu.Lock()
	w.unsubs = append(w.unsubs, unsubs...)
	w.mu.Unlock()

	FetchLyricsForCurrentTrack()
}

// Unmount releases the signal subscriptions.
func (w *LyricsViewWidget) Unmount() {
	w.mu.Lock()
	unsubs := w.unsubs
	w.unsubs = nil
	w.mu.Unlock()

	for _, unsub := range unsubs {
		if unsub != nil {
			unsub()
		}
	}
}

func (w *LyricsViewWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	width := constraints.MaxWidth
	height := constraints.MaxHeight
	if height <= 0 {
		height = 500
	}
	size := geometry.Sz(width, height)
	w.SetBounds(geometry.FromPointSize(w.Bounds().Min, size))
	return size
}

func (w *LyricsViewWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	if bounds.IsEmpty() {
		return
	}

	doc := ActiveLyricsDoc.Get()
	track := appstate.CurrentTrack.Get()
	isLoading := LyricsLoading.Get()

	// 1. Semi-transparent backdrop blur card
	canvas.DrawRoundRect(bounds, theme.ColorSurfaceLight, 16)

	// 2. Loading state
	if isLoading {
		loadingText := fmt.Sprintf("Finding synchronized lyrics for %s...", track.Title)
		canvas.DrawText(loadingText, bounds, 15, theme.ColorTextSecondary, false, widget.TextAlignCenter)
		return
	}

	// 3. Empty / No lyrics state
	if doc == nil || len(doc.Lines) == 0 {
		emptyMsg := "No lyrics available for this song"
		if track.Title != "" {
			emptyMsg = fmt.Sprintf("No synchronized lyrics found for \"%s\"", track.Title)
		}
		canvas.DrawText(emptyMsg, bounds, 15, theme.ColorTextMuted, false, widget.TextAlignCenter)
		return
	}

	playhead := appstate.Playhead.Get()
	activeIdx := -1
	if doc.IsSynced {
		activeIdx = doc.ActiveLineIndex(playhead)
	}

	// Auto-scroll target calculation: keep active line centered at ~40% of viewport
	viewportHeight := bounds.Height()
	targetCenterY := viewportHeight * 0.40

	if activeIdx >= 0 {
		idealOffset := float32(activeIdx)*w.lineHeight - targetCenterY
		if idealOffset < 0 {
			idealOffset = 0
		}
		maxScroll := float32(len(doc.Lines))*w.lineHeight - viewportHeight/2
		if maxScroll < 0 {
			maxScroll = 0
		}
		if idealOffset > maxScroll {
			idealOffset = maxScroll
		}

		// Smoothly interpolate scroll unless user actively scrolled recently
		if time.Since(w.lastWheelTime) > 3*time.Second {
			w.targetScrollY = idealOffset
			// LERP step
			w.scrollY += (w.targetScrollY - w.scrollY) * 0.18
		}
		w.lastActiveIdx = activeIdx
	}

	// 4. Render lines
	startY := bounds.Min.Y + 20 - w.scrollY
	for i, line := range doc.Lines {
		lineY := startY + float32(i)*w.lineHeight

		// Clip off-screen lines
		if lineY < bounds.Min.Y-w.lineHeight || lineY > bounds.Max.Y+w.lineHeight {
			continue
		}

		lineRect := geometry.Rect{
			Min: geometry.Pt(bounds.Min.X+32, lineY),
			Max: geometry.Pt(bounds.Max.X-32, lineY+w.lineHeight),
		}

		if i == activeIdx {
			// Active singing line: Mint glow accent, 20pt bold
			// Indicator bar on left
			indicatorRect := geometry.Rect{
				Min: geometry.Pt(bounds.Min.X+16, lineY+14),
				Max: geometry.Pt(bounds.Min.X+20, lineY+w.lineHeight-14),
			}
			canvas.DrawRoundRect(indicatorRect, theme.ColorAccent, 2)

			canvas.DrawText(line.Text, lineRect, 20, theme.ColorAccent, true, widget.TextAlignLeft)
		} else if activeIdx >= 0 && i < activeIdx {
			// Past lines: subdued slate
			canvas.DrawText(line.Text, lineRect, 16, widget.RGBA8(150, 175, 185, 170), false, widget.TextAlignLeft)
		} else {
			// Future lines: muted
			canvas.DrawText(line.Text, lineRect, 16, widget.RGBA8(110, 135, 145, 130), false, widget.TextAlignLeft)
		}
	}
}

func (w *LyricsViewWidget) Event(ctx widget.Context, e event.Event) bool {
	bounds := w.Bounds()

	// Mouse wheel scrolling
	if we, ok := e.(*event.WheelEvent); ok {
		if bounds.Contains(we.Position) {
			w.lastWheelTime = time.Now()
			w.scrollY += float32(we.Delta.Y) * 20
			if w.scrollY < 0 {
				w.scrollY = 0
			}
			return true
		}
	}

	// Click to seek on a lyric line
	if me, ok := e.(*event.MouseEvent); ok {
		if me.IsPress() && bounds.Contains(me.GlobalPosition) {
			doc := ActiveLyricsDoc.Get()
			if doc == nil || len(doc.Lines) == 0 || !doc.IsSynced {
				return false
			}

			// Calculate which line was clicked
			relY := (me.GlobalPosition.Y - bounds.Min.Y - 20) + w.scrollY
			if relY >= 0 {
				clickedIdx := int(math.Floor(float64(relY / w.lineHeight)))
				if clickedIdx >= 0 && clickedIdx < len(doc.Lines) {
					targetTS := doc.Lines[clickedIdx].Timestamp
					appstate.Playhead.Set(targetTS)
					SeekRequest.Set(targetTS)
					return true
				}
			}
		}
	}

	return false
}

func (w *LyricsViewWidget) Children() []widget.Widget {
	return nil
}

// NewLyricsOverlay builds the full overlay panel with header controls,
// track metadata, synced pill badge, and the lyrics canvas. Passing a router
// makes the close button return to the previous page.
func NewLyricsOverlay(routers ...*router.Router) widget.Widget {
	track := appstate.CurrentTrack.Get()

	// Header
	titleText := primitives.Text("LYRICS").FontSize(11).Bold().Color(theme.ColorAccent)
	trackText := primitives.Text(fmt.Sprintf("%s • %s", track.Title, track.Artist)).
		FontSize(13).
		Color(theme.ColorTextSecondary)

	// Close Button
	closeBtn := NewVectorIconButton(IconMore, 30, 14, func() {
		IsLyricsOpen.Set(false)
		if len(routers) > 0 && routers[0] != nil {
			routers[0].Back()
		}
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextMuted
		s.BorderWidth = 0
	})

	header := primitives.HBox(
		primitives.VBox(titleText, trackText).Gap(2),
		primitives.Expanded(primitives.Box()),
		closeBtn,
	).CrossAlign(primitives.CrossAxisCenter).PaddingXY(20, 14)

	lyricsWidget := NewLyricsViewWidget()

	return primitives.VBox(
		header,
		primitives.Expanded(lyricsWidget),
	).
		Background(theme.ColorSurfaceLight).
		Rounded(16).
		BorderStyle(1, theme.ColorBorder)
}
