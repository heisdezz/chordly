package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
)

// NewUpNextPanel renders the "Up next" queue panel on the right.
func NewUpNextPanel() widget.Widget {
	header := primitives.HBox(
		primitives.Text("Up next").FontSize(18).Bold().Color(theme.ColorTextPrimary),
		primitives.Expanded(primitives.Box()),
		NewVectorIcon(IconQueue, theme.ColorTextMuted, 16),
	).CrossAlign(primitives.CrossAxisCenter)

	queue := state.Queue.Get()
	currentIdx := state.QueueIndex.Get()

	items := make([]widget.Widget, 0, len(queue))
	for i, track := range queue {
		isActive := i == currentIdx
		items = append(items, newQueueRow(track, isActive))
	}

	queueList := primitives.VBox(items...).Gap(8)

	return primitives.VBox(
		header,
		primitives.Box().Height(14),
		queueList,
	).
		PaddingXY(18, 16).
		Width(260).
		Background(widget.Hex(0x10171D)).
		Rounded(20).
		BorderStyle(1, theme.ColorBorder)
}

func newQueueRow(track model.Track, isActive bool) widget.Widget {
	// Thumbnail (38x38)
	var thumbColor widget.Color = widget.Hex(0x1D2A33)
	if isActive {
		thumbColor = widget.Hex(0x183C34)
	}

	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 14),
	).
		PaddingXY(12, 10).
		Background(thumbColor).
		Rounded(8)

	// Play indicator for active track (Vector Play Triangle)
	var leadingIndicator widget.Widget
	if isActive {
		leadingIndicator = NewVectorIcon(IconPlay, theme.ColorAccent, 11)
	} else {
		leadingIndicator = primitives.Box().Width(0)
	}

	// Title & Artist
	titleColor := theme.ColorTextPrimary
	if !isActive {
		titleColor = theme.ColorTextPrimary.WithAlpha(0.9)
	}

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(13).Bold().Color(titleColor),
		primitives.Text(track.Artist).FontSize(11).Color(theme.ColorTextMuted),
	).Gap(2)

	// Duration (mint if active, muted if inactive)
	durColor := theme.ColorTextMuted
	if isActive {
		durColor = theme.ColorAccent
	}
	durText := primitives.Text(track.DurationStr).FontSize(11).Color(durColor)

	row := primitives.HBox(
		leadingIndicator,
		thumb,
		primitives.Box().Width(10),
		meta,
		primitives.Expanded(primitives.Box()),
		durText,
	).
		PaddingXY(10, 8).
		Rounded(12).
		CrossAlign(primitives.CrossAxisCenter)

	if isActive {
		row.Background(widget.RGBA8(92, 225, 182, 30)).
			BorderStyle(1, widget.RGBA8(92, 225, 182, 60))
	} else {
		row.Background(widget.ColorTransparent)
	}

	return row
}
