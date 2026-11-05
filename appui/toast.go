package appui

import (
	"fmt"
	"sync"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// ToastKind identifies the visual tone of a toast.
type ToastKind uint8

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// Toast is one transient message shown by ToastHost.
type Toast struct {
	ID      uint64
	Kind    ToastKind
	Message string

	Duration  time.Duration
	shownAt   time.Time
	dismissed time.Time
}

const (
	toastMaxVisible = 4
	toastWidth      = float32(320)
	toastHeight     = float32(46)
	toastGap        = float32(8)
	toastMargin     = float32(24)
)

var (
	// SigToasts holds the active toast stack. Widgets subscribe through ToastHost.
	SigToasts = gpstate.NewSignal([]Toast(nil))

	toastEnterDuration = 200 * time.Millisecond
	toastExitDuration  = 160 * time.Millisecond

	toastMu     sync.Mutex
	toastNextID uint64
)

// ShowInfo shows a neutral informational message.
func ShowInfo(message string) uint64 { return ShowToast(ToastInfo, message) }

// ShowSuccess shows a positive confirmation message.
func ShowSuccess(message string) uint64 { return ShowToast(ToastSuccess, message) }

// ShowWarning shows a non-fatal warning.
func ShowWarning(message string) uint64 { return ShowToast(ToastWarning, message) }

// ShowError shows an error message.
func ShowError(message string) uint64 { return ShowToast(ToastError, message) }

// ShowErrorf formats and shows an error message.
func ShowErrorf(format string, args ...any) uint64 {
	return ShowError(fmt.Sprintf(format, args...))
}

// ShowToast appends a toast, keeping only the newest few visible.
func ShowToast(kind ToastKind, message string) uint64 {
	duration := toastDuration(kind)

	toastMu.Lock()
	toastNextID++
	t := Toast{
		ID:       toastNextID,
		Kind:     kind,
		Message:  message,
		Duration: duration,
		shownAt:  time.Now(),
	}

	current := SigToasts.Get()
	next := make([]Toast, 0, len(current)+1)
	next = append(next, current...)
	next = append(next, t)
	if len(next) > toastMaxVisible {
		next = next[len(next)-toastMaxVisible:]
	}
	SigToasts.Set(next)
	toastMu.Unlock()

	if duration > 0 {
		time.AfterFunc(duration, func() { DismissToast(t.ID) })
	}
	return t.ID
}

// DismissToast starts the exit animation and removes the toast shortly after.
func DismissToast(id uint64) {
	toastMu.Lock()
	current := SigToasts.Get()
	next := make([]Toast, len(current))
	copy(next, current)

	changed := false
	for i := range next {
		if next[i].ID == id && next[i].dismissed.IsZero() {
			next[i].dismissed = time.Now()
			changed = true
			break
		}
	}
	if changed {
		SigToasts.Set(next)
	}
	toastMu.Unlock()

	if changed {
		time.AfterFunc(toastExitDuration, func() { removeToast(id) })
	}
}

func removeToast(id uint64) {
	toastMu.Lock()
	defer toastMu.Unlock()

	current := SigToasts.Get()
	next := make([]Toast, 0, len(current))
	for _, t := range current {
		if t.ID != id {
			next = append(next, t)
		}
	}
	if len(next) != len(current) {
		SigToasts.Set(next)
	}
}

func clearToasts() {
	toastMu.Lock()
	SigToasts.Set(nil)
	toastMu.Unlock()
}

func toastDuration(kind ToastKind) time.Duration {
	switch kind {
	case ToastError:
		return 6 * time.Second
	case ToastWarning:
		return 5 * time.Second
	default:
		return 3 * time.Second
	}
}

// ToastHost wraps a widget tree and paints the toast stack above it. It avoids
// the overlay package so it works with the currently vendored GoGPU tree.
type ToastHost struct {
	widget.WidgetBase

	child widget.Widget

	mu    sync.Mutex
	hits  []toastHit
	unsub func()
}

type toastHit struct {
	id   uint64
	rect geometry.Rect
}

// NewToastHost wraps child with a toast layer.
func NewToastHost(child widget.Widget) *ToastHost {
	h := &ToastHost{child: child}
	h.SetVisible(true)
	h.SetEnabled(true)
	h.adopt(child)
	return h
}

func (h *ToastHost) adopt(child widget.Widget) {
	if child == nil {
		return
	}
	if setter, ok := child.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(h)
	}
}

