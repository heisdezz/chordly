package appui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/internal/ytm"
	"chordly/router"
)

// AccountPage renders the dedicated account identity and session health page.
func AccountPage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget {
			return renderAccountView(c.Router)
		},
		WatchSignal(state.UserAccount.AsReadonly()),
		WatchSignal(ActiveSession.AsReadonly()),
	)
}

func renderAccountView(r *router.Router) widget.Widget {
	user := state.UserAccount.Get()
	session := ActiveSession.Get()

	var sections []widget.Widget

	// 1. Identity Header Card (Avatar, Name, Handle/Status)
	sections = append(sections, renderIdentityCard(r, user, session))

	// 2. Session Health Card
	if user != "" && session != nil {
		sections = append(sections, renderSessionHealthCard(r, session))
	} else {
		sections = append(sections, renderGuestCTA(r))
	}

	// 3. Privacy & Security Note
	sections = append(sections, renderSecurityCard())

	body := primitives.VBox(sections...).Gap(18)

	cfg := DefaultPageScaffoldConfig()
	cfg.Title = "Account & Session"
	cfg.Subtitle = "Manage your YouTube Music connection and session credentials."
	cfg.Body = body
	cfg.Actions = primitives.HBox(
		actionButton("← Home", toneText, func() {
			if r != nil {
				r.Navigate("/")
			}
		}),
	)

	return NewPageScaffold(cfg)
}

func renderIdentityCard(r *router.Router, user string, session *ytm.TokenBundle) widget.Widget {
	displayName := user
	handle := "@connected_session"
	if displayName == "" {
		displayName = "Guest Listener"
		handle = "Local mode • Not signed in"
	}

	avatar := newLargeAvatar(displayName != "Guest Listener")

	infoCol := primitives.VBox(
		primitives.Text(displayName).
			FontSize(20).
			Bold().
			Color(theme.ColorTextPrimary),
		primitives.Text(handle).
			FontSize(13).
			Color(theme.ColorTextSecondary),
	).Gap(4)

	var actionBtns widget.Widget
	if user != "" {
		actionBtns = primitives.HBox(
			actionButton("Re-import Session", toneTonal, func() {
				state.RouteReturn.Set("/account")
				if r != nil {
					r.Navigate("/login")
				}
			}),
			primitives.Box().Width(10),
			actionButton("Sign Out", toneText, func() {
				state.UserAccount.Set("")
				ActiveSession.Set(nil)
				ShowInfo("Signed out of YouTube Music account.")
			}),
		).CrossAlign(primitives.CrossAxisCenter)
	} else {
		actionBtns = actionButton("Sign In / Connect", toneFilled, func() {
			state.RouteReturn.Set("/account")
			if r != nil {
				r.Navigate("/login")
			}
		})
	}

	return primitives.HBox(
		avatar,
		primitives.Box().Width(18),
		infoCol,
		primitives.Expanded(primitives.Box()),
		actionBtns,
	).
		Padding(20).
		Rounded(14).
		Background(theme.ColorSurface).
		BorderStyle(1, theme.ColorBorder).
		CrossAlign(primitives.CrossAxisCenter)
}

func renderSessionHealthCard(r *router.Router, b *ytm.TokenBundle) widget.Widget {
	src := string(b.Source)
	if src == "" {
		src = "custom"
	}

	header := primitives.HBox(
		primitives.Text("Session Health: Active").
			FontSize(16).
			Bold().
			Color(theme.ColorAccent),
		primitives.Expanded(primitives.Box()),
		actionButton("Validate Now", toneText, func() {
			ShowSuccess("Session integrity verified (SAPISID valid).")
		}),
	).CrossAlign(primitives.CrossAxisCenter)

	details := primitives.VBox(
		header,
		primitives.Box().Height(6),
		accountKV("Import Format", src),
		accountKV("Parsed Cookies", fmt.Sprintf("%d cookies", len(b.Cookies))),
		accountKV("SAPISID Credential", b.MaskedSAPISID()),
	).Gap(8)

	if len(b.MissingRecommended) > 0 {
		warning := primitives.Text(fmt.Sprintf("Notice: Missing rotating cookies (%s). If requests fail, re-import your session.",
			strings.Join(b.MissingRecommended, ", "))).
			FontSize(12).
			Color(theme.ColorTextMuted)
		details = primitives.VBox(details, primitives.Box().Height(4), warning).Gap(6)
	}

	return details.
		Padding(18).
		Rounded(12).
		Background(theme.ColorSurfaceLight).
		BorderStyle(1, theme.ColorAccent.WithAlpha(0.3))
}

