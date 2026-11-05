package appui

import (
	"fmt"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"chordly/internal/state"
	"chordly/internal/theme"
	"chordly/router"
)

// accentSwatches are the quick-pick custom accents. A full hex text field is
// not available yet because the vendored GoGPU build has no text-input widget.
var accentSwatches = []widget.Color{
	widget.Hex(0x5CE1B6), // mint (default)
	widget.Hex(0x6750A4), // material violet
	widget.Hex(0x3B82F6), // blue
	widget.Hex(0xE2556E), // rose
	widget.Hex(0xD85D38), // amber
	widget.Hex(0x10B981), // emerald
	widget.Hex(0x06B6D4), // cyan
	widget.Hex(0xEC4899), // pink
}

// SettingsPage renders appearance, playback, and about sections.
// It rebuilds on theme changes (via the shell) and on reduced-motion changes.
func SettingsPage(c router.RouteContext) widget.Widget {
	return newReactiveView(
		func() widget.Widget { return buildSettingsContent(c) },
		WatchSignal(SigReducedMotion.AsReadonly()),
		WatchSignal(state.UserAccount),
	)
}

func buildSettingsContent(c router.RouteContext) widget.Widget {
	header := primitives.VBox(
		primitives.Text("Settings").FontSize(28).Bold().Color(theme.ColorTextPrimary),
		primitives.Text("Appearance, account, playback, and application details.").
			FontSize(13).
			Color(theme.ColorTextSecondary),
	).Gap(4)

	pageVBox := primitives.VBox(
		header,
		primitives.Box().Height(4),
		buildAccountCard(c.Router),
		buildAppearanceCard(),
		primitives.HBox(
			primitives.Expanded(buildPlaybackCard()),
			primitives.Box().Width(12),
			primitives.Expanded(buildAboutCard()),
		),
	).
		Padding(24).
		Gap(16).
		CrossAlign(primitives.CrossAxisStretch).
		Background(theme.ColorCanvas)

	return NewScrollableBody(pageVBox)
}

func buildAccountCard(r *router.Router) widget.Widget {
	currAccount := state.UserAccount.Get()
	var body []widget.Widget

	if currAccount == "" {
		body = append(body,
			settingsKeyValue("Status", "Guest Mode (Not signed in)"),
			settingsKeyValue("Library", "Guest preview; sign in to sync favorites, history, and playlists"),
			primitives.Box().Height(6),
			primitives.HBox(
				button.New(
					button.Text("Sign In / Import Session"),
					button.VariantOpt(button.Filled),
					button.OnClick(func() {
						state.RouteReturn.Set("/settings")
						if r != nil {
							r.Navigate("/login")
						}
					}),
				),
			),
		)
	} else {
		body = append(body,
			settingsKeyValue("Status", "Connected"),
			settingsKeyValue("Account", currAccount),
			primitives.Box().Height(6),
			primitives.HBox(
				button.New(
					button.Text("Manage Session"),
					button.VariantOpt(button.Tonal),
					button.OnClick(func() {
						state.RouteReturn.Set("/settings")
						if r != nil {
							r.Navigate("/login")
						}
					}),
				),
				primitives.Box().Width(10),
				button.New(
					button.Text("Sign Out"),
					button.VariantOpt(button.Outlined),
					button.OnClick(func() {
						state.UserAccount.Set("")
						ActiveSession.Set(nil)
						ShowInfo("Signed out of YouTube Music account.")
					}),
				),
			),
		)
	}

	return settingsCard(
		"Account & Sync",
		"YouTube Music web session and library synchronization.",
		body...,
	)
}

