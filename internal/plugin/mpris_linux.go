//go:build linux

package plugin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"chordly/internal/model"
	"chordly/internal/state"
)

const (
	mprisObjectPath = "/org/mpris/MediaPlayer2"
	mprisRootIface  = "org.mpris.MediaPlayer2"
	mprisPlayerIface = "org.mpris.MediaPlayer2.Player"
	propsIface      = "org.freedesktop.DBus.Properties"
	introIface      = "org.freedesktop.DBus.Introspectable"

	defaultBusName  = "org.mpris.MediaPlayer2.chordly"
)

const introspectXML = `<!DOCTYPE node PUBLIC "-//freedesktop//DTD D-BUS Object Introspection 1.0//EN"
"http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd">
<node>
  <interface name="org.freedesktop.DBus.Introspectable">
    <method name="Introspect">
      <arg name="out" direction="out" type="s"/>
    </method>
  </interface>
  <interface name="org.freedesktop.DBus.Properties">
    <method name="Get">
      <arg name="interface_name" direction="in" type="s"/>
      <arg name="property_name" direction="in" type="s"/>
      <arg name="value" direction="out" type="v"/>
    </method>
    <method name="Set">
      <arg name="interface_name" direction="in" type="s"/>
      <arg name="property_name" direction="in" type="s"/>
      <arg name="value" direction="in" type="v"/>
    </method>
    <method name="GetAll">
      <arg name="interface_name" direction="in" type="s"/>
      <arg name="properties" direction="out" type="a{sv}"/>
    </method>
    <signal name="PropertiesChanged">
      <arg name="interface_name" type="s"/>
      <arg name="changed_properties" type="a{sv}"/>
      <arg name="invalidated_properties" type="as"/>
    </signal>
  </interface>
  <interface name="org.mpris.MediaPlayer2">
    <method name="Raise"/>
    <method name="Quit"/>
    <property name="CanQuit" type="b" access="read"/>
    <property name="CanRaise" type="b" access="read"/>
    <property name="HasTrackList" type="b" access="read"/>
    <property name="Identity" type="s" access="read"/>
    <property name="SupportedUriSchemes" type="as" access="read"/>
    <property name="SupportedMimeTypes" type="as" access="read"/>
  </interface>
  <interface name="org.mpris.MediaPlayer2.Player">
    <method name="Next"/>
    <method name="Previous"/>
    <method name="Pause"/>
    <method name="PlayPause"/>
    <method name="Stop"/>
    <method name="Play"/>
    <method name="Seek">
      <arg name="Offset" direction="in" type="x"/>
    </method>
    <method name="SetPosition">
      <arg name="TrackId" direction="in" type="o"/>
      <arg name="Position" direction="in" type="x"/>
    </method>
    <method name="OpenUri">
      <arg name="Uri" direction="in" type="s"/>
    </method>
    <signal name="Seeked">
      <arg name="Position" type="x"/>
    </signal>
    <property name="PlaybackStatus" type="s" access="read"/>
    <property name="LoopStatus" type="s" access="readwrite"/>
    <property name="Rate" type="d" access="readwrite"/>
    <property name="Shuffle" type="b" access="readwrite"/>
    <property name="Metadata" type="a{sv}" access="read"/>
    <property name="Volume" type="d" access="readwrite"/>
    <property name="Position" type="x" access="read"/>
    <property name="MinimumRate" type="d" access="read"/>
    <property name="MaximumRate" type="d" access="read"/>
    <property name="CanControl" type="b" access="read"/>
    <property name="CanPlay" type="b" access="read"/>
    <property name="CanPause" type="b" access="read"/>
    <property name="CanSeek" type="b" access="read"/>
    <property name="CanGoNext" type="b" access="read"/>
    <property name="CanGoPrevious" type="b" access="read"/>
  </interface>
</node>`

// MPRISServer hosts the MPRIS v2 D-Bus interface for Chordly on Linux.
type MPRISServer struct {
	mu       sync.RWMutex
	conn     *dbus.Conn
	busName  string
	cancel   context.CancelFunc
	closed   bool
	onQuit   func()
	onRaise  func()

	// Cached state to detect property changes
	lastTrackID string
	lastStatus  model.PlaybackState
	lastVolume  float32
}

