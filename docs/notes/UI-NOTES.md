# UI Task Notes — 2026-11-05

Working notes for the next WS-UI agent. Read alongside `docs/PROGRESS.md`,
`docs/plan-1-basic-capabilities.md` (Module 4–5), and `docs/plan-2-advanced-extensibility.md`
(Module 6–9).

---

## 1. What the docs claim vs. what the code actually did

`PROGRESS.md` marks Phase 0 and `TSK-204` / `TSK-205` as `DONE`, but several of those
deliverables were **standalone and never mounted**. Before this session:

| Documented capability                    | Reality                                                                                                                                                |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Metrolist shell + navigation (`TSK-004`) | `NewShell` ignored its `*router.Router` argument and rendered a static VBox.                                                                           |
| Router pages (`main.go`)                 | Routes were registered, but `routerView` (`appui/routerview.go`) was only used by its own test. Pages were dead code.                                  |
| Sidebar navigation                       | Clicking rail icons only set `state.ActiveNavTab`; it never called `Router.Navigate`.                                                                  |
| Search page (`TSK-205`)                  | `NewSearchView()` had zero callers. Top bar called `ytm.LiveSearch("Daft Punk")` directly.                                                             |
| Lyrics overlay (`TSK-204`)               | `NewLyricsOverlay()` had zero callers. `FetchLyricsForCurrentTrack()` had zero callers. `IsLyricsOpen` was only ever set to `false`.                   |
| Home feed                                | Hero / Made-for-you / New-releases were assembled directly in the shell, not in the `/` route page. `HomePage` showed demo "Chordly projects" content. |
| Theme presets (`TSK-203`)                | Presets / `ParseHexColor` / `TransitionEngine` existed, but the UI read the global `theme.Color*` vars only and there was no settings UI.              |

Resolved since: Settings applies presets and custom accents through `theme.ApplyPalette`
(UI-INT-02), and the shell rebuilds on palette change.

Two palettes coexisted and did **not** match:

- `internal/theme` = dark slate/mint (used by shell, sidebar, player, sections).
- `appui/style.go` = light Material demo palette (used by `page()`, demo project routes,
  `NotFoundPage`).

Settings, Queue, Search, Lyrics, and Home now use theme tokens. The demo `page()` helper and the
unreachable project scaffolding still use `appui/style.go`.

---

## 2. UI wiring added in this session

Files changed:

- `appui/home_view.go` (new): `NewHomeContent()` composes `NewHeroBanner`,
  `NewMadeForYouSection`, `NewNewReleasesSection`.
- `appui/reactive.go` (new): `newReactiveView` + `WatchSignal` rebuild container (see UI-INT-01).
- `internal/theme/active.go` (new): `SigActivePalette`, `SigActivePresetID`, `SigActiveIsDark`,
  `ApplyPalette`, `ApplyPreset`, `ApplyCustomAccent` (see UI-INT-02).
- `appui/settings_view.go` (new): dark/light-aware Settings page with preset cards,
  Dark/Light toggle, accent swatches, playback and about sections.
- `NewShell` now rebuilds the whole app when the palette changes; `QueuePage` was
  rewritten with theme tokens instead of the light demo `page()` wrapper.
- Lyrics wiring (UI-INT-05): `LyricsViewWidget` lifecycle subscriptions, guarded fetching,
  `IconLyrics` + player-dock button, reactive bottom-player metadata, and repainting
  scrubber/volume sliders.
- Search input (UI-INT-03): `appui/textfield.go` with focus, editing keys, caret; top bar now
  submits real queries and `/search` no longer auto-runs a hardcoded placeholder.
- Converted `NewHeroBanner`, `NewUpNextPanel`, `NewMadeForYouSection`,
  `NewNewReleasesSection`, and `QueuePage` to rebuild from signals instead of
  taking one-time `.Get()` snapshots. Hero now shows `state.CurrentTrack` instead
  of hardcoded title/artist text.
- `appui/pages.go`: `HomePage` now returns the live home feed; added `QueuePage`,
  `SearchPage`, `LyricsPage`.
- `appui/shell.go`: content area is now `newRouterView(r)`; bottom player receives the router.
- `appui/sidebar.go`: `NewNavRail(r)` navigates (`/`, `/explore`, `/library`, `/settings`)
  and derives active state from `r.Match()`.
- `appui/topbar.go`: `NewTopBar(r)` navigates to `/search` instead of calling YTM directly.
- `appui/bottom_player.go`: `NewBottomPlayer(r)` queue button navigates to `/queue`.
- `main.go`: added `/search`, `/queue`, `/lyrics` routes.

