package appui

import (
	"sync"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// reactiveView rebuilds its child whenever one of its watched signals changes.
//
// Signal callbacks only mark the view dirty and invalidate the window; the
// actual widget swap happens at the start of Layout or Draw. This mirrors
// routerView and prevents the widget tree from being mutated while an event
// handler (for example a button click) is still on the stack.
type reactiveView struct {
	widget.WidgetBase

	build    func() widget.Widget
	watchers []func(notify func()) func()

	mu     sync.Mutex
	ctx    widget.Context
	child  widget.Widget
	dirty  bool
	unsubs []func()
}

// ReactiveOption adds a signal subscription to a reactive view.
type ReactiveOption func(*reactiveView)

// WatchSignal rebuilds the view when sig notifies a change.
func WatchSignal[T any](sig gpstate.ReadonlySignal[T]) ReactiveOption {
	return func(v *reactiveView) {
		v.watchers = append(v.watchers, func(notify func()) func() {
			return sig.SubscribeForever(func(T) { notify() })
		})
	}
}

func newReactiveView(build func() widget.Widget, opts ...ReactiveOption) *reactiveView {
	v := &reactiveView{build: build}
	v.SetVisible(true)
	v.SetEnabled(true)
	for _, opt := range opts {
		opt(v)
	}
	v.child = v.build()
	v.adopt(v.child)
	return v
}

func (v *reactiveView) adopt(child widget.Widget) {
	if child == nil {
		return
	}
	if setter, ok := child.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(v)
	}
}

// Mount subscribes to every watched signal. The initial child is mounted by
// the framework's MountTree walk through Children().
func (v *reactiveView) Mount(ctx widget.Context) {
	v.mu.Lock()
	v.ctx = ctx
	watchers := append([]func(notify func()) func(){}, v.watchers...)
	v.mu.Unlock()

	notify := func() {
		v.mu.Lock()
		v.dirty = true
		v.mu.Unlock()

		v.MarkNeedsLayout()
		ctx.Invalidate()
		v.SetNeedsRedraw(true)
	}

	unsubs := make([]func(), 0, len(watchers))
	for _, subscribe := range watchers {
		unsubs = append(unsubs, subscribe(notify))
	}

	v.mu.Lock()
	v.unsubs = append(v.unsubs, unsubs...)
	v.mu.Unlock()
}

// Unmount releases all subscriptions. The framework unmounts the child first
// because it is returned from Children().
func (v *reactiveView) Unmount() {
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

func (v *reactiveView) Children() []widget.Widget {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.child == nil {
		return nil
	}
	return []widget.Widget{v.child}
}

// sync swaps in a freshly built child when a watched signal changed.
func (v *reactiveView) sync() {
	v.mu.Lock()
	if !v.dirty {
		v.mu.Unlock()
		return
	}
	v.dirty = false
	ctx := v.ctx
	old := v.child
	v.mu.Unlock()

	if old != nil {
		widget.UnmountTree(old)
	}

	child := v.build()
	if child == nil {
		v.mu.Lock()
		v.child = nil
		v.mu.Unlock()
		return
	}
	v.adopt(child)
	if ctx != nil {
		widget.MountTree(child, ctx)
	}

	v.mu.Lock()
	v.child = child
	v.mu.Unlock()

	v.SetNeedsRedraw(true)
}

func (v *reactiveView) currentChild() widget.Widget {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.child
}

func (v *reactiveView) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	v.mu.Lock()
	v.ctx = ctx
	v.mu.Unlock()

	v.sync()

	child := v.currentChild()
	if child == nil {
		v.SetBounds(geometry.FromPointSize(v.Position(), geometry.Size{}))
		return geometry.Size{}
	}

	size := widget.LayoutChild(child, ctx, constraints)
	if setter, ok := child.(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(geometry.FromPointSize(geometry.Point{}, size))
	}
	v.SetBounds(geometry.FromPointSize(v.Position(), size))
	return size
}

func (v *reactiveView) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !v.IsVisible() {
		return
	}
	v.sync()

	child := v.currentChild()
	if child == nil {
		return
	}

	bounds := v.Bounds()
	canvas.PushTransform(bounds.Min)
	widget.StampScreenOrigin(child, canvas)
	child.Draw(ctx, canvas)
	canvas.PopTransform()
}

func (v *reactiveView) Event(ctx widget.Context, e event.Event) bool {
	if !v.IsVisible() || !v.IsEnabled() {
		return false
	}

	child := v.currentChild()
	if child == nil {
		return false
	}

	switch ev := e.(type) {
	case *event.MouseEvent:
		local := *ev
		local.Position = ev.Position.Sub(v.Bounds().Min)
		return child.Event(ctx, &local)
	case *event.WheelEvent:
		local := *ev
		local.Position = ev.Position.Sub(v.Bounds().Min)
		return child.Event(ctx, &local)
	default:
		return child.Event(ctx, e)
	}
}

var _ widget.Widget = (*reactiveView)(nil)
var _ widget.Lifecycle = (*reactiveView)(nil)
