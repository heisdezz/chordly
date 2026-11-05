package appui

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/model"
	intstate "chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
	"chordly/router"
)

// AlbumPage renders the detail view for an album (/album/:id).
func AlbumPage(c router.RouteContext) widget.Widget {
	albumID := c.Param("id")
	if albumID == "" {
		albumID = "unknown"
	}

	albumSig := state.NewSignal[*ytm.AlbumDetails](nil)
	loadingSig := state.NewSignal(true)
	errSig := state.NewSignal("")

	// Asynchronously fetch album details
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		details, err := ytm.DefaultService.GetAlbum(ctx, albumID)
		if err != nil {
			errSig.Set(fmt.Sprintf("Failed to load album: %v", err))
			loadingSig.Set(false)
			return
		}
		albumSig.Set(details)
		loadingSig.Set(false)
	}()

	return newReactiveView(
		func() widget.Widget {
			if loadingSig.Get() {
				return NewPageScaffold(PageScaffoldConfig{
					Title:      "Loading Album...",
					Scrollable: false,
					Body: primitives.VBox(
						primitives.Box().Height(40),
						NewSpinner(36),
						primitives.Box().Height(16),
						primitives.Text("Fetching tracklist from YouTube Music...").
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

			album := albumSig.Get()
			if album == nil {
				return primitives.Box()
			}

			return buildAlbumDetailView(c, album)
		},
		WatchSignal(albumSig.AsReadonly()),
		WatchSignal(loadingSig.AsReadonly()),
		WatchSignal(errSig.AsReadonly()),
	)
}

func buildAlbumDetailView(c router.RouteContext, album *ytm.AlbumDetails) widget.Widget {
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

	// Album Artwork Box (140x140)
	artBox := primitives.HBox(
		NewVectorIcon(IconWaveform, palette.Accent, 48),
	).
		Width(140).
		Height(140).
		Background(widget.Hex(0x142028)).
		Rounded(16).
		BorderStyle(1, palette.Border).
		CrossAlign(primitives.CrossAxisCenter)

	// Album Badge & Title
	badge := primitives.Text("ALBUM").
		FontSize(11).
		Bold().
		Color(palette.Accent)

	title := primitives.Text(album.Title).
		FontSize(28).
		Bold().
		Color(palette.TextPrimary)

	// Artist Link / Subtitle
	artistLink := button.New(
		button.Text(album.Artist),
		button.OnClick(func() {
			artistID := album.ArtistID
			if artistID == "" {
				artistID = album.Artist
			}
			Navigate("/artist/" + artistID)
		}),
		button.PainterOpt(subtleLinkPainter{color: palette.Accent}),
	)

	subInfo := fmt.Sprintf("• %s • %d songs", album.Year, album.TrackCount)
	if album.DurationStr != "" {
		subInfo += " • " + album.DurationStr
	}
	subText := primitives.Text(subInfo).
		FontSize(13).
		Color(palette.TextSecondary)

	metaRow := primitives.HBox(
		artistLink,
		primitives.Box().Width(6),
		subText,
	).CrossAlign(primitives.CrossAxisCenter)

	// Play All & Shuffle Buttons
	playAllBtn := button.New(
		button.Text("Play All"),
		button.OnClick(func() {
			playAlbumTracks(album.Tracks, 0)
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
			shuffleAlbumTracks(album.Tracks)
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
		metaRow,
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
	trackRows := make([]widget.Widget, 0, len(album.Tracks))
	for i, tr := range album.Tracks {
		index := i
		track := tr
		trackRows = append(trackRows, buildAlbumTrackRow(index, track, album.Tracks))
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

func buildAlbumTrackRow(index int, track model.Track, allTracks []model.Track) widget.Widget {
	palette := theme.ActivePalette()

	indexLabel := primitives.Text(fmt.Sprintf("%d", index+1)).
		FontSize(13).
		Color(palette.TextMuted)

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(palette.TextPrimary),
		primitives.Text(track.Artist).FontSize(12).Color(palette.TextSecondary),
	).Gap(2)

	durationText := primitives.Text(track.DurationStr).
		FontSize(12).
		Color(palette.TextMuted)

	playBtn := NewVectorIconButton(IconPlay, 30, 13, func() {
		playAlbumTracks(allTracks, index)
	}, func(s *IconButtonStyle) {
		s.Background = palette.SurfaceLight
		s.Foreground = palette.Accent
		s.BorderWidth = 0
	})

	heartBtn := NewVectorIconButton(IconHeart, 30, 13, func() {
		track.IsLiked = !track.IsLiked
		if track.IsLiked {
			ShowSuccess("Added to Liked Songs")
		} else {
			ShowInfo("Removed from Liked Songs")
		}
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.Foreground = palette.TextMuted
		s.BorderWidth = 0
	})

	moreBtn := NewVectorIconButton(IconMore, 30, 13, func() {
		curQueue := intstate.Queue.Get()
		intstate.Queue.Set(append(curQueue, track))
		ShowSuccess("Added to queue: " + track.Title)
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.Foreground = palette.TextMuted
		s.BorderWidth = 0
	})

	return primitives.HBox(
		indexLabel,
		primitives.Box().Width(16),
		primitives.Expanded(meta),
		durationText,
		primitives.Box().Width(12),
		playBtn,
		primitives.Box().Width(4),
		heartBtn,
		moreBtn,
	).
		PaddingXY(14, 10).
		Background(palette.Surface).
		Rounded(12).
		BorderStyle(1, palette.Border).
		CrossAlign(primitives.CrossAxisCenter)
}

func playAlbumTracks(tracks []model.Track, startIdx int) {
	if len(tracks) == 0 || startIdx < 0 || startIdx >= len(tracks) {
		return
	}
	intstate.Queue.Set(tracks)
	intstate.QueueIndex.Set(startIdx)
	intstate.CurrentTrack.Set(tracks[startIdx])
	intstate.Playhead.Set(0)
	intstate.PlaybackStatus.Set(model.StatePlaying)
}

func shuffleAlbumTracks(tracks []model.Track) {
	if len(tracks) == 0 {
		return
	}
	shuffled := make([]model.Track, len(tracks))
	copy(shuffled, tracks)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	playAlbumTracks(shuffled, 0)
}

// Action button painters
type primaryActionPainter struct {
	bg   widget.Color
	fg   widget.Color
	icon IconType
}

func (p primaryActionPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}
	bg := p.bg
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.12)
	}
	canvas.DrawRoundRect(s.Bounds, bg, 18)

	// Draw icon and label
	iconRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+14, s.Bounds.Min.Y+(s.Bounds.Height()-14)/2),
		Max: geometry.Pt(s.Bounds.Min.X+28, s.Bounds.Min.Y+(s.Bounds.Height()+14)/2),
	}
	DrawVectorIcon(canvas, p.icon, iconRect, p.fg, 12)

	textRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+34, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Max.X-10, s.Bounds.Max.Y),
	}
	canvas.DrawText(s.Text, textRect, 13, p.fg, true, widget.TextAlignLeft)
}

