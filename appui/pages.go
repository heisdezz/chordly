package appui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

type project struct {
	ID          string
	Name        string
	Description string
}

type chord struct {
	ID    string
	Name  string
	Notes string
}

var projectCatalog = []project{
	{ID: "ambient", Name: "Ambient Studies", Description: "Slow pads, drones, and long fades."},
	{ID: "neon", Name: "Neon Lines", Description: "Synthwave sketches with bright leads."},
	{ID: "paper", Name: "Paper Folk", Description: "Acoustic loops and fingerpicking."},
}

var chordCatalog = map[string][]chord{
	"ambient": {
		{ID: "verse-1", Name: "Verse 1", Notes: "Cmaj7 → Am9"},
		{ID: "chorus", Name: "Chorus", Notes: "Fmaj7 → G6"},
	},
	"neon": {
		{ID: "intro", Name: "Intro", Notes: "Am → F → C → G"},
		{ID: "drop", Name: "Drop", Notes: "Dm → Bb → F → C"},
	},
	"paper": {
		{ID: "verse", Name: "Verse", Notes: "G → D → Em → C"},
	},
}

func projectByID(id string) project {
	for _, p := range projectCatalog {
		if p.ID == id {
			return p
		}
	}
	return project{ID: id, Name: titleCase(id), Description: "No metadata for this project."}
}

func chordsFor(projectID string) []chord {
	if list, ok := chordCatalog[projectID]; ok {
		return list
	}
	return []chord{{ID: "sketch-1", Name: "Untitled sketch", Notes: "No notes yet."}}
}

// HomePage is the index route and renders the live home feed.
func HomePage(c router.RouteContext) widget.Widget {
	return NewHomeContent()
}

// QueuePage renders the active playback queue with the current track marked.
// It rebuilds whenever the queue or the active index changes.
func QueuePage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget { return buildQueueContent(c) },
		WatchSignal(state.Queue.AsReadonly()),
		WatchSignal(state.QueueIndex.AsReadonly()),
	)
}

func buildQueueContent(c router.RouteContext) widget.Widget {
	queue := state.Queue.Get()
	currentIdx := state.QueueIndex.Get()

	rows := make([]widget.Widget, 0, len(queue))
	for i, track := range queue {
		rows = append(rows, newQueueRow(track, i == currentIdx))
	}

	header := primitives.VBox(
		primitives.Text("Queue").FontSize(28).Bold().Color(theme.ColorTextPrimary),
		primitives.Text("Tracks that will play next, straight from the active queue.").
			FontSize(13).
			Color(theme.ColorTextSecondary),
	).Gap(4)

	var body widget.Widget
	if len(rows) == 0 {
		body = primitives.VBox(
			primitives.Text("Nothing queued yet. Play a track from Home or Search to start a radio.").
				FontSize(13).
				Color(theme.ColorTextMuted),
		).Padding(18).
			Background(theme.ColorSurface).
			Rounded(16).
			BorderStyle(1, theme.ColorBorder)
	} else {
		body = primitives.VBox(
			primitives.VBox(rows...).Gap(8),
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				queueClearButton(func() {
					state.Queue.Set(nil)
					state.QueueIndex.Set(-1)
					ShowInfo("Queue cleared")
					c.Router.Replace("/")
				}),
			),
		).Padding(16).
			Gap(12).
			Background(theme.ColorSurface).
			Rounded(16).
			BorderStyle(1, theme.ColorBorder)
	}

	return primitives.VBox(
		header,
		primitives.Box().Height(8),
		body,
	).
		Padding(28).
		Gap(10).
		CrossAlign(primitives.CrossAxisStretch).
		Background(theme.ColorCanvas)
}

// queueClearButton is a small theme-aware text action used by the queue page.
type queueClearButtonPainter struct{}

func (p queueClearButtonPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}
	bg := widget.ColorTransparent
	if s.Hovered {
		bg = theme.ColorSurfaceLight
	}
	canvas.DrawRoundRect(s.Bounds, bg, 10)
	canvas.DrawText(s.Text, s.Bounds, 12, theme.ColorTextMuted, false, widget.TextAlignCenter)
}

func (p queueClearButtonPainter) ButtonHeight(button.Size) float32             { return 32 }
func (p queueClearButtonPainter) ButtonPadding(button.Size) (float32, float32) { return 14, 0 }
func (p queueClearButtonPainter) ButtonFontSize(button.Size) float32           { return 12 }
func (p queueClearButtonPainter) ButtonRadius() float32                        { return 10 }

var _ button.Painter = queueClearButtonPainter{}
var _ button.LayoutMetrics = queueClearButtonPainter{}

func queueClearButton(onClick func()) *button.Widget {
	return button.New(
		button.Text("Clear queue"),
		button.OnClick(onClick),
		button.PainterOpt(queueClearButtonPainter{}),
	)
}

// LyricsPage hosts the synchronized lyrics overlay as a routable page.
func LyricsPage(c router.RouteContext) widget.Widget {
	return primitives.VBox(
		NewLyricsOverlay(c.Router),
	).Padding(24).Background(theme.ColorCanvas)
}

// SearchPage renders the standalone search view. Queries are submitted from
// the top-bar text field, which writes to state.SearchQuery.
func SearchPage(c router.RouteContext) widget.Widget {
	return primitives.VBox(NewSearchView()).Padding(24).Background(theme.ColorCanvas)
}

