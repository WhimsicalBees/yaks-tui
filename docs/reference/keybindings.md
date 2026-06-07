# Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | move cursor up |
| `↓` / `j` | move cursor down |
| `→` / `l` | expand |
| `←` / `h` | collapse |
| `enter` | toggle fold |
| `tab` | switch pane focus |
| `w` | set state: wip |
| `b` | set state: blocked |
| `d` | set state: done |
| `t` | set state: todo |
| `e` | edit context body inline |
| `H` | hide done yaks (toggle) |
| `W` | focus wip / blocked (toggle) |
| `f` | search by name |
| `/` | fuzzy jump (requires fzf) |
| `r` | reload |
| `?` | toggle help |
| `q` / `ctrl+c` | quit |

## Filtering & search

`H` and `W` are toggles: press once to activate, press again to clear.

`f` opens an incremental name search. While the search input is open, the tree
narrows live as you type. Press `enter` to keep the filter applied after closing
the input; press `esc` to clear the search and restore the full tree.

The filters compose: `H`, `W`, and `f` can all be active at the same time. The
status bar shows which filters are currently on. When any are active, `esc` (in
normal mode) clears them all at once.

## While editing context

| Key | Action |
|-----|--------|
| `ctrl+s` | save and exit the editor |
| `esc` | cancel and discard |

All other keys are text input for the editor while editing — including
`ctrl+c`, which is *not* quit in this mode. Press `esc` first, then quit.
