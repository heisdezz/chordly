package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

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

// HomePage is the index route.
func HomePage(c router.RouteContext) widget.Widget {
	return page(c.Router, c.Match,
		heading("Chordly", "A sidebar, nested pages, and signal-driven routing."),
		bodyText("Pick a section in the sidebar, or jump into a deep link."),
		primitives.VBox(
			bodyText("Every page comes from the same route tree."),
			bodyText("Parameters captured at any depth stay available through the match."),
		).Gap(4),
		primitives.HBox(
			actionButton("Browse projects", toneFilled, func() { c.Navigate("/projects") }),
			actionButton("Open deep link", toneTonal, func() { c.Navigate("/projects/ambient/chords/verse-1") }),
			actionButton("Trigger 404", toneText, func() { c.Navigate("/missing/page") }),
		).Gap(8),
		routeInspector(c.Match),
	)
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

// SettingsPage is a sibling top-level route.
func SettingsPage(c router.RouteContext) widget.Widget {
	return page(c.Router, c.Match,
		heading("Settings", "A second top-level route, so sidebar highlighting can switch."),
		card(
			keyValue("theme", "light (default)"),
			keyValue("router", "signal location + computed match"),
			keyValue("history", "Back tracks every Navigate call"),
		),
		primitives.HBox(
			actionButton("Back", toneTonal, func() { c.Router.Back() }),
			actionButton("Home", toneText, func() { c.Navigate("/") }),
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