func buildAppearanceCard() widget.Widget {
	presets := theme.ListPresets()
	rows := make([]widget.Widget, 0, (len(presets)+2)/3)
	for i := 0; i < len(presets); i += 3 {
		cells := make([]widget.Widget, 0, 5)
		for j := 0; j < 3; j++ {
			cell := widget.Widget(primitives.Box())
			if i+j < len(presets) {
				cell = presetCard(presets[i+j])
			}
			if j > 0 {
				cells = append(cells, primitives.Box().Width(10))
			}
			cells = append(cells, primitives.Expanded(cell))
		}
		rows = append(rows, primitives.HBox(cells...))
	}

	modeRow := primitives.HBox(
		modeButton("Dark", true),
		primitives.Box().Width(8),
		modeButton("Light", false),
	)

	swatches := make([]widget.Widget, 0, len(accentSwatches))
	for i, color := range accentSwatches {
		if i > 0 {
			swatches = append(swatches, primitives.Box().Width(8))
		}
		swatches = append(swatches, accentSwatch(color))
	}

	activeName := "Custom accent"
	if preset, ok := theme.GetPreset(theme.SigActivePresetID.Get()); ok {
		activeName = preset.Name
	}

	controls := primitives.HBox(
		primitives.VBox(settingsLabel("Mode"), modeRow).Gap(6),
		primitives.Box().Width(28),
		primitives.VBox(settingsLabel("Accent"), primitives.HBox(swatches...)).Gap(6),
		primitives.Expanded(primitives.Box()),
		primitives.VBox(settingsLabel("Motion"), motionToggle()).Gap(6),
		primitives.Box().Width(20),
		primitives.VBox(
			primitives.Text(activeName).FontSize(12).Bold().Color(theme.ColorAccent),
			primitives.Text(theme.FormatHexColor(theme.ColorAccent)).
				FontSize(11).
				Color(theme.ColorTextMuted),
		).Gap(2),
	).CrossAlign(primitives.CrossAxisEnd)

	return settingsCard(
		"Appearance",
		"Theme presets and accent color. Changes apply instantly.",
		primitives.VBox(rows...).Gap(10),
		settingsDivider(),
		controls,
	)
}

func buildPlaybackCard() widget.Widget {
	track := state.CurrentTrack.Get()
	queue := state.Queue.Get()

	nowPlaying := track.Title
	if nowPlaying == "" {
		nowPlaying = "Nothing playing"
	}

	return settingsCard(
		"Playback",
		"Current output and queue state.",
		settingsKeyValue("Now playing", nowPlaying),
		settingsKeyValue("Queue", fmt.Sprintf("%d tracks", len(queue))),
		settingsKeyValue("Volume", fmt.Sprintf("%.0f%%", state.Volume.Get()*100)),
		settingsKeyValue("Stream", "Opus 48 kHz preferred, AAC fallback"),
	)
}

func buildAboutCard() widget.Widget {
	return settingsCard(
		"About",
		"Chordly desktop YouTube Music player.",
		settingsKeyValue("Renderer", "GoGPU UI / WGSL"),
		settingsKeyValue("Backend", "InnerTube WEB_REMIX"),
		settingsKeyValue("Version", "0.1.0"),
	)
}

// --- Layout helpers ---

func settingsCard(title, subtitle string, body ...widget.Widget) widget.Widget {
	children := make([]widget.Widget, 0, len(body)+3)
	children = append(children,
		primitives.Text(title).FontSize(17).Bold().Color(theme.ColorTextPrimary),
		primitives.Text(subtitle).FontSize(12).Color(theme.ColorTextMuted),
		primitives.Box().Height(4),
	)
	children = append(children, body...)

	return primitives.VBox(children...).
		Padding(18).
		Gap(10).
		CrossAlign(primitives.CrossAxisStretch).
		Background(theme.ColorSurface).
		Rounded(16).
		BorderStyle(1, theme.ColorBorder)
}

func settingsLabel(text string) widget.Widget {
	return primitives.Text(text).FontSize(12).Bold().Color(theme.ColorTextSecondary)
}

func settingsDivider() widget.Widget {
	return primitives.Box().Height(1).Background(theme.ColorBorderMuted)
}