Resulting flow:

```mermaid
flowchart TD
    A[main.go route tree] --> B[appui.NewShell]
    B --> C[NewNavRail -> Router.Navigate]
    B --> D[newRouterView]
    D --> E[Route.Page builder]
    E --> F[HomePage / SearchPage / QueuePage / LyricsPage / SettingsPage]
    B --> G[NewTopBar -> /search]
    B --> H[NewBottomPlayer -> /queue]
    B --> I[NewUpNextPanel static snapshot]
```

`go build ./...` passed immediately after these edits. `go test ./appui ./router ./internal/theme ./internal/lyrics` passes.
A later concurrent change to `go.mod` / `go.sum` broke plain vendored builds (see B10);
`go build -mod=mod ./...` still passes.

### Layout overflow fixes

GoGPU canvases do not wrap or clip text, so every painter must fit its own strings.

- **Up next vs. New releases:** the release row used four fixed `Width(310)` cards, which painted
  under the fixed 260px Up-next panel. Releases now render as a responsive 2-column grid inside
  `Expanded` cells (all four fit, no overlap).
- **Made for you titles:** mix cards were fixed at 135px with only ~79px of title space. Cards are now
  `Expanded` (five share the row), and `fitText` truncates with an ellipsis before the play button.
- **Appearance descriptions:** preset card title/description are fitted with ellipsis to the card's
  inner width instead of spilling outside the rounded card.
- New helper: `appui/textfit.go` (`fitText(canvas, text, size, bold, maxWidth)`), reusable for any
  future painter that draws text in a fixed rect.

`go test ./...` currently fails **only** in `internal/audio` because of untracked,
in-progress `internal/audio/device.go` / `device_test.go` from Agent-Audio
(`TestPausePreservesPosition`, `TestRingReaderFeedsDevice`, `TestLoadReplacesSource`,
plus `close of closed channel` panic in `Device.pump`). Do not attribute these to UI work.

---

## 3. Startable WS-UI tasks

These are the highest-value tasks that need no new backend. Suggested IDs are new; add them
to `PROGRESS.md` when claiming so `TSK` numbers stay traceable.

### UI-INT-01 — Reactive page container — DONE

Implemented in `appui/reactive.go`:

- `newReactiveView(build func() widget.Widget, WatchSignal(...)...)`.
- Signal callbacks only mark the view dirty + invalidate; the child swap happens at
  the start of `Layout`/`Draw`, matching the existing `routerView` pattern.
- `Unmount` releases all `SubscribeForever` handles.
- `appui/reactive_test.go` covers single-signal rebuild, multi-signal rebuild, and
  unsubscribe-on-unmount.

Converted call sites: `NewHeroBanner`, `NewUpNextPanel`, `NewMadeForYouSection`,
`NewNewReleasesSection`, `QueuePage`.

Still snapshot-based (intentionally, next tasks):
`NewBottomPlayer` (rebuilding on `Playhead` at 30 Hz would be wasteful),
`NewLyricsOverlay` (needs a draw-invalidation strategy, see UI-INT-05),
`NewSearchView` (already subscribes internally).

### UI-INT-02 — Settings / theming UI — DONE

Implemented:

- `internal/theme/active.go`: `SigActivePalette`, `SigActivePresetID`, `SigActiveIsDark`,
  `ApplyPalette`, `ApplyPreset`, `ApplyCustomAccent`, `ActivePalette`, `IsDarkPalette`.
  `ApplyPalette` writes the legacy `Color*` tokens (so existing paint code keeps working) and
  then notifies the signal. It must run on the UI thread.
- `appui/settings_view.go`: preset cards (all six presets) with live mini palettes, Dark/Light
  toggle, eight accent swatches, active-theme summary, plus Playback and About cards.
- `appui/shell.go`: `NewShell` is now a `reactiveView` watching `SigActivePalette`, so a palette
  change rebuilds the shell once on the next layout pass.
- `appui/pages.go`: the demo `SettingsPage` was removed; `QueuePage` was rewritten with theme
  tokens because it previously forced the light demo palette inside the dark shell.

Remaining / deliberately deferred:

- **Custom hex text input** is not implemented yet. A single-line `TextField` now exists
  (`appui/textfield.go`), so this is a straightforward follow-up: feed it into
  `theme.ParseHexColor` and apply with `theme.ApplyCustomAccent`.
