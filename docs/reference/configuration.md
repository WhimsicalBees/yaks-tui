# Configuration

yaks-tui has no config file. Its appearance adapts automatically:

- **Markdown theme.** At startup yaks-tui detects whether the terminal has a
  dark or light background and picks the matching glamour style. When output is
  not a terminal (for example, piped), it falls back to a no-color style. This
  detection happens once, before the event loop starts.
- **Layout.** The two panes split roughly 40/60 (tree/detail) and resize with
  the terminal window. A minimum of 40×8 is required; below that, yaks-tui shows
  a "terminal too small" message.

There are no environment variables or flags to set. yaks-tui does not use
`$EDITOR` — context editing happens inline (see
[edit a yak's context](../how-to/edit-context.md)).
