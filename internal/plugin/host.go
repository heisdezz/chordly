package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"chordly/internal/model"
)

var (
	// ErrPluginQuarantined is returned when an operation is attempted on a quarantined plugin.
	ErrPluginQuarantined = errors.New("plugin is quarantined due to a previous crash or timeout")
)

// PluginMetadata holds plugin description and version info.
type PluginMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
}

// PluginInstance represents a loaded and sandboxed WebAssembly plugin.
type PluginInstance struct {
	Meta             PluginMetadata
	module           api.Module
	isQuarantined    bool
	quarantineReason string

	onTrackChange  api.Function
	onPlayheadTick api.Function
}

// IsQuarantined returns whether the plugin has been isolated from the event bus.
func (p *PluginInstance) IsQuarantined() bool {
	return p.isQuarantined
}

// QuarantineReason returns the error that triggered the quarantine.
func (p *PluginInstance) QuarantineReason() string {
	return p.quarantineReason
}

// HostConfig holds options for initializing the Wazero plugin runtime.
type HostConfig struct {
	CurrentTrackFunc func() model.Track
	LogHook          func(pluginID string, level int, message string)
	HTTPClient       *http.Client
}

// Host manages the sandboxed Wazero WASM runtime and plugin lifecycles.
type Host struct {
	runtime wazero.Runtime
	cfg     HostConfig

	mu      sync.RWMutex
	plugins map[string]*PluginInstance
}

// NewHost creates a new WebAssembly plugin host with safe sandboxing.
func NewHost(ctx context.Context, cfg HostConfig) (*Host, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}

	r := wazero.NewRuntime(ctx)

	h := &Host{
		runtime: r,
		cfg:     cfg,
		plugins: make(map[string]*PluginInstance),
	}

	// Register the "env" host module with exported ABI functions
	_, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(h.hostLog).
		Export("host_log").
		NewFunctionBuilder().
		WithFunc(h.hostGetCurrentTrack).
		Export("host_get_current_track").
		NewFunctionBuilder().
		WithFunc(h.hostHTTPFetch).
		Export("host_http_fetch").
		Instantiate(ctx)

	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("instantiate env host module: %w", err)
	}

	return h, nil
}

// hostLog ABI: host_log(level i32, msg_ptr i32, msg_len i32)
func (h *Host) hostLog(ctx context.Context, mod api.Module, level uint32, msgPtr, msgLen uint32) {
	mem := mod.Memory()
	if mem == nil {
		return
	}
	bytes, ok := mem.Read(msgPtr, msgLen)
	if !ok {
		return
	}
	msg := string(bytes)
	if h.cfg.LogHook != nil {
		h.cfg.LogHook(mod.Name(), int(level), msg)
	}
}

// hostGetCurrentTrack ABI: host_get_current_track(out_ptr i32, out_max_len i32) -> i32 (bytes written, or -1)
func (h *Host) hostGetCurrentTrack(ctx context.Context, mod api.Module, outPtr, outMaxLen uint32) int32 {
	if h.cfg.CurrentTrackFunc == nil {
		return 0
	}
	track := h.cfg.CurrentTrackFunc()
	raw, err := json.Marshal(track)
	if err != nil {
		return -1
	}

	if uint32(len(raw)) > outMaxLen {
		return -1
	}

	mem := mod.Memory()
	if mem == nil || !mem.Write(outPtr, raw) {
		return -1
	}

	return int32(len(raw))
}

// hostHTTPFetch ABI: host_http_fetch(url_ptr i32, url_len i32, out_ptr i32, out_max_len i32) -> i32
func (h *Host) hostHTTPFetch(ctx context.Context, mod api.Module, urlPtr, urlLen, outPtr, outMaxLen uint32) int32 {
	mem := mod.Memory()
	if mem == nil {
		return -1
	}

	urlBytes, ok := mem.Read(urlPtr, urlLen)
	if !ok {
		return -1
	}

	reqURL := string(urlBytes)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return -1
	}

	resp, err := h.cfg.HTTPClient.Do(req)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(outMaxLen)))
	if err != nil {
		return -1
	}

	if !mem.Write(outPtr, body) {
		return -1
	}

	return int32(len(body))
}