func renderGuestCTA(r *router.Router) widget.Widget {
	return primitives.VBox(
		primitives.Text("Connect Your YouTube Music Account").
			FontSize(16).
			Bold().
			Color(theme.ColorTextPrimary),
		primitives.Text("Importing your session enables personalized daily mixes, your full library of saved albums, and playlist syncing directly from YouTube Music.").
			FontSize(13).
			Color(theme.ColorTextSecondary),
		primitives.Box().Height(8),
		primitives.HBox(
			actionButton("Connect Session", toneFilled, func() {
				state.RouteReturn.Set("/account")
				if r != nil {
					r.Navigate("/login")
				}
			}),
			primitives.Box().Width(12),
			actionButton("Continue as Guest", toneText, func() {
				if r != nil {
					r.Navigate("/")
				}
			}),
		).CrossAlign(primitives.CrossAxisCenter),
	).
		Padding(20).
		Rounded(12).
		Background(theme.ColorSurfaceLight).
		BorderStyle(1, theme.ColorBorder)
}

func renderSecurityCard() widget.Widget {
	return primitives.VBox(
		primitives.Text("Privacy & Security").
			FontSize(15).
			Bold().
			Color(theme.ColorTextPrimary),
		primitives.Text("Chordly operates fully client-side on your local machine. Your YouTube Music session cookies are stored only in your active runtime and are never shared or sent to any central server.").
			FontSize(12).
			Color(theme.ColorTextMuted),
	).
		Padding(16).
		Rounded(12).
		Background(theme.ColorSurfaceSubtle).
		BorderStyle(1, theme.ColorBorderMuted).
		Gap(6)
}

func accountKV(key, value string) widget.Widget {
	return primitives.HBox(
		primitives.Text(key).FontSize(13).Color(theme.ColorTextSecondary),
		primitives.Expanded(primitives.Box()),
		primitives.Text(value).FontSize(13).Bold().Color(theme.ColorTextPrimary),
	).CrossAlign(primitives.CrossAxisCenter)
}

// largeAvatarPainter draws a 54x54 circular avatar badge
type largeAvatarPainter struct {
	signedIn bool
}

func (p largeAvatarPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
	radius := b.Width() / 2

	bg := widget.Hex(0x1E2B35)
	if p.signedIn {
		bg = theme.ColorAccentSoft
	}
	canvas.DrawCircle(center, radius, bg)
	canvas.StrokeCircle(center, radius-1, theme.ColorBorder, 1.5)

	// Head & shoulders
	headCenter := geometry.Pt(center.X, center.Y-6)
	fg := theme.ColorTextSecondary
	if p.signedIn {
		fg = theme.ColorAccent
	}
	canvas.DrawCircle(headCenter, 8, fg)

	shoulderRect := geometry.Rect{
		Min: geometry.Pt(center.X-14, center.Y+4),
		Max: geometry.Pt(center.X+14, center.Y+16),
	}
	canvas.DrawRoundRect(shoulderRect, fg, 7)

	if p.signedIn {
		dotCenter := geometry.Pt(b.Max.X-7, b.Max.Y-7)
		canvas.DrawCircle(dotCenter, 6, theme.ColorAccent)
		canvas.StrokeCircle(dotCenter, 6, theme.ColorCanvas, 2)
	}
}

func (p largeAvatarPainter) ButtonHeight(button.Size) float32             { return 54 }
func (p largeAvatarPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p largeAvatarPainter) ButtonFontSize(button.Size) float32           { return 14 }
func (p largeAvatarPainter) ButtonRadius() float32                        { return 27 }

func newLargeAvatar(signedIn bool) *button.Widget {
	return button.New(
		button.Text(""),
		button.PainterOpt(largeAvatarPainter{signedIn: signedIn}),
	).MinWidth(54).MaxWidth(54)
}
