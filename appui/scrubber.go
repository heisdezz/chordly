package appui

import (
	"fmt"
	"sync"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
)

// ScrubberWidget renders the playback progress bar with elapsed & duration labels.
type ScrubberWidget struct {
	widget.WidgetBase

	isDragging bool

	mu     sync.Mutex
	unsubs []func()
}

func NewScrubberWidget() *ScrubberWidget {
	s := &ScrubberWidget{}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

// Mount repaints the scrubber whenever the playhead advances.
func (s *ScrubberWidget) Mount(ctx widget.Context) {
	unsub := state.Playhead.SubscribeForever(func(time.Duration) {
		s.SetNeedsRedraw(true)
		ctx.Invalidate()
	})

	s.mu.Lock()
	s.unsubs = append(s.unsubs, unsub)
	s.mu.Unlock()
}

// Unmount releases the playhead subscription.
func (s *ScrubberWidget) Unmount() {
	s.mu.Lock()
	unsubs := s.unsubs
	s.unsubs = nil
	s.mu.Unlock()

	for _, unsub := range unsubs {
		if unsub != nil {
			unsub()
		}
	}
}

func (s *ScrubberWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	width := constraints.MaxWidth
	if width > 460 {
		width = 460
	}
	if width < 260 {
		width = 260
	}
	size := geometry.Sz(width, 24)
	s.SetBounds(geometry.FromPointSize(s.Bounds().Min, size))
	return size
}

func (s *ScrubberWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := s.Bounds()
	if bounds.IsEmpty() {
		return
	}

	playhead := state.Playhead.Get()
	track := state.CurrentTrack.Get()
	total := track.Duration
	if total <= 0 {
		total = 237 * time.Second
	}

	elapsedSec := int(playhead.Seconds())
	totalSec := int(total.Seconds())
	elapsedText := fmt.Sprintf("%d:%02d", elapsedSec/60, elapsedSec%60)
	totalText := fmt.Sprintf("%d:%02d", totalSec/60, totalSec%60)

	// Draw elapsed text
	elapsedRect := geometry.Rect{
		Min: bounds.Min,
		Max: geometry.Pt(bounds.Min.X+40, bounds.Max.Y),
	}
	canvas.DrawText(elapsedText, elapsedRect, 11, theme.ColorTextMuted, false, widget.TextAlignRight)

	// Draw total text
	totalRect := geometry.Rect{
		Min: geometry.Pt(bounds.Max.X-40, bounds.Min.Y),
		Max: bounds.Max,
	}
	canvas.DrawText(totalText, totalRect, 11, theme.ColorTextMuted, false, widget.TextAlignLeft)

	// Track bar dimensions
	trackStartX := bounds.Min.X + 48
	trackEndX := bounds.Max.X - 48
	trackWidth := trackEndX - trackStartX
	trackY := bounds.Min.Y + bounds.Height()/2 - 2

	// Background track (unfilled)
	bgTrackRect := geometry.Rect{
		Min: geometry.Pt(trackStartX, trackY),
		Max: geometry.Pt(trackEndX, trackY+4),
	}
	canvas.DrawRoundRect(bgTrackRect, theme.ColorScrubberTrack, 2)

	// Progress ratio
	ratio := float32(playhead.Seconds()) / float32(total.Seconds())
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	activeEndX := trackStartX + trackWidth*ratio
	if activeEndX > trackStartX {
		fillTrackRect := geometry.Rect{
			Min: geometry.Pt(trackStartX, trackY),
			Max: geometry.Pt(activeEndX, trackY+4),
		}
		canvas.DrawRoundRect(fillTrackRect, theme.ColorAccent, 2)
	}

	// Scrubber knob circle
	knobCenter := geometry.Pt(activeEndX, trackY+2)
	canvas.DrawCircle(knobCenter, 5, theme.ColorAccent)
}

func (s *ScrubberWidget) Event(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}

	bounds := s.Bounds()
	trackStartX := bounds.Min.X + 48
	trackEndX := bounds.Max.X - 48
	trackWidth := trackEndX - trackStartX

	switch {
	case me.IsPress() && bounds.Contains(me.GlobalPosition):
		s.isDragging = true
		s.seekFromPosition(me.GlobalPosition.X, trackStartX, trackWidth)
		return true

	case me.IsDrag() && s.isDragging:
		s.seekFromPosition(me.GlobalPosition.X, trackStartX, trackWidth)
		return true

	case me.IsRelease():
		if s.isDragging {
			s.isDragging = false
			return true
		}
	}

	return false
}

