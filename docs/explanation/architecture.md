# Architecture

yaks-tui is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) program
following the Model-View-Update (MVU) pattern.

## Layers

- **`internal/yaks`** — the only code that talks to the `yx` binary, behind a
  `Runner` interface so tests inject a fake. Decodes `yx list --format json`
  into a `Yak` tree and exposes `SetState` / `SetContext`.
- **`internal/tree`** — pure logic: flattening the tree into visible rows given
  an expansion map, and cursor math. No Bubble Tea, no I/O, so it's unit-tested
  directly.
- **`internal/ui`** — the Bubble Tea `Model`: key handling, the two-pane layout,
  the inline context editor, and async commands that call the client.
- **`internal/shell`** — the fzf integration.

## Two design decisions worth knowing

**Markdown rendering is resolved once, never in the render loop.** glamour can
auto-detect the terminal background to choose a light/dark theme, but that
detection writes an OSC escape query to the terminal and reads the reply from
stdin. Done during rendering, it races Bubble Tea's own input reader and
swallows keystrokes — multi-second input lag. yaks-tui resolves the style a
single time in `ui.New`, before the event loop, and reuses it. A PTY regression
test (`main_pty_test.go`) guards against reintroducing a per-keystroke terminal
query.

**fzf gets the real terminal.** Fuzzy find runs via `tea.ExecProcess`, which
suspends Bubble Tea, hands the TTY to fzf, then restores the program. Because
fzf needs both a candidate list and an interactive terminal, candidates are
passed on a file (read via a shell redirect) while fzf draws its UI on
`/dev/tty` and writes the selection to a second file.

For the full history of these decisions, see `docs/superpowers/specs/`.