- **Animated transition** (`theme.TransitionEngine`, 250 ms LERP) is not wired. Doing so needs a
  UI-driven ticker that calls `ApplyPalette` per frame; note that each frame currently rebuilds the
  whole shell, which is too heavy. Implement `UI-INT-07` first.
- **Persistence** of the selected preset/accent is not written to the SQLite `settings` table yet.
- Light palettes make the remaining demo pages (`NotFoundPage`, project scaffolding) look out of
  place because they still use `appui/style.go`.

### UI-INT-07 — Palette transition without full rebuild (follow-up to UI-INT-02)

**Problem:** palette changes currently rebuild the entire shell (safe, but one full tree swap).
`TSK-202`'s `TransitionEngine` needs ~15 frames per switch; rebuilding the shell 15 times is too much.
**Do:** add a paint-time palette lookup (for example a `themedCanvas` or per-widget `WatchSignal`
for color-only widgets), then drive `TransitionEngine.TransitionTo` from a UI ticker. Keep the
whole-shell rebuild only for structural changes.

### UI-INT-03 — Real search input — DONE

Implemented in `appui/textfield.go`:

- Single-line field with click-to-focus, typing (`Rune` events), Backspace/Delete,
  Left/Right/Home/End, Space, Enter to submit, Escape to blur.
- Caret positioned with `Canvas.MeasureText`; click-to-caret is resolved on the next draw
  (the click handler records the local X, `Draw` maps it to a rune index).
- Paint uses `theme.Color*` tokens and switches border/icon color on focus.
- `appui/topbar.go` now hosts the field: `onChange` mirrors into `state.SearchQuery`,
  `onSubmit` calls `RunSearch(query)` and navigates to `/search`.
- `SearchPage` no longer auto-runs `"Daft Punk"`; it shows the normal empty state until
  a query is submitted.
- Tests: `appui/textfield_test.go` covers sizing, unfocused key rejection, focus-on-click,
  typing, backspace, cursor movement, submit, escape blur, space, and delete.

Limitations (acceptable for the search bar, document before reuse elsewhere):

- No text selection, clipboard, undo, or IME composition.
- No horizontal scrolling inside the field; long values are clipped.
- No caret blink (solid caret while focused).
- Key handling ignores Ctrl/Alt shortcuts.

### UI-INT-04 — Queue view completion

The `QueuePage` list is now reactive (UI-INT-01) but still snapshot-simple. Plan 5.4 also requires:

- animated “now playing” indicator,
- reorder (drag-and-drop; `github.com/gogpu/ui/dnd` is vendored),
- “Save as playlist”,
- auto-append when `QueueIndex >= len(Queue)-2` (`ytm.Service.FetchRadioContinuation`).
  **Scope:** `appui/queue_view.go` (extract from `pages.go`), `internal/state` additive actions.
  **Dependency:** none remaining for live updates.

### UI-INT-05 — Lyrics page wiring — DONE

Implemented:

- `LyricsViewWidget` now implements `widget.Lifecycle`. `Mount` subscribes to `Playhead`,
  `ActiveLyricsDoc`, `LyricsLoading`, and `CurrentTrack`, and repaints paint-only via
  `SetNeedsRedraw` + `ctx.Invalidate` (no tree rebuild at playhead rate). `Unmount` releases them.
- `FetchLyricsForCurrentTrack` now has a request-ID + title/artist guard: duplicate track
  notifications do not re-fetch, stale responses are dropped, and failures allow a later retry.
- The bottom player has a lyrics button (`IconLyrics`) that navigates to `/lyrics`.
- `NewLyricsOverlay(router)` is variadic for compatibility; passing the router makes the close
  button call `Router.Back()`.
- Track changes now also rebuild the bottom player metadata (`CurrentTrack` + `PlaybackStatus`),
  and the scrubber/volume slider repaint from `Playhead` / `Volume` subscriptions.

Remaining:

- `Playhead` is not advanced by the audio engine yet (`internal/audio` is WIP), so karaoke motion
  only animates as far as whatever updates `state.Playhead`.
- Click-to-seek updates `state.Playhead` only; it does not call an audio player `Seek` because the
  player controller is not wired into the UI yet.

### UI-INT-06 — Album / artist detail routes

Plan 4.2 lists `/album/:id` and `/artist/:id`; neither exists. `ytm` has `GetAlbum` (and search/radio)
but no artist-detail method yet. `search_view.go` already builds artist/album cards but they have no
navigation targets.
**Do:** add `appui/album_view.go`, `appui/artist_view.go`, extend `internal/ytm/browse.go`,
register nested routes. Coordinate with WS-YTM before touching `internal/ytm`.

