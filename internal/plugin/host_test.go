package plugin

import (
	"context"
	"sync"
	"testing"
	"time"

	"chordly/internal/model"
)

// validWasmBytecode defines a valid WASM binary exporting memory, on_track_change, and on_playhead_tick.
var validWasmBytecode = []byte{
	// WASM Header
	0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00,

	// Type Section (1): 2 types (length = 8 bytes)
	// 0: () -> ()
	// 1: (i32) -> ()
	0x01, 0x08, 0x02,
	0x60, 0x00, 0x00,
	0x60, 0x01, 0x7F, 0x00,

	// Function Section (3): 2 functions (length = 3 bytes)
	0x03, 0x03, 0x02, 0x00, 0x01,

	// Memory Section (5): 1 memory (min 1 page) (length = 3 bytes)
	0x05, 0x03, 0x01, 0x00, 0x01,

	// Export Section (7): 3 exports (length = 47 / 0x2F bytes)
	// "memory" -> mem 0
	// "on_track_change" -> func 0
	// "on_playhead_tick" -> func 1
	0x07, 0x2F, 0x03,
	0x06, 'm', 'e', 'm', 'o', 'r', 'y', 0x02, 0x00,
	0x0F, 'o', 'n', '_', 't', 'r', 'a', 'c', 'k', '_', 'c', 'h', 'a', 'n', 'g', 'e', 0x00, 0x00,
	0x10, 'o', 'n', '_', 'p', 'l', 'a', 'y', 'h', 'e', 'a', 'd', '_', 't', 'i', 'c', 'k', 0x00, 0x01,

	// Code Section (10): 2 function bodies (length = 9 bytes)
	// func 0: 0 locals, nop, end
	// func 1: 0 locals, nop, end
	0x0A, 0x09, 0x02,
	0x03, 0x00, 0x01, 0x0B,
	0x03, 0x00, 0x01, 0x0B,
}

// trappingWasmBytecode defines a valid WASM binary whose on_track_change executes unreachable (traps).
var trappingWasmBytecode = []byte{
	// WASM Header
	0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00,

	// Type Section (1): 1 type: () -> () (length = 4 bytes)
	0x01, 0x04, 0x01, 0x60, 0x00, 0x00,

	// Function Section (3): 1 function: type 0 (length = 2 bytes)
	0x03, 0x02, 0x01, 0x00,

	// Export Section (7): 1 export: "on_track_change" -> func 0 (length = 19 / 0x13 bytes)
	0x07, 0x13, 0x01,
	0x0F, 'o', 'n', '_', 't', 'r', 'a', 'c', 'k', '_', 'c', 'h', 'a', 'n', 'g', 'e', 0x00, 0x00,

	// Code Section (10): 1 body: 0 locals, unreachable (0x00), end (0x0B) (length = 5 bytes)
	0x0A, 0x05, 0x01,
	0x03, 0x00, 0x00, 0x0B,
}

func TestHostLifecycleAndEvents(t *testing.T) {
	ctx := context.Background()

	var mu sync.Mutex
	logs := make([]string, 0)

	cfg := HostConfig{
		CurrentTrackFunc: func() model.Track {
			return model.Track{
				ID:       "test-id",
				Title:    "Voyager",
				Artist:   "Daft Punk",
				Duration: 227 * time.Second,
			}
		},
		LogHook: func(pluginID string, level int, message string) {
			mu.Lock()
			logs = append(logs, message)
			mu.Unlock()
		},
	}

	host, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer host.Close(ctx)

	// Load valid plugin
	meta := PluginMetadata{
		ID:          "my-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Author:      "Chordly",
		Description: "Sample plugin",
	}

	inst, err := host.LoadPlugin(ctx, "my-plugin", validWasmBytecode, meta)
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}

	if inst.IsQuarantined() {
		t.Errorf("expected plugin to not be quarantined")
	}

	// Dispatch track change
	host.NotifyTrackChange(ctx)

	// Dispatch playhead tick
	host.NotifyPlayheadTick(ctx, 15000) // 15 seconds

	// Unload plugin
	if err := host.UnloadPlugin(ctx, "my-plugin"); err != nil {
		t.Fatalf("UnloadPlugin failed: %v", err)
	}
}

func TestHostQuarantineOnCrash(t *testing.T) {
	ctx := context.Background()

	var mu sync.Mutex
	quarantinedLogged := false

	cfg := HostConfig{
		LogHook: func(pluginID string, level int, message string) {
			mu.Lock()
			if pluginID == "crashing-plugin" {
				quarantinedLogged = true
			}
			mu.Unlock()
		},
	}

	host, err := NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer host.Close(ctx)

	meta := PluginMetadata{
		ID:      "crashing-plugin",
		Name:    "Buggy Plugin",
		Version: "0.1.0",
	}

	inst, err := host.LoadPlugin(ctx, "crashing-plugin", trappingWasmBytecode, meta)
	if err != nil {
		t.Fatalf("LoadPlugin failed: %v", err)
	}

	// Notify track change should trigger the unreachable trap
	host.NotifyTrackChange(ctx)

	// Verify plugin is safely quarantined without crashing host
	if !inst.IsQuarantined() {
		t.Fatalf("expected plugin to be quarantined after unreachable trap")
	}

	if inst.QuarantineReason() == "" {
		t.Errorf("expected non-empty quarantine reason")
	}

	mu.Lock()
	logged := quarantinedLogged
	mu.Unlock()

	if !logged {
		t.Errorf("expected quarantine log message")
	}

	// Further notifications should skip quarantined plugin gracefully
	host.NotifyTrackChange(ctx)
}