// NewMPRISServer creates, initializes, and starts the MPRIS D-Bus server.
func NewMPRISServer() (*MPRISServer, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}

	s := &MPRISServer{
		conn: conn,
	}

	// Try default name, fallback to instance PID name if taken
	name := defaultBusName
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		name = fmt.Sprintf("%s.instance%d", defaultBusName, os.Getpid())
		reply, err = conn.RequestName(name, dbus.NameFlagDoNotQueue)
		if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
			conn.Close()
			return nil, fmt.Errorf("failed to claim D-Bus name %q: %v", name, err)
		}
	}
	s.busName = name

	// Export D-Bus objects
	path := dbus.ObjectPath(mprisObjectPath)
	if err := conn.Export(s, path, mprisRootIface); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to export %s: %w", mprisRootIface, err)
	}
	if err := conn.Export(s, path, mprisPlayerIface); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to export %s: %w", mprisPlayerIface, err)
	}
	if err := conn.Export(s, path, propsIface); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to export %s: %w", propsIface, err)
	}
	if err := conn.Export(s, path, introIface); err != nil {
		s.Close()
		return nil, fmt.Errorf("failed to export %s: %w", introIface, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Start state synchronization worker
	go s.syncLoop(ctx)

	return s, nil
}

// BusName returns the registered D-Bus bus name.
func (s *MPRISServer) BusName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.busName
}

// SetQuitHandler sets a callback for when an external controller calls Quit().
func (s *MPRISServer) SetQuitHandler(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onQuit = fn
}

// SetRaiseHandler sets a callback for when an external controller calls Raise().
func (s *MPRISServer) SetRaiseHandler(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onRaise = fn
}

// Close gracefully releases the D-Bus connection and unregisters the bus name.
func (s *MPRISServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		_, _ = s.conn.ReleaseName(s.busName)
		return s.conn.Close()
	}
	return nil
}

// -----------------------------------------------------------------------------
// D-Bus Methods: org.freedesktop.DBus.Introspectable
// -----------------------------------------------------------------------------

func (s *MPRISServer) Introspect() (string, *dbus.Error) {
	return introspectXML, nil
}

// -----------------------------------------------------------------------------
// D-Bus Methods: org.mpris.MediaPlayer2
// -----------------------------------------------------------------------------

func (s *MPRISServer) Raise() *dbus.Error {
	s.mu.RLock()
	fn := s.onRaise
	s.mu.RUnlock()
	if fn != nil {
		fn()
	}
	return nil
}

func (s *MPRISServer) Quit() *dbus.Error {
	s.mu.RLock()
	fn := s.onQuit
	s.mu.RUnlock()
	if fn != nil {
		fn()
	} else {
		go func() {
			time.Sleep(50 * time.Millisecond)
			os.Exit(0)
		}()
	}
	return nil
}

// -----------------------------------------------------------------------------
// D-Bus Methods: org.mpris.MediaPlayer2.Player
// -----------------------------------------------------------------------------

func (s *MPRISServer) Next() *dbus.Error {
	state.NextTrack()
	return nil
}

func (s *MPRISServer) Previous() *dbus.Error {
	state.PreviousTrack()
	return nil
}

func (s *MPRISServer) Pause() *dbus.Error {
	state.PlaybackStatus.Set(model.StatePaused)
	return nil
}

func (s *MPRISServer) PlayPause() *dbus.Error {
	state.TogglePlayPause()
	return nil
}

func (s *MPRISServer) Stop() *dbus.Error {
	state.PlaybackStatus.Set(model.StateStopped)
	state.Playhead.Set(0)
	return nil
}

func (s *MPRISServer) Play() *dbus.Error {
	state.PlaybackStatus.Set(model.StatePlaying)
	return nil
}

func (s *MPRISServer) Seek(offsetMicroseconds int64) *dbus.Error {
	cur := state.Playhead.Get()
	target := cur + time.Duration(offsetMicroseconds)*time.Microsecond
	if target < 0 {
		target = 0
	}
	t := state.CurrentTrack.Get()
	if t.Duration > 0 && target > t.Duration {
		target = t.Duration
	}
	state.Playhead.Set(target)
	s.emitSeeked(target)
	return nil
}

