# yaks-tui OSS Readiness — Design

**Date:** 2026-06-06
**Status:** Approved, ready for implementation plan

## Goal

Prepare `yaks-tui` to be published as an open-source project: complete
user-facing documentation (Diátaxis), a demo GIF in the README, an MIT license,
committed AI-tooling config for Claude Code and OpenCode, and the standard OSS
scaffolding (CI, contributing guide, badges, changelog).

This is packaging and documentation work. It does not change application
behavior. The one code change is the Go module path.

## Workstreams

Six independent workstreams. Each can be built and committed on its own.

1. Documentation (Diátaxis)
2. Demo media (VHS)
3. License (MIT)
4. AI tooling config (Claude Code + OpenCode)
5. OSS scaffolding (CI, CONTRIBUTING, badges, CHANGELOG)
6. Module path (`go install` support)

---

## 1. Documentation (Diátaxis)

Full four-quadrant structure, right-sized for a small tool. Organized by **user
need**, never by feature. Reference pages are pure spec — every key and
precondition verified against source, nothing invented.

New tree (the existing `docs/superpowers/` stays as internal build history and
is not part of user docs):

```
docs/
  README.md                  # index linking into the four sections
  tutorials/
    getting-started.md       # install → open TUI → first triage session
  how-to/
    edit-context.md          # edit a yak's body inline
    fuzzy-jump.md            # jump to a yak with /
    triage-workflow.md       # move yaks through todo→wip→done
  reference/
    keybindings.md           # complete key table (verified vs internal/ui/keys.go)
    cli-preconditions.md     # yx on PATH, .yaks/ initialized, gitignore, fzf optional
    configuration.md         # terminal/glamour style detection; no $EDITOR dependency
  explanation/
    why-a-tui.md             # the "compose existing tools" philosophy
    architecture.md          # MVU, in-process glamour, fzf via ExecProcess, input-lag fix
```

**Quadrant discipline:**

- **Tutorial** (`getting-started.md`): gets a brand-new user from nothing to a
  first successful triage session. Minimal explanation; links out to
  explanation/reference instead of teaching theory inline.
- **How-to** guides: each solves one real goal, assumes the reader knows the
  basics. No onboarding, no concept-teaching.
