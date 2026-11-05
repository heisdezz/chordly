package appui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

func reactiveLabel(t *testing.T, view *reactiveView) string {
	t.Helper()
	children := view.Children()
	if len(children) != 1 {
		t.Fatalf("reactive view children = %d, want 1", len(children))
	}
	text, ok := children[0].(*primitives.TextWidget)
	if !ok {
		t.Fatalf("child = %T, want *primitives.TextWidget", children[0])
	}
	return text.Content()
}

func mountReactive(t *testing.T, view *reactiveView) (widget.Context, geometry.Constraints) {
	t.Helper()
	ctx := widget.NewContext()
	ctx.SetWindowSize(geometry.Sz(800, 600))
	widget.MountTree(view, ctx)
	t.Cleanup(func() { widget.UnmountTree(view) })
	return ctx, geometry.Tight(geometry.Sz(800, 600))
}

func TestReactiveViewRebuildsOnSignal(t *testing.T) {
	label := gpstate.NewSignal("first")
	view := newReactiveView(
		func() widget.Widget { return primitives.Text(label.Get()) },
		WatchSignal(label.AsReadonly()),
	)
	ctx, constraints := mountReactive(t, view)

	widget.LayoutChild(view, ctx, constraints)
	if got := reactiveLabel(t, view); got != "first" {
		t.Fatalf("initial child = %q, want first", got)
	}

	label.Set("second")
	widget.LayoutChild(view, ctx, constraints)
	if got := reactiveLabel(t, view); got != "second" {
		t.Fatalf("after signal change child = %q, want second", got)
	}
}

func TestReactiveViewRebuildsOnAnyWatchedSignal(t *testing.T) {
	first := gpstate.NewSignal(1)
	second := gpstate.NewSignal("a")
	view := newReactiveView(
		func() widget.Widget {
			return primitives.Text(second.Get() + ":" + string(rune('0'+first.Get())))
		},
		WatchSignal(first.AsReadonly()),
		WatchSignal(second.AsReadonly()),
	)
	ctx, constraints := mountReactive(t, view)
	widget.LayoutChild(view, ctx, constraints)

	second.Set("b")
	widget.LayoutChild(view, ctx, constraints)
	if got := reactiveLabel(t, view); got != "b:1" {
		t.Fatalf("after second signal change child = %q, want b:1", got)
	}
}

func TestReactiveViewUnmountStopsRebuilds(t *testing.T) {
	label := gpstate.NewSignal("first")
	view := newReactiveView(
		func() widget.Widget { return primitives.Text(label.Get()) },
		WatchSignal(label.AsReadonly()),
	)
	ctx, constraints := mountReactive(t, view)
	widget.LayoutChild(view, ctx, constraints)

	widget.UnmountTree(view)
	label.Set("second")

	view.mu.Lock()
	dirty := view.dirty
	view.mu.Unlock()
	if dirty {
		t.Fatal("view was marked dirty after unmount")
	}
	if got := reactiveLabel(t, view); got != "first" {
		t.Fatalf("child after unmount = %q, want first", got)
	}
}
