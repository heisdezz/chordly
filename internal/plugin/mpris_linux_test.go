//go:build linux

package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"

	"chordly/internal/model"
	"chordly/internal/state"
)

func TestSanitizeTrackPath(t *testing.T) {
	tests := []struct {
		input    string
		expected dbus.ObjectPath
	}{
		{"", "/org/mpris/MediaPlayer2/track/none"},
		{"abc123", "/org/mpris/MediaPlayer2/track/abc123"},
		{"track-with-hyphens", "/org/mpris/MediaPlayer2/track/track_with_hyphens"},
		{"track.with.dots&specials!", "/org/mpris/MediaPlayer2/track/track_with_dots_specials_"},
		{"a_b_c", "/org/mpris/MediaPlayer2/track/a_b_c"},
	}

	for _, tt := range tests {
		got := sanitizeTrackPath(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeTrackPath(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestBuildMetadata(t *testing.T) {
	s := &MPRISServer{}
	track := model.Track{
		ID:        "track-001",
		Title:     "Solar Drift",
		Artist:    "Aura",
		Album:     "Cosmic Horizons",
		Duration:  180 * time.Second,
		Thumbnail: "https://example.com/art.jpg",
	}

	meta := s.BuildMetadata(track)

	if id, ok := meta["mpris:trackid"].Value().(dbus.ObjectPath); !ok || id != "/org/mpris/MediaPlayer2/track/track_001" {
		t.Errorf("mpris:trackid mismatch: got %v", meta["mpris:trackid"])
	}
	if length, ok := meta["mpris:length"].Value().(int64); !ok || length != 180000000 {
		t.Errorf("mpris:length mismatch: got %v", meta["mpris:length"])
	}
	if title, ok := meta["xesam:title"].Value().(string); !ok || title != "Solar Drift" {
		t.Errorf("xesam:title mismatch: got %v", meta["xesam:title"])
	}
	if artists, ok := meta["xesam:artist"].Value().([]string); !ok || len(artists) != 1 || artists[0] != "Aura" {
		t.Errorf("xesam:artist mismatch: got %v", meta["xesam:artist"])
	}
	if album, ok := meta["xesam:album"].Value().(string); !ok || album != "Cosmic Horizons" {
		t.Errorf("xesam:album mismatch: got %v", meta["xesam:album"])
	}
	if art, ok := meta["mpris:artUrl"].Value().(string); !ok || art != "https://example.com/art.jpg" {
		t.Errorf("mpris:artUrl mismatch: got %v", meta["mpris:artUrl"])
	}
}

func TestMPRISServerLiveLifecycle(t *testing.T) {
	srv, err := NewMPRISServer()
	if err != nil {
		t.Skipf("Skipping live D-Bus test: %v", err)
		return
	}
	defer srv.Close()

	if !strings.HasPrefix(srv.BusName(), "org.mpris.MediaPlayer2.chordly") {
		t.Fatalf("Unexpected bus name: %s", srv.BusName())
	}

	// Verify Introspection
	intro, dbusErr := srv.Introspect()
	if dbusErr != nil || !strings.Contains(intro, "org.mpris.MediaPlayer2.Player") {
		t.Fatalf("Introspect failed or missing player interface: %v", dbusErr)
	}

	// Verify Root Properties
	rootProps, dbusErr := srv.GetAll(mprisRootIface)
	if dbusErr != nil {
		t.Fatalf("GetAll(root) failed: %v", dbusErr)
	}
	if id, ok := rootProps["Identity"].Value().(string); !ok || id != "Chordly" {
		t.Errorf("Identity mismatch: got %v", rootProps["Identity"])
	}

	// Test Transport Controls via Player Interface
	_ = srv.Play()
	if state.PlaybackStatus.Get() != model.StatePlaying {
		t.Errorf("Play() failed to set StatePlaying: got %v", state.PlaybackStatus.Get())
	}

	_ = srv.Pause()
	if state.PlaybackStatus.Get() != model.StatePaused {
		t.Errorf("Pause() failed to set StatePaused: got %v", state.PlaybackStatus.Get())
	}

	_ = srv.PlayPause()
	if state.PlaybackStatus.Get() != model.StatePlaying {
		t.Errorf("PlayPause() failed to toggle: got %v", state.PlaybackStatus.Get())
	}

	// Test Seek and Playhead
	state.Playhead.Set(10 * time.Second)
	_ = srv.Seek(5000000) // +5 seconds in microseconds
	if state.Playhead.Get() != 15*time.Second {
		t.Errorf("Seek() playhead mismatch: got %v, want 15s", state.Playhead.Get())
	}

	// Test Volume Set
	_ = srv.Set(mprisPlayerIface, "Volume", dbus.MakeVariant(float64(0.55)))
	if state.Volume.Get() < 0.54 || state.Volume.Get() > 0.56 {
		t.Errorf("Set Volume mismatch: got %v", state.Volume.Get())
	}

	// Test Raise & Quit Callbacks
	raised := false
	srv.SetRaiseHandler(func() { raised = true })
	_ = srv.Raise()
	if !raised {
		t.Errorf("Raise() handler not called")
	}

	quitted := false
	srv.SetQuitHandler(func() { quitted = true })
	_ = srv.Quit()
	if !quitted {
		t.Errorf("Quit() handler not called")
	}
}

func TestMPRISClientDBusQuery(t *testing.T) {
	srv, err := NewMPRISServer()
	if err != nil {
		t.Skipf("Skipping live client D-Bus query: %v", err)
		return
	}
	defer srv.Close()

	// Connect as an independent external client over session bus
	clientConn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatalf("Client failed to connect to session bus: %v", err)
	}
	defer clientConn.Close()

	obj := clientConn.Object(srv.BusName(), dbus.ObjectPath(mprisObjectPath))

	// Remote Call: PlayPause
	state.PlaybackStatus.Set(model.StatePlaying)
	call := obj.Call("org.mpris.MediaPlayer2.Player.PlayPause", 0)
	if call.Err != nil {
		t.Fatalf("Remote PlayPause failed: %v", call.Err)
	}

	if state.PlaybackStatus.Get() != model.StatePaused {
		t.Errorf("Remote PlayPause did not toggle state: got %v", state.PlaybackStatus.Get())
	}

	// Remote Property Query: Identity
	variant, err := obj.GetProperty("org.mpris.MediaPlayer2.Identity")
	if err != nil {
		t.Fatalf("Remote GetProperty(Identity) failed: %v", err)
	}
	if name, ok := variant.Value().(string); !ok || name != "Chordly" {
		t.Errorf("Remote Identity property mismatch: got %v", variant.Value())
	}
}