func settingsKeyValue(key, value string) widget.Widget {
	return primitives.HBox(
		primitives.Box(primitives.Text(key).FontSize(12).Color(theme.ColorTextMuted)).Width(110),
		primitives.Text(value).FontSize(12).Color(theme.ColorTextPrimary),
	).Gap(12).CrossAlign(primitives.CrossAxisCenter)
}

// --- Preset cards ---

type presetCardPainter struct {
	preset theme.Preset
}

func (p presetCardPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	pal := p.preset.Palette
	active := theme.SigActivePresetID.Get() == p.preset.ID

	bg := pal.SurfaceLight
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.04)
	}
	canvas.DrawRoundRect(b, bg, 14)

	if active {
		canvas.StrokeRoundRect(b, pal.Accent, 14, 2)
	} else {
		canvas.StrokeRoundRect(b, pal.Border, 14, 1)
	}

	pad := float32(14)
	titleRect := geometry.Rect{
		Min: geometry.Pt(b.Min.X+pad, b.Min.Y+12),
		Max: geometry.Pt(b.Max.X-pad, b.Min.Y+32),
	}
	title := fitText(canvas, p.preset.Name, 14, true, titleRect.Width())
	canvas.DrawText(title, titleRect, 14, pal.TextPrimary, true, widget.TextAlignLeft)

	descRect := geometry.Rect{
		Min: geometry.Pt(b.Min.X+pad, b.Min.Y+32),
		Max: geometry.Pt(b.Max.X-pad, b.Min.Y+50),
	}
	desc := fitText(canvas, p.preset.Description, 10, false, descRect.Width())
	canvas.DrawText(desc, descRect, 10, pal.TextSecondary, false, widget.TextAlignLeft)

	// Mini palette: canvas, surface, accent.
	chipY := b.Max.Y - 28
	chipW := float32(26)
	chipH := float32(14)
	chips := []struct {
		color  widget.Color
		border widget.Color
	}{
		{pal.Canvas, pal.Border},
		{pal.Surface, pal.Border},
		{pal.Accent, pal.Accent},
	}
	for i, chip := range chips {
		x := b.Min.X + pad + float32(i)*(chipW+6)
		rect := geometry.Rect{
			Min: geometry.Pt(x, chipY),
			Max: geometry.Pt(x+chipW, chipY+chipH),
		}
		canvas.DrawRoundRect(rect, chip.color, 4)
		canvas.StrokeRoundRect(rect, chip.border, 4, 1)
	}

	if active {
		dotCenter := geometry.Pt(b.Max.X-pad-4, b.Min.Y+18)
		canvas.DrawCircle(dotCenter, 4, pal.Accent)
	}
}

func (p presetCardPainter) ButtonHeight(button.Size) float32             { return 84 }
func (p presetCardPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p presetCardPainter) ButtonFontSize(button.Size) float32           { return 14 }
func (p presetCardPainter) ButtonRadius() float32                        { return 14 }

var _ button.Painter = presetCardPainter{}
var _ button.LayoutMetrics = presetCardPainter{}

func presetCard(p theme.Preset) *button.Widget {
	return button.New(
		button.Text(p.Name),
		button.OnClick(func() { theme.ApplyPreset(p) }),
		button.PainterOpt(presetCardPainter{preset: p}),
	)
}

// --- Mode toggle ---

type modeButtonPainter struct {
	dark bool
}

func (p modeButtonPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	dark := theme.SigActiveIsDark.Get()
	active := dark == p.dark

	bg := theme.ColorSurfaceLight
	fg := theme.ColorTextSecondary
	if active {
		bg = theme.ColorAccentSoft
		fg = theme.ColorAccent
	}
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.04)
	}

	canvas.DrawRoundRect(b, bg, 10)
	if !active {
		canvas.StrokeRoundRect(b, theme.ColorBorder, 10, 1)
	}

	text := "Dark"
	if !p.dark {
		text = "Light"
	}
	canvas.DrawText(text, b, 12, fg, active, widget.TextAlignCenter)
}