func (s *MPRISServer) SetPosition(trackID dbus.ObjectPath, positionMicroseconds int64) *dbus.Error {
	expectedPath := sanitizeTrackPath(state.CurrentTrack.Get().ID)
	if string(trackID) != string(expectedPath) {
		// Ignore requests for non-matching tracks as per MPRIS spec
		return nil
	}
	target := time.Duration(positionMicroseconds) * time.Microsecond
	if target < 0 {
		target = 0
	}
	t := state.CurrentTrack.Get()
	if t.Duration > 0 && target > t.Duration {
		target = t.Duration
	}
	state.Playhead.Set(target)
	s.emitSeeked(target)
	return nil
}

func (s *MPRISServer) OpenUri(uri string) *dbus.Error {
	// Not supported for streaming URLs in standalone mode
	return nil
}

// -----------------------------------------------------------------------------
// D-Bus Methods: org.freedesktop.DBus.Properties
// -----------------------------------------------------------------------------

func (s *MPRISServer) Get(iface string, prop string) (dbus.Variant, *dbus.Error) {
	all, err := s.GetAll(iface)
	if err != nil {
		return dbus.Variant{}, err
	}
	val, ok := all[prop]
	if !ok {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", []any{prop})
	}
	return val, nil
}

func (s *MPRISServer) Set(iface string, prop string, val dbus.Variant) *dbus.Error {
	switch iface {
	case mprisPlayerIface:
		switch prop {
		case "Volume":
			v, ok := val.Value().(float64)
			if !ok {
				return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"Volume must be double"})
			}
			if v < 0 {
				v = 0
			}
			if v > 1.0 {
				v = 1.0
			}
			state.Volume.Set(float32(v))
			return nil
		case "Rate":
			// Playback rate 1.0 only
			return nil
		case "Shuffle":
			if shuf, ok := val.Value().(bool); ok {
				state.IsShuffle.Set(shuf)
				return nil
			}
			return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"Shuffle must be bool"})
		case "LoopStatus":
			if loop, ok := val.Value().(string); ok {
				switch strings.ToLower(loop) {
				case "none":
					state.Repeat.Set(model.RepeatOff)
				case "track":
					state.Repeat.Set(model.RepeatOne)
				case "playlist":
					state.Repeat.Set(model.RepeatAll)
				}
				return nil
			}
			return dbus.NewError("org.freedesktop.DBus.Error.InvalidArgs", []any{"LoopStatus must be string"})
		}
	}
	return dbus.NewError("org.freedesktop.DBus.Error.PropertyReadOnly", []any{prop})
}

func (s *MPRISServer) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	res := make(map[string]dbus.Variant)
	switch iface {
	case mprisRootIface:
		res["CanQuit"] = dbus.MakeVariant(true)
		res["CanRaise"] = dbus.MakeVariant(true)
		res["HasTrackList"] = dbus.MakeVariant(false)
		res["Identity"] = dbus.MakeVariant("Chordly")
		res["SupportedUriSchemes"] = dbus.MakeVariant([]string{"http", "https"})
		res["SupportedMimeTypes"] = dbus.MakeVariant([]string{"audio/webm", "audio/mp4", "audio/mpeg"})
		return res, nil

	case mprisPlayerIface:
		// PlaybackStatus
		var statusStr string
		switch state.PlaybackStatus.Get() {
		case model.StatePlaying:
			statusStr = "Playing"
		case model.StatePaused:
			statusStr = "Paused"
		default:
			statusStr = "Stopped"
		}
		res["PlaybackStatus"] = dbus.MakeVariant(statusStr)

		// LoopStatus
		var loopStr string
		switch state.Repeat.Get() {
		case model.RepeatOne:
			loopStr = "Track"
		case model.RepeatAll:
			loopStr = "Playlist"
		default:
			loopStr = "None"
		}
		res["LoopStatus"] = dbus.MakeVariant(loopStr)

		res["Rate"] = dbus.MakeVariant(float64(1.0))
		res["MinimumRate"] = dbus.MakeVariant(float64(1.0))
		res["MaximumRate"] = dbus.MakeVariant(float64(1.0))
		res["Shuffle"] = dbus.MakeVariant(state.IsShuffle.Get())
		res["Volume"] = dbus.MakeVariant(float64(state.Volume.Get()))
		res["Position"] = dbus.MakeVariant(state.Playhead.Get().Microseconds())

		res["CanControl"] = dbus.MakeVariant(true)
		res["CanPlay"] = dbus.MakeVariant(true)
		res["CanPause"] = dbus.MakeVariant(true)
		res["CanSeek"] = dbus.MakeVariant(true)
		res["CanGoNext"] = dbus.MakeVariant(true)
		res["CanGoPrevious"] = dbus.MakeVariant(true)

		// Metadata
		res["Metadata"] = dbus.MakeVariant(s.BuildMetadata(state.CurrentTrack.Get()))
		return res, nil

	default:
		return nil, dbus.NewError("org.freedesktop.DBus.Error.UnknownInterface", []any{iface})
	}
}

