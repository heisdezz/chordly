# Task Note: UI-AUTH-03 (Anonymous Stream Resolution)

**Workstream:** `WS-YTM`
**Task series:** `UI-AUTH-01` (token parser) → `UI-AUTH-02` (session injection) → `UI-AUTH-03`
**File Scope:** `internal/ytm/player.go`, `internal/ytm/client.go`, `internal/ytm/player_fallback_test.go`, `internal/ytm/player_rotation_test.go`, `internal/ytm/client_auth_test.go`
**Status:** `DONE` — commit `caaf24f`

---

## 1. Objective

Playback must work **without a Google sign-in**, the way BitChord / Metrolist
do it: resolve a direct `googlevideo` URL from YouTube's undocumented InnerTube
API using spoofed clients, then hand that URL to the audio engine.

Sign-in remains supported (cookie bundle import from InnerTune), but an imported
session must never make playback *worse* than a guest session.

## 2. Resolution Pipeline

`PlayerService.ResolveStreamURL(ctx, videoID)`

1. **Cache** — in-memory `StreamInfo` with a 4 h expiry (URLs usually last ~6 h).
2. **InnerTube player** — `POST /youtubei/v1/player`, rotating through spoofed
   clients until one returns `playabilityStatus.status == "OK"`:
   `ANDROID_MUSIC` → `TVHTML5` (Samsung) → `VISIONOS` → `ANDROID_VR` → `IOS` →
   `MWEB` → `WEB_REMIX`.
3. **Format selection** — prefer Opus/WebM itag 251, then AAC/MP4 itag 140, then
   any audio-only format. Video formats are skipped.
4. **CDN validation** — `Range: bytes=0-4095` probe; a URL that does not return
   bytes is rejected and the rotation continues.
5. **yt-dlp fallback** — only when InnerTube fails with a sign-in/HTTP gating
   error (`PlayabilityError`, HTTP 400/401/403). `CHORDLY_YTDLP` overrides the
   binary path. This is what handles the JS `n`-parameter; do not try to
   reimplement that.

### Signed-in retry (the important part)

When `client.HasAuth()` is true, the rotation runs **twice**:

```
for _, useAuth := range []bool{true, false} { tryClients(useAuth) }
```

Imported cookies are sometimes rejected outright (`LOGIN_REQUIRED` or
`400 INVALID_ARGUMENT`), while the anonymous clients still serve a playable
stream. Before this change a signed-in user would fall through to a slow
`yt-dlp` call (or hang) while the same track played fine as a guest.

Guest requests also use the **anonymous visitor**, not the session's
`VisitorData` (`requestStateFor(useAuth)`), so a retry does not look like a
logged-in visitor with its credentials stripped.

## 3. Header Requirements for Authenticated Requests

InnerTube account endpoints accept **web cookies + SID hashes**, never OAuth
`ya29.*` tokens.

- `Cookie` — imported cookie header.
- `Authorization` — include **all** available hashes, not just SAPISID:
  `SAPISIDHASH`, `SAPISID1PHASH`, `SAPISID3PHASH` (each
  `SCHEME <ts>_<sha1("<ts> <sid> <origin>")>`).
- `X-Goog-AuthUser: 0`, `Origin` + `X-Origin` (must match the request host:
  `https://music.youtube.com` or `https://www.youtube.com`),
  `X-Youtube-Bootstrap-Logged-In: true`.
- A lone `SAPISIDHASH` has produced `400 INVALID_ARGUMENT`; the multi-hash form
  mirrors yt-dlp.

## 4. Visitor Data

- `Client.RefreshVisitor` mints an anonymous visitor from
  `https://www.youtube.com/sw.js_data`.
- It runs only for the **guest** pass. Authenticated requests keep the
  session's own visitor data, so refreshing cannot destabilise search/browse
  calls made with the account.
- `Client.updateVisitor` also learns visitor ids from response headers.

## 5. Pitfalls (do not repeat)

| Attempt | Result |
| --- | --- |
| OAuth token / WebView login | InnerTube ignores OAuth; not viable |
| `Cookie` alone, no `Authorization` | `LOGIN_REQUIRED` |
| `SAPISIDHASH` only | `200 + LOGIN_REQUIRED` or `400 INVALID_ARGUMENT` |
| `WEB_REMIX` / `WEB` player calls | `UNPLAYABLE` |
| Bare `TVHTML5_SIMPLY_EMBEDDED_PLAYER` | "no longer supported" |
| `ANDROID_VR` / `TVHTML5` alone | "confirm you're not a bot" |
| Reimplementing `n`-parameter deciphering in Go | Use yt-dlp; it already tracks player changes |

## 6. Known Limits / Follow-ups

- **AAC/MP4 decoding is unsupported** — the decoder handles WebM/Opus, Ogg Opus,
  WAV. A stream resolved as `audio/mp4` will resolve but fail to play. Prefer
  Opus formats in `SelectAudioFormat`.
- **Seeking is not implemented** — the scrubber snaps back.
- **Sessions are in-memory only** — re-import after every restart. Persisting
  the bundle (keyring or `0600` file) is the next auth task.
- Resolution is bounded by the caller (`appui/playback.go` uses a 35 s timeout).
