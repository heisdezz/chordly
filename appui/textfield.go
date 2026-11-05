package appui

import (
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"chordly/internal/theme"
)

// TextField is a minimal single-line text input. The vendored GoGPU build has
// no text-input primitive, so this implements the subset needed by the top bar:
// click to focus, typing, backspace/delete, cursor movement, Enter to submit,
// and Escape to blur. It is intentionally not a full text editor (no selection
// or clipboard yet).
type TextField struct {
	widget.WidgetBase

	value       []rune
	cursor      int
	placeholder string
	fontSize    float32
	fieldWidth  float32
	height      float32

	onSubmit func(string)
	onChange func(string)

	clickPending bool
	clickLocalX  float32

	ctx widget.Context
}

// TextFieldOption configures a TextField.
type TextFieldOption func(*TextField)

// TextFieldInitial seeds the field with an existing query.
func TextFieldInitial(value string) TextFieldOption {
	return func(f *TextField) {
		f.value = []rune(value)
		f.cursor = len(f.value)
	}
}

// TextFieldOnSubmit is called when Enter is pressed.
func TextFieldOnSubmit(fn func(string)) TextFieldOption {
	return func(f *TextField) { f.onSubmit = fn }
}

// TextFieldOnChange is called after each edit.
func TextFieldOnChange(fn func(string)) TextFieldOption {
	return func(f *TextField) { f.onChange = fn }
}

// NewTextField creates a single-line input.
func NewTextField(placeholder string, opts ...TextFieldOption) *TextField {
	f := &TextField{
		placeholder: placeholder,
		fontSize:    13,
		fieldWidth:  400,
		height:      42,
	}
	f.SetVisible(true)
	f.SetEnabled(true)
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Value returns the current text.
func (f *TextField) Value() string {
	return string(f.value)
}

// SetValue replaces the text and moves the cursor to the end.
func (f *TextField) SetValue(value string) {
	f.value = []rune(value)
	f.cursor = len(f.value)
	f.notifyChange()
}

// Layout gives the field a fixed pill size, capped by available width.
func (f *TextField) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	width := f.fieldWidth
	if constraints.MaxWidth > 0 && constraints.MaxWidth < width {
		width = constraints.MaxWidth
	}
	size := geometry.Sz(width, f.height)
	f.SetBounds(geometry.FromPointSize(f.Bounds().Min, size))
	return size
}

// Draw paints the pill, optional icon, text or placeholder, and caret.
func (f *TextField) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := f.Bounds()
	if b.IsEmpty() {
		return
	}

	radius := b.Height() / 2
	bg := theme.ColorSurfaceSubtle
	if f.IsFocused() {
		bg = theme.ColorSurfaceLight
	}
	canvas.DrawRoundRect(b, bg, radius)

	border := theme.ColorBorder
	width := float32(1)
	if f.IsFocused() {
		border = theme.ColorAccent
		width = 1.5
	}
	canvas.StrokeRoundRect(b, border, radius, width)

	iconRect := geometry.Rect{
		Min: geometry.Pt(b.Min.X+14, b.Min.Y),
		Max: geometry.Pt(b.Min.X+32, b.Max.Y),
	}
	iconColor := theme.ColorTextMuted
	if f.IsFocused() {
		iconColor = theme.ColorAccent
	}
	DrawVectorIcon(canvas, IconSearch, iconRect, iconColor, 16)

	textX := b.Min.X + 38
	textRect := geometry.Rect{
		Min: geometry.Pt(textX, b.Min.Y),
		Max: geometry.Pt(b.Max.X-14, b.Max.Y),
	}

	canvas.PushClip(textRect)

	if len(f.value) == 0 {
		canvas.DrawText(f.placeholder, textRect, f.fontSize, theme.ColorTextMuted, false, widget.TextAlignLeft)
	} else {
		canvas.DrawText(string(f.value), textRect, f.fontSize, theme.ColorTextPrimary, false, widget.TextAlignLeft)
	}

	if f.IsFocused() {
		if f.clickPending {
			f.cursor = f.cursorForX(canvas, textX)
			f.clickPending = false
		}
		caretX := textX + canvas.MeasureText(string(f.value[:f.cursor]), f.fontSize, false)
		caretTop := b.Min.Y + 11
		caretBottom := b.Max.Y - 11
		canvas.DrawLine(geometry.Pt(caretX, caretTop), geometry.Pt(caretX, caretBottom), theme.ColorAccent, 1.5)
	}

	canvas.PopClip()
}

