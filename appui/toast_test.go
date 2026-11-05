package appui

import (
	"fmt"
	"testing"
)

func TestShowToastKeepsNewest(t *testing.T) {
	clearToasts()
	t.Cleanup(clearToasts)

	for i := 0; i < 6; i++ {
		ShowInfo(fmt.Sprintf("message %d", i))
	}

	got := SigToasts.Get()
	if len(got) != toastMaxVisible {
		t.Fatalf("toast count = %d, want %d", len(got), toastMaxVisible)
	}
	if got[0].Message != "message 2" {
		t.Errorf("oldest kept = %q, want message 2", got[0].Message)
	}
	if got[len(got)-1].Message != "message 5" {
		t.Errorf("newest = %q, want message 5", got[len(got)-1].Message)
	}
}

func TestToastKindsAndDurations(t *testing.T) {
	clearToasts()
	t.Cleanup(clearToasts)

	info := ShowToast(ToastInfo, "info")
	success := ShowToast(ToastSuccess, "success")
	warning := ShowToast(ToastWarning, "warning")
	failure := ShowError("failure")

	got := SigToasts.Get()
	if len(got) != 4 {
		t.Fatalf("toast count = %d, want 4", len(got))
	}
	if got[0].ID != info || got[1].ID != success || got[2].ID != warning || got[3].ID != failure {
		t.Fatal("toast IDs are not in insertion order")
	}
	if got[3].Kind != ToastError {
		t.Errorf("error kind = %v, want ToastError", got[3].Kind)
	}
	if got[3].Duration <= got[0].Duration {
		t.Errorf("error duration %v should exceed info duration %v", got[3].Duration, got[0].Duration)
	}
}

func TestDismissToast(t *testing.T) {
	clearToasts()
	t.Cleanup(clearToasts)

	id := ShowWarning("dismiss me")

	DismissToast(id)
	got := SigToasts.Get()
	if len(got) != 1 {
		t.Fatalf("toast count after dismiss = %d, want 1", len(got))
	}
	if got[0].dismissed.IsZero() {
		t.Fatal("toast was not marked as dismissing")
	}

	removeToast(id)
	if got := SigToasts.Get(); len(got) != 0 {
		t.Fatalf("toast count after removal = %d, want 0", len(got))
	}
}

func TestShowToastUnknownKindDefaultsToInfoTone(t *testing.T) {
	clearToasts()
	t.Cleanup(clearToasts)

	if accent := toastAccent(ToastKind(99)); accent != toastAccent(ToastInfo) {
		t.Error("unknown toast kind should use the info accent")
	}
}
