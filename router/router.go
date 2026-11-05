// Package router provides a small client-side router for gogpu/ui.
//
// Routes form a tree. Every route owns exactly one URL segment, which may be a
// literal ("projects"), a parameter (":projectID"), or a catch-all ("*").
// Nesting routes therefore produces arbitrarily deep paths, and every matched
// level contributes parameters to the final Match.
//
// The current location is stored in a signal. A computed signal derives the
// current Match, so any widget can subscribe to route changes through
// github.com/gogpu/ui/state.
package router

import (
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// Params holds the path parameters captured by a matched route.
type Params map[string]string

// Get returns the value of the named parameter, or "" when absent.
func (p Params) Get(name string) string {
	return p[name]
}

// Int returns the named parameter parsed as an int. Invalid values yield 0.
func (p Params) Int(name string) int {
	n, _ := strconv.Atoi(p[name])
	return n
}

// Match is the result of resolving a path against the route tree.
type Match struct {
	// Path is the normalized request path, e.g. "/projects/alpha/chords/verse-1".
	Path string
	// Pattern is the pattern that matched, e.g. "/projects/:projectID/chords/:chordID".
	Pattern string
	// Name is the name of the deepest named route in the chain.
	Name string
	// Params contains parameters captured at every nesting level.
	Params Params
	// Chain is the matched route chain from top level to leaf.
	Chain []*Route
	// Route is the leaf route whose Page renders this match.
	Route *Route
	// Found reports whether any declared route matched. A catch-all route still
	// counts as found; check Name (for example "not-found") to distinguish it.
	Found bool
}

// Param returns the value of the named parameter, or "" when absent.
func (m Match) Param(name string) string {
	return m.Params[name]
}

// PageBuilder builds the widget tree for a matched route.
type PageBuilder func(RouteContext) widget.Widget

// RouteContext is passed to a PageBuilder. It exposes the match, its params,
// and helpers for navigating or building links.
type RouteContext struct {
	Router *Router
	Match  Match
}

// Param returns the value of the named parameter.
func (c RouteContext) Param(name string) string {
	return c.Match.Param(name)
}

// Params returns all captured parameters.
func (c RouteContext) Params() Params {
	return c.Match.Params
}

// Navigate pushes path onto the router history.
func (c RouteContext) Navigate(path string) {
	c.Router.Navigate(path)
}

// Link builds a concrete path from a pattern and parameters.
func (c RouteContext) Link(pattern string, params Params) string {
	return Build(pattern, params)
}

// Route is one node in the route tree.
type Route struct {
	// Segment is a single URL segment: "projects", ":projectID", or "*".
	// Use "" for the index route at the current level.
	Segment string
	// Name is an optional route name, useful for breadcrumbs and debugging.
	Name string
	// Page renders the route. It may be nil for grouping-only nodes.
	Page PageBuilder
	// Children are routes nested below this one.
	Children []*Route
}

// Add appends children to the route and returns the route for chaining.
func (r *Route) Add(children ...*Route) *Route {
	r.Children = append(r.Children, children...)
	return r
}

// Router owns the current location signal and the derived match signal.
//
// Navigate must be called from the UI thread, because subscribers may mark
// widgets dirty.
type Router struct {
	routes   []*Route
	location state.Signal[string]
	match    state.ReadonlySignal[Match]

	mu      sync.Mutex
	history []string
}

// New creates a router from a set of top-level routes.
func New(routes ...*Route) *Router {
	r := &Router{routes: routes}
	r.location = state.NewSignal("/")
	r.match = state.NewComputed(func() Match {
		return r.resolve(r.location.Get())
	}, r.location.AsReadonly())
	return r
}

// Location returns a read-only signal of the current path.
func (r *Router) Location() state.ReadonlySignal[string] {
	return r.location.AsReadonly()
}

// Match returns a computed signal of the current match.
func (r *Router) Match() state.ReadonlySignal[Match] {
	return r.match
}

// Current returns the current match synchronously.
func (r *Router) Current() Match {
	return r.match.Get()
}

// Navigate pushes path onto the history and resolves it.
func (r *Router) Navigate(path string) {
	next := normalize(path)

	r.mu.Lock()
	current := r.location.Get()
	if next == current {
		r.mu.Unlock()
		return
	}
	r.history = append(r.history, current)
	r.mu.Unlock()

	r.location.Set(next)
}

// Replace resolves path without adding a history entry. Use it for redirects.
func (r *Router) Replace(path string) {
	r.location.Set(normalize(path))
}

// Back returns to the previous location. It reports false when history is empty.
func (r *Router) Back() bool {
	r.mu.Lock()
	if len(r.history) == 0 {
		r.mu.Unlock()
		return false
	}
	prev := r.history[len(r.history)-1]
	r.history = r.history[:len(r.history)-1]
	r.mu.Unlock()

	r.location.Set(prev)
	return true
}

// CanGoBack reports whether there is a previous location.
func (r *Router) CanGoBack() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.history) > 0
}