- **Reference**: accurate, complete, opinion-free. `keybindings.md` is generated
  from the actual bindings in `internal/ui/keys.go`. `cli-preconditions.md`
  mirrors the checks in `main.go`. No "use this when…" advice (that's how-to).
- **Explanation**: the *why*, zero step-by-step. `architecture.md` draws on the
  existing specs (two-pane MVU, glamour in-process, the
  `WithAutoStyle`/OSC-query input-lag fix, fzf via `tea.ExecProcess`).

`docs/README.md` is a short index that links into the four sections. The
top-level project `README.md` remains the marketing entry point and links to
`docs/`.

**Verification:** before writing reference content, read `internal/ui/keys.go`
and `main.go` so every documented key, precondition, and message is grounded in
the code.

---

## 2. Demo media (VHS)

A committed [VHS](https://github.com/charmbracelet/vhs) tape that generates the
README GIF deterministically.

```
docs/
  demo.tape          # VHS script (scripted keystrokes + timing)
  demo.gif           # generated output, committed so GitHub renders it
```

**Tape contents:** the tape is self-contained — it first sets up a throwaway
demo yaks repo in a temp dir (a handful of `yx add` commands) so the recording
never depends on the author's real `.yaks/`. Then it scripts a readable session:

1. Launch the TUI
2. Navigate the tree (`j`/`k`), expand a yak (`l`)
3. Triage: `w` to set wip
4. `e` to edit, type a context line, `ctrl+s` to save
5. `/` fuzzy jump to another yak
6. `q` to quit

Timing tuned for readability (not frantic).

**Build target:** `make demo` runs `vhs docs/demo.tape` to regenerate
`docs/demo.gif`.

**Constraints (documented):**

- Generating the GIF requires `vhs` installed (`brew install vhs`) and is run by
  a human in a real terminal — it cannot be produced in this automated session.
- The committed `demo.gif` means contributors and GitHub viewers never need VHS;
  only someone regenerating the GIF does.
- The README references `![demo](docs/demo.gif)`; the reference is wired in even
  before the GIF is first generated, so it lights up as soon as `make demo` is
  run.

---

## 3. License (MIT)

```
LICENSE            # MIT, © 2026 Lena Anne Krug
```

- Standard MIT text, copyright holder **Lena Anne Krug**, year 2026.
- README header gets the badge:
  `![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)`.
- README gets a short License section linking to `LICENSE`.

---

## 4. AI tooling config (Claude Code + OpenCode)

Single source of truth, both tools supported.

```
AGENTS.md                     # committed: project guidance for any AI agent
CLAUDE.md                     # symlink → AGENTS.md
.claude/settings.json         # committed: sanitized, portable allowlist
.claude/settings.local.json   # gitignored (personal overrides, incl. rtk)
```

**`AGENTS.md`** — project-specific, public-facing. Contents:

- What the project is (one paragraph).
- Build/test/run commands: `make build`, `go test ./...`, `go vet ./...`,
  `gofmt -l`, `go run . .`.
- Code conventions: Bubble Tea MVU with value receivers; pure logic in
  `internal/tree`; table-driven tests; graceful UX (no raw stack traces).
- The critical gotcha: **never call glamour `WithAutoStyle` (or otherwise query
  the terminal) in the render loop** — it races Bubble Tea's input reader and
  causes input lag. Resolve the style once at startup. Link the spec.
- yaks workflow note: this repo tracks work in `.yaks/` via `yx`.

This is *project* guidance only — none of the author's global `~/.claude`
personal rules.

**`CLAUDE.md`** — a symlink to `AGENTS.md`. Claude Code follows the symlink;
OpenCode reads `AGENTS.md` natively. One file to maintain.

**`.claude/settings.json`** — sanitized permission allowlist covering the tools
this project actually uses: `go`, `git`, `gofmt`, `yx`, `make`, `staticcheck`.
No `rtk`, no `/Users/lkrug` absolute paths, no personal MCP permissions.

**`.gitignore`** — add `.claude/settings.local.json` so personal overrides never
get committed.

**Privacy gate (build-time check):** before committing anything under
`.claude/` or the AGENTS/CLAUDE files, grep the staged content for `lkrug`,
`rtk`, and absolute home paths. Abort the commit if any match — nothing personal
leaks into the public repo.

---

## 5. OSS scaffolding

**CI** — `.github/workflows/ci.yml`, runs on push and PR:

- `go build ./...`
- `go vet ./...`
- `gofmt -l .` (fails if any file is unformatted)
- `go test ./...`

The PTY and e2e tests skip gracefully when `yx`/`fzf` are absent (they already
guard with `exec.LookPath` and `testing.Short()`), so CI stays green on a stock
runner without installing `yx`.

**CONTRIBUTING.md** — short: how to build and test, the TDD expectation, code
conventions (point at `AGENTS.md`), and the yaks workflow.

**README badges** — in the header: License (MIT), Go version, CI status.

**CHANGELOG.md** — [Keep a Changelog](https://keepachangelog.com) format, with
entries for v1 (browse + triage, fuzzy find) and v1.1 (inline context editor).

---

## 6. Module path

Change the module path so `go install` works from the public repo.

- `go.mod`: `module yaks-tui` → `module github.com/whimsicalbees/yaks-tui`.
- Update internal imports: `yaks-tui/internal/...` →
  `github.com/whimsicalbees/yaks-tui/internal/...` across all `.go` files.
- README gets an install line: `go install github.com/whimsicalbees/yaks-tui@latest`.
- Verify with `go build ./...` and `go test ./...` after the rename.

**Note:** creating the GitHub remote and pushing are separate actions the author
authorizes; this spec only changes the local module path and imports.

---

## Out of scope

- Creating the GitHub repository and pushing (author-authorized, separate step).
- Issue/PR templates (add when there are contributors).
- SPDX headers in source files (root LICENSE suffices for MIT).
- Any change to application behavior.

## Testing / verification

- `go build ./...` and `go test ./...` pass after the module-path rename.
- `gofmt -l .` reports nothing.
- Privacy grep over staged `.claude/` and AGENTS/CLAUDE content finds no
  `lkrug` / `rtk` / absolute-home-path matches.
- Manual (author, real terminal): `make demo` produces `docs/demo.gif`; the
  README renders it.
- Diátaxis reference docs cross-checked against `internal/ui/keys.go` and
  `main.go`.
