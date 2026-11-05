package appui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gogpu/ui/primitives"
	gpstate "github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
	"chordly/router"
)

var (
	// ActiveSession holds the currently validated and active TokenBundle.
	ActiveSession = gpstate.NewSignal[*ytm.TokenBundle](nil)

	// In-flight state signals for the sign-in screen
	sigStagedBundle = gpstate.NewSignal[*ytm.TokenBundle](nil)
	sigAuthError    = gpstate.NewSignal("")
	sigInputText    = gpstate.NewSignal("")
	sigShowHelp     = gpstate.NewSignal(false)
)

// SignInPage creates the full `/login` page widget.
func SignInPage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget {
			return renderSignInView(c.Router)
		},
		WatchSignal(sigStagedBundle.AsReadonly()),
		WatchSignal(sigAuthError.AsReadonly()),
		WatchSignal(sigInputText.AsReadonly()),
		WatchSignal(sigShowHelp.AsReadonly()),
		WatchSignal(state.UserAccount.AsReadonly()),
	)
}

func renderSignInView(r *router.Router) widget.Widget {
	staged := sigStagedBundle.Get()
	authErr := sigAuthError.Get()
	currAccount := state.UserAccount.Get()

	var sections []widget.Widget

	// 1. If currently signed in, show connected account card
	if currAccount != "" {
		sections = append(sections, renderConnectedAccountCard(r, currAccount))
	}

	// 2. If a session has just been validated and staged, show preview & confirmation card
	if staged != nil {
		sections = append(sections, renderValidatedCard(r, staged))
	} else {
		// 3. Otherwise show input import card
		sections = append(sections, renderImportCard(r, authErr))
	}

	// 4. Instructions / Help Section
	sections = append(sections, renderHelpSection())

	// 5. Guest Mode / Return Option
	sections = append(sections, renderGuestFooter(r))

	body := primitives.VBox(sections...).Gap(20)

	cfg := DefaultPageScaffoldConfig()
	cfg.Title = "Connect YouTube Music"
	cfg.Subtitle = "Sign in to sync your library, favorites, and custom recommendations."
	cfg.Body = body
	cfg.Actions = primitives.HBox(
		actionButton("← Back", toneText, func() {
			navigateReturn(r)
		}),
	)

	return NewPageScaffold(cfg)
}

func renderConnectedAccountCard(r *router.Router, accountName string) widget.Widget {
	return primitives.VBox(
		primitives.HBox(
			primitives.Text("✓ Connected Account: "+accountName).
				FontSize(15).
				Bold().
				Color(theme.ColorAccent),
			primitives.Expanded(primitives.Box()),
			actionButton("Sign Out", toneText, func() {
				state.UserAccount.Set("")
				ActiveSession.Set(nil)
				ytm.DefaultService.ClearAuth()
				sigStagedBundle.Set(nil)
				ShowInfo("Signed out of YouTube Music account.")
			}),
		).CrossAlign(primitives.CrossAxisCenter),
	).
		Padding(16).
		Rounded(12).
		Background(theme.ColorSurfaceLight).
		BorderStyle(1, theme.ColorBorder)
}

