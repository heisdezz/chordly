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
	PlaybackStatus = state.NewSignal(model.StateStopped)
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

	// Continuation trigger callback for endless radio automix
	OnQueueNeedsContinuation func(lastTrack model.Track)
)

// Actions
func TogglePlayPause() {
	if PlaybackStatus.Get() == model.StatePlaying {
		PlaybackStatus.Set(model.StatePaused)
	} else {
		PlaybackStatus.Set(model.StatePlaying)
	}
}

// TogglePlay is an alias for TogglePlayPause.
func TogglePlay() {
	TogglePlayPause()
}

// ToggleLike flips the IsLiked property on CurrentTrack.
func ToggleLike() {
	t := CurrentTrack.Get()
	t.IsLiked = !t.IsLiked
	CurrentTrack.Set(t)
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

	checkContinuation(nextIdx, q)
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

// PrevTrack is an alias for PreviousTrack.
func PrevTrack() {
	PreviousTrack()
}

func PlaySpecificTrack(t model.Track) {
	CurrentTrack.Set(t)
	Playhead.Set(0)
	PlaybackStatus.Set(model.StatePlaying)
}

// PlayQueueIndex sets the active queue index and immediately starts playing it.
func PlayQueueIndex(index int) {
	q := Queue.Get()
	if index < 0 || index >= len(q) {
		return
	}
	QueueIndex.Set(index)
	CurrentTrack.Set(q[index])
	Playhead.Set(0)
	PlaybackStatus.Set(model.StatePlaying)

	checkContinuation(index, q)
}

// MoveQueueItem relocates a track from 'from' position to 'to' position.
// Automatically updates QueueIndex if the current track was moved or shifted.
func MoveQueueItem(from, to int) {
	q := Queue.Get()
	n := len(q)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}

	newQ := make([]model.Track, n)
	copy(newQ, q)
	item := newQ[from]

	if from < to {
		copy(newQ[from:to], newQ[from+1:to+1])
	} else {
		copy(newQ[to+1:from+1], newQ[to:from])
	}
	newQ[to] = item

	currentIdx := QueueIndex.Get()
	if currentIdx == from {
		QueueIndex.Set(to)
	} else if from < currentIdx && to >= currentIdx {
		QueueIndex.Set(currentIdx - 1)
	} else if from > currentIdx && to <= currentIdx {
		QueueIndex.Set(currentIdx + 1)
	}

	Queue.Set(newQ)
}

// MoveQueueItemUp moves a track one position earlier in the queue.
func MoveQueueItemUp(index int) {
	if index > 0 {
		MoveQueueItem(index, index-1)
	}
}

// MoveQueueItemDown moves a track one position later in the queue.
func MoveQueueItemDown(index int) {
	q := Queue.Get()
	if index >= 0 && index < len(q)-1 {
		MoveQueueItem(index, index+1)
	}
}

// RemoveQueueItem removes the track at the specified index from the queue.
func RemoveQueueItem(index int) {
	q := Queue.Get()
	n := len(q)
	if index < 0 || index >= n {
		return
	}

	newQ := make([]model.Track, 0, n-1)
	newQ = append(newQ, q[:index]...)
	newQ = append(newQ, q[index+1:]...)

	currentIdx := QueueIndex.Get()
	if len(newQ) == 0 {
		QueueIndex.Set(-1)
	} else if currentIdx == index {
		if currentIdx >= len(newQ) {
			currentIdx = len(newQ) - 1
		}
		QueueIndex.Set(currentIdx)
		CurrentTrack.Set(newQ[currentIdx])
	} else if currentIdx > index {
		QueueIndex.Set(currentIdx - 1)
	}

	Queue.Set(newQ)
}

// ClearQueue empties the queue and resets the active index.
func ClearQueue() {
	Queue.Set(nil)
	QueueIndex.Set(-1)
}

// AppendToQueue adds tracks to the end of the queue. If queue was empty, it starts playing the first track.
func AppendToQueue(tracks ...model.Track) {
	if len(tracks) == 0 {
		return
	}
	q := Queue.Get()
	wasEmpty := len(q) == 0

	newQ := append(q, tracks...)
	Queue.Set(newQ)

	if wasEmpty {
		PlayQueueIndex(0)
	}
}

func checkContinuation(idx int, q []model.Track) {
	if len(q) > 0 && idx >= len(q)-2 && OnQueueNeedsContinuation != nil {
		go OnQueueNeedsContinuation(q[len(q)-1])
	}
}
