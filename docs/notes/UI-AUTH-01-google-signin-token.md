# Task Specification & Resolution Note: UI-AUTH-01 (Google Sign-In Token Bundle Parser)

**Workstream:** `WS-YTM` (auth logic lives beside the InnerTube client)
**Task series:** `UI-AUTH-01` → `UI-AUTH-03` → `UI-PAGE-01` → `UI-PAGE-02` (plan: `docs/ui/auth-signin-plan.md`)
**File Scope:** `internal/ytm/token.go`, `internal/ytm/token_test.go`
**Dependencies:** None
**Status:** `DONE` — commit `1b0…` (UI-AUTH-01 token bundle parser, adapters, required-cookie checks implemented)

---

## 1. Objective

Implement the Google Sign-In / InnerTune token import pipeline for Chordly:

- Token **bundle model** (`TokenBundle`) + source enum (`TokenSource`).
- Detection-driven **adapters** that recognize and parse InnerTune session payloads
  (cookie header, JSON bundle, Netscape cookies.txt, base64, bare cookie strings).
- **Required-cookie validation** with clear failure reporting.

This task must be panic-free and fixture-driven; it performs no network I/O.

## 2. Key Spec Points (from `docs/ui/auth-signin-plan.md`)

- **Philosophy:** cookie import, not OAuth. InnerTune holds a WebView cookie jar;
  the session is transferred as cookies/JSON.
- **Detection order:** trim/empty → reject `ya29.` OAuth tokens (`ErrOAuthTokenUnsupported`)
  → JSON (`{`/`[`) → base64 (decode once, re-run detection) → cookies.txt (tab rows /
  `Netscape` marker) → `Cookie:` prefix strip → bare cookie (`=`/`;`) → else `ErrUnrecognized`.
- **Caps:** 2 MiB input size cap before parsing.
- **Required cookies:** `SAPISID` **or** `__Secure-3PAPISID` (required; SAPISID derived from
  `__Secure-3PAPISID` if needed); `__Secure-3PSID` strongly recommended; `SID`/`HSID`/`SSID`/`APISID`
  recommended; `LOGIN_INFO`, `VISITOR_INFO1_LIVE`, `PREF` optional. Missing → fail listing missing names.
- **Adapter registry:** `TokenParser{Name, CanParse, Parse}` with adapters `cssCookie`, `netscape`,
  `json`, `base64`, future `innertuneBackup`. Public entry: `ParseToken(raw)`.

## 3. Acceptance

- Fixtures pass; parser never panics on adversarial input.
- Required-cookie enforcement behaves per spec.
- No panics; unit tests cover empty/unrecognized/JSON/cookie/base64/cookies.txt/required-missing paths.