// LoadPlugin compiles and instantiates a WebAssembly plugin from raw .wasm bytecode.
func (h *Host) LoadPlugin(ctx context.Context, id string, wasmBytes []byte, meta PluginMetadata) (*PluginInstance, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.plugins[id]; exists {
		return nil, fmt.Errorf("plugin %s is already loaded", id)
	}

	config := wazero.NewModuleConfig().WithName(id)
	mod, err := h.runtime.InstantiateWithConfig(ctx, wasmBytes, config)
	if err != nil {
		return nil, fmt.Errorf("instantiate wasm plugin %s: %w", id, err)
	}

	instance := &PluginInstance{
		Meta:           meta,
		module:         mod,
		onTrackChange:  mod.ExportedFunction("on_track_change"),
		onPlayheadTick: mod.ExportedFunction("on_playhead_tick"),
	}

	h.plugins[id] = instance
	return instance, nil
}

// UnloadPlugin safely terminates and removes a loaded plugin.
func (h *Host) UnloadPlugin(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	instance, exists := h.plugins[id]
	if !exists {
		return fmt.Errorf("plugin %s not found", id)
	}

	delete(h.plugins, id)
	return instance.module.Close(ctx)
}

// Quarantine isolates a failing or panicking plugin to protect the audio pipeline.
func (h *Host) Quarantine(id string, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if p, ok := h.plugins[id]; ok {
		p.isQuarantined = true
		p.quarantineReason = reason
		if h.cfg.LogHook != nil {
			h.cfg.LogHook(id, 3, "QUARANTINED: "+reason)
		}
	}
}

// NotifyTrackChange broadcasts track updates to all non-quarantined plugins with a strict timeout.
func (h *Host) NotifyTrackChange(ctx context.Context) {
	h.mu.RLock()
	plugins := make([]*PluginInstance, 0, len(h.plugins))
	for _, p := range h.plugins {
		if !p.isQuarantined && p.onTrackChange != nil {
			plugins = append(plugins, p)
		}
	}
	h.mu.RUnlock()

	for _, p := range plugins {
		func(inst *PluginInstance) {
			defer func() {
				if r := recover(); r != nil {
					h.Quarantine(inst.Meta.ID, fmt.Sprintf("panic in on_track_change: %v", r))
				}
			}()

			callCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
			defer cancel()

			_, err := inst.onTrackChange.Call(callCtx)
			if err != nil {
				h.Quarantine(inst.Meta.ID, fmt.Sprintf("error in on_track_change: %v", err))
			}
		}(p)
	}
}

// NotifyPlayheadTick sends playhead positions to all active plugins.
func (h *Host) NotifyPlayheadTick(ctx context.Context, posMs int32) {
	h.mu.RLock()
	plugins := make([]*PluginInstance, 0, len(h.plugins))
	for _, p := range h.plugins {
		if !p.isQuarantined && p.onPlayheadTick != nil {
			plugins = append(plugins, p)
		}
	}
	h.mu.RUnlock()

	for _, p := range plugins {
		func(inst *PluginInstance) {
			defer func() {
				if r := recover(); r != nil {
					h.Quarantine(inst.Meta.ID, fmt.Sprintf("panic in on_playhead_tick: %v", r))
				}
			}()

			callCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			defer cancel()

			_, err := inst.onPlayheadTick.Call(callCtx, uint64(posMs))
			if err != nil {
				h.Quarantine(inst.Meta.ID, fmt.Sprintf("error in on_playhead_tick: %v", err))
			}
		}(p)
	}
}

// Close terminates all plugins and the Wazero runtime.
func (h *Host) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.runtime.Close(ctx)
}
