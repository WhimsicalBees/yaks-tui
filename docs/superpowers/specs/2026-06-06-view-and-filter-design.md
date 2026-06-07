# yaks-tui — View & Filter — Design

**Date:** 2026-06-06
**Status:** Approved, ready for implementation plan

## Goal

Give the user ways to narrow what the tree pane shows, without leaving the
keyboard and without mutating any yak:

1. **Filter toggles** — `H` hides done yaks; `W` focuses on active work (wip +
   blocked).
2. **Text search** — `f` filters the tree as you type by name.

All three are view-only: they change which rows render, never the underlying
yaks. This is the first of three planned sub-rounds (View & filter → simple
CRUD → structural); add/remove/rename/re-parent/tags are explicitly out of scope
here.

## Background / constraint

The tree is built by `tree.Flatten(roots, expanded) []Row` (pure, no I/O). Today
it emits every yak honoring only fold state. This round generalizes filtering by
adding **predicate-based visibility** to the flatten step, with
**ancestor preservation**: a yak is shown if it matches the active filters OR if
any descendant matches (so matches stay reachable in the hierarchy).

All filters compose: a yak renders only if it passes *every* active filter (or
is the ancestor of something that does).

## Architecture

### Pure layer: `internal/tree`

Add a filtered flatten that takes a predicate. The predicate decides whether a
yak "matches"; ancestor preservation is handled by the walk.

```go
// Predicate reports whether a yak matches on its own merits (not counting
// descendants). nil children handling and ancestor-keeping live in the walk.
type Predicate func(y *yaks.Yak) bool

// FlattenFiltered is Flatten plus a predicate. A node is emitted if it matches
// the predicate or has any descendant that does. Fold state still hides the
// children of a collapsed (but visible) node. When pred is nil, behaves exactly
// like Flatten.
func FlattenFiltered(roots []yaks.Yak, expanded map[string]bool, pred Predicate) []Row
```

`Flatten` stays as-is (or becomes a thin wrapper calling `FlattenFiltered` with
a nil predicate — implementer's choice, as long as existing behavior and tests
are unchanged).

**Ancestor preservation rule:** during the depth-first walk, a node is emitted
when `pred(node)` is true OR any node in its subtree matches. A node kept *only*
because a descendant matched is still shown (so the path to the match is
visible), and its matching descendants are shown; non-matching siblings of the
match are not.

**Interaction with fold state:** if a visible node is collapsed
(`expanded[id] == false`), its children are not walked for display — but
collapse must not hide a node that itself matches. Practically: compute subtree
match first; emit matched/ancestor nodes; only descend into children when the
node is expanded. A matching node under a collapsed ancestor is unreachable
until the user expands — acceptable and consistent with how fold already works.

### Model layer: `internal/ui`

New view-state fields on `Model`:

```go
hideDone  bool            // H: hide done yaks (and done subtrees with no active descendant)
wipFocus  bool            // W: show only wip/blocked (+ ancestors)
searching bool            // true while the search input line is open
search    textinput.Model // one-line incremental name filter
query     string          // committed search text (applied even when input closed)
```

`rebuildRows` composes the active filters into one predicate and calls
`FlattenFiltered`:

- `hideDone`: a yak matches if its state != done.
- `wipFocus`: a yak matches if its state == wip or blocked.
- text query (from `search.Value()` while searching, or `query` once committed):
  a yak matches if its name contains the query, case-insensitive.

The composed predicate ANDs the active ones; inactive filters contribute no
constraint. Ancestor preservation (from `FlattenFiltered`) keeps parents of
matches visible even if the parent itself wouldn't match.

After any toggle or query change, `rebuildRows` runs, then the cursor is
re-resolved: keep it on the same yak id if still visible (via `IndexOfID`),
otherwise `ClampCursor` to the nearest valid row. Detail pane refreshes.

### Keys (`internal/ui/keys.go`)

Add bindings (capitals are free; lowercase w/b/d/t are triage):

- `H` → toggle `hideDone`
- `W` → toggle `wipFocus`
- `f` → enter search mode

In **search mode** (`searching == true`), key routing mirrors the existing
edit-mode pattern in `handleKey`: keystrokes go to the `textinput` first, except:

- `enter` → commit: `query = search.Value()`, `searching = false`. The filter
  stays applied; the user can now navigate/triage the narrowed tree.
- `esc` → cancel: clear `search`, `query = ""`, `searching = false`, restoring
  the full tree (subject to any H/W toggles still on).

While searching, nav/triage keys are text input, not commands — same contract
as edit mode.

### View (`internal/ui/model.go`)

- While `searching`, the status/help bar area shows the search input line
  (e.g. `search: auth▍`).
- When not searching but filters/query are active, the bar shows a compact
  indicator of active view state, e.g. `[hide-done] [wip-focus] [search: auth]`,
  so the user always knows why the tree looks narrowed.
- Empty-result case: if the active filters hide everything, the tree pane shows
  a graceful message (e.g. `No yaks match the current filters — press esc / H /
  W to clear`) rather than a blank pane. (Consistent with the existing graceful
  empty-state guard.)

## Data flow

```
press H / W
  → toggle field → rebuildRows (compose predicate) → re-resolve cursor → refreshDetail

press f
  → searching = true, search.Focus()
  → each keystroke: search.Update → rebuildRows (live) → re-resolve cursor
  → enter: query = search.Value(); searching = false   (filter persists)
  → esc:   search reset; query = ""; searching = false  (full tree)
```

## Error handling / edge cases

- **Everything filtered out:** graceful "no matches" message, never a blank or
  crashing pane. Cursor clamps to 0.
- **Cursor on a now-hidden yak:** falls back to nearest valid row (existing
  clamp logic), never dangles out of bounds.
- **Reload while filtered:** a reload (`r`, or post-triage) rebuilds rows
  through the same predicate, so the active view is preserved across reloads.
- **Search + fold:** a match under a collapsed node stays hidden until expanded;
  documented, consistent with current fold behavior. No special-casing.

## Testing

Pure-layer (`internal/tree`), table-driven, the bulk of the coverage:

- `FlattenFiltered` with nil predicate == `Flatten` (behavior parity).
- Predicate matches a leaf → leaf shown, its ancestors shown, unrelated branches
  hidden.
- Predicate matches only a deep descendant → full ancestor chain preserved.
- No matches → empty result.
- Collapsed ancestor → matching descendant not emitted (fold wins for display).

Model-layer (`internal/ui`):

- `H` toggles hideDone; done yak with an active child stays (ancestor rule);
  done leaf disappears.
- `W` shows only wip/blocked + ancestors.
- `f` enters search; typing narrows rows live; `enter` commits and keeps the
  filter; `esc` clears it.
- While searching, a triage key (`d`) is text input, not "done" — mirrors the
  edit-mode routing test.
- Filters compose (H + search both applied).
- Cursor re-resolution: cursor on a yak hidden by a toggle clamps to a valid
  row.

## Out of scope (future sub-rounds)

- Add / remove / rename yaks (round 2: simple CRUD).
- Re-parent (move under/to-root) and tag management (round 3: structural).
- True sibling reordering — **not possible**: yx has no reorder command; order
  is insertion order and `yx move` only re-parents. Noted so it isn't
  re-proposed.
- Tag-based filtering — natural follow-on once tag management exists; deferred.
- Persisting filter/search state across TUI restarts.

## Notes

- New dependency surface: `bubbles/textinput` (same family as the already-used
  `bubbles/textarea`; likely already in the module cache). The plan verifies and
  runs `go mod tidy` if needed.
- No new yx calls — this round is read-only over data already loaded.
