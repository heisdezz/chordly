# Task Specification & Resolution Note: TSK-213 / UI-INF-01 (GoGPU Core Component Adoption)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-213` (alias `UI-INF-01`)  
**Parent Plan:** [`docs/ui/pages-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/pages-plan.md)  
**File Scope:**

- [`appui/textfield.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/textfield.go)
- [`appui/page_scaffold.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/page_scaffold.go)
- [`appui/page_scaffold_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/page_scaffold_test.go)

---

## 1. Objective

Integrate GoGPU upstream core components (`core/textfield`, `core/scrollview`, `core/listview`, `core/dialog`) into `appui` to power standard pages, virtualized lists, scroll containers, and input fields.

---

## 2. Core Components Adopted

1. **`core/textfield`**:
   - Exposed via `NewSearchField(placeholder, initial, onChange, onSubmit)` in [`appui/textfield.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/textfield.go).
   - Provides native selection, clipboard shortcuts (Ctrl+A/C/X/V), and search input styling.
2. **`core/scrollview`**:
   - Integrated into `NewPageScaffold(cfg)` and `NewScrollableBody(content)` in [`appui/page_scaffold.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/page_scaffold.go).
   - Automatically handles vertical smooth scrolling with auto-hiding scrollbars and bottom dock padding (96px).
3. **`core/listview`**:
   - Wrapped via `NewListView(itemCount, itemHeight, buildItem)` in [`appui/page_scaffold.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/page_scaffold.go).
   - Provides virtualized item recycling for track lists and long scroll views.
4. **`core/dialog`**:
   - `NewAlertDialog(title, message, onDismiss)` and `NewConfirmDialog(title, message, onCancel, onConfirm)` for themed modal prompts.

---

## 3. Verification & Compatibility

- Replaced non-standard layout methods with GoGPU primitive methods (`PaddingLeft`, `PaddingRight`, `PaddingTop`, `PaddingBottom`, `PaddingXY`).
- Themed text with Metrolist slate/mint colors (`theme.ColorTextPrimary`, `theme.ColorTextSecondary`).
- Passes all tests with `go test -v ./appui/...` and compiles cleanly in `go build ./...`.

---

## 4. Status

- **Status:** `DONE`
