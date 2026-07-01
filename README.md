# TunaOS Niri Installer — TUI frontend for fisherman

**Terminal UI (ratatui/crossterm) installer** for TunaOS that drives the fisherman bootc install backend. Designed for the Niri scrollable-tiling Wayland compositor.

## Workflow

1. **Welcome** — brief intro
2. **Disk Selection** — `lsblk -J` lists disks; navigate with ↑/↓, select with Enter
3. **Confirm** — review and press Enter to install
4. **Install Progress** — streams fisherman output (runs synchronously in TUI)
5. **Done** — success/failure, press any key to exit

## Build

```bash
cargo build --release
cargo run --release
```

## Recipe

Produces the same JSON recipe as the Qt/KDE and COSMIC frontends.

## License

GPL-3.0-only
