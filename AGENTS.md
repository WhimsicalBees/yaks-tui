# AGENTS.md

Guidance for AI coding agents (Claude Code, OpenCode, etc.) working in this repo.

## What this is

`yaks-tui` is an interactive, keyboard-driven terminal UI for
[yaks](https://github.com/mattwynne/yaks), built with Go and Bubble Tea. It
shells out to the `yx` CLI for all data; it never touches `.yaks/` directly.

## Build, test, run

```bash
make build          # go build -o bin/yaks-tui .
go test ./...       # full suite (PTY/e2e tests skip without yx/fzf)
go vet ./...
gofmt -l .          # must report nothing
go run . .          # run the TUI in the current repo
```

## Conventions

- **Bubble Tea MVU.** `Model` methods use value receivers; `Update` mutates a
  local copy and returns it. Keep `Init/Update/View` honest.
- **Pure logic is isolated.** Tree flattening/cursor math lives in
  `internal/tree` with no Bubble Tea or I/O dependency, and is unit-tested
  directly.
- **All `yx` access goes through `internal/yaks`** behind the `Runner`
  interface, so tests inject a fake instead of running the binary.
- **Table-driven tests.** Follow the existing patterns in `*_test.go`.
- **Graceful UX always.** No raw stack traces — every error and edge state
  shows a friendly message.

## Critical gotcha: never query the terminal in the render loop

Do **not** call glamour's `WithAutoStyle` (or anything that calls
`termenv.HasDarkBackground`) while rendering. It writes an OSC query to the
terminal and blocks reading the reply from stdin, racing Bubble Tea's input
reader and causing multi-second input lag. The markdown style is resolved
**once** at startup in `ui.New` and stored on the model. See
`docs/superpowers/specs/2026-06-04-yaks-tui-design.md` and the PTY regression
test in `main_pty_test.go`.

## Task tracking

Work is tracked in `.yaks/` via `yx`. Run `yx list` to orient before starting.
