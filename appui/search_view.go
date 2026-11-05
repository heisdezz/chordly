package appui

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
)

// Search results page (Metrolist style): query header, categorized filter
// pills (All / Songs / Albums / Artists / Playlists), and result lists fed by
// the live InnerTube search backend (WS-YTM).
//
// This view is a standalone constructor (NewSearchView) so it can be wired
// into the shell or router during an INTEGRATION phase without editing shared
// files. To launch a search, call RunSearch(query); the view rebuilds itself
// from its own reactive signals afterwards.

// --- Local reactive signals (additive declarations) ---

var (
	searchResultsSignal = gpstate.NewSignal([]model.Track(nil))
	searchFilterSignal  = gpstate.NewSignal(searchFilterAll)
	searchLoadingSignal = gpstate.NewSignal(false)
	searchErrorSignal   = gpstate.NewSignal("")
)

// searchFilter identifies a category pill on the search results page.
type searchFilter int

const (
	searchFilterAll searchFilter = iota
	searchFilterSongs
	searchFilterAlbums
	searchFilterArtists
	searchFilterPlaylists
)

var searchFilterLabels = [...]string{"All", "Songs", "Albums", "Artists", "Playlists"}

// --- Search request manager (debounced, async) ---

const searchDebounceInterval = 300 * time.Millisecond

var (
	searchMu          sync.Mutex
	searchRequestID   uint64
	searchLastQuery   string
	searchLastFetched string
)

// RunSearch submits a query to the live InnerTube search backend.
//
// Setting-part: it mirrors the query into the public state.SearchQuery signal
// so the header (and future top bar integration) can display it, then performs
// the actual fetch 300ms later unless a newer request superseded it. Rapid
// bursts therefore only issue one network call for the final query.
func RunSearch(query string) {
	query = strings.TrimSpace(query)

	if query == "" {
		searchMu.Lock()
		searchRequestID++ // invalidate any debounced fetch still pending
		searchLastQuery = ""
		searchLastFetched = ""
		searchMu.Unlock()
		searchLoadingSignal.Set(false)
		searchErrorSignal.Set("")
		searchResultsSignal.Set(nil)
		return
	}

	if state.SearchQuery.Get() != query {
		state.SearchQuery.Set(query)
	}

	searchMu.Lock()
	searchRequestID++
	id := searchRequestID
	searchLastQuery = query
	shouldFetch := query != searchLastFetched
	searchMu.Unlock()

	if !shouldFetch {
		return
	}

	time.AfterFunc(searchDebounceInterval, func() {
		if !searchFetchStillCurrent(id, query) {
			return
		}
		searchFetch(id, query)
	})
}

func searchFetchStillCurrent(id uint64, query string) bool {
	searchMu.Lock()
	defer searchMu.Unlock()
	return searchRequestID == id && searchLastQuery == query
}

func searchFetch(id uint64, query string) {
	searchLoadingSignal.Set(true)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	tracks, err := ytm.DefaultService.Client().Search(ctx, query)

	searchLoadingSignal.Set(false)
	if !searchFetchStillCurrent(id, query) {
		return
	}

	if err != nil {
		log.Printf("[UI] Search failed for %q: %v", query, err)
		searchErrorSignal.Set(err.Error())
		searchResultsSignal.Set(nil)
		ShowError("Search failed — check your connection and try again.")
		return
	}

	searchErrorSignal.Set("")
	searchResultsSignal.Set(tracks)

	searchMu.Lock()
	searchLastFetched = query
	searchMu.Unlock()
	log.Printf("[UI] Search for %q returned %d tracks", query, len(tracks))
}

// playSearchResultAt seeds the playback queue with the current search results
// and starts the track at index. The stream URL is resolved asynchronously so
// the UI reacts instantly (same pattern as ytm.Service.LiveSearch).
func playSearchResultAt(index int) {
	queue := searchResultsSignal.Get()
	if index < 0 || index >= len(queue) {
		return
	}

	track := queue[index]
	state.Queue.Set(queue)
	state.QueueIndex.Set(index)
	state.CurrentTrack.Set(track)
	state.Playhead.Set(0)
	state.PlaybackStatus.Set(model.StatePlaying)

	go func(t model.Track) {
		resCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ytm.DefaultService.PrepareTrackStream(resCtx, &t); err == nil {
			if state.QueueIndex.Get() == index {
				state.CurrentTrack.Set(t)
			}
		} else {
			log.Printf("[UI] Stream resolve failed for %q: %v", t.Title, err)
		}
	}(track)
}

