# TunaOS Niri Installer — Roadmap

**Last updated**: 2026-08-24 | **Maintainer**: tuna-os (hanthor)

---

## Mission

Ship the Niri desktop's install experience: a Quickshell/QML + Go installer
(modeled on DankMaterialShell's architecture) that drives the fisherman bootc
backend, so a first-time Niri user gets a native install on the
scrollable-tiling Wayland compositor.

---

## Current Status

- **App**: Quickshell/QML frontend + Go backend for fisherman; CI-rendered
  walkthrough in docs/gui-walkthrough.md.
- **Distribution**: image-baked flatpak — no standalone GitHub Releases (by
  design, not yet documented as policy).
- **Parity**: covered by `installer-smoke.yml` + `docs/INSTALLER-FRONTENDS.md`
  checks (readiness stamp, non-blank, advances, per-screen OCR).
- **Health**: active (pushed 08-24); open issues concentrate on LUKS
  passphrase handling (#22/#26/#28) and missing backend CI (#27).

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| P0 | LUKS passphrase handling — feed recipe over stdin, no CLI arg, no QML↔Go boundary leak | #22/#26/#28 | 🟡 Open |
| P0 | No CI runs on `installer/**` — Go backend untested in CI | #27 | 🟡 Open |
| P1 | User-account creation broken — recipe missing required field | #25 | 🟡 Open |
| P2 | ROADMAP-coverage entry in org ROADMAP tally | #1295 | ⬜ Not started |

---

## Quarterly Goals

### Current Quarter (2026 Q3)

**Theme**: harden the install path

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Green LUKS handling (stdin + boundary) | hanthor | #22/#26/#28 | ⬜ Not started |
| CI coverage on the Go backend | hanthor | #27 | ⬜ Not started |

### Next Quarter (2026 Q4)

**Theme**: parity and cadence

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Fix user-account creation | hanthor | #25 | ⬜ Not started |
| Document release/versioning model (image-baked vs tagged) | tuna-os | (org #2020) | ⬜ Not started |

---

*ROADMAP added by strategist agent (ACMM L6 — full mode). Signed-off-by: hanthor-hive-agent[bot] <290068839+hanthor-hive-agent[bot]@users.noreply.github.com>*
