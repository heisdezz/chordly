package appui

import (
	"github.com/gogpu/ui/core/dialog"
	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// PageScaffoldConfig defines options for a scrollable page layout.
type PageScaffoldConfig struct {
	Title         string
	Subtitle      string
	Actions       widget.Widget
	Body          widget.Widget
	Scrollable    bool
	PaddingX      float32
	PaddingTop    float32
	PaddingBottom float32
}

// DefaultPageScaffoldConfig returns defaults suitable for Metrolist pages.
func DefaultPageScaffoldConfig() PageScaffoldConfig {
	return PageScaffoldConfig{
		Scrollable:    true,
		PaddingX:      24,
		PaddingTop:    20,
		PaddingBottom: 96, // clearance for persistent bottom player dock
	}
}

// NewPageScaffold wraps a page header and content in a scrollable container
// with responsive padding, header actions, and theme styling.
func NewPageScaffold(cfg PageScaffoldConfig) widget.Widget {
	if cfg.PaddingX <= 0 {
		cfg.PaddingX = 24
	}
	if cfg.PaddingTop <= 0 {
		cfg.PaddingTop = 20
	}
	if cfg.PaddingBottom <= 0 {
		cfg.PaddingBottom = 96
	}

	palette := theme.ActivePalette()
	var elements []widget.Widget

	// Header row
	if cfg.Title != "" || cfg.Subtitle != "" || cfg.Actions != nil {
		var titleCol []widget.Widget
		if cfg.Title != "" {
			t := primitives.Text(cfg.Title).
				FontSize(28).
				Color(palette.TextPrimary)
			titleCol = append(titleCol, t)
		}
		if cfg.Subtitle != "" {
			st := primitives.Text(cfg.Subtitle).
				FontSize(14).
				Color(palette.TextSecondary)
			titleCol = append(titleCol, st)
		}

		headerLeft := primitives.VBox(titleCol...).Gap(4)

		var headerRow widget.Widget
		if cfg.Actions != nil {
			headerRow = primitives.HBox(
				headerLeft,
				primitives.Expanded(primitives.Box()),
				cfg.Actions,
			).CrossAlign(primitives.CrossAxisCenter)
		} else {
			headerRow = headerLeft
		}

		elements = append(elements, headerRow)
	}

	// Body
	if cfg.Body != nil {
		elements = append(elements, cfg.Body)
	}

	pageContent := primitives.VBox(elements...).
		Gap(20).
		PaddingXY(cfg.PaddingX, 0).
		PaddingTop(cfg.PaddingTop).
		PaddingBottom(cfg.PaddingBottom)

	if !cfg.Scrollable {
		return pageContent
	}

	// Wrap in core/scrollview for modern scrolling
	sv := scrollview.New(
		pageContent,
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
	)

	return sv
}

// NewSimplePageScaffold creates a standard scrollable page scaffold with title and body.
func NewSimplePageScaffold(title string, body widget.Widget) widget.Widget {
	cfg := DefaultPageScaffoldConfig()
	cfg.Title = title
	cfg.Body = body
	return NewPageScaffold(cfg)
}

// NewScrollableBody wraps content in a vertical scrollview with standard bottom clearance.
func NewScrollableBody(content widget.Widget) widget.Widget {
	padded := primitives.VBox(content).
		PaddingXY(20, 0).
		PaddingTop(10).
		PaddingBottom(96)
	return scrollview.New(
		padded,
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
	)
}

// NewListView creates a virtualized vertical list using GoGPU core/listview.
func NewListView(itemCount int, itemHeight float32, buildItem func(index int) widget.Widget) widget.Widget {
	return listview.New(
		listview.ItemCount(itemCount),
		listview.FixedItemHeight(itemHeight),
		listview.BuildItem(func(ctx listview.ItemContext) widget.Widget {
			return buildItem(ctx.Index)
		}),
	)
}

// NewAlertDialog creates a themed alert dialog using core/dialog.
func NewAlertDialog(title, message string, onDismiss func()) *dialog.Widget {
	return dialog.Alert(title, message, onDismiss)
}

// NewConfirmDialog creates a themed confirmation dialog using core/dialog.
func NewConfirmDialog(title, message string, onCancel, onConfirm func()) *dialog.Widget {
	return dialog.Confirm(title, message, onCancel, onConfirm)
}
