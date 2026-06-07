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
| `/` | fuzzy jump (requires fzf) |
| `r` | reload |
| `?` | toggle help |
| `q` / `ctrl+c` | quit |

## While editing context

| Key | Action |
|-----|--------|
| `ctrl+s` | save and exit the editor |
| `esc` | cancel and discard |

All other keys are text input for the editor while editing — including
`ctrl+c`, which is *not* quit in this mode. Press `esc` first, then quit.
