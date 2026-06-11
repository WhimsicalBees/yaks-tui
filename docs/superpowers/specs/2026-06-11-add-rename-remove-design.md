# Round 2 — Add / Rename / Remove yaks in the TUI

**Date:** 2026-06-11
**Status:** Approved
**Scope:** Simple CRUD on yaks from inside the TUI: create (child/root), rename, remove.

## Goal

Let a user manage the yak tree without dropping to the `yx` CLI. Three new
keyboard actions — add, rename, remove — built on the inline-input and
transient-status patterns the TUI already uses for search and the context
editor. Re-parenting and tag management are explicitly Round 3, not here.

## Constraints (from the `yx` CLI)

- `yx add [NAME]... --under <PARENT>` creates a yak; omit `--under` for a root.
  `--id <ID>` lets the caller choose the new yak's slug instead of letting yx
  auto-generate one. `--state` defaults to `todo`.
- `yx rename <FROM> <TO>` changes a name in place; no move.
- `yx remove [NAME]...` deletes; a yak with children requires `--recursive`.
- All three accept the stable **id slug** as the locator (verified:
  `yx show <id>` resolves by id). We address yaks by id everywhere because
  names collide.
- `.yaks/` is gitignored and backed by a git event store; a removed yak is
  **not** recoverable via git. Hence: confirm every removal.

## Section 1 — Client layer (`internal/yaks`)

Three new `Client` methods behind the existing `Runner` interface, mirroring
`SetState` / `SetContext`. Tests inject the fake `Runner` and assert exact argv.

### `Add(ctx, parentID, name string) (newID string, err error)`
- Generates a **collision-proof id** for the new yak (see below) and passes it
  via `--id <slug>`.
- argv: `yx add <name> --id <slug> --under <parentID>`; when `parentID == ""`,
  omit `--under` (root yak).
- Returns the chosen id so the UI can land the cursor on the new yak.

**Id generation (collision-proof):**
1. Slugify the name: lowercase, non-alphanumerics → `-`, collapse repeated
   `-`, trim leading/trailing `-`. Matches yx's slug style.
2. Append a 4-char random suffix (same shape yx uses, e.g. `…-iwg9`).
3. **Verify against the live tree** — `Add` is given the current set of
   existing ids (or does a `List` first); if the candidate collides,
   regenerate the suffix until unique.
4. Pass the verified-unique slug as `--id`.
5. **Fallback:** if `yx add --id` still errors (e.g. a race with another
   writer via `yx sync`), surface it gracefully as a status message
   ("couldn't create yak: …") rather than failing silently.

### `Rename(ctx, id, newName string) error`
- argv: `yx rename <id> <newName>`.

### `Remove(ctx, id string, recursive bool) error`
- argv: `yx remove <id>`, adding `--recursive` when `recursive` is true.

The `dataSource` interface in `internal/ui/model.go` gains the same three
methods so the model is testable against a stub.

## Section 2 — UI state & key routing (`internal/ui`)

One new single-line input mode handles add and rename (reusing the search
`textinput` pattern); a separate boolean gates the remove confirmation.

### New model fields
```go
inputMode  inputMode      // none | addChild | addRoot | rename
inputParID string         // parent id for addChild (captured on entry)
inputTgtID string         // target id for rename (captured on entry)
input      textinput.Model

confirming bool           // remove confirmation open
removeID   string         // captured target id
removeName string         // for the prompt text
removeKids int            // child count → recursive flag + prompt wording
```

### Keys (`keys.go`)
- `a` — add child under the cursor yak
- `A` — add root yak
- `R` — rename cursor yak
- `x` — remove cursor yak

None collide with the existing map.

### Routing in `Update` (established precedence)
- When `confirming`: only `y` / `n` / `esc` are live.
- Else when `inputMode != none`: keystrokes go to the input; Enter commits,
  Esc cancels.
- Else: normal tree keys.

This mirrors how `editing` and `searching` already gate input.

### Flows
- **`a`** → capture cursor yak's id as `inputParID`, open empty input
  (`addChild`). Enter → `client.Add(parID, name)` → reload preserving cursor
  onto the returned new id.
- **`A`** → `inputParID = ""`, open empty input (`addRoot`). Same commit path.
- **`R`** → capture cursor id as `inputTgtID`, open input **pre-filled** with
  the current name, cursor at end (`rename`). Enter → `client.Rename(id, name)`
  → reload preserving cursor on the same id.
- **`x`** → capture cursor id / name / child-count, set `confirming`. `y` →
  `client.Remove(id, recursive = kids > 0)` → reload; cursor falls to nearest
  sibling/parent. Anything else → cancel.

Empty-input Enter on add/rename is a **no-op cancel** (no blank names).

## Section 3 — Rendering, errors, tests

### Rendering
Input line and confirm prompt render in the **same footer slot** the search
line uses (one line above the help bar); only one transient affordance is
visible at a time.

- Add child:    `add child of "<parent>": <input>`
- Add root:     `add root: <input>`
- Rename:       `rename: <input>` (pre-filled)
- Confirm leaf: `remove "<name>"? (y/n)`
- Confirm tree: `remove "<name>" and its <N> children? (y/n)`

Success is silent (the tree just updates). Failures show a friendly line via
the existing transient `status` field — never a raw error:
`couldn't create yak: …`, `rename failed: …`, `remove failed: …`.

Help bar (`FullHelp`) gains `a/A add`, `R rename`, `x remove`.

### Error handling
Every `client.*` call returns a `tea.Cmd` emitting either a reload message or
`errMsg`; `errMsg` already routes to the graceful status line. CLI-level
failures (id-collision race, `yx` missing) surface there.

### Tests
- `client_test.go`: argv for `Add` (root vs `--under`, `--id` passed,
  uniqueness regen on collision), `Rename`, `Remove` (with/without
  `--recursive`).
- `model_test.go`: key routing per mode; Enter-commits / Esc-cancels /
  empty-is-noop; confirm `y` vs other; recursive flag derived from child
  count; cursor lands on new id after add.
- `render_test.go`: footer shows each prompt variant; help bar includes new
  keys.
- `e2e_test.go`: add → rename → remove round-trip against real `yx` when
  available (skips otherwise, per existing pattern).

## Out of scope (YAGNI)
- Initial context/state on add (use the existing `e` editor afterward).
- A separate "sibling" key (a sibling is just a child of the parent).
- Undo, multi-select.
- Re-parenting and tag management — Round 3.
