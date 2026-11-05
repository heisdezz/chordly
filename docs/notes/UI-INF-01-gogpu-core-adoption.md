# Task Specification & Resolution Note: UI-INF-01 (GoGPU Core Component Adoption)

**Workstream:** `WS-UI`
**Task series:** `UI-INF-01` → … (docs: `docs/ui/README.md`, "Recommended P0 order")
**File Scope:** GoGPU vendor re-vendor + `appui` call sites adopting `core/textfield`,
`core/scrollview`, `core/listview` (per `docs/ui/pages-plan.md`, `docs/ui/UI-NOTES.md`)
**Dependencies:** None (planning-level); `UI-INT-01` reactive container already DONE

---

## 1. Objective

Replace hand-rolled widget shims in `appui/` with the upstream GoGPU core primitives so
the UI consumes the framework instead of local re-implementations.

## 2. Why now (2026-11-05 context)

- `docs/ui/README.md` recommends P0 order:
  1. `UI-INF-01` — re-vendor GoGPU and adopt `core/textfield`, `core/scrollview`,
     `core/listview` (module has them; the vendor tree did not).
  2. `UI-AUTH-01` → `UI-AUTH-03` — auth token/import.
  3. `UI-PAGE-01` → `UI-PAGE-03 — …
- `docs/notes/UI-NOTES.md` records that the hand-rolled `appui/textfield.go` +
  `appui/textfit.go` were created during `UI-INT-03` because the vendored tree lacked
  `core/textfield`; the loading/skeleton + spinner shims (`appui/loading.go`) are the
  latest in-flight adopters of the same pattern and are currently failing to compile
  (`*Skeleton`/`*Spinner` do not implement `widget.Widget`, missing `Event`).

## 3. Critical current state (as of this task pick)

- `go build ./...` → **RED** on untracked in-flight WIP: `appui/loading.go` uses
  `*Skeleton` / `*Spinner` where a `widget.Widget` is required (`primitives.HBox`,
  `primitives.VBox`, return positions).
- `go test ./...` was green immediately beforehand; red states appeared transiently
  mid-write during concurrent agent sessions and resolved on their own.
- Working-tree WIP batch (untracked): `appui/home_view.go`, `appui/reactive.go`,
  `appui/settings_view.go`, `appui/textfield.go`, `appui/textfit.go`, `appui/loading.go`,
  `internal/audio/{device,decoder}.{go,test}`, `internal/theme/active*.go`,
  `internal/plugin/mpris_*`. Modified (mid-flight too): `appui/shell.go`, `topbar.go`,
  `sidebar.go`, `bottom_player.go`, `hero.go`, etc.

## 4. Acceptance

- Vendored/adopted `core/textfield`, `core/scrollview`, `core/listview` compile into
  `appui` without hand-rolled duplicates.
- `go build ./...` and `go test ./...` green after adoption.