func renderImportCard(r *router.Router, authErr string) widget.Widget {
	var elements []widget.Widget

	header := primitives.VBox(
		primitives.Text("Import Session from InnerTune or Web Browser").
			FontSize(17).
			Bold().
			Color(theme.ColorTextPrimary),
		primitives.Text("Paste your cookie string, JSON token export, or Netscape cookies.txt file below.").
			FontSize(13).
			Color(theme.ColorTextSecondary),
	).Gap(4)
	elements = append(elements, header)

	// Error banner if any
	if authErr != "" {
		errBox := primitives.VBox(
			primitives.HBox(
				primitives.Text("✕").FontSize(14).Bold().Color(theme.ColorHeartActive),
				primitives.Box().Width(8),
				primitives.Expanded(
					primitives.Text(authErr).FontSize(13).Color(theme.ColorHeartActive),
				),
			).CrossAlign(primitives.CrossAxisCenter),
		).
			Padding(12).
			Rounded(8).
			Background(theme.ColorHeartActive.WithAlpha(0.12)).
			BorderStyle(1, theme.ColorHeartActive.WithAlpha(0.3))

		elements = append(elements, errBox)
	}

	// Text Input Field for token / cookies
	inputField := NewSearchField(
		"Paste session cookies, JSON bundle, or token here…",
		sigInputText.Get(),
		func(val string) {
			sigInputText.Set(val)
			if authErr != "" {
				sigAuthError.Set("")
			}
		},
		func(val string) {
			processTokenInput(val)
		},
		TextFieldWidth(580),
		TextFieldHeight(42),
	)
	elements = append(elements, inputField)

	// Action buttons
	buttons := primitives.HBox(
		actionButton("Import Session", toneFilled, func() {
			processTokenInput(sigInputText.Get())
		}),
		primitives.Box().Width(12),
		actionButton("Paste from Clipboard", toneTonal, func() {
			clipText := widget.ClipboardRead()
			if clipText == "" {
				ShowWarning("Clipboard is empty.")
				return
			}
			sigInputText.Set(clipText)
			processTokenInput(clipText)
		}),
		primitives.Box().Width(12),
		actionButton("Clear", toneText, func() {
			sigInputText.Set("")
			sigAuthError.Set("")
			sigStagedBundle.Set(nil)
		}),
	).CrossAlign(primitives.CrossAxisCenter)
	elements = append(elements, buttons)

	return primitives.VBox(elements...).
		Gap(14).
		Padding(20).
		Rounded(14).
		Background(theme.ColorSurface).
		BorderStyle(1, theme.ColorBorder)
}

func renderValidatedCard(r *router.Router, b *ytm.TokenBundle) widget.Widget {
	srcName := string(b.Source)
	if srcName == "" {
		srcName = "custom session"
	}

	details := []widget.Widget{
		primitives.Text("✓ Session Successfully Validated!").
			FontSize(17).
			Bold().
			Color(theme.ColorAccent),
		primitives.Box().Height(6),
		primitives.Text(fmt.Sprintf("• Format: %s", srcName)).
			FontSize(13).
			Color(theme.ColorTextPrimary),
		primitives.Text(fmt.Sprintf("• Cookies parsed: %d", len(b.Cookies))).
			FontSize(13).
			Color(theme.ColorTextPrimary),
		primitives.Text(fmt.Sprintf("• SAPISID: %s", b.MaskedSAPISID())).
			FontSize(13).
			Color(theme.ColorTextPrimary),
	}

	if len(b.MissingRecommended) > 0 {
		warnText := fmt.Sprintf("Notice: Missing recommended cookies (%s). Basic playback works, but some library feeds may be limited.",
			strings.Join(b.MissingRecommended, ", "))
		details = append(details,
			primitives.Box().Height(4),
			primitives.Text(warnText).FontSize(12).Color(theme.ColorTextSecondary),
		)
	}

	actions := primitives.HBox(
		actionButton("Save & Continue", toneFilled, func() {
			applyValidatedSession(r, b)
		}),
		primitives.Box().Width(12),
		actionButton("Use Different Session", toneTonal, func() {
			sigStagedBundle.Set(nil)
			sigInputText.Set("")
			sigAuthError.Set("")
		}),
	).CrossAlign(primitives.CrossAxisCenter)

	details = append(details, primitives.Box().Height(10), actions)

	return primitives.VBox(details...).
		Gap(6).
		Padding(20).
		Rounded(14).
		Background(theme.ColorSurfaceLight).
		BorderStyle(1, theme.ColorAccent.WithAlpha(0.4))
}

