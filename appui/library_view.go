package appui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/cache"
	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

// LibraryPage renders the primary user catalog page with tab-based views:
// Favorites, Playlists, History, and Downloads.
func LibraryPage(c router.RouteContext) widget.Widget {
	// Sync active tab signal from route path if provided (e.g. /library/downloads)
	if c.Match.Path != "" {
		path := strings.ToLower(c.Match.Path)
		switch {
		case strings.HasSuffix(path, "/playlists"):
			SigLibraryTab.Set(LibraryTabPlaylists)
		case strings.HasSuffix(path, "/history"):
			SigLibraryTab.Set(LibraryTabHistory)
		case strings.HasSuffix(path, "/downloads"):
			SigLibraryTab.Set(LibraryTabDownloads)
		case strings.HasSuffix(path, "/favorites"):
			SigLibraryTab.Set(LibraryTabFavorites)
		}
	}

	return newReactiveView(
		func() widget.Widget {
			return renderLibraryView(c.Router)
		},
		WatchSignal(SigLibraryTab.AsReadonly()),
		WatchSignal(state.UserAccount.AsReadonly()),
		WatchSignal(ActiveSession.AsReadonly()),
		WatchSignal(theme.SigActivePalette.AsReadonly()),
	)
}

func isUserAuthenticated() bool {
	return state.UserAccount.Get() != "" || ActiveSession.Get() != nil
}

func renderLibraryView(r *router.Router) widget.Widget {
	tabBar := buildLibraryTabBar(r)
	activeTab := SigLibraryTab.Get()

	db, _ := cache.GetDefaultDB()
	dc, _ := cache.GetDefaultCache()

	var content widget.Widget
	switch activeTab {
	case LibraryTabFavorites:
		content = renderLibraryFavorites(r, db)
	case LibraryTabPlaylists:
		content = renderLibraryPlaylists(r, db)
	case LibraryTabHistory:
		content = renderLibraryHistory(r, db)
	case LibraryTabDownloads:
		content = renderLibraryDownloads(r, dc, db)
	default:
		content = renderLibraryFavorites(r, db)
	}

	pageLayout := primitives.VBox(
		tabBar,
		primitives.Box().Height(16),
		content,
	)

	cfg := DefaultPageScaffoldConfig()
	cfg.Title = "Library"
	cfg.Subtitle = "Your personal collection of tracks, playlists, and offline audio."
	cfg.Body = pageLayout

	return NewPageScaffold(cfg)
}

func renderLibraryFavorites(r *router.Router, db *cache.DB) widget.Widget {
	if !isUserAuthenticated() {
		return renderLibraryGuestCTA(
			r,
			"Favorites",
			"Sign in with your YouTube Music account to view and synchronize your liked tracks, favorite albums, and top artists across all your devices.",
		)
	}

	if db == nil {
		return renderEmptyState(
			"Database Unavailable",
			"Could not connect to the local storage database.",
			"Retry",
			func() { SigLibraryTab.Set(LibraryTabFavorites) },
		)
	}

	items, err := db.ListFavorites(context.Background(), "")
	if err != nil || len(items) == 0 {
		return renderEmptyState(
			"No Favorites Yet",
			"Tracks and albums you like will appear here for instant offline and online access.",
			"Discover Music",
			func() {
				if r != nil {
					r.Navigate("/")
				}
			},
		)
	}

	rows := []widget.Widget{
		primitives.HBox(
			primitives.Text(fmt.Sprintf("%d Saved Items", len(items))).
				FontSize(14).
				Color(theme.ColorTextSecondary),
		),
		primitives.Box().Height(12),
	}

	for i, item := range items {
		idx := i
		track := model.Track{
			ID:          item.ID,
			Title:       item.Title,
			Artist:      item.Artist,
			Duration:    time.Duration(item.DurationSec) * time.Second,
			DurationStr: fmt.Sprintf("%d:%02d", item.DurationSec/60, item.DurationSec%60),
			Thumbnail:   item.ThumbnailURL,
		}
		rows = append(rows,
			buildLibraryTrackRow(idx, track, item.Album, func() {
				state.PlaySpecificTrack(track)
			}),
			primitives.Box().Height(8),
		)
	}

	return primitives.VBox(rows...)
}

