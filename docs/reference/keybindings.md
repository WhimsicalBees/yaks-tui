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
| `a` | add a child yak under the cursor |
| `A` | add a root yak |
| `R` | rename the selected yak |
| `x` | remove the selected yak (confirms first) |
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

## Managing yaks

`a` adds a child under the highlighted yak; `A` adds a root. Both open a
one-line input — type a name and press `enter` to create the yak (it lands on
the new one), or `esc` to cancel. An empty name cancels.

`R` renames the selected yak. The input opens pre-filled with the current name;
edit it and press `enter`, or `esc` to cancel.

`x` removes the selected yak after a confirmation. Press `y` to confirm,
anything else to cancel. A yak with children is removed recursively (the prompt
shows the child count first). Removal is not undoable, so the confirmation is
always shown.

## While editing context

| Key | Action |
|-----|--------|
| `ctrl+s` | save and exit the editor |
| `esc` | cancel and discard |

All other keys are text input for the editor while editing — including
`ctrl+c`, which is *not* quit in this mode. Press `esc` first, then quit.