func (p modeButtonPainter) ButtonHeight(button.Size) float32             { return 34 }
func (p modeButtonPainter) ButtonPadding(button.Size) (float32, float32) { return 16, 0 }
func (p modeButtonPainter) ButtonFontSize(button.Size) float32           { return 12 }
func (p modeButtonPainter) ButtonRadius() float32                        { return 10 }

var _ button.Painter = modeButtonPainter{}
var _ button.LayoutMetrics = modeButtonPainter{}

func modeButton(label string, dark bool) *button.Widget {
	return button.New(
		button.Text(label),
		button.OnClick(func() {
			theme.ApplyCustomAccent(theme.ActivePalette().Accent, dark)
		}),
		button.PainterOpt(modeButtonPainter{dark: dark}),
	)
}

// --- Motion toggle ---

type motionTogglePainter struct{}

func (p motionTogglePainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	if s.Bounds.IsEmpty() {
		return
	}

	active := SigReducedMotion.Get()
	bg := theme.ColorSurfaceLight
	fg := theme.ColorTextSecondary
	if active {
		bg = theme.ColorAccentSoft
		fg = theme.ColorAccent
	}
	if s.Hovered {
		bg = bg.Lerp(widget.ColorWhite, 0.04)
	}

	canvas.DrawRoundRect(s.Bounds, bg, 9)
	if !active {
		canvas.StrokeRoundRect(s.Bounds, theme.ColorBorder, 9, 1)
	}
	canvas.DrawText(s.Text, s.Bounds, 11, fg, active, widget.TextAlignCenter)
}

func (p motionTogglePainter) ButtonHeight(button.Size) float32             { return 30 }
func (p motionTogglePainter) ButtonPadding(button.Size) (float32, float32) { return 12, 0 }
func (p motionTogglePainter) ButtonFontSize(button.Size) float32           { return 11 }
func (p motionTogglePainter) ButtonRadius() float32                        { return 9 }

var _ button.Painter = motionTogglePainter{}
var _ button.LayoutMetrics = motionTogglePainter{}

func motionToggle() *button.Widget {
	return button.New(
		button.Text("Reduce motion"),
		button.OnClick(func() { SigReducedMotion.Set(!SigReducedMotion.Get()) }),
		button.PainterOpt(motionTogglePainter{}),
	)
}

// --- Accent swatches ---

type accentSwatchPainter struct {
	color widget.Color
}

func (p accentSwatchPainter) PaintButton(canvas widget.Canvas, s button.PaintState) {
	b := s.Bounds
	if b.IsEmpty() {
		return
	}

	center := geometry.Pt(b.Min.X+b.Width()/2, b.Min.Y+b.Height()/2)
	radius := b.Width()/2 - 4

	current := theme.FormatHexColor(theme.ActivePalette().Accent)
	active := current == theme.FormatHexColor(p.color)

	canvas.DrawCircle(center, radius, p.color)

	if active {
		canvas.StrokeCircle(center, radius+2.5, theme.ColorTextPrimary, 2)
	} else {
		canvas.StrokeCircle(center, radius, theme.ColorBorder, 1)
	}
	if s.Hovered {
		canvas.StrokeCircle(center, radius+2.5, theme.ColorTextMuted, 1)
	}
}

func (p accentSwatchPainter) ButtonHeight(button.Size) float32             { return 36 }
func (p accentSwatchPainter) ButtonPadding(button.Size) (float32, float32) { return 0, 0 }
func (p accentSwatchPainter) ButtonFontSize(button.Size) float32           { return 12 }
func (p accentSwatchPainter) ButtonRadius() float32                        { return 18 }

var _ button.Painter = accentSwatchPainter{}
var _ button.LayoutMetrics = accentSwatchPainter{}

func accentSwatch(color widget.Color) *button.Widget {
	return button.New(
		button.Text(""),
		button.OnClick(func() {
			theme.ApplyCustomAccent(color, theme.SigActiveIsDark.Get())
		}),
		button.PainterOpt(accentSwatchPainter{color: color}),
	).MinWidth(36).MaxWidth(36)
}