// Mount subscribes to toast changes so the host repaints.
func (h *ToastHost) Mount(ctx widget.Context) {
	h.mu.Lock()
	h.unsub = SigToasts.SubscribeForever(func([]Toast) {
		h.SetNeedsRedraw(true)
		ctx.Invalidate()
	})
	h.mu.Unlock()
}

// Unmount releases the toast subscription.
func (h *ToastHost) Unmount() {
	h.mu.Lock()
	unsub := h.unsub
	h.unsub = nil
	h.mu.Unlock()

	if unsub != nil {
		unsub()
	}
}

func (h *ToastHost) Children() []widget.Widget {
	if h.child == nil {
		return nil
	}
	return []widget.Widget{h.child}
}

func (h *ToastHost) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	if h.child == nil {
		return geometry.Size{}
	}
	size := widget.LayoutChild(h.child, ctx, constraints)
	if setter, ok := h.child.(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
	}
	h.SetBounds(geometry.FromPointSize(h.Position(), size))
	return size
}

func (h *ToastHost) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !h.IsVisible() {
		return
	}

	bounds := h.Bounds()
	if h.child != nil {
		canvas.PushTransform(bounds.Min)
		widget.StampScreenOrigin(h.child, canvas)
		h.child.Draw(ctx, canvas)
		canvas.PopTransform()
	}

	toasts := SigToasts.Get()
	if len(toasts) == 0 {
		h.mu.Lock()
		h.hits = nil
		h.mu.Unlock()
		return
	}

	h.drawToasts(ctx, canvas, bounds, toasts)
}

func (h *ToastHost) drawToasts(ctx widget.Context, canvas widget.Canvas, bounds geometry.Rect, toasts []Toast) {
	now := ctx.Now()
	animating := false

	width := toastWidth
	if bounds.Width() < width+2*toastMargin {
		width = bounds.Width() - 2*toastMargin
	}
	if width < 160 {
		return
	}

	x := bounds.Max.X - width - toastMargin
	if x < bounds.Min.X+toastMargin {
		x = bounds.Min.X + toastMargin
	}
	y := bounds.Max.Y - 100 // sit above the 78px mini-player

	hits := make([]toastHit, 0, len(toasts))

	// Newest at the bottom, older messages stack upward.
	for i := len(toasts) - 1; i >= 0; i-- {
		t := toasts[i]
		in, out := toastProgress(t, now)
		if out >= 1 {
			continue
		}

		opacity := in * (1 - out)
		if opacity <= 0.01 && !t.dismissed.IsZero() {
			continue
		}
		if in < 1 || out > 0 {
			animating = true
		}

		offset := (1-in)*10 - out*8
		rect := geometry.Rect{
			Min: geometry.Pt(x, y-toastHeight+offset),
			Max: geometry.Pt(x+width, y+offset),
		}
		y -= toastHeight + toastGap

		hits = append(hits, toastHit{id: t.ID, rect: rect})

		pushToastPaint(canvas, opacity, func() {
			drawToast(canvas, rect, t)
		})
	}

	h.mu.Lock()
	h.hits = hits
	h.mu.Unlock()

	if animating {
		if sched, ok := ctx.(widget.AnimationScheduler); ok {
			sched.ScheduleAnimationFrame()
		} else {
			ctx.Invalidate()
		}
		h.SetNeedsRedraw(true)
	}
}

