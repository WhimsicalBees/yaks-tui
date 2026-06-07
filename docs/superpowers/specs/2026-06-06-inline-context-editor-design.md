# yaks-tui v1.1 — Inline context editor

**Date:** 2026-06-06
**Status:** Approved, ready for implementation plan

## Goal

Let the user edit a yak's **context body** (the markdown shown by `yx show` /
`yx context --show`) without leaving the TUI. Editing happens inline in the
detail pane using `bubbles/textarea` — no `$EDITOR` subprocess, no separate
screen.

Editing other fields (name, tags, custom fields) is explicitly **out of scope**
for this iteration. We do one thing at a time; context body first.

## UX flow

- **Enter edit mode:** press `e` from either pane. The selected yak's current
  context loads into a textarea that replaces the glamour-rendered detail. The
  tree pane stays visible so the user keeps their place. If the yak has no
  context, the textarea starts empty.
- **Save:** `ctrl+s` writes the body via the client, exits edit mode, reloads
  the detail pane.
- **Cancel:** `esc` discards edits and exits edit mode.
- **No yak selected** (empty tree): `e` is a no-op.

The help bar shows `ctrl+s save  esc cancel` while editing.

## Architecture

### Model (`internal/ui/model.go`)

Two new fields:

```go
editing bool
ta      textarea.Model
```

- `ta` is initialized in `New()` and resized in the existing
  `tea.WindowSizeMsg` handler alongside the rest of the layout.
- While `editing == true`, the detail pane renders the textarea instead of the
  glamour output. The tree pane is unchanged.

**Key routing while editing:** all keypresses go to `ta.Update()` first, except:

- `ctrl+s` → intercept, trigger save command
- `esc` → intercept, cancel (set `editing = false`, discard)

Everything else — including `ctrl+c` — passes through to the textarea so normal
text editing (newlines, navigation, etc.) works.

**Entering edit mode:** `e` (when not already editing and a yak is selected)
sets `editing = true` and loads the selected yak's `Context` (or empty string if
nil) into the textarea via `ta.SetValue`.

### Save command + messages

A `saveContextCmd(id, content string) tea.Cmd` calls `client.SetContext` and
returns either:

- `contextSavedMsg` → exit edit mode, reload detail (reuse the existing
  reload-preserving-cursor path so the tree state is kept)
- `errMsg` → show the error in the message line, **stay in edit mode** so edits
  are not lost

### Client (`internal/yaks/client.go`)

New method:

```go
// SetContext replaces a yak's context body by id. The content is piped to
// `yx context <id>` on stdin, which yx reads when stdin is present.
func (c *Client) SetContext(ctx context.Context, id, content string) error
```

This requires a stdin-aware addition to the `Runner` interface:

```go
type Runner interface {
    Run(ctx context.Context, args ...string) ([]byte, error)
    RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error)
}
```

`ExecRunner.RunWithInput` mirrors `Run` but sets `cmd.Stdin` to a reader over
the content string. The same stderr-surfacing / error-wrapping logic as `Run`
applies. Test fakes (`dirRunner`, any UI fake) implement the new method; the
fake can assert on the piped stdin.

The consumer-side `dataSource` interface in the `ui` package gains
`SetContext(ctx, id, content) error`.

## Data flow

```
press e
  → editing = true, ta.SetValue(selected.Context)
  → detail pane renders textarea

type / edit
  → ta.Update handles keystrokes

press ctrl+s
  → saveContextCmd(id, ta.Value())
      → client.SetContext
          → RunWithInput(content, "context", id)   # pipes body on stdin
      → contextSavedMsg | errMsg

contextSavedMsg
  → editing = false, reload detail (cursor preserved)

errMsg
  → show message, stay in editing

press esc
  → editing = false (discard)
```

## Error handling

- **Save fails:** surface the error in the message line, stay in edit mode.
  Same graceful pattern as triage failures — no raw stack traces.
- **Empty content:** valid. Saving an empty textarea clears the context body.
- **No yak selected:** `e` is a no-op.

## Testing

- **`internal/yaks`:**
  - Unit-test `SetContext` with a fake Runner asserting the piped stdin equals
    the content and args are `context <id>`.
  - Extend the e2e test to round-trip: `SetContext`, then `List`/show, verify
    the body persisted.
- **`internal/ui`:**
  - Table test for key routing while `editing == true`: `ctrl+s` saves, `esc`
    cancels, other keys reach the textarea.
  - Test that `e` enters edit mode and loads existing context into the textarea.
- No PTY test needed: the input-lag class of bug is already guarded by the
  existing PTY regression test, and edit-mode routing is pure model logic.

## Out of scope (future)

- Editing name, tags, custom fields (next iteration, one at a time)
- Adding / moving / removing yaks
- TUI-driven `yx sync`
- Configurable layout
