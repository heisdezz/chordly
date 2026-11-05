package appui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
)

// NewNewReleasesSection renders the "New releases" row.
func NewNewReleasesSection() widget.Widget {
	header := primitives.HBox(
		primitives.Text("New releases").FontSize(20).Bold().Color(theme.ColorTextPrimary),
		primitives.Expanded(primitives.Box()),
		primitives.Text("See all →").FontSize(13).Color(theme.ColorTextMuted),
	).CrossAlign(primitives.CrossAxisCenter)

	releases := state.NewReleases.Get()
	items := make([]widget.Widget, 0, len(releases))
	for _, track := range releases {
		items = append(items, newReleaseTrackRow(track))
	}

	tracksRow := primitives.HBox(items...).Gap(16)

	return primitives.VBox(
		header,
		primitives.Box().Height(12),
		tracksRow,
	)
}

func newReleaseTrackRow(track model.Track) widget.Widget {
	// Thumbnail Box (46x46)
	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 16),
	).
		PaddingXY(14, 12).
		Background(widget.Hex(0x1B2830)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	// Title & Artist
	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(track.Artist).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	// Duration
	durationText := primitives.Text(track.DurationStr).FontSize(12).Color(theme.ColorTextMuted)

	// Action buttons (Vector Heart & More)
	heartBtn := NewVectorIconButton(IconHeart, 32, 14, func() {
		// Toggle like
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	moreBtn := NewVectorIconButton(IconMore, 32, 14, func() {
		// Options
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorTextMuted
	})

	return primitives.HBox(
		thumb,
		primitives.Box().Width(12),
		meta,
		primitives.Expanded(primitives.Box()),
		durationText,
		primitives.Box().Width(8),
		heartBtn,
		moreBtn,
	).
		PaddingXY(12, 8).
		Width(310).
		Background(widget.Hex(0x121A20)).
		Rounded(14).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}