func (h *ToastHost) Event(ctx widget.Context, e event.Event) bool {
	if !h.IsVisible() || !h.IsEnabled() {
		return false
	}

	if me, ok := e.(*event.MouseEvent); ok && me.IsPress() {
		if id, found := h.toastAt(me.GlobalPosition); found {
			DismissToast(id)
			return true
		}
	}

	if h.child == nil {
		return false
	}

	switch ev := e.(type) {
	case *event.MouseEvent:
		local := *ev
		local.Position = ev.Position.Sub(h.Bounds().Min)
		return h.child.Event(ctx, &local)
	case *event.WheelEvent:
		local := *ev
		local.Position = ev.Position.Sub(h.Bounds().Min)
		return h.child.Event(ctx, &local)
	default:
		return h.child.Event(ctx, e)
	}
}

func (h *ToastHost) toastAt(pos geometry.Point) (uint64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, hit := range h.hits {
		if hit.rect.Contains(pos) {
			return hit.id, true
		}
	}
	return 0, false
}

func toastProgress(t Toast, now time.Time) (in, out float32) {
	in = 1
	if !t.shownAt.IsZero() && toastEnterDuration > 0 {
		in = clampToast(float32(float64(now.Sub(t.shownAt)) / float64(toastEnterDuration)))
	}
	in = easeOutToast(in)

	if !t.dismissed.IsZero() && toastExitDuration > 0 {
		out = clampToast(float32(float64(now.Sub(t.dismissed)) / float64(toastExitDuration)))
	}
	out = easeOutToast(out)
	return in, out
}

func clampToast(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func easeOutToast(t float32) float32 {
	inv := 1 - t
	return 1 - inv*inv*inv
}

type toastOpacityCanvas interface {
	PushOpacity(opacity float64)
	PopOpacity()
}

func pushToastPaint(canvas widget.Canvas, opacity float32, draw func()) {
	op, ok := canvas.(toastOpacityCanvas)
	if !ok || opacity >= 1 {
		draw()
		return
	}
	op.PushOpacity(float64(opacity))
	draw()
	op.PopOpacity()
}

func drawToast(canvas widget.Canvas, rect geometry.Rect, t Toast) {
	accent := toastAccent(t.Kind)

	canvas.PushClip(rect)
	defer canvas.PopClip()

	canvas.DrawRoundRect(rect, theme.ColorSurface, 12)
	canvas.StrokeRoundRect(rect, theme.ColorBorder, 12, 1)

	bar := geometry.Rect{
		Min: geometry.Pt(rect.Min.X+1, rect.Min.Y+10),
		Max: geometry.Pt(rect.Min.X+4, rect.Max.Y-10),
	}
	canvas.DrawRoundRect(bar, accent, 1.5)

	icon := IconInfo
	switch t.Kind {
	case ToastSuccess:
		icon = IconCheck
	case ToastWarning, ToastError:
		icon = IconAlert
	}

	iconRect := geometry.Rect{
		Min: geometry.Pt(rect.Min.X+14, rect.Min.Y),
		Max: geometry.Pt(rect.Min.X+34, rect.Max.Y),
	}
	DrawVectorIcon(canvas, icon, iconRect, accent, 16)

	textRect := geometry.Rect{
		Min: geometry.Pt(rect.Min.X+42, rect.Min.Y),
		Max: geometry.Pt(rect.Max.X-14, rect.Max.Y),
	}
	message := fitText(canvas, t.Message, 12, false, textRect.Width())
	canvas.DrawText(message, textRect, 12, theme.ColorTextPrimary, false, widget.TextAlignLeft)
}

func toastAccent(kind ToastKind) widget.Color {
	switch kind {
	case ToastSuccess:
		return theme.ColorAccent
	case ToastWarning:
		return widget.Hex(0xEAB308)
	case ToastError:
		return theme.ColorHeartActive
	default:
		return theme.ColorAccent
	}
}

var _ widget.Widget = (*ToastHost)(nil)
var _ widget.Lifecycle = (*ToastHost)(nil)