func (p primaryActionPainter) ButtonHeight(button.Size) float32             { return 36 }
func (p primaryActionPainter) ButtonPadding(button.Size) (float32, float32) { return 14, 0 }
func (p primaryActionPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p primaryActionPainter) ButtonRadius() float32                        { return 18 }

type secondaryActionPainter struct {
	bg   widget.Color
	fg   widget.Color
	icon IconType
}

func (p secondaryActionPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}
	bg := p.bg
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.08)
	}
	canvas.DrawRoundRect(s.Bounds, bg, 18)
	canvas.StrokeRoundRect(s.Bounds, theme.ActivePalette().Border, 18, 1)

	iconRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+14, s.Bounds.Min.Y+(s.Bounds.Height()-14)/2),
		Max: geometry.Pt(s.Bounds.Min.X+28, s.Bounds.Min.Y+(s.Bounds.Height()+14)/2),
	}
	DrawVectorIcon(canvas, p.icon, iconRect, p.fg, 12)

	textRect := geometry.Rect{
		Min: geometry.Pt(s.Bounds.Min.X+34, s.Bounds.Min.Y),
		Max: geometry.Pt(s.Bounds.Max.X-10, s.Bounds.Max.Y),
	}
	canvas.DrawText(s.Text, textRect, 13, p.fg, true, widget.TextAlignLeft)
}

func (p secondaryActionPainter) ButtonHeight(button.Size) float32             { return 36 }
func (p secondaryActionPainter) ButtonPadding(button.Size) (float32, float32) { return 14, 0 }
func (p secondaryActionPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p secondaryActionPainter) ButtonRadius() float32                        { return 18 }

type subtleLinkPainter struct {
	color widget.Color
}

func (p subtleLinkPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}
	col := p.color
	if s.Hovered {
		col = col.Lerp(widget.ColorWhite, 0.2)
	}
	canvas.DrawText(s.Text, s.Bounds, 13, col, true, widget.TextAlignLeft)
}

func (p subtleLinkPainter) ButtonHeight(button.Size) float32             { return 20 }
func (p subtleLinkPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p subtleLinkPainter) ButtonFontSize(button.Size) float32           { return 13 }
func (p subtleLinkPainter) ButtonRadius() float32                        { return 0 }
