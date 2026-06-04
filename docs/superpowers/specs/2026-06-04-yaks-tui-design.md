# yaks-tui — Design Spec (v1)

**Date:** 2026-06-04
**Status:** Approved for planning

## Summary

`yaks-tui` is a keyboard-driven, two-pane terminal UI built on top of the
[yaks](https://github.com/mattwynne/yaks) CLI (`yx`). It lets a human navigate
the yak discovery tree interactively — moving a cursor with the keyboard,
expanding/collapsing nodes, viewing a yak's rendered markdown context, and
triaging state (todo / wip / blocked / done) with single keystrokes.

It is a **hybrid orchestrator**: it owns the screen and event loop, but treats
`yx` as the single source of truth for all data and mutations, and composes
best-in-class tools for specific jobs (glamour for markdown, fzf for fuzzy
find).

## Goals (v1)

The v1 scope is the **browse + triage** loop:

- Navigate the yak tree (up/down, expand/collapse, jump-to via fzf).
- View a yak's detail: rendered markdown context + fields + tags + breadcrumb.
- Change state (todo → wip → blocked → done) with single keys.
- Reload from disk (after every local mutation, and on demand).
- Graceful behavior in every edge/error case.

## Non-Goals (v1)

Deferred to later versions (see **Future Work**):

- Creating / renaming / moving / removing yaks.
- Editing context and fields.
- Tag add/remove.
- TUI-driven `yx sync`.
- Live file-watching of external changes.
- Configurable layout.

## Core Principle

**`yx` is the only thing that touches `.yaks/`.** We never read or write the
yaks store directly. Every read is `yx list/show --format json`; every write is
a `yx` subcommand. This inherits yaks' git-CRDT correctness for free and makes
it impossible for the TUI to corrupt state.

## Technology

- **Language:** Go.
- **TUI framework:** Bubble Tea (Model-View-Update), with Lip Gloss (styling)
  and Bubbles (widgets: `viewport`, `help`, `key`, and later `textarea`).
- **Markdown rendering:** [glamour](https://github.com/charmbracelet/glamour),
  in-process (no `glow` subprocess). Chosen for direct library access and
  because it's the same renderer glow uses.
- **Distribution:** single static binary, drops in next to `yx`.

### Composition Map

| Job | Tool | When |
|-----|------|------|
| Data model & mutations | `yx` (via `os/exec`, JSON in/out) | v1 |
| Markdown **viewing** | glamour (in-process) | v1 |
| Fuzzy jump-to-yak | `fzf` (shell out) | v1 |
| Markdown **editing** | `textarea` + live glamour preview, **and** `$EDITOR` shell-out | v1.1 |
| Editing context/fields trigger | `$EDITOR` | v1.1 |

## yaks Data Model (reference)

From `yx list --format json` (full recursive tree) and `yx show <id|name>
--format json`. Key fields per yak:

- `id` — stable slug, e.g. `write-tests-hgny`. **Commands accept both id and
  name; the TUI addresses yaks by `id`** because names can collide.
- `name` — display name.
- `state` — one of `todo`, `wip`, `blocked`, `done`.
- `context` — markdown body (nullable).
- `fields` — custom key/value map (short + long fields; long fields are
  multi-line markdown).
- `tags` — list (stored with `@` prefix, e.g. `@bug`).
- `parent_id`, `children` — tree structure.
- `depth`, `connector`, `prefix` — yaks' own rendering hints (we compute our
  own, but they're available).

`show --format json` additionally provides `breadcrumb`, `created_at`,
`created_by`, and split `short_fields` / `long_fields`.

**Mutations used in v1:** `yx state <id> <state>` (and equivalently `yx start`,
`yx done`). Structural mutations (`add`, `move`, `rename`, `remove`, `context`,
`field`, `tag`, `sync`) are v1.1+.

## Layout

**Two-pane, side by side** (left: tree, right: detail). The detail pane updates
live as the cursor moves in the tree. Chosen for fast triage. Configurable
layout (selectable A/B/C arrangements) is deferred to Future Work.

```
┌────────────────────┬───────────────────────────────┐
│ ● deploy app       │ set up CI                      │
│ ├─ ● write tests   │ deploy app ›                   │
│ ╰─ ◌ set up CI  ◀  │ ## Notes                       │
│    ╰─ ◌ fix linter │ Configure the pipeline...      │
│                    │ @bug  #blocked                 │
├────────────────────┴───────────────────────────────┤
│ j/k move · h/l fold · w/b/d/t state · / find · ? help│
└──────────────────────────────────────────────────────┘
```

## Components & Structure

```
yaks-tui/
├── main.go                 # entry: locate repo, bootstrap, run program
├── internal/
│   ├── yaks/               # the ONLY package that knows about `yx`
│   │   ├── client.go       # exec wrapper: List(), Show(id), SetState(id,s)...
│   │   └── types.go        # Yak struct ↔ JSON from `yx ... --format json`
│   ├── tree/               # flatten nested yaks → navigable rows + expand/collapse
│   ├── ui/
│   │   ├── model.go        # top-level Bubble Tea model (layout, focus, state)
│   │   ├── treepane.go     # left pane: cursor, rendering, selection
│   │   ├── detailpane.go   # right pane: glamour + viewport, fields, tags, crumb
│   │   ├── keys.go         # key bindings (one place; feeds help bar + future config)
│   │   └── styles.go       # Lip Gloss styles, status→color mapping
│   └── shell/              # generic helpers to shell out (fzf, later $EDITOR)
└── ...
```

### Boundaries

- `internal/yaks` is the **only** code that runs `yx`. Everything else speaks Go
  structs. Swappable and testable via a fake `execer`.
- `internal/tree` is **pure logic** (nested → flat rows, expansion state) — no
  I/O, no Bubble Tea — so it is trivially unit-testable. Most logic bugs would
  live here, so it gets the most coverage.
- The `ui` panes are **dumb renderers** driven by the top-level model; they do
  not fetch data themselves.

## Data Flow (one cycle)

1. Startup → `yaks.Client.List()` → `[]Yak` tree.
2. `tree.Flatten(tree, expansionState)` → `[]Row` the left pane renders.
3. Cursor moves → selected `Row.ID` → detail pane renders that yak. **v1 renders
   the detail pane from the tree JSON we already have** (`list --format json`
   already includes `context`, `fields`, `tags`); fall back to `yx show <id>`
   only if something needed turns out to be missing.
4. Triage key (e.g. `w` = wip) → `yaks.Client.SetState(id, "wip")` → on success,
   re-run `List()` → re-flatten, **preserving cursor position and expansion
   state**. This is the "reload after every local mutation" model.

## Data Freshness

yaks is a git-CRDT system; external changes arrive in a batch when `yx sync`
pulls remote events into `.yaks/`, not continuously. The freshness model:

- **Reload after every local mutation** — keeps the user's own actions
  consistent (covers ~95% of the value).
- **Manual reload (`r`)** — catch-all, including when someone runs `yx sync` in
  another terminal while the TUI is open.
- **(v1.1) Reload after TUI-driven sync** — once `sync` becomes a TUI action, it
  triggers a reload as part of completing, so freshly-pulled remote yaks appear
  immediately.

No file-watching in v1; live external-change watching is deferred.

## State, Focus & Keybindings

**Focus model:** exactly one pane focused at a time (tree or detail). `Tab`
toggles. Tree focus = navigation + triage; detail focus = scroll the markdown.
The focused pane gets a highlighted border (Lip Gloss).

**v1 keymap** (centralized in `keys.go`, surfaced via the `bubbles/help` widget;
vim keys are the default, arrows also work):

| Key | Action |
|-----|--------|
| `↑`/`k`, `↓`/`j` | move cursor up/down |
| `→`/`l`, `←`/`h` | expand / collapse node |
| `Enter` | toggle expand/collapse |
| `Tab` | switch focus (tree ↔ detail) |
| `w` | set state → wip |
| `b` | set state → blocked |
| `d` | set state → done |
| `t` | set state → todo (reset) |
| `/` | fuzzy jump-to-yak (fzf) |
| `r` | reload from disk |
| `?` | toggle full help |
| `q` / `Ctrl-C` | quit |

**Deferred key (v1.1):** delete on **`#`** (Shift+3), Gmail-style — deliberate
and paired with a confirmation prompt. `d` remains "done".

## Error Handling & Graceful States

Every external dependency (`yx`, `fzf`, glamour) **degrades gracefully** — a
missing or failing dependency disables its feature with an explanation, never
takes down the app. No raw stack traces are ever shown to the user.

| Situation | Behavior |
|-----------|----------|
| **No `.yaks/` repo** | Friendly full-screen empty state: "No yaks here yet. Start one with `yx add \"…\"`, then reopen. Press `q` to quit." |
| **`yx` not on PATH** | Startup check; clear message: "`yx` not found — install yaks first," with the repo link. Exit cleanly. |
| **Empty repo (0 yaks)** | Tree-pane empty state: "No yaks yet. (v1.1 will let you add them here.)" Detail pane shows a hint. |
| **A `yx` command fails** (mutation error, bad JSON) | Transient red status-line message: "Couldn't set state: <reason>". UI stays usable; we re-read to resync rather than crash. |
| **`fzf` not installed** | `/` shows a status note: "Fuzzy find needs `fzf` installed." Everything else keeps working. |
| **Terminal too small** | Minimum-size guard: show a "resize me" message rather than rendering garbage. |
| **glamour render error** | Fall back to showing raw markdown text rather than blanking the pane. |

## Testing Strategy

Business logic (tree math, JSON mapping, update reducers) is thoroughly
unit-tested; rendering is tested lightly. TDD where it pays — especially
`internal/tree` and `internal/yaks`.

| Layer | How we test it | Why it's testable |
|-------|---------------|-------------------|
| `internal/tree` (flatten, expand/collapse, cursor math) | Plain Go table-driven unit tests | Pure functions — nested `[]Yak` in, `[]Row` out. No I/O, no TUI. |
| `internal/yaks` (the `yx` wrapper) | Unit tests against captured real JSON fixtures + a fake `execer` interface | Snapshot real `yx ... --format json` into testdata; test parsing/mapping against it. Exec is behind an interface, so inject a fake. |
| `internal/ui` (model update logic) | Bubble Tea `teatest` harness: key-event → state assertions on critical flows (cursor move, triage key → correct mutation, focus toggle) | Headless model testing; assert on model state, not pixels. |
| Rendering / styles | Light golden-file tests on a couple of representative views; mostly manual | Pixel-exact TUI output is brittle to over-test; keep thin. |
| End-to-end | One smoke test: temp git repo, real `yx` seeds yaks, launch model headless, drive a triage key, assert state changed | Validates the real `yx` integration once, end to end. |

## Future Work

Ordered roughly by expected priority:

**v1.1 — structural editing & mutation**
- Add / rename / move / remove yaks (remove on `#` with confirm).
- Edit context & fields: in-app **`textarea` + live glamour preview** editor
  (headline feature), plus `$EDITOR` shell-out for power users.
- Tag add/remove.
- TUI-driven `yx sync`, which triggers a reload on completion.

**v2 — quality of life**
- Multi-select for batch triage (set state on several yaks at once).
- Filtering/scoping the tree by state or tag (mirrors `yx list --tag`/`--only`).
- `prune` (clear done yaks) from the TUI.

**v3 — configurable layout**
- Selectable layouts: two-pane (A), stacked (B), tree-first + overlay (C).

**Later / parked**
- Live file-watching (fsnotify) for external changes with cursor/expansion
  preservation.
