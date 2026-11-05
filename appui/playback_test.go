package appui

import (
	"testing"
	"time"

	"chordly/internal/model"
	"chordly/internal/state"
)

func TestScrubberTargetFromPosition(t *testing.T) {
	previous := state.CurrentTrack.Get()
	defer state.CurrentTrack.Set(previous)

	state.CurrentTrack.Set(model.Track{ID: "scrub", Duration: 200 * time.Second})

	s := NewScrubberWidget()
	const startX, width = float32(48), float32(304)

	cases := []struct {
		name   string
		mouseX float32
		want   time.Duration
	}{
		{"start", startX, 0},
		{"middle", startX + width/2, 100 * time.Second},
		{"before start clamps", startX - 40, 0},
		{"after end clamps", startX + width + 40, 200 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.targetFromPosition(tc.mouseX, startX, width)
			if diff := got - tc.want; diff > time.Millisecond || diff < -time.Millisecond {
				t.Errorf("targetFromPosition(%v) = %v, want %v", tc.mouseX, got, tc.want)
			}
		})
	}
}

func TestSeekRequestNotifiesRepeatedTargets(t *testing.T) {
	var seen []time.Duration
	unsub := SeekRequest.SubscribeForever(func(d time.Duration) {
		seen = append(seen, d)
	})
	defer unsub()

	SeekRequest.Set(42 * time.Second)
	SeekRequest.Set(42 * time.Second)

	if len(seen) != 2 {
		t.Fatalf("SeekRequest delivered %d notifications, want 2 (same target must re-notify)", len(seen))
	}
	for _, d := range seen {
		if d != 42*time.Second {
			t.Errorf("delivered %v, want 42s", d)
		}
	}
}

func TestIsScrubbingSignal(t *testing.T) {
	IsScrubbing.Set(true)
	if !IsScrubbing.Get() {
		t.Fatal("IsScrubbing should be true")
	}
	IsScrubbing.Set(false)
	if IsScrubbing.Get() {
		t.Fatal("IsScrubbing should be false")
	}
}
