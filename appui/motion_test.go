package appui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
)

func TestMotionDurationCollapsesWhenReduced(t *testing.T) {
	previous := SigReducedMotion.Get()
	t.Cleanup(func() { SigReducedMotion.Set(previous) })

	SigReducedMotion.Set(false)
	if got := motionDuration(MotionBase); got != MotionBase {
		t.Fatalf("motion duration = %v, want %v", got, MotionBase)
	}

	SigReducedMotion.Set(true)
	if got := motionDuration(MotionBase); got != 0 {
		t.Fatalf("reduced motion duration = %v, want 0", got)
	}
}

func TestSpinnerLayout(t *testing.T) {
	s := NewSpinner(24)
	size := s.Layout(nil, geometry.Constraints{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 24 || size.Height != 24 {
		t.Fatalf("spinner size = %v, want 24x24", size)
	}
}

func TestSkeletonLayoutFillsAvailableWidth(t *testing.T) {
	s := NewSkeleton(0, 12, 6)
	size := s.Layout(nil, geometry.Constraints{MaxWidth: 200, MaxHeight: 100})
	if size.Width != 200 {
		t.Fatalf("skeleton width = %v, want 200", size.Width)
	}
	if size.Height != 12 {
		t.Fatalf("skeleton height = %v, want 12", size.Height)
	}
}

func TestLoadingBuildersAreNonNil(t *testing.T) {
	if NewSkeletonTrackRows(3) == nil {
		t.Fatal("NewSkeletonTrackRows returned nil")
	}
	if NewSkeletonCardRow(4) == nil {
		t.Fatal("NewSkeletonCardRow returned nil")
	}
	if NewSkeletonHero() == nil {
		t.Fatal("NewSkeletonHero returned nil")
	}
	if NewSearchLoading("daft punk") == nil {
		t.Fatal("NewSearchLoading returned nil")
	}
}