---

## 4. Blockers / pitfalls found

**B1. No scroll or list primitives.**
Vendored `github.com/gogpu/ui/primitives` contains only `box.go`, `expanded.go`, `image.go`, `text.go`,
`style.go`, `themescope.go`, `repaint_boundary.go`, `raster_cache.go`. There is no scroll view or lazy
list. A single-line `TextField` now exists in `appui/textfield.go`, but long result lists still need a
dedicated scroll/list widget.

**B2. No layout scrolling in the shell.**
`NewHomeContent` and long search/queue pages are plain `VBox`es. Content taller than the viewport
will overflow; there is no scroll container to fix it yet.

**B3. Snapshots, not signals (mostly resolved).**
`reactiveView` (UI-INT-01) rebuilds home sections, queue page, and bottom-player metadata.
The lyrics view, scrubber, and volume slider use paint-only signal subscriptions.
Any new `.Get()` inside a `NewXxx()` builder is still a one-time read — wrap it in
`newReactiveView` or subscribe in `Mount`.

**B4. Route pages are rebuilt on every navigation.**
`routerView.sync()` unmounts the old page and builds a new one. Keep per-page state in signals
(as `search_view.go` does), not in the widget struct, or it will reset on every visit.

**B5. Demo scaffolding still in `appui/pages.go`.**
`projectCatalog`, `chordCatalog`, `ProjectsPage`, `ProjectPage`, `ChordsPage`, `ChordPage`, and
`routeInspector` are unreachable demo route-inspector code. Safe to delete once nothing references
them, but they currently keep `legacy_helpers.go` (`page`, `heading`, `card`, `keyValue`) in use.

**B6. Mixed palettes (mostly resolved).**
Shell, home, search, queue, lyrics, and settings now read `theme.Color*`. The remaining light demo
`appui/style.go` palette is still used by `page()` and the unreachable project-scaffolding pages
(plus `NotFoundPage`), so a dark-theme 404 still renders a light card.

**B7. Search playlist filter is a stub.**
`buildSearchContent` explicitly says playlist categories are unavailable until WS-YTM adds filter tokens.
Do not “fix” it in UI without the backend.

**B8. `appui/lyrics_view.go` naming collision.**
It imports `chordly/internal/state` as `appstate` because `github.com/gogpu/ui/state` is also imported.
Keep that alias when editing.

**B9. Shared-file ownership.**
`appui/shell.go` is listed as a shared extension point. This session edited it without an explicit
`INTEGRATION` claim in `PROGRESS.md`; before further shared-file work, add a task row and claim it.

**B10. Vendor directory is currently out of sync.**
While this session was running, `go.mod` / `go.sum` were modified (SQLite-related requirements) without
regenerating `vendor/modules.txt`, so plain `go build ./...` now fails with "inconsistent vendoring".
This is **not** a UI change. Until the owner runs `go mod vendor` (or the change is reverted), build and
test with `-mod=mod`:

```sh
go build -mod=mod ./...
go test -mod=mod ./appui ./router ./internal/theme ./internal/lyrics
```

---

## 5. Verification checklist for WS-UI changes

```sh
go build ./...
go test ./appui ./router ./internal/theme ./internal/lyrics
```

If plain `go build` reports "inconsistent vendoring" (see B10), prefix both commands with `-mod=mod`.
Earlier in this session plain `go build ./...` and `go test ./...` did run; the vendoring error appeared
only after another workstream edited `go.mod`/`go.sum`.

Then manual smoke check:

1. Launch app → home feed sections render inside the shell.
2. Rail: Home / Explore / Library / Settings swap pages; active icon updates.
3. Type in the top-bar field → Enter → `/search` shows results for that query; filter pills swap sections;
   clicking a result plays it. Empty query stays on the search empty state.
4. Player queue button → `/queue`; current track highlighted.
5. Back/forward (`Router.Back`) still resolves correctly.
6. Player lyrics button → `/lyrics`; the page fetches lyrics for the current track and shows the
   loading/empty/synced states. Close returns to the previous page.
7. Settings → each of the six preset cards repaints the whole app (sidebar, player, home cards).
8. Settings → Dark/Light toggle and accent swatches derive a new palette and stay readable in both modes.
9. Settings → active preset name + accent hex update after each selection.
10. Change track (Next/Previous) → bottom player title/artist and liked state update without navigation.

Avoid running the full `go test ./...` as a UI gate until the untracked `internal/audio/device*.go`
work is committed/fixed.
