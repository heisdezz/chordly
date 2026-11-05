package appui

import (
	"time"

	gpstate "github.com/gogpu/ui/state"
)

// SigReducedMotion disables non-essential motion. It is intended to mirror the
// OS "reduce motion" preference and is also exposed as a Settings toggle.
var SigReducedMotion = gpstate.NewSignal(false)

// Motion duration tokens. Keep spatial motion >= 100ms because the framework's
// animation pump runs at ~30 fps.
const (
	MotionFast = 100 * time.Millisecond
	MotionBase = 200 * time.Millisecond
	MotionSlow = 400 * time.Millisecond
)

var pageTransitionDuration = MotionBase

// motionReduced reports whether non-essential animation should be skipped.
func motionReduced() bool {
	return SigReducedMotion.Get()
}

// motionDuration collapses a duration to 0 when reduced motion is enabled.
func motionDuration(d time.Duration) time.Duration {
	if motionReduced() {
		return 0
	}
	return d
}

// easeOutCubic is the shared easing for paint-only time-based effects.
func easeOutCubic(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	inv := 1 - t
	return 1 - inv*inv*inv
}
