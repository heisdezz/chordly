# Task Specification & Resolution Note: TSK-215 (Account Page)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-215` (alias `UI-PAGE-02`)  
**Parent Plan:** [`docs/ui/pages-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/pages-plan.md)  
**File Scope:**  
- [`appui/account_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/account_view.go)
- [`appui/account_view_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/account_view_test.go)
- [`appui/sidebar.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/sidebar.go)
- [`main.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/main.go)

---

## 1. Objective

Implement the dedicated `/account` surface in Chordly:
1. **Account Identity**: Show display name, handle / account ID, and avatar placeholder.
2. **Session Health**: Show active session details (parsed cookie count, masked `SAPISID ••••`, source format `innertune`, `cookies_txt`, etc., and missing recommended cookie warnings if applicable).
3. **Account Management Actions**:
   - "Re-import Session": deep link to `/login` with return route back to `/account`.
   - "Validate Session": re-checks session integrity.
   - "Sign Out": clears active session and returns to guest mode.
4. **Guest State**: When no account is connected, provide a prominent call to action to connect an account or continue listening as guest.
5. **Sidebar Entry**: Navigation rail item or account profile button at the bottom of [`appui/sidebar.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/sidebar.go) leading directly to `/account`.

---

## 2. Technical Design & Integration

- Reactive view observing `state.UserAccount` and `ActiveSession`.
- Rendered using `NewPageScaffold(cfg)`.
- Route registered in `main.go` under `/account` pointing to `appui.AccountPage`.
- Unit tests in `appui/account_view_test.go` verifying layout in both authenticated and guest states, as well as sign-out logic.

---

## 3. Status
- **Status:** `IN_PROGRESS`