func renderLibraryPlaylists(r *router.Router, db *cache.DB) widget.Widget {
	if !isUserAuthenticated() {
		return renderLibraryGuestCTA(
			r,
			"Playlists",
			"Sign in with your YouTube Music account to view your YouTube playlists, cloud mixes, and custom collections.",
		)
	}

	if db == nil {
		return renderEmptyState(
			"Database Unavailable",
			"Could not connect to the local storage database.",
			"Retry",
			func() { SigLibraryTab.Set(LibraryTabPlaylists) },
		)
	}

	playlists, err := db.ListPlaylists(context.Background())
	if err != nil || len(playlists) == 0 {
		return renderEmptyState(
			"No Playlists Found",
			"Create custom playlists or import your existing YouTube Music library.",
			"Browse Explore",
			func() {
				if r != nil {
					r.Navigate("/explore")
				}
			},
		)
	}

	rows := []widget.Widget{
		primitives.HBox(
			primitives.Text(fmt.Sprintf("%d Playlists", len(playlists))).
				FontSize(14).
				Color(theme.ColorTextSecondary),
		),
		primitives.Box().Height(12),
	}

	for i, pl := range playlists {
		idx := i
		playlist := pl
		rows = append(rows,
			buildPlaylistRow(idx, playlist, func() {
				if r != nil {
					r.Navigate("/playlist/" + playlist.ID)
				}
			}),
			primitives.Box().Height(8),
		)
	}

	return primitives.VBox(rows...)
}

func renderLibraryHistory(r *router.Router, db *cache.DB) widget.Widget {
	if !isUserAuthenticated() {
		return renderLibraryGuestCTA(
			r,
			"Listening History",
			"Sign in with your YouTube Music account to sync your playback history across sessions.",
		)
	}

	if db == nil {
		return renderEmptyState(
			"Database Unavailable",
			"Could not connect to the local storage database.",
			"Retry",
			func() { SigLibraryTab.Set(LibraryTabHistory) },
		)
	}

	ctx := context.Background()
	history, err := db.GetRecentHistory(ctx, 50)
	if err != nil || len(history) == 0 {
		return renderEmptyState(
			"Listening History is Empty",
			"Songs you listen to will appear here for easy replay and discovery.",
			"Listen Now",
			func() {
				if r != nil {
					r.Navigate("/")
				}
			},
		)
	}

	header := primitives.HBox(
		primitives.Text(fmt.Sprintf("%d Recently Played Songs", len(history))).
			FontSize(14).
			Color(theme.ColorTextSecondary),
		primitives.Expanded(primitives.Box()),
		actionButton("Clear History", toneText, func() {
			_ = db.ClearHistory(ctx)
			SigLibraryTab.Set(LibraryTabHistory)
		}),
	).CrossAlign(primitives.CrossAxisCenter)

	rows := []widget.Widget{
		header,
		primitives.Box().Height(12),
	}

	for i, h := range history {
		idx := i
		track := h.Track
		playedTimeStr := h.PlayedAt.Format("Jan 02, 15:04")
		rows = append(rows,
			buildLibraryTrackRow(idx, track, "Played "+playedTimeStr, func() {
				state.PlaySpecificTrack(track)
			}),
			primitives.Box().Height(8),
		)
	}

	return primitives.VBox(rows...)
}

func renderLibraryDownloads(r *router.Router, dc *cache.DiskCache, db *cache.DB) widget.Widget {
	// Downloads are local and available to both guests and signed-in users!
	if dc == nil {
		return renderEmptyState(
			"Cache Storage Unavailable",
			"Could not open the local disk cache directory.",
			"Retry",
			func() { SigLibraryTab.Set(LibraryTabDownloads) },
		)
	}

	items, err := dc.ListDownloaded()
	if err != nil || len(items) == 0 {
		return renderEmptyState(
			"No Downloaded Tracks",
			"Streamed audio is automatically cached on disk (~/.cache/chordly/audio) for offline playback.",
			"Start Listening",
			func() {
				if r != nil {
					r.Navigate("/")
				}
			},
		)
	}

	var totalBytes int64
	for _, item := range items {
		totalBytes += item.Size
	}

	header := primitives.HBox(
		primitives.Text(fmt.Sprintf("%d Offline Tracks • %s total", len(items), formatBytes(totalBytes))).
			FontSize(14).
			Color(theme.ColorTextSecondary),
	)

	rows := []widget.Widget{
		header,
		primitives.Box().Height(12),
	}

	ctx := context.Background()
	for i, item := range items {
		idx := i
		videoID := item.VideoID
		track := model.Track{
			ID:    videoID,
			Title: videoID,
		}
		if db != nil {
			if meta, found, _ := db.GetTrackMeta(ctx, videoID); found && meta.Title != "" {
				track = meta
			}
		}

		infoText := fmt.Sprintf("WebM / Opus • %s", formatBytes(item.Size))
		rows = append(rows,
			buildLibraryTrackRow(idx, track, infoText, func() {
				state.PlaySpecificTrack(track)
			}),
			primitives.Box().Height(8),
		)
	}

	return primitives.VBox(rows...)
}

