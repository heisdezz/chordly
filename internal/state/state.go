package state

import (
	"time"

	"github.com/gogpu/ui/state"

	"chordly/internal/model"
)

var (
	// Hero / Current Playing Track
	CurrentTrack = state.NewSignal(model.Track{
		ID:          "echoes-of-tomorrow",
		Title:       "Echoes of Tomorrow",
		Artist:      "Rexha Mavrick",
		Duration:    237 * time.Second, // 3:57
		DurationStr: "3:57",
		Listeners:   "12.5M listeners",
		Thumbnail:   "echoes_art",
		IsLiked:     false,
	})

	// Playback Transport
	PlaybackStatus = state.NewSignal(model.StatePlaying)
	Playhead       = state.NewSignal(84 * time.Second) // 1:24
	Volume         = state.NewSignal[float32](0.78)    // 78%
	IsShuffle      = state.NewSignal(false)
	Repeat         = state.NewSignal(model.RepeatOff)

	// Up Next Queue
	Queue = state.NewSignal([]model.Track{
		{
			ID:          "echoes-of-tomorrow",
			Title:       "Echoes of Tomorrow",
			Artist:      "Rexha Mavrick",
			DurationStr: "3:45",
			Duration:    225 * time.Second,
		},
		{
			ID:          "midnight-drive",
			Title:       "Midnight Drive",
			Artist:      "The Weeknd",
			DurationStr: "3:52",
			Duration:    232 * time.Second,
		},
		{
			ID:          "starlight",
			Title:       "Starlight",
			Artist:      "Dua Lipa",
			DurationStr: "3:37",
			Duration:    217 * time.Second,
		},
		{
			ID:          "better-days",
			Title:       "Better Days",
			Artist:      "SZA",
			DurationStr: "4:12",
			Duration:    252 * time.Second,
		},
		{
			ID:          "lost-in-you",
			Title:       "Lost in You",
			Artist:      "The Neighbourhood",
			DurationStr: "3:29",
			Duration:    209 * time.Second,
		},
	})
	QueueIndex = state.NewSignal(0)

	// "Made for you" Mixes
	MadeForYouMixes = state.NewSignal([]model.MixCard{
		{
			ID:       "mix-daily-1",
			Title:    "Daily Mix 1",
			ColorHex: 0xE2556E, // Vibrant Rose
		},
		{
			ID:       "mix-chill",
			Title:    "Chill Vibes",
			ColorHex: 0x2A7EB7, // Deep Ocean Blue
		},
		{
			ID:       "mix-rnb",
			Title:    "R&B Mix",
			ColorHex: 0x8C49E6, // Purple Glow
		},
		{
			ID:       "mix-amapiano",
			Title:    "Amapiano Mix",
			ColorHex: 0xD85D38, // Sunset Amber
		},
		{
			ID:       "mix-focus",
			Title:    "Focus Flow",
			ColorHex: 0x367F4F, // Emerald Green
		},
	})

	// "New releases" Tracks
	NewReleases = state.NewSignal([]model.Track{
		{
			ID:          "all-i-want",
			Title:       "All I Want",
			Artist:      "Daniel Caesar",
			DurationStr: "3:45",
			Duration:    225 * time.Second,
			IsLiked:     false,
		},
		{
			ID:          "maybe-in-another-life",
			Title:       "Maybe in Another Life",
			Artist:      "Tame Impala",
			DurationStr: "4:23",
			Duration:    263 * time.Second,
			IsLiked:     false,
		},
	})

	// Navigation & UI States
	ActiveNavTab = state.NewSignal("home") // "home", "explore", "library", "settings"
	SearchQuery  = state.NewSignal("")
	IsQueueOpen  = state.NewSignal(true)
	IsFullscreen = state.NewSignal(false)

	// Authentication & Return Navigation
	RouteReturn = state.NewSignal("/")
	UserAccount = state.NewSignal("") // Empty if guest, otherwise display name / handle
)

// Actions
func TogglePlayPause() {
	if PlaybackStatus.Get() == model.StatePlaying {
		PlaybackStatus.Set(model.StatePaused)
	} else {
		PlaybackStatus.Set(model.StatePlaying)
	}
}

func NextTrack() {
	q := Queue.Get()
	idx := QueueIndex.Get()
	if len(q) == 0 {
		return
	}
	nextIdx := (idx + 1) % len(q)
	QueueIndex.Set(nextIdx)
	CurrentTrack.Set(q[nextIdx])
	Playhead.Set(0)
	PlaybackStatus.Set(model.StatePlaying)
}

func PreviousTrack() {
	q := Queue.Get()
	idx := QueueIndex.Get()
	if len(q) == 0 {
		return
	}
	prevIdx := (idx - 1 + len(q)) % len(q)
	QueueIndex.Set(prevIdx)
	CurrentTrack.Set(q[prevIdx])
	Playhead.Set(0)
	PlaybackStatus.Set(model.StatePlaying)
}

func PlaySpecificTrack(t model.Track) {
	CurrentTrack.Set(t)
	Playhead.Set(0)
	PlaybackStatus.Set(model.StatePlaying)
}