// Build substitutes parameters into a route pattern.
//
//	Build("/projects/:projectID/chords/:chordID", Params{
//	    "projectID": "ambient",
//	    "chordID":   "verse 1",
//	}) // "/projects/ambient/chords/verse%201"
func Build(pattern string, params Params) string {
	parts := strings.Split(pattern, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, ":") {
			value, ok := params[strings.TrimPrefix(part, ":")]
			if ok {
				segments = append(segments, url.PathEscape(value))
				continue
			}
		}
		segments = append(segments, part)
	}
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

// --- Matching ---

type candidate struct {
	route  *Route
	chain  []*Route
	params Params
	score  int
}

func (r *Router) resolve(path string) Match {
	segments := splitPath(path)

	var best *candidate
	for _, route := range r.routes {
		matchChild(route, nil, Params{}, segments, 0, 0, &best)
	}

	if best == nil {
		return Match{
			Path:    path,
			Pattern: path,
			Params:  Params{},
			Found:   false,
		}
	}

	pattern, name := patternFor(best.chain)
	return Match{
		Path:    path,
		Pattern: pattern,
		Name:    name,
		Params:  best.params,
		Chain:   best.chain,
		Route:   best.route,
		Found:   true,
	}
}

// matchChild consumes the route's own segment against segs[idx], then walks
// its children.
func matchChild(route *Route, chain []*Route, params Params, segs []string, idx, score int, best **candidate) {
	switch {
	case route.Segment == "*":
		// A catch-all consumes the remainder of the path.
		matchRoute(route, chain, params, segs, len(segs), score+1, best)

	case route.Segment == "":
		// Index route: only matches when the path is exhausted here.
		if idx == len(segs) {
			matchRoute(route, chain, params, segs, idx, score+3, best)
		}

	case idx >= len(segs):
		return

	case strings.HasPrefix(route.Segment, ":"):
		name := strings.TrimPrefix(route.Segment, ":")
		segment := segs[idx]
		if name == "" || segment == "" {
			return
		}
		next := cloneParams(params)
		next[name] = decodeSegment(segment)
		matchRoute(route, chain, next, segs, idx+1, score+2, best)

	case route.Segment == segs[idx]:
		matchRoute(route, chain, params, segs, idx+1, score+3, best)
	}
}

// matchRoute records the route as a candidate when it fully matches and has a
// page, then recurses into children.
func matchRoute(route *Route, chain []*Route, params Params, segs []string, idx, score int, best **candidate) {
	chain = appendRoute(chain, route)

	if idx == len(segs) && route.Page != nil {
		consider(&candidate{
			route:  route,
			chain:  chain,
			params: params,
			score:  score,
		}, best)
	}

	for _, child := range route.Children {
		matchChild(child, chain, params, segs, idx, score, best)
	}
}

func consider(c *candidate, best **candidate) {
	if *best == nil ||
		c.score > (*best).score ||
		(c.score == (*best).score && len(c.chain) > len((*best).chain)) {
		*best = c
	}
}

func patternFor(chain []*Route) (pattern, name string) {
	parts := make([]string, 0, len(chain))
	for _, route := range chain {
		if route.Segment != "" {
			parts = append(parts, route.Segment)
		}
		if route.Name != "" {
			name = route.Name
		}
	}
	if len(parts) == 0 {
		return "/", name
	}
	return "/" + strings.Join(parts, "/"), name
}

func appendRoute(chain []*Route, route *Route) []*Route {
	next := make([]*Route, len(chain)+1)
	copy(next, chain)
	next[len(chain)] = route
	return next
}

func cloneParams(params Params) Params {
	next := make(Params, len(params)+1)
	for key, value := range params {
		next[key] = value
	}
	return next
}

func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

func normalize(path string) string {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	segments := splitPath(path)
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func decodeSegment(segment string) string {
	value, err := url.PathUnescape(segment)
	if err != nil {
		return segment
	}
	return value
}