// --- Reactive view ---

// SearchViewWidget is a single-child container that swaps its content whenever
// a search signal changes. It mirrors the routerView swap protocol: signal
// callbacks only mark the change pending, and the actual widget rebuild
// happens at the start of Layout, on the UI thread.
type SearchViewWidget struct {
	widget.WidgetBase

	mu          sync.Mutex
	ctx         widget.Context
	child       widget.Widget
	hasPending  bool
	unsubscribe []func()
}

// NewSearchView creates the standalone search results page.
func NewSearchView() widget.Widget {
	view := &SearchViewWidget{}
	view.SetVisible(true)
	view.SetEnabled(true)
	view.child = buildSearchContent()
	view.adoptChild(view.child)
	return view
}

func (v *SearchViewWidget) adoptChild(child widget.Widget) {
	if setter, ok := child.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(v)
	}
}

// Mount subscribes to every signal the page renders from.
func (v *SearchViewWidget) Mount(ctx widget.Context) {
	v.mu.Lock()
	v.ctx = ctx
	v.mu.Unlock()

	rebuild := func() {
		v.mu.Lock()
		v.hasPending = true
		subCtx := v.ctx
		v.mu.Unlock()

		v.MarkNeedsLayout()
		v.SetNeedsRedraw(true)
		if subCtx != nil {
			subCtx.Invalidate()
		}
	}

	v.unsubscribe = append(v.unsubscribe,
		gpstate.SubscribeForever(searchResultsSignal.AsReadonly(), func([]model.Track) { rebuild() }),
		gpstate.SubscribeForever(searchFilterSignal.AsReadonly(), func(searchFilter) { rebuild() }),
		gpstate.SubscribeForever(searchLoadingSignal.AsReadonly(), func(bool) { rebuild() }),
		gpstate.SubscribeForever(searchErrorSignal.AsReadonly(), func(string) { rebuild() }),
	)
}

// Unmount releases the signal subscriptions.
func (v *SearchViewWidget) Unmount() {
	v.mu.Lock()
	subs := v.unsubscribe
	v.unsubscribe = nil
	v.mu.Unlock()

	for _, unsub := range subs {
		if unsub != nil {
			unsub()
		}
	}
}

// Children exposes the current content so the framework mounts it.
func (v *SearchViewWidget) Children() []widget.Widget {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.child == nil {
		return nil
	}
	return []widget.Widget{v.child}
}

// sync swaps in the rebuilt content. It runs on the UI thread, before layout.
func (v *SearchViewWidget) sync() {
	v.mu.Lock()
	if !v.hasPending {
		v.mu.Unlock()
		return
	}
	v.hasPending = false
	ctx := v.ctx
	old := v.child
	v.mu.Unlock()

	child := buildSearchContent()
	if child == nil {
		return
	}

	if old != nil {
		widget.UnmountTree(old)
	}
	v.adoptChild(child)
	if ctx != nil {
		widget.MountTree(child, ctx)
	}

	v.mu.Lock()
	v.child = child
	v.mu.Unlock()

	v.SetNeedsRedraw(true)
}

