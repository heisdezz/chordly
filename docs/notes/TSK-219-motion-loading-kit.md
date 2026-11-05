# Task Specification & Resolution Note: TSK-219 (Motion & Loading Kit)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-219`  
**Parent Plan:** [`docs/ui/animations-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/animations-plan.md)  
**File Scope:**

- [`appui/motion.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/motion.go)
- [`appui/motion_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/motion_test.go)
- [`appui/loading.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/loading.go)
- [`appui/routerview.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/routerview.go)
- [`appui/routerview_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/routerview_test.go)
- [`appui/settings_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/settings_view.go)
- [`appui/search_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/search_view.go)

---

## 1. Objective

Implement Chordly's motion language, loading indicators, skeleton placeholders, and page transitions while respecting accessibility guidelines (reduced motion preference).

---

## 2. Architecture & Design

### 2.1 Motion Tokens & Reduced Motion

- **Tokens**: `MotionFast` (100ms), `MotionBase` (200ms), `MotionSlow` (400ms)
  in `appui/motion.go`; `pageTransitionDuration` uses `MotionBase`.
  (The GoGPU animation engine's M3 tokens are available for a later pass, but
  this kit uses the local paint-only, time-based helpers so it works with the
  current dependency set.)
- **Accessibility**: Global `SigReducedMotion` signal. When enabled,
  `motionDuration()` returns `0`, decorative pulsing stops, and transitions
  jump instantly without scheduling animation frames.
- **Settings Toggle**: "Reduce motion" toggle in the Appearance section of
  [`appui/settings_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/settings_view.go).

### 2.2 Loading Indicators (`appui/loading.go`)

- **`Spinner`**: A vector-rendered indeterminate circular spinner with phase-offset arc sweeping. Implements `widget.Widget` with paint-only animation via `ctx.ScheduleAnimationFrame()` and `s.SetNeedsRedraw(true)`.
- **`Skeleton`**: A soft pulsing placeholder box with rounded corners. Alpha pulses smoothly between 0.35 and 0.75 using a 1200ms sine wave.
- **Pre-built Placeholders**:
  - `NewSkeletonTrackRows(count)`: Simulated track list items with artwork box, title, subtitle, and duration pill.
  - `NewSkeletonCardRow(count)`: Horizontal card grid placeholders for albums/mixes.
  - `NewSkeletonHero()`: Prominent hero showcase placeholder.
  - `NewSearchLoading(query)`: Header spinner with track skeletons displayed while querying InnerTube.

### 2.3 Page Navigation Transitions (`appui/routerview.go`)

- Retains the outgoing page for a crossfade while the incoming page fades in and
  slides up 8px over `pageTransitionDuration` (200ms), using
  `Canvas.PushOpacity` when supported.
- Short-circuits to an instant cut when `SigReducedMotion` is active or the
  canvas cannot fade.
- Rapid navigations unmount any previous outgoing page before starting the next
  transition.

---

## 3. Implementation Verification & Gotchas

1. **`widget.Widget` Interface Compliance**:
   `Spinner` and `Skeleton` must implement `Event(ctx widget.Context, e event.Event) bool { return false }` in addition to `Layout`, `Draw`, and `Children`. `WidgetBase` in GoGPU does not provide a default `Event` handler.
2. **Animation Frame Scheduling**:
   Always check `sched, ok := ctx.(widget.AnimationScheduler); ok` and invoke `sched.ScheduleAnimationFrame()` to trigger 30fps animation loops without pegging the event loop.
3. **Reduced Motion Safety**:
   If reduced motion is active, animated widgets skip scheduling animation frames and render static fallback states to conserve CPU/GPU cycles.

---

## 4. Status

- **Status:** `DONE`
- **Verification:** Unit tests passing (`go test ./appui/...`) and all packages building cleanly (`go build ./...`).
