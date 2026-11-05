package appui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/router"
)

func labelPage(label string) router.PageBuilder {
	return func(router.RouteContext) widget.Widget {
		return primitives.Text(label)
	}
}

func viewLabel(t *testing.T, view *routerView) string {
	t.Helper()
	children := view.Children()
	if len(children) != 1 {
		t.Fatalf("router view children = %d, want 1", len(children))
	}
	text, ok := children[0].(*primitives.TextWidget)
	if !ok {
		t.Fatalf("page = %T, want *primitives.TextWidget", children[0])
	}
	return text.Content()
}

func TestRouterViewSwapsPageWhenLocationChanges(t *testing.T) {
	r := router.New(
		&router.Route{Segment: "", Name: "home", Page: labelPage("home")},
		&router.Route{
			Segment: "projects", Name: "projects", Page: labelPage("projects"),
			Children: []*router.Route{
				{Segment: ":projectID", Name: "project", Page: labelPage("project")},
			},
		},
	)

	view := newRouterView(r)
	ctx := widget.NewContext()
	ctx.SetWindowSize(geometry.Sz(800, 600))
	widget.MountTree(view, ctx)
	defer widget.UnmountTree(view)

	constraints := geometry.Tight(geometry.Sz(800, 600))
	widget.LayoutChild(view, ctx, constraints)
	if got := viewLabel(t, view); got != "home" {
		t.Fatalf("initial page = %q, want home", got)
	}

	r.Navigate("/projects/ambient")
	widget.LayoutChild(view, ctx, constraints)
	if got := viewLabel(t, view); got != "project" {
		t.Errorf("after navigate page = %q, want project", got)
	}

	r.Navigate("/projects")
	widget.LayoutChild(view, ctx, constraints)
	if got := viewLabel(t, view); got != "projects" {
		t.Errorf("after second navigate page = %q, want projects", got)
	}
}

func TestRouterViewRetainsOutgoingPageForTransition(t *testing.T) {
	previous := SigReducedMotion.Get()
	SigReducedMotion.Set(false)
	t.Cleanup(func() { SigReducedMotion.Set(previous) })

	r := router.New(
		&router.Route{Segment: "", Name: "home", Page: labelPage("home")},
		&router.Route{Segment: "projects", Name: "projects", Page: labelPage("projects")},
	)
	view := newRouterView(r)
	ctx := widget.NewContext()
	ctx.SetWindowSize(geometry.Sz(800, 600))
	widget.MountTree(view, ctx)
	defer widget.UnmountTree(view)

	constraints := geometry.Tight(geometry.Sz(800, 600))
	widget.LayoutChild(view, ctx, constraints)

	r.Navigate("/projects")
	widget.LayoutChild(view, ctx, constraints)

	view.mu.Lock()
	from := view.from
	view.mu.Unlock()
	if from == nil {
		t.Fatal("expected outgoing page to be retained during the transition")
	}
}

func TestRouterViewSkipsTransitionWhenReducedMotion(t *testing.T) {
	previous := SigReducedMotion.Get()
	SigReducedMotion.Set(true)
	t.Cleanup(func() { SigReducedMotion.Set(previous) })

	r := router.New(
		&router.Route{Segment: "", Name: "home", Page: labelPage("home")},
		&router.Route{Segment: "projects", Name: "projects", Page: labelPage("projects")},
	)
	view := newRouterView(r)
	ctx := widget.NewContext()
	ctx.SetWindowSize(geometry.Sz(800, 600))
	widget.MountTree(view, ctx)
	defer widget.UnmountTree(view)

	constraints := geometry.Tight(geometry.Sz(800, 600))
	widget.LayoutChild(view, ctx, constraints)

	r.Navigate("/projects")
	widget.LayoutChild(view, ctx, constraints)

	view.mu.Lock()
	from := view.from
	view.mu.Unlock()
	if from != nil {
		t.Fatal("reduced motion should not retain the outgoing page")
	}
}

func TestRouterViewPageIsMounted(t *testing.T) {
	r := router.New(&router.Route{Segment: "", Name: "home", Page: labelPage("home")})
	view := newRouterView(r)
	ctx := widget.NewContext()
	widget.MountTree(view, ctx)
	defer widget.UnmountTree(view)

	child := view.Children()[0]
	mounted, ok := child.(interface{ IsMounted() bool })
	if !ok {
		t.Fatalf("%T does not expose IsMounted", child)
	}
	if !mounted.IsMounted() {
		t.Error("initial page was not mounted by the framework walk")
	}
}