func renderLibraryGuestCTA(r *router.Router, tabName, description string) widget.Widget {
	return primitives.VBox(
		primitives.HBox(
			NewVectorIcon(IconAlert, theme.ColorAccent, 24),
			primitives.Box().Width(12),
			primitives.Text("Sign in to sync "+tabName).FontSize(18).Bold().Color(theme.ColorTextPrimary),
		).CrossAlign(primitives.CrossAxisCenter),
		primitives.Box().Height(10),
		primitives.Text(description).FontSize(14).Color(theme.ColorTextSecondary),
		primitives.Box().Height(20),
		primitives.HBox(
			actionButton("Connect YouTube Music", toneFilled, func() {
				state.RouteReturn.Set("/library")
				if r != nil {
					r.Navigate("/login")
				}
			}),
			primitives.Box().Width(12),
			actionButton("Browse Offline Downloads", toneTonal, func() {
				SigLibraryTab.Set(LibraryTabDownloads)
				if r != nil {
					r.Navigate("/library/downloads")
				}
			}),
		),
	).
		PaddingXY(28, 24).
		Background(theme.ColorSurface).
		Rounded(16).
		BorderStyle(1, theme.ColorBorder)
}

func renderEmptyState(title, subtitle, actionText string, onAction func()) widget.Widget {
	return primitives.VBox(
		primitives.HBox(
			NewVectorIcon(IconLibrary, theme.ColorTextMuted, 26),
			primitives.Box().Width(12),
			primitives.Text(title).FontSize(18).Bold().Color(theme.ColorTextPrimary),
		).CrossAlign(primitives.CrossAxisCenter),
		primitives.Box().Height(8),
		primitives.Text(subtitle).FontSize(14).Color(theme.ColorTextSecondary),
		primitives.Box().Height(18),
		actionButton(actionText, toneTonal, onAction),
	).
		PaddingXY(28, 24).
		Background(theme.ColorSurfaceLight).
		Rounded(16).
		BorderStyle(1, theme.ColorBorder)
}

func buildLibraryTrackRow(index int, track model.Track, subtext string, onPlay func()) widget.Widget {
	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, theme.ColorAccent, 16),
	).
		PaddingXY(14, 12).
		Background(widget.Hex(0x1B2830)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	displaySub := track.Artist
	if subtext != "" {
		if displaySub != "" {
			displaySub += " • " + subtext
		} else {
			displaySub = subtext
		}
	}

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(displaySub).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	playBtn := NewVectorIconButton(IconPlay, 32, 14, onPlay, func(s *IconButtonStyle) {
		s.Background = theme.ColorAccent
		s.Foreground = theme.ColorOnAccent
		s.BorderWidth = 0
	})

	return primitives.HBox(
		primitives.Text(fmt.Sprintf("%d", index+1)).FontSize(13).Color(theme.ColorTextMuted),
		primitives.Box().Width(10),
		thumb,
		primitives.Box().Width(14),
		meta,
		primitives.Expanded(primitives.Box()),
		playBtn,
	).
		CrossAlign(primitives.CrossAxisCenter).
		PaddingXY(14, 10).
		Background(theme.ColorSurfaceLight).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)
}

func buildPlaylistRow(index int, pl cache.Playlist, onClick func()) widget.Widget {
	thumb := primitives.HBox(
		NewVectorIcon(IconQueue, theme.ColorAccent, 16),
	).
		PaddingXY(14, 12).
		Background(widget.Hex(0x1B2830)).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)

	sub := fmt.Sprintf("%d tracks • Updated %s", pl.ItemCount, pl.UpdatedAt.Format("Jan 02, 2006"))
	if pl.Description != "" {
		sub = pl.Description + " • " + sub
	}

	meta := primitives.VBox(
		primitives.Text(pl.Name).FontSize(14).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(sub).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	openBtn := actionButton("Open", toneTonal, onClick)

	return primitives.HBox(
		primitives.Text(fmt.Sprintf("%d", index+1)).FontSize(13).Color(theme.ColorTextMuted),
		primitives.Box().Width(10),
		thumb,
		primitives.Box().Width(14),
		meta,
		primitives.Expanded(primitives.Box()),
		openBtn,
	).
		CrossAlign(primitives.CrossAxisCenter).
		PaddingXY(14, 10).
		Background(theme.ColorSurfaceLight).
		Rounded(10).
		BorderStyle(1, theme.ColorBorder)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
