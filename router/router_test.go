package router

import (
	"testing"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func page(name string) PageBuilder {
	return func(RouteContext) widget.Widget {
		return primitives.Text(name)
	}
}

func testRouter() *Router {
	return New(
		&Route{Segment: "", Name: "home", Page: page("home")},
		&Route{
			Segment: "projects",
			Name:    "projects",
			Page:    page("projects"),
			Children: []*Route{
				{
					Segment: ":projectID",
					Name:    "project",
					Page:    page("project"),
					Children: []*Route{
						{
							Segment: "chords",
							Name:    "chords",
							Page:    page("chords"),
							Children: []*Route{
								{Segment: ":chordID", Name: "chord", Page: page("chord")},
							},
						},
					},
				},
			},
		},
		&Route{Segment: "settings", Name: "settings", Page: page("settings")},
		&Route{Segment: "*", Name: "not-found", Page: page("not-found")},
	)
}

func TestResolveDeepNestedParams(t *testing.T) {
	r := testRouter()

	tests := []struct {
		path    string
		pattern string
		name    string
		params  Params
	}{
		{path: "/", pattern: "/", name: "home"},
		{path: "/projects", pattern: "/projects", name: "projects"},
		{
			path:    "/projects/ambient",
			pattern: "/projects/:projectID",
			name:    "project",
			params:  Params{"projectID": "ambient"},
		},
		{
			path:    "/projects/ambient/chords",
			pattern: "/projects/:projectID/chords",
			name:    "chords",
			params:  Params{"projectID": "ambient"},
		},
		{
			path:    "/projects/ambient/chords/verse-1",
			pattern: "/projects/:projectID/chords/:chordID",
			name:    "chord",
			params:  Params{"projectID": "ambient", "chordID": "verse-1"},
		},
		{
			path:    "/projects/a%20b/chords/c%2Fd",
			pattern: "/projects/:projectID/chords/:chordID",
			name:    "chord",
			params:  Params{"projectID": "a b", "chordID": "c/d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			m := r.resolve(normalize(tt.path))

			if !m.Found {
				t.Fatalf("expected %q to match", tt.path)
			}
			if m.Pattern != tt.pattern {
				t.Errorf("pattern = %q, want %q", m.Pattern, tt.pattern)
			}
			if m.Name != tt.name {
				t.Errorf("name = %q, want %q", m.Name, tt.name)
			}
			for key, want := range tt.params {
				if got := m.Param(key); got != want {
					t.Errorf("param %q = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestResolveChainIncludesEveryLevel(t *testing.T) {
	m := testRouter().resolve("/projects/ambient/chords/verse-1")

	if len(m.Chain) != 4 {
		t.Fatalf("chain length = %d, want 4 (%v)", len(m.Chain), m.Pattern)
	}
	wantPatterns := []string{"projects", ":projectID", "chords", ":chordID"}
	for i, want := range wantPatterns {
		if m.Chain[i].Segment != want {
			t.Errorf("chain[%d] = %q, want %q", i, m.Chain[i].Segment, want)
		}
	}
	if m.Route == nil || m.Route.Name != "chord" {
		t.Errorf("leaf route = %v, want chord", m.Route)
	}
}

func TestResolveCatchAllAndMiss(t *testing.T) {
	r := testRouter()

	m := r.resolve("/does/not/exist")
	if !m.Found || m.Name != "not-found" || m.Pattern != "/*" {
		t.Errorf("catch-all match = %+v", m)
	}
	if len(m.Params) != 0 {
		t.Errorf("catch-all params = %v, want none", m.Params)
	}

	// Without a catch-all, unknown paths report Found=false.
	strict := New(&Route{Segment: "", Name: "home", Page: page("home")})
	miss := strict.resolve("/nope")
	if miss.Found {
		t.Errorf("expected miss, got %+v", miss)
	}
}

func TestResolvePrefersLiteralThenParamThenWildcard(t *testing.T) {
	r := New(
		&Route{Segment: "*", Name: "wild", Page: page("wild")},
		&Route{Segment: ":slug", Name: "param", Page: page("param")},
		&Route{Segment: "settings", Name: "literal", Page: page("literal")},
	)

	for path, want := range map[string]string{
		"/settings": "literal",
		"/anything": "param",
		"/a/b":      "wild",
	} {
		if got := r.resolve(path).Name; got != want {
			t.Errorf("resolve(%q).Name = %q, want %q", path, got, want)
		}
	}
}

func TestSignalsUpdateOnNavigate(t *testing.T) {
	r := testRouter()

	var notified []string
	unsub := r.Location().SubscribeForever(func(path string) {
		notified = append(notified, path)
	})
	defer unsub()

	r.Navigate("/projects")
	r.Navigate("/projects/ambient/chords/verse-1")

	if got := r.Current().Param("projectID"); got != "ambient" {
		t.Errorf("projectID = %q, want ambient", got)
	}
	if got := r.Current().Param("chordID"); got != "verse-1" {
		t.Errorf("chordID = %q, want verse-1", got)
	}
	if got := r.Location().Get(); got != "/projects/ambient/chords/verse-1" {
		t.Errorf("location = %q", got)
	}
	if len(notified) != 2 {
		t.Errorf("location notifications = %v, want 2", notified)
	}
}

func TestHistoryAndBack(t *testing.T) {
	r := testRouter()

	if r.CanGoBack() {
		t.Fatal("new router should not have history")
	}

	r.Navigate("/projects")
	r.Navigate("/projects/ambient")
	if !r.CanGoBack() {
		t.Fatal("expected history after navigation")
	}

	if !r.Back() {
		t.Fatal("Back() = false, want true")
	}
	if got := r.Current().Path; got != "/projects" {
		t.Errorf("after Back path = %q, want /projects", got)
	}
	if !r.Back() {
		t.Fatal("second Back() = false, want true")
	}
	if got := r.Current().Path; got != "/" {
		t.Errorf("after second Back path = %q, want /", got)
	}
	if r.Back() {
		t.Error("Back() on empty history = true, want false")
	}
}

func TestNavigateSamePathIsNoOp(t *testing.T) {
	r := testRouter()
	calls := 0
	unsub := r.Location().SubscribeForever(func(string) { calls++ })
	defer unsub()

	r.Navigate("/projects")
	r.Navigate("/projects/")
	r.Navigate("/projects?tab=chords")

	if calls != 1 {
		t.Errorf("notifications = %d, want 1", calls)
	}
}

func TestBuild(t *testing.T) {
	got := Build("/projects/:projectID/chords/:chordID", Params{
		"projectID": "ambient",
		"chordID":   "verse 1",
	})
	want := "/projects/ambient/chords/verse%201"
	if got != want {
		t.Errorf("Build() = %q, want %q", got, want)
	}

	if got := Build("/", nil); got != "/" {
		t.Errorf("Build(/) = %q", got)
	}
}
