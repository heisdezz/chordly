# Task Specification & Architecture Note: TSK-302 (Linux MPRIS & Media Keys Integration)

**Workstream:** `WS-PLG`  
**File Scope:** `internal/plugin/mpris_linux.go`, `internal/plugin/mpris_linux_test.go`  
**Dependencies:** None  
**Claimed Status:** `IN_PROGRESS [Agent-MPRIS]`  
**Target Package:** `package plugin` (`chordly/internal/plugin`)  

---

## 1. Executive Summary & Objective

The **Linux MPRIS & Media Keys Integration (`TSK-302`)** implements the desktop standard **MPRIS v2 (Media Player Remote Interfacing Specification)** over the Linux D-Bus session bus.

This allows Chordly to seamlessly integrate with:
- Desktop environment shell media widgets (GNOME top bar, KDE Plasma system tray, XFCE panel, Hyprland/Waybar mpris modules).
- Dedicated hardware media keys (Play/Pause, Next, Previous, Stop, Volume) via standard desktop daemons.
- CLI media player controllers like `playerctl` (`playerctl -p chordly play-pause`).
- Lock screen playback controls with live track titles, artist names, and album artwork.

---

## 2. MPRIS v2 Specification Contracts

Chordly registers on the D-Bus **Session Bus** with:
- **Well-known Bus Name:** `org.mpris.MediaPlayer2.chordly` (or `org.mpris.MediaPlayer2.chordly.instance<PID>` on multi-instance).
- **Object Path:** `/org/mpris/MediaPlayer2`

### 2.1. Supported D-Bus Interfaces

1. `org.freedesktop.DBus.Introspectable`: Provides XML schema introspection.
2. `org.freedesktop.DBus.Properties`: Standard property getter (`Get`), setter (`Set`), bulk query (`GetAll`), and reactive change signal (`PropertiesChanged`).
3. `org.mpris.MediaPlayer2`: Root application controls (`Identity`, `Quit`, `Raise`, `CanQuit`, `CanRaise`, `SupportedUriSchemes`, `SupportedMimeTypes`).
4. `org.mpris.MediaPlayer2.Player`: Playback controls (`Play`, `Pause`, `PlayPause`, `Next`, `Previous`, `Stop`, `Seek`, `SetPosition`, `Volume`, `PlaybackStatus`, `Metadata`, `Position`).

---

## 3. Metadata Mapping (`org.mpris.MediaPlayer2.Player.Metadata`)

MPRIS uses the **XESAM** metadata vocabulary. The table below outlines how Chordly maps `model.Track` and `internal/state` into MPRIS metadata variant entries:

| MPRIS Metadata Key | D-Bus Variant Type | Source Field (`model.Track` / `state`) | Description |
| :--- | :--- | :--- | :--- |
| `mpris:trackid` | Object Path (`o`) | `"/org/mpris/MediaPlayer2/track/" + track.ID` | Unique D-Bus object path identifier |
| `mpris:length` | 64-bit int (`x`) | `track.Duration.Microseconds()` | Total duration in **microseconds** |
| `mpris:artUrl` | String (`s`) | `track.Thumbnail` | Web thumbnail or local cached file URI |
| `xesam:title` | String (`s`) | `track.Title` | Song title |
| `xesam:artist` | Array of strings (`as`)| `[]string{track.Artist}` | Performing artist(s) |
| `xesam:album` | String (`s`) | `track.Album` | Album name |
| `xesam:url` | String (`s`) | `"https://music.youtube.com/watch?v=" + track.ID` | Canonical web URL |

---

## 4. Playback Status & State Synchronization

Chordly bridges reactive signals (`internal/state`) to MPRIS properties:

```
[Chordly State Ticker / Playback Change]
                 │
                 ▼
     SigPlaybackState.Get()
                 │
        ┌────────┴────────┐
        ▼                 ▼
   StatePlaying      StatePaused
   ("Playing")       ("Paused")
                 │
                 ▼
  Emit D-Bus Signal: PropertiesChanged(
      "org.mpris.MediaPlayer2.Player",
      { "PlaybackStatus": variant(status), "Metadata": variant(meta) },
      []
  )
```

### Supported Remote Player Commands

| MPRIS Method | Handled Action in Chordly |
| :--- | :--- |
| `Play()` | Resumes playback via `state.SigPlaybackState.Set(model.StatePlaying)` |
| `Pause()` | Pauses audio engine via `state.SigPlaybackState.Set(model.StatePaused)` |
| `PlayPause()` | Toggles current playback state |
| `Stop()` | Stops audio stream and resets position to 0 |
| `Next()` | Advances to next track in queue via `state.SigQueue` |
| `Previous()` | Jumps to previous history track or restarts song if > 3s |
| `Seek(offset_us)` | Offsets current playhead position by microseconds |
| `SetPosition(track_id, pos_us)` | Seeks directly to absolute position |

---

## 5. Pure-Go Zero-CGO Linux D-Bus Transport

To preserve cross-compilation without requiring libdbus headers or CGO:
- Connect directly to the Unix Domain Socket specified by `$DBUS_SESSION_BUS_ADDRESS` (standard `unix:path=/run/user/<uid>/bus`).
- Implement the SASL `EXTERNAL` authentication handshake (`AUTH EXTERNAL <hex_uid>\r\nNEGOTIATE_UNIX_FD\r\nBEGIN\r\n`).
- Binary message serialization per the D-Bus specification (Little-Endian / byte 'l', 16-byte fixed header, body alignment).
- Fallback gracefully when running in headless, non-Linux, or containerized environments without active D-Bus sessions.

---

## 6. Verification Gate Checklist

- [ ] Successful registration of bus name `org.mpris.MediaPlayer2.chordly`.
- [ ] Verification with `playerctl -p chordly status` (reports `Playing` or `Paused`).
- [ ] Verification with `playerctl -p chordly metadata` (reports Title, Artist, ArtURL, and Length).
- [ ] Verification of `playerctl -p chordly play-pause` triggering internal state change.
- [ ] Ensure non-Linux environments compile safely via `//go:build linux` and stubs for non-Linux.
- [ ] Pass all unit tests in `internal/plugin/mpris_linux_test.go`.
