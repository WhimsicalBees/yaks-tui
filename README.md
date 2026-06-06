# yaks-tui

An interactive, keyboard-driven terminal UI for [yaks](https://github.com/mattwynne/yaks).

Browse your yak tree in two panes — tree on the left, rendered markdown detail on
the right — and triage state without leaving the keyboard.

## Requirements

- `yx` (yaks) on your PATH
- `fzf` (optional, enables `/` fuzzy jump)
- A repo with `.yaks/` initialized (and `.yaks` gitignored)

Markdown detail is rendered in-process with
[glamour](https://github.com/charmbracelet/glamour) — no `glow` or other external
renderer is required.

## Build & run

```bash
make build
./bin/yaks-tui      # run inside a yaks repo
```

## Keys

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | move cursor |
| `→`/`l`, `←`/`h` | expand / collapse |
| `enter` | toggle fold |
| `tab` | switch pane focus |
| `w` / `b` / `d` / `t` | set state wip / blocked / done / todo |
| `/` | fuzzy jump (needs fzf) |
| `r` | reload |
| `?` | toggle help |
| `q` / `ctrl+c` | quit |

## Scope

v1 is browse + triage. Editing, adding/moving yaks, tags, sync, and configurable
layouts are planned — see `docs/superpowers/specs/`.