func renderHelpSection() widget.Widget {
	isOpen := sigShowHelp.Get()

	var helpContent []widget.Widget

	helpTitle := "▶ How to get your YouTube Music session cookies"
	if isOpen {
		helpTitle = "▼ How to get your YouTube Music session cookies"
	}

	toggleBtn := actionButton(helpTitle, toneText, func() {
		sigShowHelp.Set(!isOpen)
	})
	helpContent = append(helpContent, toggleBtn)

	if isOpen {
		instructions := primitives.VBox(
			primitives.Text("Option A: From InnerTune (Android)").
				FontSize(14).Bold().Color(theme.ColorTextPrimary),
			primitives.Text("1. In InnerTune, go to Settings > Backup or Account.\n2. Export session data or copy the logged-in cookie header.\n3. Paste the contents directly into Chordly.").
				FontSize(12).Color(theme.ColorTextSecondary),
			primitives.Box().Height(8),
			primitives.Text("Option B: From Desktop Web Browser (Chrome, Firefox, Brave)").
				FontSize(14).Bold().Color(theme.ColorTextPrimary),
			primitives.Text("1. Log in to https://music.youtube.com in your web browser.\n2. Press F12 to open Developer Tools > Application (or Storage) > Cookies.\n3. Copy the cookie string or use a cookie export extension (e.g. 'Get cookies.txt LOCALLY').\n4. Paste the text or file contents into Chordly.").
				FontSize(12).Color(theme.ColorTextSecondary),
			primitives.Box().Height(8),
			primitives.Text("Security Notice:").
				FontSize(13).Bold().Color(theme.ColorTextPrimary),
			primitives.Text("Chordly runs locally on your computer. Your session cookies never leave your device and are never sent to third-party servers.").
				FontSize(12).Color(theme.ColorTextMuted),
		).
			Gap(4).
			Padding(14).
			Background(theme.ColorSurfaceSubtle).
			Rounded(10).
			BorderStyle(1, theme.ColorBorderMuted)

		helpContent = append(helpContent, instructions)
	}

	return primitives.VBox(helpContent...).Gap(8)
}

func renderGuestFooter(r *router.Router) widget.Widget {
	return primitives.HBox(
		primitives.Text("Don't have an account or want to listen anonymously?").
			FontSize(13).
			Color(theme.ColorTextSecondary),
		primitives.Expanded(primitives.Box()),
		actionButton("Continue as Guest", toneTonal, func() {
			ShowInfo("Continuing in Guest Mode.")
			navigateReturn(r)
		}),
	).
		CrossAlign(primitives.CrossAxisCenter).
		Padding(16).
		Rounded(12).
		Background(theme.ColorSurfaceSubtle)
}

// -----------------------------------------------------------------------------
// Logic Handlers
// -----------------------------------------------------------------------------

func processTokenInput(input string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		sigAuthError.Set("Please enter or paste a session token or cookie string.")
		return
	}

	bundle, err := ytm.ParseToken([]byte(trimmed))
	if err != nil {
		if errors.Is(err, ytm.ErrOAuthTokenUnsupported) {
			sigAuthError.Set("Google OAuth tokens (ya29.*) are not accepted by YouTube Music InnerTube web endpoints. Please paste your web session cookies instead.")
			return
		}
		if errors.Is(err, ytm.ErrMissingRequiredCookies) {
			sigAuthError.Set("Missing required authentication cookies: SAPISID or __Secure-3PAPISID was not found.")
			return
		}
		if errors.Is(err, ytm.ErrInputTooLarge) {
			sigAuthError.Set("Input exceeds the 2 MiB limit.")
			return
		}
		sigAuthError.Set(fmt.Sprintf("Invalid token format: %v", err))
		return
	}

	sigAuthError.Set("")
	sigStagedBundle.Set(bundle)
	ShowSuccess("Session token parsed successfully! Review and save.")
}

func applyValidatedSession(r *router.Router, b *ytm.TokenBundle) {
	ActiveSession.Set(b)
	// Apply immediately so search/player requests use the session even if the
	// main-thread subscription is not wired yet.
	ytm.DefaultService.SetAuth(b)

	accountName := b.AccountName
	if accountName == "" {
		accountName = "YouTube Music User"
	}
	state.UserAccount.Set(accountName)

	ShowSuccess("Connected to YouTube Music as " + accountName + "!")
	sigStagedBundle.Set(nil)
	sigInputText.Set("")
	sigAuthError.Set("")

	navigateReturn(r)
}

func navigateReturn(r *router.Router) {
	returnPath := state.RouteReturn.Get()
	if returnPath == "" || returnPath == "/login" {
		returnPath = "/"
	}
	if r != nil {
		r.Navigate(returnPath)
	}
}
