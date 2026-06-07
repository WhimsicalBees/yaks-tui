# Filter and search the tree

Narrow what the tree pane shows without changing any yak.

## Hide completed work

Press `H` to hide done yaks. A done yak that still has an active (todo/wip/
blocked) descendant stays visible, so you don't lose the path to live work.
Press `H` again to show everything.

## Focus on what's active

Press `W` to show only `wip` and `blocked` yaks (plus their parents). This is
the "what am I working on right now" view. Press `W` again to clear it.

## Search by name

Press `f` and type — the tree narrows to yaks whose name contains what you typed
(case-insensitive), keeping their parents visible. Then:

- `enter` keeps the filter applied so you can navigate and triage the narrowed
  set.
- `esc` clears the search and restores the full tree.

The filters and search stack: with `H` on and a search active, you see only
matching, non-done yaks. The status bar shows which filters are active.
