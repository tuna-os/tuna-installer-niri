# TunaOS Niri Installer — Roadmap

**Last updated**: 2026-09-16 | **Evidence through**: 2026-09-16 | **Maintainer**: tuna-os (hanthor)

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
- **Distribution**: published multi-arch (`x86_64` + `aarch64`) to the TunaOS
  Flatpak index on every `main` push (#67); the fisherman backend target was updated
  to `tuna-os/fisherman` commit `027fa25c` (#72). Standalone GitHub Releases / tags
  remain open under org-level decision (tunaOS #2020).
- **Parity**: covered by `installer-smoke.yml` + `docs/INSTALLER-FRONTENDS.md`
  checks, plus simplified technical english (STE) linter checks (#68).
- **Health**: active (pushed 09-07). Go backend test suite and CI execution landed
  in #52, #62, and #70; privileged fisherman source pinned in #42.

### Priorities

| Priority | Item | Tracking | Status |
|----------|------|----------|--------|
| P0 | Pin the privileged fisherman source and reusable publishing workflow | #40/#41/#42 | 🟢 Landed |
| P0 | Go backend gate and CLI test coverage | #27/#52/#62/#70 | 🟢 Landed |
| P0 | Multi-arch Flatpak publishing (`aarch64` + `x86_64`) | #67 | 🟢 Landed |
| P1 | User-account creation broken — recipe missing required field | #25 | 🟡 Open |
| P1 | Reconcile the superseded LUKS boundary tracker after the stdin fix landed | #26/#28 | 🟢 Resolved |
| P2 | Keep roadmap outcomes synchronized with repository evidence | #43 | 🟢 Active |

---

## Quarterly Goals

### Current Quarter (2026 Q3)

**Theme**: harden the privileged install and publishing paths

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Pin privileged install and publishing dependencies | hanthor | #39-#42 | 🟢 Completed |
| CI coverage on the Go backend | hanthor | #27/#52/#62 | 🟢 Completed |
| Multi-arch Flatpak distribution for arm64 ISO support | hanthor | #67 | 🟢 Completed |

### Next Quarter (2026 Q4)

**Theme**: parity and cadence

| Goal | Owner | Tracking | Status |
|------|-------|----------|--------|
| Fix user-account creation field gap | hanthor | #25 | ⬜ Not started |
| Document release/versioning model (image-baked vs tagged) | tuna-os | (org #2020) | ⬜ Not started |

---

## Review Cadence

Review this roadmap monthly and after any material install-path, distribution,
or privileged-supply-chain change. Each refresh must record an evidence-through
date and reconcile priorities against merged changes, open issues, releases or
tags, and the latest publishing runs. A merged outcome is evidence even when a
follow-up tracker has not yet been closed; record that mismatch explicitly
rather than reporting completed work as not started.
