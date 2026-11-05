package appui

import (
	"sync"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/router"
)

// routerView is a single-child container that swaps its page whenever the
// router's match signal changes.
//
// The signal callback only records the pending match and invalidates layout.
// The actual widget swap happens at the start of Layout, so the tree is never
// mutated in the middle of event dispatch (for example while a button's click
// handler is still running).
type routerView struct {
	widget.WidgetBase

	router *router.Router

	mu         sync.Mutex
	ctx        widget.Context
	current    router.Match
	pending    router.Match
	hasPending bool
	child      widget.Widget

	unsubscribe func()
}

func newRouterView(r *router.Router) *routerView {
	match := r.Current()
	view := &routerView{
		router:  r,
		current: match,
	}
	view.SetVisible(true)
	view.SetEnabled(true)
	view.child = view.build(match)
	view.adopt(view.child)
	return view
}

func (v *routerView) build(match router.Match) widget.Widget {
	if match.Route == nil || match.Route.Page == nil {
		return primitives.Text("No page registered for this route").
			FontSize(14).
			Color(colorMuted)
	}
	return match.Route.Page(router.RouteContext{
		Router: v.router,
		Match:  match,
	})
}

func (v *routerView) adopt(child widget.Widget) {
	if child == nil {
		return
	}
	if setter, ok := child.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(v)
	}
}

// Mount subscribes to the match signal. The wrapped page is mounted by the
// framework's MountTree walk through Children().
func (v *routerView) Mount(ctx widget.Context) {
	v.mu.Lock()
	v.ctx = ctx
	v.mu.Unlock()

	v.unsubscribe = v.router.Match().SubscribeForever(func(match router.Match) {
		v.mu.Lock()
		v.pending = match
		v.hasPending = true
		v.mu.Unlock()

		// Ask the window to relayout and repaint. MarkNeedsLayout invalidates
		// this view's cached measurement so the parent actually calls Layout
		// (and therefore sync). The swap itself is deferred to sync(), which
		// runs on the UI thread at the start of Layout.
		v.MarkNeedsLayout()
		ctx.Invalidate()
		v.SetNeedsRedraw(true)
	})
}

// Unmount releases the signal subscription. The framework unmounts the page
// first because it is returned from Children().
func (v *routerView) Unmount() {
	v.mu.Lock()
	unsubscribe := v.unsubscribe
	v.unsubscribe = nil
	v.mu.Unlock()

	if unsubscribe != nil {
		unsubscribe()
	}
}

// Children exposes the current page so the framework mounts/unmounts it and
// propagates the parent chain.
func (v *routerView) Children() []widget.Widget {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.child == nil {
		return nil
	}
	return []widget.Widget{v.child}
}

// sync swaps in the pending page. It runs on the UI thread, before layout.
func (v *routerView) sync() {
	v.mu.Lock()
	if !v.hasPending {
		v.mu.Unlock()
		return
	}
	next := v.pending
	v.hasPending = false
	ctx := v.ctx
	current := v.current
	old := v.child
	v.mu.Unlock()

	if next.Path == current.Path {
		return
	}

	if old != nil {
		widget.UnmountTree(old)
	}

	child := v.build(next)
	if child == nil {
		return
	}
	v.adopt(child)
	if ctx != nil {
		widget.MountTree(child, ctx)
	}

	v.mu.Lock()
	v.current = next
	v.child = child
	v.mu.Unlock()

	v.SetNeedsRedraw(true)
}

func (v *routerView) currentChild() widget.Widget {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.child
}

// Layout delegates to the active page and records the context needed to mount
// pages created later.
func (v *routerView) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
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

// Draw delegates to the active page with the view's origin applied.
func (v *routerView) Draw(ctx widget.Context, canvas widget.Canvas) {
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

// Event forwards input to the active page with local coordinates. The parent
// container has already hit-tested this view's bounds.
func (v *routerView) Event(ctx widget.Context, e event.Event) bool {
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

var _ widget.Widget = (*routerView)(nil)
var _ widget.Lifecycle = (*routerView)(nil)
