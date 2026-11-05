package appui

import (
	"context"
	"fmt"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	"chordly/internal/theme"
	"chordly/internal/ytm"
	"chordly/router"
)

// ArtistPage renders the rich detail view for an artist (/artist/:id).
func ArtistPage(c router.RouteContext) widget.Widget {
	artistID := c.Param("id")
	if artistID == "" {
		artistID = "unknown"
	}

	artistSig := state.NewSignal[*ytm.ArtistDetails](nil)
	loadingSig := state.NewSignal(true)
	errSig := state.NewSignal("")

	// Asynchronously fetch artist details
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		details, err := ytm.DefaultService.GetArtist(ctx, artistID)
		if err != nil {
			errSig.Set(fmt.Sprintf("Failed to load artist: %v", err))
			loadingSig.Set(false)
			return
		}
		artistSig.Set(details)
		loadingSig.Set(false)
	}()

	return newReactiveView(
		func() widget.Widget {
			if loadingSig.Get() {
				return NewPageScaffold(PageScaffoldConfig{
					Title:      "Loading Artist...",
					Scrollable: false,
					Body: primitives.VBox(
						primitives.Box().Height(40),
						NewSpinner(36),
						primitives.Box().Height(16),
						primitives.Text("Fetching discography from YouTube Music...").
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

			artist := artistSig.Get()
			if artist == nil {
				return primitives.Box()
			}

			return buildArtistDetailView(c, artist)
		},
		WatchSignal(artistSig.AsReadonly()),
		WatchSignal(loadingSig.AsReadonly()),
		WatchSignal(errSig.AsReadonly()),
	)
}

func buildArtistDetailView(c router.RouteContext, artist *ytm.ArtistDetails) widget.Widget {
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

	// Circular Avatar with real thumbnail
	avatarBox := NewCircularThumbnail(artist.Thumbnail, 120, artist.Name)

	// Artist Meta
	badge := primitives.Text("VERIFIED ARTIST").
		FontSize(11).
		Bold().
		Color(palette.Accent)

	name := primitives.Text(artist.Name).
		FontSize(32).
		Bold().
		Color(palette.TextPrimary)

	subscribers := primitives.Text(artist.SubscriberCount).
		FontSize(13).
		Color(palette.TextSecondary)

	// Play Top Songs & Shuffle Actions
	playAllBtn := button.New(
		button.Text("Play Top Songs"),
		button.OnClick(func() {
			playAlbumTracks(artist.TopSongs, 0)
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
			shuffleAlbumTracks(artist.TopSongs)
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
		name,
		subscribers,
		primitives.Box().Height(10),
		actionRow,
	).Gap(4)

	heroSection := primitives.HBox(
		avatarBox,
		primitives.Box().Width(24),
		primitives.Expanded(headerMeta),
	).CrossAlign(primitives.CrossAxisCenter)

	// Top Songs Section
	topSongsHeader := primitives.Text("Top Songs").
		FontSize(20).
		Bold().
		Color(palette.TextPrimary)

	topSongRows := make([]widget.Widget, 0, len(artist.TopSongs))
	for i, tr := range artist.TopSongs {
		index := i
		track := tr
		topSongRows = append(topSongRows, buildAlbumTrackRow(index, track, artist.TopSongs))
	}

	topSongsSection := primitives.VBox(
		topSongsHeader,
		primitives.Box().Height(10),
		primitives.VBox(topSongRows...).Gap(6),
	)

	// Albums & Discography Section
	var discographySection widget.Widget
	allReleases := append(artist.Albums, artist.Singles...)
	if len(allReleases) > 0 {
		discogHeader := primitives.Text("Albums & Singles").
			FontSize(20).
			Bold().
			Color(palette.TextPrimary)

		cards := make([]widget.Widget, 0, len(allReleases))
		for _, card := range allReleases {
			cards = append(cards, buildArtistReleaseCard(card))
		}

		discographySection = primitives.VBox(
			discogHeader,
			primitives.Box().Height(10),
			primitives.HBox(cards...).Gap(14),
		)
	}

	// About Section
	var aboutSection widget.Widget
	if artist.Description != "" {
		aboutHeader := primitives.Text("About").
			FontSize(20).
			Bold().
			Color(palette.TextPrimary)

		aboutText := primitives.Text(artist.Description).
			FontSize(13).
			Color(palette.TextSecondary)

		aboutBox := primitives.VBox(
			aboutHeader,
			primitives.Box().Height(8),
			aboutText,
		).
			Padding(18).
			Background(palette.Surface).
			Rounded(14).
			BorderStyle(1, palette.Border)

		aboutSection = aboutBox
	}

	elements := []widget.Widget{
		backBtn,
		primitives.Box().Height(14),
		heroSection,
		primitives.Box().Height(28),
		topSongsSection,
	}

	if discographySection != nil {
		elements = append(elements, primitives.Box().Height(28), discographySection)
	}
	if aboutSection != nil {
		elements = append(elements, primitives.Box().Height(28), aboutSection)
	}

	return NewPageScaffold(PageScaffoldConfig{
		Scrollable:    true,
		PaddingBottom: 110,
		Body:          primitives.VBox(elements...),
	})
}

func buildArtistReleaseCard(card model.MixCard) widget.Widget {
	palette := theme.ActivePalette()

	thumb := NewThumbnailWidget(card.Thumbnail, 136, 12, card.Title)

	meta := primitives.VBox(
		primitives.Text(card.Title).FontSize(14).Bold().Color(palette.TextPrimary),
		primitives.Text(card.Subtitle).FontSize(12).Color(palette.TextSecondary),
	).Gap(2)

	cardContent := primitives.VBox(
		thumb,
		primitives.Box().Height(10),
		meta,
	).
		Width(164).
		Padding(14).
		Background(palette.Surface).
		Rounded(16).
		BorderStyle(1, palette.Border).
		CrossAlign(primitives.CrossAxisCenter)

	return button.New(
		button.Text(""),
		button.OnClick(func() {
			Navigate("/album/" + card.ID)
		}),
		button.PainterOpt(wrapperCardPainter{child: cardContent}),
	)
}

type wrapperCardPainter struct {
	child widget.Widget
}

func (p wrapperCardPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() || p.child == nil {
		return
	}
	palette := theme.ActivePalette()
	bg := palette.Surface
	if s.Hovered {
		bg = palette.SurfaceLight
	}
	canvas.DrawRoundRect(s.Bounds, bg, 16)
	canvas.StrokeRoundRect(s.Bounds, palette.Border, 16, 1)
}

func (p wrapperCardPainter) ButtonHeight(button.Size) float32             { return 205 }
func (p wrapperCardPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p wrapperCardPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p wrapperCardPainter) ButtonRadius() float32                        { return 16 }
