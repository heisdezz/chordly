package appui

import (
	"sort"
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/router"
)

const (
	sidebarWidth   float32 = 236
	sidebarPadding float32 = 16
)

// Legacy scaffolding helpers used by router pages and inspection.

func page(r *router.Router, m router.Match, body ...widget.Widget) widget.Widget {
	children := make([]widget.Widget, 0, len(body)+2)
	children = append(children, breadcrumbs(r, m))
	children = append(children, primitives.Box().Height(2))
	children = append(children, body...)

	return primitives.VBox(children...).
		Padding(30).
		Gap(14).
		CrossAlign(primitives.CrossAxisStretch).
		Background(colorSurface)
}

func heading(title, subtitle string) widget.Widget {
	return primitives.VBox(
		primitives.Text(title).FontSize(28).Bold().Color(colorInk),
		primitives.Text(subtitle).FontSize(14).Color(colorMuted),
	).Gap(4)
}

func bodyText(text string) widget.Widget {
	return primitives.Text(text).FontSize(13).Color(colorMuted)
}

func card(children ...widget.Widget) *primitives.BoxWidget {
	return primitives.VBox(children...).
		Padding(16).
		Gap(8).
		Background(widget.RGBA8(250, 249, 252, 255)).
		Rounded(12).
		BorderStyle(1, colorBorder)
}

func keyValue(key, value string) widget.Widget {
	return primitives.HBox(
		primitives.Box(primitives.Text(key).FontSize(12).Color(colorMuted)).Width(88),
		primitives.Text(value).FontSize(12).Bold().Color(colorInk),
	).Gap(8).CrossAlign(primitives.CrossAxisCenter)
}

func routeInspector(m router.Match) widget.Widget {
	items := []widget.Widget{
		primitives.Text("Route inspector").FontSize(14).Bold().Color(colorInk),
		keyValue("Path", m.Path),
		keyValue("Pattern", m.Pattern),
		keyValue("Leaf route", m.Name),
	}

	if len(m.Params) > 0 {
		keys := make([]string, 0, len(m.Params))
		for k := range m.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, keyValue("param:"+k, m.Params[k]))
		}
	}

	return card(items...)
}

func breadcrumbs(r *router.Router, m router.Match) widget.Widget {
	items := make([]widget.Widget, 0, len(m.Chain)*2)

	acc := ""
	for i, route := range m.Chain {
		label := route.Name
		if label == "" {
			label = "home"
		}

		if route.Segment != "" {
			if strings.HasPrefix(route.Segment, ":") {
				paramName := route.Segment[1:]
				if val, ok := m.Params[paramName]; ok {
					label = val
					acc += "/" + val
				} else {
					acc += "/" + route.Segment
				}
			} else {
				acc += "/" + route.Segment
			}
		} else {
			acc = "/"
		}

		if i > 0 {
			items = append(items, primitives.Text(" / ").FontSize(12).Color(colorMuted))
		}

		target := acc
		items = append(items, crumbButton(r, label, target))
	}

	return primitives.HBox(items...).CrossAlign(primitives.CrossAxisCenter)
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
