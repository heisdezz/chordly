# Task Specification & Resolution Note: TSK-214 (Sign-In Page & InnerTune Import Flow)

**Workstream:** `WS-UI`  
**Task ID:** `TSK-214`  
**Parent Plan:** [`docs/ui/auth-signin-plan.md`](file:///run/media/destiny/Ventoy/dr_projects/chord/docs/ui/auth-signin-plan.md)  
**File Scope:**  
- [`appui/signin_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/signin_view.go)
- [`appui/signin_view_test.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/signin_view_test.go)
- [`appui/settings_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/settings_view.go)
- [`main.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/main.go)

---

## 1. Objective

Provide the user-facing `/login` page allowing users to connect their YouTube Music session:
1. First-class **InnerTune session import** (raw cookie string, JSON token bundle, or Netscape `cookies.txt`).
2. Immediate format validation and user-friendly error diagnostics (e.g. explicitly explaining why OAuth tokens like `ya29.*` do not work on InnerTube `WEB_REMIX`).
3. Session validation preview (display name, masked SAPISID, detected cookie count).
4. Save session to state / client and navigate back via `SigRouteReturn` (fallback `/`).
5. Account settings integration: "Sign In / Connect YouTube Music" card in [`appui/settings_view.go`](file:///run/media/destiny/Ventoy/dr_projects/chord/appui/settings_view.go).

---

## 2. Technical Design & Flow

### 2.1 State Management
- `SigAuthBundle`: Active or staged `*ytm.TokenBundle`.
- `SigRouteReturn`: Route to return to after successful login (default `/`).
- `SigAuthStatus`: `AuthIdle`, `AuthValidating`, `AuthSuccess`, `AuthError`.

### 2.2 UI Structure (`SignInView`)
- Embedded in `NewPageScaffold`:
  - **Header**: Title "Connect YouTube Music", subtitle explaining session import philosophy.
  - **Cards**:
    - **Method 1: InnerTune / Cookie Import**: Input textfield (masked or multi-line paste) + "Import Session" button + "Paste from Clipboard" button.
    - **Method 2: Guest Mode**: "Continue as Guest" button returning immediately.
    - **Instructions Expander**: Step-by-step guidance for exporting cookies from InnerTune or a desktop web browser.
  - **Success / Validated Card**:
    - Masked SAPISID (`••••AB12`), cookie count, detected format (`innertune`, `cookies_txt`, etc.).
    - "Save & Continue" button.

### 2.3 Route Registration (`main.go`)
- Register `/login` route mapping to `appui.SignInPage`.