func (s *ScrubberWidget) seekFromPosition(mouseX, trackStartX, trackWidth float32) {
	ratio := (mouseX - trackStartX) / trackWidth
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	track := state.CurrentTrack.Get()
	total := track.Duration
	if total <= 0 {
		total = 237 * time.Second
	}

	target := time.Duration(float64(total) * float64(ratio))
	state.Playhead.Set(target)
}

func (s *ScrubberWidget) Children() []widget.Widget {
	return nil
}

// VolumeSliderWidget renders an interactive volume slider with a speaker icon.
type VolumeSliderWidget struct {
	widget.WidgetBase
	isDragging bool

	mu     sync.Mutex
	unsubs []func()
}

func NewVolumeSliderWidget() *VolumeSliderWidget {
	v := &VolumeSliderWidget{}
	v.SetVisible(true)
	v.SetEnabled(true)
	return v
}

// Mount repaints the slider when the volume signal changes.
func (v *VolumeSliderWidget) Mount(ctx widget.Context) {
	unsub := state.Volume.SubscribeForever(func(float32) {
		v.SetNeedsRedraw(true)
		ctx.Invalidate()
	})

	v.mu.Lock()
	v.unsubs = append(v.unsubs, unsub)
	v.mu.Unlock()
}

// Unmount releases the volume subscription.
func (v *VolumeSliderWidget) Unmount() {
	v.mu.Lock()
	unsubs := v.unsubs
	v.unsubs = nil
	v.mu.Unlock()

	for _, unsub := range unsubs {
		if unsub != nil {
			unsub()
		}
	}
}

func (v *VolumeSliderWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := geometry.Sz(110, 24)
	v.SetBounds(geometry.FromPointSize(v.Bounds().Min, size))
	return size
}

func (v *VolumeSliderWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := v.Bounds()
	if bounds.IsEmpty() {
		return
	}

	// Vector Speaker icon
	iconRect := geometry.Rect{
		Min: bounds.Min,
		Max: geometry.Pt(bounds.Min.X+22, bounds.Max.Y),
	}
	DrawVectorIcon(canvas, IconVolume, iconRect, theme.ColorTextMuted, 16)

	// Slider track
	trackStartX := bounds.Min.X + 26
	trackEndX := bounds.Max.X - 8
	trackWidth := trackEndX - trackStartX
	trackY := bounds.Min.Y + bounds.Height()/2 - 2

	bgTrackRect := geometry.Rect{
		Min: geometry.Pt(trackStartX, trackY),
		Max: geometry.Pt(trackEndX, trackY+4),
	}
	canvas.DrawRoundRect(bgTrackRect, theme.ColorScrubberTrack, 2)

	vol := state.Volume.Get()
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}

	activeEndX := trackStartX + trackWidth*vol
	if activeEndX > trackStartX {
		fillTrackRect := geometry.Rect{
			Min: geometry.Pt(trackStartX, trackY),
			Max: geometry.Pt(activeEndX, trackY+4),
		}
		canvas.DrawRoundRect(fillTrackRect, theme.ColorAccent, 2)
	}

	// Knob
	knobCenter := geometry.Pt(activeEndX, trackY+2)
	canvas.DrawCircle(knobCenter, 4.5, theme.ColorAccent)
}

func (v *VolumeSliderWidget) Event(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}

	bounds := v.Bounds()
	trackStartX := bounds.Min.X + 26
	trackEndX := bounds.Max.X - 8
	trackWidth := trackEndX - trackStartX

	switch {
	case me.IsPress() && bounds.Contains(me.GlobalPosition):
		v.isDragging = true
		v.setVolumeFromPos(me.GlobalPosition.X, trackStartX, trackWidth)
		return true

	case me.IsDrag() && v.isDragging:
		v.setVolumeFromPos(me.GlobalPosition.X, trackStartX, trackWidth)
		return true

	case me.IsRelease():
		if v.isDragging {
			v.isDragging = false
			return true
		}
	}

	return false
}

func (v *VolumeSliderWidget) setVolumeFromPos(mouseX, startX, width float32) {
	ratio := (mouseX - startX) / width
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	state.Volume.Set(ratio)
}

func (v *VolumeSliderWidget) Children() []widget.Widget {
	return nil
}
