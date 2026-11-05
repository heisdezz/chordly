package appui

import (
	"context"
	"fmt"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
	"chordly/internal/ytm"
	"chordly/router"
)

// PlaylistPage renders the detail view for a playlist (/playlist/:id).
func PlaylistPage(c router.RouteContext) widget.Widget {
	playlistID := c.Param("id")
	if playlistID == "" {
		playlistID = "unknown"
	}

	return renderPlaylistView(c, playlistID, "PLAYLIST", false)
}

// MixPage renders the detail view for an endless mix or radio (/mix/:id).
func MixPage(c router.RouteContext) widget.Widget {
	mixID := c.Param("id")
	if mixID == "" {
		mixID = "unknown"
	}

	return renderPlaylistView(c, mixID, "MADE FOR YOU • MIX", true)
}

func renderPlaylistView(c router.RouteContext, id, badgeText string, isMix bool) widget.Widget {
	plSig := state.NewSignal[*ytm.PlaylistDetails](nil)
	loadingSig := state.NewSignal(true)
	errSig := state.NewSignal("")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		details, err := ytm.DefaultService.GetPlaylist(ctx, id)
		if err != nil {
			errSig.Set(fmt.Sprintf("Failed to load playlist: %v", err))
			loadingSig.Set(false)
			return
		}
		plSig.Set(details)
		loadingSig.Set(false)
	}()

	return newReactiveView(
		func() widget.Widget {
			if loadingSig.Get() {
				title := "Loading Playlist..."
				if isMix {
					title = "Loading Mix..."
				}
				return NewPageScaffold(PageScaffoldConfig{
					Title:      title,
					Scrollable: false,
					Body: primitives.VBox(
						primitives.Box().Height(40),
						NewSpinner(36),
						primitives.Box().Height(16),
						primitives.Text("Fetching tracks from YouTube Music...").
							FontSize(14).
							Color(theme.ColorTextMuted),
					).CrossAlign(primitives.CrossAxisCenter),
				})
			}

			if errMsg := errSig.Get(); errMsg != "" {
				return NewPageScaffold(PageScaffoldConfig{
					Title:      "Error",
					Scrollable: false,
					Body: primitives.VBox(
						primitives.Text(errMsg).FontSize(14).Color(theme.ColorHeartActive),
						primitives.Box().Height(12),
						actionButton("← Back", toneTonal, func() {
							if !c.Router.Back() {
								c.Router.Replace("/")
							}
						}),
					),
				})
			}

			pl := plSig.Get()
			if pl == nil {
				return primitives.Box()
			}

			return buildPlaylistDetailContent(c, pl, badgeText, isMix)
		},
		WatchSignal(plSig.AsReadonly()),
		WatchSignal(loadingSig.AsReadonly()),
		WatchSignal(errSig.AsReadonly()),
	)
}

func buildPlaylistDetailContent(c router.RouteContext, pl *ytm.PlaylistDetails, badgeText string, isMix bool) widget.Widget {
	palette := theme.ActivePalette()

	// Back button
	backBtn := NewVectorIconButton(IconPrev, 32, 14, func() {
		if !c.Router.Back() {
			c.Router.Replace("/")
		}
	}, func(s *IconButtonStyle) {
		s.Background = palette.Surface
		s.BorderWidth = 1
		s.BorderColor = palette.Border
		s.Foreground = palette.TextPrimary
	})

	// Artwork box (140x140)
	iconType := IconQueue
	if isMix {
		iconType = IconWaveform
	}
	artBox := primitives.HBox(
		NewVectorIcon(iconType, palette.Accent, 48),
	).
		Width(140).
		Height(140).
		Background(widget.Hex(0x16232B)).
		Rounded(16).
		BorderStyle(1, palette.Border).
		CrossAlign(primitives.CrossAxisCenter)

	// Badge & Title
	badge := primitives.Text(badgeText).
		FontSize(11).
		Bold().
		Color(palette.Accent)

	title := primitives.Text(pl.Title).
		FontSize(28).
		Bold().
		Color(palette.TextPrimary)

	subInfo := fmt.Sprintf("%s • %d songs", pl.Author, pl.TrackCount)
	if isMix {
		subInfo += " • Endless radio enabled"
	}
	subText := primitives.Text(subInfo).
		FontSize(13).
		Color(palette.TextSecondary)

	// Play All & Shuffle Buttons
	playAllBtn := button.New(
		button.Text("Play All"),
		button.OnClick(func() {
			playAlbumTracks(pl.Tracks, 0)
			if isMix && len(pl.Tracks) > 0 {
				ytm.DefaultService.StartRadioForTrack(context.Background(), pl.Tracks[0].ID)
			}
		}),
		button.PainterOpt(primaryActionPainter{
			bg:   palette.Accent,
			fg:   palette.OnAccent,
			icon: IconPlay,
		}),
	)

	shuffleBtn := button.New(
		button.Text("Shuffle"),
		button.OnClick(func() {
			shuffleAlbumTracks(pl.Tracks)
		}),
		button.PainterOpt(secondaryActionPainter{
			bg:   palette.Surface,
			fg:   palette.TextPrimary,
			icon: IconShuffle,
		}),
	)

	actionRow := primitives.HBox(
		playAllBtn,
		primitives.Box().Width(10),
		shuffleBtn,
	).CrossAlign(primitives.CrossAxisCenter)

	headerMeta := primitives.VBox(
		badge,
		title,
		subText,
		primitives.Box().Height(10),
		actionRow,
	).Gap(4)

	heroSection := primitives.HBox(
		artBox,
		primitives.Box().Width(20),
		primitives.Expanded(headerMeta),
	).CrossAlign(primitives.CrossAxisCenter)

	// Tracklist Header
	tableHeader := primitives.HBox(
		primitives.Text("#").FontSize(12).Bold().Color(palette.TextMuted),
		primitives.Box().Width(18),
		primitives.Expanded(primitives.Text("TITLE").FontSize(12).Bold().Color(palette.TextMuted)),
		primitives.Text("DURATION").FontSize(12).Bold().Color(palette.TextMuted),
		primitives.Box().Width(80),
	).PaddingXY(14, 8)

	// Track Rows
	trackRows := make([]widget.Widget, 0, len(pl.Tracks))
	for i, tr := range pl.Tracks {
		index := i
		track := tr
		trackRows = append(trackRows, buildAlbumTrackRow(index, track, pl.Tracks))
	}

	trackListContainer := primitives.VBox(trackRows...).Gap(6)

	body := primitives.VBox(
		backBtn,
		primitives.Box().Height(14),
		heroSection,
		primitives.Box().Height(24),
		tableHeader,
		primitives.Box().Height(6),
		trackListContainer,
	)

	return NewPageScaffold(PageScaffoldConfig{
		Scrollable:    true,
		PaddingBottom: 110,
		Body:          body,
	})
}
