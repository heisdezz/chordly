package appui

import (
	"context"
	"fmt"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/cache"
	"chordly/internal/model"
	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

// QueuePage renders the full-featured interactive play queue view.
func QueuePage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget { return renderQueuePage(c.Router) },
		WatchSignal(state.Queue.AsReadonly()),
		WatchSignal(state.QueueIndex.AsReadonly()),
		WatchSignal(state.PlaybackStatus.AsReadonly()),
		WatchSignal(theme.SigActivePalette.AsReadonly()),
	)
}

func renderQueuePage(r *router.Router) widget.Widget {
	queue := state.Queue.Get()
	currentIdx := state.QueueIndex.Get()

	var totalDuration time.Duration
	for _, track := range queue {
		totalDuration += track.Duration
	}

	totalMin := int(totalDuration.Minutes())
	totalSec := int(totalDuration.Seconds()) % 60
	durationStr := fmt.Sprintf("%d:%02d", totalMin, totalSec)
	subtitle := fmt.Sprintf("%d tracks queued • %s total duration", len(queue), durationStr)
	if len(queue) == 0 {
		subtitle = "No tracks in playback queue"
	}

	var actions widget.Widget
	if len(queue) > 0 {
		actions = primitives.HBox(
			actionButton("Save as Playlist", toneFilled, func() {
				saveQueueAsPlaylist()
			}),
			primitives.Box().Width(10),
			actionButton("Clear Queue", toneText, func() {
				state.ClearQueue()
				ShowInfo("Queue cleared")
			}),
		).CrossAlign(primitives.CrossAxisCenter)
	}

	var body widget.Widget
	if len(queue) == 0 {
		body = renderEmptyQueueState(r)
	} else {
		rows := make([]widget.Widget, 0, len(queue)*2)
		for i, track := range queue {
			idx := i
			t := track
			isActive := idx == currentIdx
			rows = append(rows,
				buildQueueItemRow(idx, t, isActive, len(queue)),
				primitives.Box().Height(6),
			)
		}
		body = primitives.VBox(rows...)
	}

	cfg := DefaultPageScaffoldConfig()
	cfg.Title = "Play Queue"
	cfg.Subtitle = subtitle
	cfg.Actions = actions
	cfg.Body = body

	return NewPageScaffold(cfg)
}

func renderEmptyQueueState(r *router.Router) widget.Widget {
	return primitives.VBox(
		primitives.HBox(
			NewVectorIcon(IconQueue, theme.ColorTextMuted, 28),
			primitives.Box().Width(14),
			primitives.Text("Your Queue is Empty").FontSize(18).Bold().Color(theme.ColorTextPrimary),
		).CrossAlign(primitives.CrossAxisCenter),
		primitives.Box().Height(8),
		primitives.Text("Play any song, album, or mix to automatically start your playback queue.").
			FontSize(14).
			Color(theme.ColorTextSecondary),
		primitives.Box().Height(18),
		actionButton("Discover Music", toneTonal, func() {
			if r != nil {
				r.Navigate("/")
			}
		}),
	).
		PaddingXY(28, 24).
		Background(theme.ColorSurfaceLight).
		Rounded(16).
		BorderStyle(1, theme.ColorBorder)
}

func buildQueueItemRow(index int, track model.Track, isActive bool, total int) widget.Widget {
	var thumbColor widget.Color = widget.Hex(0x1B2830)
	var iconColor widget.Color = theme.ColorAccent
	var rowBg widget.Color = theme.ColorSurfaceLight
	var rowBorder widget.Color = theme.ColorBorder
	var titleColor widget.Color = theme.ColorTextPrimary
	var durColor widget.Color = theme.ColorTextMuted

	if isActive {
		thumbColor = widget.Hex(0x183C34)
		rowBg = widget.RGBA8(92, 225, 182, 24)
		rowBorder = widget.RGBA8(92, 225, 182, 80)
		titleColor = theme.ColorAccent
		durColor = theme.ColorAccent
	}

	thumb := primitives.HBox(
		NewVectorIcon(IconWaveform, iconColor, 16),
	).
		PaddingXY(14, 12).
		Background(thumbColor).
		Rounded(10).
		BorderStyle(1, rowBorder)

	var indicator widget.Widget
	if isActive {
		indicator = primitives.HBox(
			NewVectorIcon(IconPlay, theme.ColorAccent, 12),
		).PaddingXY(4, 2)
	} else {
		indicator = primitives.Text(fmt.Sprintf("%d", index+1)).
			FontSize(13).
			Color(theme.ColorTextMuted)
	}

	meta := primitives.VBox(
		primitives.Text(track.Title).FontSize(14).Bold().Color(titleColor),
		primitives.Text(track.Artist).FontSize(12).Color(theme.ColorTextSecondary),
	).Gap(2)

	// Action controls per row: Up, Down, Play, Remove
	var moveUpBtn widget.Widget
	if index > 0 {
		moveUpBtn = actionButton("▲", toneText, func() {
			state.MoveQueueItemUp(index)
		})
	} else {
		moveUpBtn = primitives.Box().Width(24)
	}

	var moveDownBtn widget.Widget
	if index < total-1 {
		moveDownBtn = actionButton("▼", toneText, func() {
			state.MoveQueueItemDown(index)
		})
	} else {
		moveDownBtn = primitives.Box().Width(24)
	}

	playBtn := NewVectorIconButton(IconPlay, 28, 12, func() {
		state.PlayQueueIndex(index)
	}, func(s *IconButtonStyle) {
		s.Background = widget.ColorTransparent
		s.Foreground = theme.ColorTextPrimary
		s.BorderWidth = 0
	})

	removeBtn := actionButton("✕", toneText, func() {
		state.RemoveQueueItem(index)
	})

	controls := primitives.HBox(
		moveUpBtn,
		moveDownBtn,
		playBtn,
		removeBtn,
	).CrossAlign(primitives.CrossAxisCenter).Gap(4)

	return primitives.HBox(
		primitives.HBox(indicator).Width(24).CrossAlign(primitives.CrossAxisCenter),
		primitives.Box().Width(8),
		thumb,
		primitives.Box().Width(12),
		meta,
		primitives.Expanded(primitives.Box()),
		primitives.Text(track.DurationStr).FontSize(12).Color(durColor),
		primitives.Box().Width(16),
		controls,
	).
		CrossAlign(primitives.CrossAxisCenter).
		PaddingXY(14, 10).
		Background(rowBg).
		Rounded(12).
		BorderStyle(1, rowBorder)
}

func saveQueueAsPlaylist() {
	queue := state.Queue.Get()
	if len(queue) == 0 {
		ShowError("Playback queue is empty; nothing to save")
		return
	}

	db, err := cache.GetDefaultDB()
	if err != nil || db == nil {
		ShowError("Database unavailable; cannot save playlist")
		return
	}

	ctx := context.Background()
	playlistName := fmt.Sprintf("Queue (%s)", time.Now().Format("Jan 02, 15:04"))
	pl, err := db.CreatePlaylist(ctx, playlistName, "Saved from active play queue")
	if err != nil {
		ShowError("Failed to create playlist: " + err.Error())
		return
	}

	for _, track := range queue {
		_ = db.AddTrackToPlaylist(ctx, pl.ID, track)
	}

	ShowSuccess(fmt.Sprintf("Saved %d tracks to playlist %q", len(queue), playlistName))
}