// BuildMetadata constructs the XESAM metadata map for a model.Track.
func (s *MPRISServer) BuildMetadata(track model.Track) map[string]dbus.Variant {
	m := make(map[string]dbus.Variant)
	m["mpris:trackid"] = dbus.MakeVariant(sanitizeTrackPath(track.ID))

	if track.Duration > 0 {
		m["mpris:length"] = dbus.MakeVariant(track.Duration.Microseconds())
	}
	if track.Thumbnail != "" {
		m["mpris:artUrl"] = dbus.MakeVariant(track.Thumbnail)
	}
	if track.Title != "" {
		m["xesam:title"] = dbus.MakeVariant(track.Title)
	}
	if track.Artist != "" {
		m["xesam:artist"] = dbus.MakeVariant([]string{track.Artist})
	}
	if track.Album != "" {
		m["xesam:album"] = dbus.MakeVariant(track.Album)
	}
	if track.ID != "" {
		m["xesam:url"] = dbus.MakeVariant("https://music.youtube.com/watch?v=" + track.ID)
	}
	return m
}

// -----------------------------------------------------------------------------
// Synchronization & Signal Emission
// -----------------------------------------------------------------------------

func (s *MPRISServer) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkStateChanges()
		}
	}
}

func (s *MPRISServer) checkStateChanges() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	curTrack := state.CurrentTrack.Get()
	curStatus := state.PlaybackStatus.Get()
	curVol := state.Volume.Get()

	changed := make(map[string]dbus.Variant)

	if curTrack.ID != s.lastTrackID {
		s.lastTrackID = curTrack.ID
		changed["Metadata"] = dbus.MakeVariant(s.BuildMetadata(curTrack))
	}

	if curStatus != s.lastStatus {
		s.lastStatus = curStatus
		var st string
		switch curStatus {
		case model.StatePlaying:
			st = "Playing"
		case model.StatePaused:
			st = "Paused"
		default:
			st = "Stopped"
		}
		changed["PlaybackStatus"] = dbus.MakeVariant(st)
	}

	if curVol != s.lastVolume {
		s.lastVolume = curVol
		changed["Volume"] = dbus.MakeVariant(float64(curVol))
	}

	if len(changed) > 0 {
		path := dbus.ObjectPath(mprisObjectPath)
		_ = s.conn.Emit(path, "org.freedesktop.DBus.Properties.PropertiesChanged",
			mprisPlayerIface,
			changed,
			[]string{},
		)
	}
}

func (s *MPRISServer) emitSeeked(target time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.conn == nil {
		return
	}
	path := dbus.ObjectPath(mprisObjectPath)
	_ = s.conn.Emit(path, "org.mpris.MediaPlayer2.Player.Seeked", target.Microseconds())
}

// sanitizeTrackPath converts arbitrary track IDs into valid D-Bus ObjectPaths.
func sanitizeTrackPath(id string) dbus.ObjectPath {
	if id == "" {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/track/none")
	}
	var sb strings.Builder
	sb.WriteString("/org/mpris/MediaPlayer2/track/")
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return dbus.ObjectPath(sb.String())
}