// Layout delegates to the active content and records the mount context.
func (v *SearchViewWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	v.mu.Lock()
	v.ctx = ctx
	v.mu.Unlock()

	v.sync()

	v.mu.Lock()
	child := v.child
	v.mu.Unlock()
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

// Draw delegates to the active content with the view's origin applied.
func (v *SearchViewWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !v.IsVisible() {
		return
	}
	v.sync()

	v.mu.Lock()
	child := v.child
	v.mu.Unlock()
	if child == nil {
		return
	}

	bounds := v.Bounds()
	canvas.PushTransform(bounds.Min)
	widget.StampScreenOrigin(child, canvas)
	child.Draw(ctx, canvas)
	canvas.PopTransform()
}

// Event forwards input to the active content with local coordinates.
func (v *SearchViewWidget) Event(ctx widget.Context, e event.Event) bool {
	if !v.IsVisible() || !v.IsEnabled() {
		return false
	}

	v.mu.Lock()
	child := v.child
	v.mu.Unlock()
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

var _ widget.Widget = (*SearchViewWidget)(nil)
var _ widget.Lifecycle = (*SearchViewWidget)(nil)

// --- Content builders ---

func buildSearchContent() widget.Widget {
	query := state.SearchQuery.Get()
	filter := searchFilterSignal.Get()

	rows := []widget.Widget{
		buildSearchHeader(query),
		primitives.Box().Height(14),
		buildFilterPillsRow(),
		primitives.Box().Height(16),
	}

	results := searchResultsSignal.Get()
	loading := searchLoadingSignal.Get()
	errMsg := searchErrorSignal.Get()

	switch {
	case loading && len(results) > 0:
		// Keep previous results visible while the next query loads.
		rows = append(rows, buildTrackList(results)...)
	case loading:
		rows = append(rows, NewSearchLoading(query))
	case errMsg != "":
		rows = append(rows, buildSearchStateCard("Search failed", errMsg, IconSearch))
	case query == "" && len(results) == 0:
		rows = append(rows, buildSearchStateCard(
			"Search YouTube Music",
			"Submit a query from the top bar to find songs, albums, artists, and playlists.",
			IconSearch,
		))
	case len(results) == 0:
		rows = append(rows, buildSearchStateCard(
			"No results",
			fmt.Sprintf("YouTube Music returned nothing for %q. Try a different spelling.", query),
			IconWaveform,
		))
	default:
		switch filter {
		case searchFilterArtists:
			rows = append(rows, buildArtistSection(results)...)
		case searchFilterAlbums:
			rows = append(rows, buildAlbumSection(results)...)
		case searchFilterPlaylists:
			// The InnerTube search backend currently parses song shelves only;
			// playlist categories arrive with the WS-YTM filter-token work.
			rows = append(rows, buildSearchStateCard(
				"Playlists unavailable",
				"Playlist categories land with the InnerTube filter tokens in the WS-YTM Search Service.",
				IconQueue,
			))
		default:
			// "All" and "Songs" both render the ranked track list.
			rows = append(rows, buildTrackList(results)...)
		}
	}

	return primitives.VBox(rows...).CrossAlign(primitives.CrossAxisStretch)
}

func buildSearchHeader(query string) widget.Widget {
	subtitle := "Results come straight from the InnerTube WEB_REMIX client."
	if query != "" {
		subtitle = fmt.Sprintf("Showing results for %q.", query)
	}

	header := primitives.VBox(
		primitives.Text("Search").FontSize(22).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(subtitle).FontSize(12).Color(theme.ColorTextMuted),
	).Gap(2)

	countText := ""
	if count := len(searchResultsSignal.Get()); count > 0 {
		countText = fmt.Sprintf("%d results", count)
	}

	var trailing widget.Widget
	switch {
	case searchLoadingSignal.Get():
		trailing = NewSpinner(14)
	case countText != "":
		trailing = primitives.Text(countText).FontSize(12).Color(theme.ColorTextSecondary)
	default:
		trailing = primitives.Box()
	}

	return primitives.HBox(
		header,
		primitives.Expanded(primitives.Box()),
		trailing,
	).CrossAlign(primitives.CrossAxisCenter)
}

// --- Filter pills ---

type searchFilterPillPainter struct {
	filter searchFilter
}

func (p searchFilterPillPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	active := searchFilterSignal.Get() == p.filter

	bg := theme.ColorSurfaceLight
	fg := theme.ColorTextSecondary
	if active {
		bg = theme.ColorAccentSoft
		fg = theme.ColorAccent
	}
	switch {
	case s.Pressed:
		bg = bg.Lerp(widget.ColorBlack, 0.15)
	case s.Hovered:
		bg = bg.Lerp(widget.ColorWhite, 0.06)
	}

	canvas.DrawRoundRect(s.Bounds, bg, s.Bounds.Height()/2)
	if !active {
		canvas.StrokeRoundRect(s.Bounds, theme.ColorBorder, s.Bounds.Height()/2, 1)
	}

	textRect := s.Bounds
	textRect.Min.X += 16
	textRect.Max.X -= 16
	canvas.DrawText(s.Text, textRect, 13, fg, active, widget.TextAlignCenter)
}

func (p searchFilterPillPainter) ButtonHeight(button.Size) float32             { return 32 }
func (p searchFilterPillPainter) ButtonPadding(button.Size) (float32, float32) { return 16, 0 }
func (p searchFilterPillPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p searchFilterPillPainter) ButtonRadius() float32                        { return 16 }

func buildFilterPillsRow() widget.Widget {
	pills := make([]widget.Widget, 0, len(searchFilterLabels))
	for i, label := range searchFilterLabels {
		filter := searchFilter(i)
		pill := button.New(
			button.Text(label),
			button.OnClick(func() { searchFilterSignal.Set(filter) }),
			button.PainterOpt(searchFilterPillPainter{filter: filter}),
		)
		pills = append(pills, pill)
	}
	return primitives.HBox(pills...).Gap(8)
}

// --- Track list ("All" / "Songs") ---

func buildTrackList(tracks []model.Track) []widget.Widget {
	rows := []widget.Widget{
		buildSectionHeader("Top results", "Tap play on a row to seed the queue."),
		primitives.Box().Height(8),
	}
	for i, track := range tracks {
		index, track := i, track
		rows = append(rows,
			buildSearchTrackRow(index, track, func() { playSearchResultAt(index) }),
			primitives.Box().Height(8),
		)
	}
	return rows
}

func buildSearchTrackRow(index int, track model.Track, onPlay func()) widget.Widget {
	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 16),
	).
		PaddingXY(14, 12).
		Background(widget.Hex(0x1B2830)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(track.Artist).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	playBtn := NewVectorIconButton(IconPlay, 32, 14, onPlay, func(s *IconButtonStyle) {
		s.Background = theme.ColorAccent
		s.Foreground = theme.ColorOnAccent
		s.BorderWidth = 0
	})

	moreBtn := NewVectorIconButton(IconMore, 32, 14, func() {
		// Context menu ("Play Next", "Add to Queue", …) lands with the
		// shared menu component.
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	return primitives.HBox(
		primitives.Text(fmt.Sprintf("%d", index+1)).FontSize(13).Color(theme.ColorTextMuted),
		primitives.Box().Width(10),
		thumb,
		primitives.Box().Width(12),
		meta,
		primitives.Expanded(primitives.Box()),
		primitives.Text(track.DurationStr).FontSize(12).Color(theme.ColorTextMuted),
		primitives.Box().Width(10),
		playBtn,
		primitives.Box().Width(6),
		moreBtn,
	).
		PaddingXY(12, 8).
		Background(theme.ColorSurface).
		Rounded(12).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}

// --- Artists ("Artists") ---

func buildArtistSection(tracks []model.Track) []widget.Widget {
	order := make([]string, 0)
	grouped := make(map[string][]model.Track)
	for _, track := range tracks {
		name := strings.TrimSpace(track.Artist)
		if name == "" {
			name = "Unknown artist"
		}
		if _, seen := grouped[name]; !seen {
			order = append(order, name)
		}
		grouped[name] = append(grouped[name], track)
	}

	rows := []widget.Widget{
		buildSectionHeader("Artists", "Grouped from the returned song results."),
		primitives.Box().Height(8),
	}

	cards := make([]widget.Widget, 0, len(order))
	for _, name := range order {
		group := grouped[name]
		name, group := name, group
		cards = append(cards, buildArtistCard(name, len(group), func() { playSearchResultAt(indexOfTrack(tracks, group[0])) }))
	}

	rows = append(rows,
		primitives.HBox(cards...).Gap(12),
		primitives.Box().Height(14),
	)
	return rows
}

func buildArtistCard(name string, songCount int, onPlay func()) widget.Widget {
	avatar := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 20),
	).
		PaddingXY(22, 18).
		Background(widget.Hex(0x16242C)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	meta := primitives.VBox(
		primitives.Text(name).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(fmt.Sprintf("%d songs", songCount)).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	playBtn := NewVectorIconButton(IconPlay, 30, 13, onPlay, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorAccent
	})

	return primitives.HBox(
		avatar,
		primitives.Box().Width(10),
		meta,
		primitives.Expanded(primitives.Box()),
		playBtn,
	).
		Width(230).
		PaddingXY(12, 10).
		Background(theme.ColorSurface).
		Rounded(12).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}

// --- Albums ("Albums") ---

func buildAlbumSection(tracks []model.Track) []widget.Widget {
	order := make([]string, 0)
	grouped := make(map[string][]model.Track)
	for _, track := range tracks {
		album := strings.TrimSpace(track.Album)
		if album == "" {
			album = "Singles" // parser does not surface album shelves yet
		}
		if _, seen := grouped[album]; !seen {
			order = append(order, album)
		}
		grouped[album] = append(grouped[album], track)
	}

	rows := []widget.Widget{
		buildSectionHeader("Albums", "Album data arrives with the WS-YTM filter tokens; parsed tracks are grouped as Singles."),
		primitives.Box().Height(8),
	}

	cards := make([]widget.Widget, 0, len(order))
	for _, album := range order {
		group := grouped[album]
		album, group := album, group
		cards = append(cards, buildAlbumCard(album, group, len(group), func() { playSearchResultAt(indexOfTrack(tracks, group[0])) }))
	}

	rows = append(rows,
		primitives.HBox(cards...).Gap(12),
		primitives.Box().Height(14),
	)
	return rows
}

func buildAlbumCard(album string, group []model.Track, count int, onPlay func()) widget.Widget {
	subtitle := "Various artists"
	if len(group) > 0 && strings.TrimSpace(group[0].Artist) != "" {
		subtitle = group[0].Artist
	}

	art := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 20),
	).
		PaddingXY(22, 18).
		Background(widget.Hex(0x18262E)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	meta := primitives.VBox(
		primitives.Text(album).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(subtitle).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	playBtn := NewVectorIconButton(IconPlay, 30, 13, onPlay, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorAccent
	})

	return primitives.HBox(
		art,
		primitives.Box().Width(10),
		meta,
		primitives.Expanded(primitives.Box()),
		primitives.Text(fmt.Sprintf("%d songs", count)).FontSize(12).Color(theme.ColorTextMuted),
		primitives.Box().Width(8),
		playBtn,
	).
		PaddingXY(12, 10).
		Width(260).
		Background(theme.ColorSurface).
		Rounded(12).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}

// --- Shared helpers ---

func buildSectionHeader(title, subtitle string) widget.Widget {
	return primitives.HBox(
		primitives.Text(title).FontSize(16).Bold().Color(theme.ColorTextPrimary),
		primitives.Expanded(primitives.Box()),
		primitives.Text(subtitle).FontSize(12).Color(theme.ColorTextMuted),
	).CrossAlign(primitives.CrossAxisCenter)
}

func buildSearchStateCard(title, body string, icon IconType) widget.Widget {
	// Vertically centering the icon inside a paintable box is left to the
	// box padding, matching the thumbnail treatment used across the shell.
	art := primitives.HBox(
		NewVectorIcon(icon, theme.ColorAccent, 22),
	).
		Padding(16).
		Background(theme.ColorSurfaceLight).
		Rounded(12)

	return primitives.HBox(
		art,
		primitives.Box().Width(14),
		primitives.VBox(
			primitives.Text(title).FontSize(15).Bold().Color(theme.ColorTextPrimary),
			primitives.Text(body).FontSize(12).Color(theme.ColorTextMuted),
		).Gap(3),
		primitives.Expanded(primitives.Box()),
	).
		Padding(18).
		Background(theme.ColorSurface).
		Rounded(14).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisStart)
}

// indexOfTrack returns the index of track inside tracks, or -1 when missing.
func indexOfTrack(tracks []model.Track, track model.Track) int {
	for i := range tracks {
		if tracks[i].ID == track.ID {
			return i
		}
	}
	return -1
}