// ProjectsPage lists the top-level project routes.
func ProjectsPage(c router.RouteContext) widget.Widget {
	rows := make([]widget.Widget, 0, len(projectCatalog))
	for _, p := range projectCatalog {
		p := p
		path := router.Build("/projects/:projectID", router.Params{"projectID": p.ID})
		rows = append(rows, projectRow(p, func() { c.Navigate(path) }))
	}

	return page(c.Router, c.Match,
		heading("Projects", "Route level 2. Each row links to /projects/:projectID."),
		primitives.VBox(rows...).Gap(10),
	)
}

// ProjectPage shows one project and links to its nested chord list.
func ProjectPage(c router.RouteContext) widget.Widget {
	id := c.Param("projectID")
	p := projectByID(id)
	chordsPath := router.Build("/projects/:projectID/chords", router.Params{"projectID": id})

	return page(c.Router, c.Match,
		heading(p.Name, "Route level 3. The project id comes from the :projectID segment."),
		bodyText(p.Description),
		primitives.HBox(
			actionButton("View chords", toneFilled, func() { c.Navigate(chordsPath) }),
			actionButton("← Projects", toneText, func() { c.Navigate("/projects") }),
		).Gap(8),
		routeInspector(c.Match),
	)
}

// ChordsPage lists chords for a project (two-segment nesting).
func ChordsPage(c router.RouteContext) widget.Widget {
	projectID := c.Param("projectID")
	p := projectByID(projectID)

	rows := make([]widget.Widget, 0, len(chordsFor(projectID)))
	for _, ch := range chordsFor(projectID) {
		ch := ch
		path := router.Build("/projects/:projectID/chords/:chordID", router.Params{
			"projectID": projectID,
			"chordID":   ch.ID,
		})
		rows = append(rows, chordRow(ch, func() { c.Navigate(path) }))
	}

	return page(c.Router, c.Match,
		heading("Chords in "+p.Name, "Route level 4. projectID is still available alongside the chord list."),
		primitives.VBox(rows...).Gap(10),
		primitives.HBox(
			actionButton("← "+p.Name, toneText, func() {
				c.Navigate(router.Build("/projects/:projectID", router.Params{"projectID": projectID}))
			}),
		).Gap(8),
		routeInspector(c.Match),
	)
}

// ChordPage is the deepest page and captures two parameters.
func ChordPage(c router.RouteContext) widget.Widget {
	projectID := c.Param("projectID")
	chordID := c.Param("chordID")

	var selected chord
	for _, ch := range chordsFor(projectID) {
		if ch.ID == chordID {
			selected = ch
			break
		}
	}
	if selected.Name == "" {
		selected = chord{ID: chordID, Name: titleCase(chordID), Notes: "No notes for this chord yet."}
	}

	return page(c.Router, c.Match,
		heading(selected.Name, "Deepest route: /projects/:projectID/chords/:chordID"),
		bodyText(selected.Notes),
		primitives.HBox(
			actionButton("← Chords", toneText, func() {
				c.Navigate(router.Build("/projects/:projectID/chords", router.Params{"projectID": projectID}))
			}),
			actionButton("Project", toneTonal, func() {
				c.Navigate(router.Build("/projects/:projectID", router.Params{"projectID": projectID}))
			}),
		).Gap(8),
		routeInspector(c.Match),
	)
}

// NotFoundPage renders for the catch-all route.
func NotFoundPage(c router.RouteContext) widget.Widget {
	return page(c.Router, c.Match,
		heading("Route not found", "No page is registered for this path."),
		card(
			keyValue("path", c.Match.Path),
			bodyText("Declare another route, or follow one of the links below."),
		),
		primitives.HBox(
			actionButton("Go home", toneFilled, func() { c.Navigate("/") }),
			actionButton("Projects", toneTonal, func() { c.Navigate("/projects") }),
			actionButton("← Back", toneText, func() { c.Router.Back() }),
		).Gap(8),
		routeInspector(c.Match),
	)
}

// --- Row helpers ---

func projectRow(p project, onOpen func()) widget.Widget {
	return primitives.HBox(
		primitives.VBox(
			primitives.Text(p.Name).FontSize(16).Bold().Color(colorInk),
			primitives.Text(p.Description).FontSize(12).Color(colorMuted),
			primitives.Text("id: "+p.ID).FontSize(11).Color(colorMuted),
		).Gap(2),
		primitives.Expanded(primitives.Box()),
		actionButton("Open", toneTonal, onOpen),
	).
		Gap(12).
		Padding(14).
		CrossAlign(primitives.CrossAxisCenter).
		Background(widget.RGBA8(250, 249, 252, 255)).
		Rounded(12).
		BorderStyle(1, colorBorder)
}

func chordRow(ch chord, onOpen func()) widget.Widget {
	return primitives.HBox(
		primitives.VBox(
			primitives.Text(ch.Name).FontSize(15).Bold().Color(colorInk),
			primitives.Text(ch.Notes).FontSize(12).Color(colorMuted),
			primitives.Text("id: "+ch.ID).FontSize(11).Color(colorMuted),
		).Gap(2),
		primitives.Expanded(primitives.Box()),
		actionButton("Open", toneTonal, onOpen),
	).
		Gap(12).
		Padding(14).
		CrossAlign(primitives.CrossAxisCenter).
		Background(widget.RGBA8(250, 249, 252, 255)).
		Rounded(12).
		BorderStyle(1, colorBorder)
}