// Event handles focus, text entry, and navigation keys.
func (f *TextField) Event(ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		if ev.IsPress() && f.Bounds().Contains(ev.GlobalPosition) {
			if ctx != nil {
				ctx.RequestFocus(f)
			}
			f.clickLocalX = ev.GlobalPosition.X - f.Bounds().Min.X
			f.clickPending = true
			f.SetNeedsRedraw(true)
			if ctx != nil {
				ctx.Invalidate()
			}
			return true
		}

	case *event.KeyEvent:
		if !ev.IsPress() || ctx == nil || !ctx.IsFocused(f) {
			return false
		}
		return f.handleKey(ctx, ev)
	}

	return false
}

func (f *TextField) handleKey(ctx widget.Context, ev *event.KeyEvent) bool {
	switch ev.Key {
	case event.KeyEnter, event.KeyNumpadEnter:
		if f.onSubmit != nil {
			f.onSubmit(strings.TrimSpace(string(f.value)))
		}
		return true

	case event.KeyEscape:
		ctx.ReleaseFocus(f)
		return true

	case event.KeyBackspace:
		if f.cursor > 0 {
			f.value = append(f.value[:f.cursor-1], f.value[f.cursor:]...)
			f.cursor--
			f.edited(ctx)
		}
		return true

	case event.KeyDelete:
		if f.cursor < len(f.value) {
			f.value = append(f.value[:f.cursor], f.value[f.cursor+1:]...)
			f.edited(ctx)
		}
		return true

	case event.KeyLeft:
		if f.cursor > 0 {
			f.cursor--
			f.redraw(ctx)
		}
		return true

	case event.KeyRight:
		if f.cursor < len(f.value) {
			f.cursor++
			f.redraw(ctx)
		}
		return true

	case event.KeyHome:
		f.cursor = 0
		f.redraw(ctx)
		return true

	case event.KeyEnd:
		f.cursor = len(f.value)
		f.redraw(ctx)
		return true

	case event.KeySpace:
		if ev.Rune != 0 {
			f.insert(ev.Rune)
		} else {
			f.insert(' ')
		}
		f.edited(ctx)
		return true
	}

	// Printable characters arrive either as Rune text-input events or as a
	// regular key press carrying a rune.
	if ev.Rune != 0 && ev.Rune >= 32 {
		f.insert(ev.Rune)
		f.edited(ctx)
		return true
	}

	return false
}

func (f *TextField) insert(r rune) {
	f.value = append(f.value, 0)
	copy(f.value[f.cursor+1:], f.value[f.cursor:])
	f.value[f.cursor] = r
	f.cursor++
}

func (f *TextField) edited(ctx widget.Context) {
	f.notifyChange()
	f.redraw(ctx)
}

func (f *TextField) notifyChange() {
	if f.onChange != nil {
		f.onChange(string(f.value))
	}
}

func (f *TextField) redraw(ctx widget.Context) {
	f.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
}

func (f *TextField) cursorForX(canvas widget.Canvas, textX float32) int {
	target := f.clickLocalX - (textX - f.Bounds().Min.X)
	best := 0
	for i := 0; i <= len(f.value); i++ {
		// Measure each prefix and pick the closest gap to the click.
		width := canvas.MeasureText(string(f.value[:i]), f.fontSize, false)
		if width > target {
			break
		}
		best = i
	}
	return best
}

func (f *TextField) Children() []widget.Widget { return nil }

// Mount records the context so Unmount can release focus safely when the shell
// rebuilds (for example after a theme change).
func (f *TextField) Mount(ctx widget.Context) {
	f.ctx = ctx
}

// Unmount releases focus if this field still owns it.
func (f *TextField) Unmount() {
	if f.ctx != nil && f.ctx.IsFocused(f) {
		f.ctx.ReleaseFocus(f)
	}
	f.ctx = nil
}

var _ widget.Widget = (*TextField)(nil)
var _ widget.Lifecycle = (*TextField)(nil)
