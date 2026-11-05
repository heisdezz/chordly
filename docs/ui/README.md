# UI Planning Docs

- [Pages Plan](pages-plan.md) — current route inventory, the target route tree,
  page-by-page specs (album, artist, playlist, explore, library, player,
  settings, errors), shared components, phasing, and the task matrix.
- [Google Sign-In & InnerTune Import](auth-signin-plan.md) — cookie/token model,
  InnerTune import formats, parser rules, validation, `SAPISIDHASH` client
  integration, secure storage, and error copy.
- [Motion & Loading Plan](animations-plan.md) — motion tokens, mini-player →
  full-player expansion (phased), loading skeletons/spinners, page transitions,
  micro-interactions, reduced motion, and performance rules.

## Recommended P0 order

1. `UI-INF-01` — re-vendor GoGPU and adopt `core/textfield`, `core/scrollview`,
   `core/listview` (currently the module has them, the vendor tree does not).
2. `UI-AUTH-01` → `UI-AUTH-03` — token parser, auth service, secure storage.
3. `UI-PAGE-01` — `/login` with **Import from InnerTune**.
4. `UI-PAGE-02` — `/account` session health and sign out.
5. `UI-PAGE-03` — `/library` shell with guest state.
6. Album, artist, explore, and full-screen player pages.
7. `UI-MOTION-01` → `UI-MOTION-04` — motion tokens, loading kit, page
   transitions, and the mini-player expansion.

## Current reality (2026-11-05)

Implemented: `/`, `/search`, `/queue`, `/lyrics`, `/settings` (appearance).
Placeholders: `/explore`, `/library` (both render `HomePage`).
Missing: `/login`, `/account`, `/album/:id`, `/artist/:id`, `/playlist/:id`,
`/mix/:id`, `/player`, settings subpages, real explore/library content.
