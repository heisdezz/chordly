package appui

import (
	"testing"

	"chordly/router"
)

func TestSettingsPageBuilds(t *testing.T) {
	page := SettingsPage(router.RouteContext{})
	if page == nil {
		t.Fatal("SettingsPage returned nil")
	}
}
