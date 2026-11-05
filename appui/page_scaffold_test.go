package appui

import (
	"testing"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

func TestPageScaffold_DefaultConfig(t *testing.T) {
	cfg := DefaultPageScaffoldConfig()
	if !cfg.Scrollable {
		t.Error("expected Scrollable to be true by default")
	}
	if cfg.PaddingBottom != 96 {
		t.Errorf("expected PaddingBottom 96, got %f", cfg.PaddingBottom)
	}
	if cfg.PaddingX != 24 {
		t.Errorf("expected PaddingX 24, got %f", cfg.PaddingX)
	}
}

func TestPageScaffold_Creation(t *testing.T) {
	body := primitives.Text("Page Content")
	actions := primitives.Text("Actions")

	scaffold := NewPageScaffold(PageScaffoldConfig{
		Title:    "Test Title",
		Subtitle: "Test Subtitle",
		Actions:  actions,
		Body:     body,
	})

	if scaffold == nil {
		t.Fatal("expected non-nil scaffold widget")
	}
}

func TestPageScaffold_NonScrollable(t *testing.T) {
	body := primitives.Text("Unscrollable Content")
	scaffold := NewPageScaffold(PageScaffoldConfig{
		Title:      "Static Page",
		Body:       body,
		Scrollable: false,
	})

	if scaffold == nil {
		t.Fatal("expected non-nil unscrollable scaffold widget")
	}
}

func TestSimplePageScaffold(t *testing.T) {
	body := primitives.Text("Simple Body")
	scaffold := NewSimplePageScaffold("Simple Page", body)
	if scaffold == nil {
		t.Fatal("expected non-nil simple scaffold widget")
	}
}

func TestScrollableBody(t *testing.T) {
	content := primitives.Text("Long content")
	sv := NewScrollableBody(content)
	if sv == nil {
		t.Fatal("expected non-nil scrollview widget")
	}
}

func TestListView(t *testing.T) {
	lv := NewListView(10, 48, func(index int) widget.Widget {
		return primitives.Box()
	})
	if lv == nil {
		t.Fatal("expected non-nil listview widget")
	}
}

func TestDialogConstructors(t *testing.T) {
	dismissed := false
	alert := NewAlertDialog("Alert Title", "Alert Message", func() {
		dismissed = true
	})
	if alert == nil {
		t.Fatal("expected non-nil alert dialog")
	}

	confirmed := false
	canceled := false
	confirm := NewConfirmDialog("Confirm Title", "Confirm Message", func() {
		canceled = true
	}, func() {
		confirmed = true
	})
	if confirm == nil {
		t.Fatal("expected non-nil confirm dialog")
	}

	_ = dismissed
	_ = confirmed
	_ = canceled
}
