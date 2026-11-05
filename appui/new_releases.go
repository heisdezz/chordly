package appui

import (
	"context"
	"log"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
)

// NewNewReleasesSection renders the "New releases" row. It rebuilds when the
// live home feed replaces the release list.
func NewNewReleasesSection() widget.Widget {
	return newReactiveView(
		buildNewReleasesContent,
		WatchSignal(state.NewReleases.AsReadonly()),
	)
}

func buildNewReleasesContent() widget.Widget {
	header := primitives.HBox(
		primitives.Text("New releases").FontSize(20).Bold().Color(theme.ColorTextPrimary),
		primitives.Expanded(primitives.Box()),
		primitives.Text("See all →").FontSize(13).Color(theme.ColorTextMuted),
	).CrossAlign(primitives.CrossAxisCenter)

	releases := state.NewReleases.Get()
	rows := make([]widget.Widget, 0, (len(releases)+1)/2)
	for i := 0; i < len(releases); i += 2 {
		cells := []widget.Widget{primitives.Expanded(newReleaseTrackRow(releases[i]))}
		if i+1 < len(releases) {
			cells = append(cells,
				primitives.Box().Width(12),
				primitives.Expanded(newReleaseTrackRow(releases[i+1])),
			)
		}
		rows = append(rows, primitives.HBox(cells...))
	}

	tracksGrid := primitives.VBox(rows...).Gap(10)

	return primitives.VBox(
		header,
		primitives.Box().Height(12),
		tracksGrid,
	)
}

func newReleaseTrackRow(track model.Track) widget.Widget {
	// Thumbnail Box with real artwork
	thumb := NewThumbnailWidget(track.Thumbnail, 46, 10, track.Title)

	// Title & Artist
	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(track.Artist).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	// Duration
	durationText := primitives.Text(track.DurationStr).FontSize(12).Color(theme.ColorTextMuted)

	// Play button: starts playback directly without unintended album navigation
	playBtn := NewVectorIconButton(IconPlay, 32, 14, func() {
		releases := state.NewReleases.Get()
		if len(releases) > 0 {
			state.Queue.Set(releases)
			for i, r := range releases {
				if r.ID == track.ID {
					state.QueueIndex.Set(i)
					break
				}
			}
		}
		state.PlaySpecificTrack(track)

		// Pre-resolve stream in background if URL is missing
		if track.StreamURL == "" {
			go func(t model.Track) {
				resCtx, resCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer resCancel()
				if err := ytm.DefaultService.PrepareTrackStream(resCtx, &t); err == nil {
					if state.CurrentTrack.Get().ID == t.ID {
						state.CurrentTrack.Set(t)
					}
				} else {
					log.Printf("[UI] Stream resolve failed for %q: %v", t.Title, err)
				}
			}(track)
		}
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		s.Foreground = theme.ColorAccent
	})

	// Action buttons (Vector Heart & More)
	heartBtn := NewVectorIconButton(IconHeart, 32, 14, func() {
		track.IsLiked = !track.IsLiked
		if track.IsLiked {
			ShowSuccess("Added to Liked Songs")
		} else {
			ShowInfo("Removed from Liked Songs")
		}
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.BorderWidth = 0
		if track.IsLiked {
			s.Foreground = theme.ColorHeartActive
		} else {
			s.Foreground = theme.ColorTextMuted
		}
	})

	moreBtn := NewVectorIconButton(IconMore, 32, 14, func() {
		if track.Artist != "" {
			artistID := track.ArtistID
			if artistID == "" {
				artistID = track.Artist
			}
			Navigate("/artist/" + artistID)
		}
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
		playBtn,
		primitives.Box().Width(4),
		heartBtn,
		moreBtn,
	).
		PaddingXY(12, 8).
		Background(theme.ColorSurface).
		Rounded(14).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}
