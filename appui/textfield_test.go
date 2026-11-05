package appui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

func textKey(r rune) *event.KeyEvent {
	return event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone)
}

func controlKey(k event.Key) *event.KeyEvent {
	return event.NewKeyEvent(event.KeyPress, k, 0, event.ModNone)
}

func clickField(t *testing.T, f *TextField, ctx widget.Context) {
	t.Helper()
	pos := geometry.Pt(f.Bounds().Min.X+10, f.Bounds().Min.Y+10)
	ev := event.NewMouseEvent(event.MousePress, event.ButtonLeft, 0, pos, pos, 0)
	if !f.Event(ctx, ev) {
		t.Fatal("mouse press was not handled")
	}
	if !ctx.IsFocused(f) {
		t.Fatal("field did not receive focus on click")
	}
}

func TestTextFieldLayout(t *testing.T) {
	f := NewTextField("Search")

	size := f.Layout(nil, geometry.Constraints{MaxWidth: 500, MaxHeight: 100})
	if size.Width != 400 || size.Height != 42 {
		t.Fatalf("default size = %v, want 400x42", size)
	}

	size = f.Layout(nil, geometry.Constraints{MaxWidth: 250, MaxHeight: 100})
	if size.Width != 250 {
		t.Fatalf("narrow width = %v, want 250", size.Width)
	}
}

func TestTextFieldTypingSubmitAndBlur(t *testing.T) {
	var submitted string
	f := NewTextField("Search", TextFieldOnSubmit(func(query string) { submitted = query }))
	ctx := widget.NewContext()
	f.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(400, 42)})

	if f.Event(ctx, textKey('x')) {
		t.Fatal("unfocused field handled a key event")
	}

	clickField(t, f, ctx)
	for _, r := range "daft" {
		if !f.Event(ctx, textKey(r)) {
			t.Fatalf("key %q was not handled", r)
		}
	}
	if got := f.Value(); got != "daft" {
		t.Fatalf("value = %q, want daft", got)
	}

	f.Event(ctx, controlKey(event.KeyBackspace))
	if got := f.Value(); got != "daf" {
		t.Fatalf("after backspace value = %q, want daf", got)
	}

	f.Event(ctx, controlKey(event.KeyLeft))
	f.Event(ctx, textKey('X'))
	if got := f.Value(); got != "daXf" {
		t.Fatalf("after left+insert value = %q, want daXf", got)
	}

	f.Event(ctx, controlKey(event.KeyEnter))
	if submitted != "daXf" {
		t.Fatalf("submitted = %q, want daXf", submitted)
	}

	f.Event(ctx, controlKey(event.KeyEscape))
	if ctx.IsFocused(f) {
		t.Fatal("escape did not blur the field")
	}
}

func TestTextFieldSpaceAndDelete(t *testing.T) {
	f := NewTextField("Search", TextFieldInitial("ab"))
	ctx := widget.NewContext()
	f.SetBounds(geometry.Rect{Min: geometry.Pt(0, 0), Max: geometry.Pt(400, 42)})
	clickField(t, f, ctx)

	f.Event(ctx, controlKey(event.KeyHome))
	f.Event(ctx, controlKey(event.KeySpace))
	if got := f.Value(); got != " ab" {
		t.Fatalf("space insert value = %q, want %q", got, " ab")
	}

	f.Event(ctx, controlKey(event.KeyEnd))
	f.Event(ctx, controlKey(event.KeyBackspace))
	if got := f.Value(); got != " a" {
		t.Fatalf("backspace value = %q, want %q", got, " a")
	}

	f.Event(ctx, controlKey(event.KeyHome))
	f.Event(ctx, controlKey(event.KeyDelete))
	if got := f.Value(); got != "a" {
		t.Fatalf("delete value = %q, want a", got)
	}
}
