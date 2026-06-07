# yaks-tui OSS Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `yaks-tui` ready to publish as open source — module path, license, Diátaxis docs, demo GIF, AI-tooling config, OSS scaffolding, and the public GitHub repo.

**Architecture:** Seven independent workstreams, each its own task and commit. The module-path rename lands first because it touches every `.go` file; everything after assumes the new path. The public push is last, behind a confirmation gate.

**Tech Stack:** Go, Bubble Tea, the `yx` CLI, VHS (demo GIF), GitHub Actions (CI), `gh` CLI (repo creation).

---

## File Structure

**Created:**
- `LICENSE` — MIT text.
- `AGENTS.md` — project guidance for AI agents (build/test/conventions/gotchas).
- `CLAUDE.md` — symlink → `AGENTS.md`.
- `.claude/settings.json` — sanitized, portable permission allowlist.
- `CONTRIBUTING.md` — contributor build/test/conventions guide.
- `CHANGELOG.md` — Keep a Changelog format.
- `.github/workflows/ci.yml` — build/vet/gofmt/test on push + PR.
- `docs/README.md` — Diátaxis index.
- `docs/tutorials/getting-started.md`
- `docs/how-to/{edit-context,fuzzy-jump,triage-workflow}.md`
- `docs/reference/{keybindings,cli-preconditions,configuration}.md`
- `docs/explanation/{why-a-tui,architecture}.md`
- `docs/demo.tape` — VHS script; `docs/demo.gif` — generated output.

**Modified:**
- `go.mod` + all 13 `.go` files importing `yaks-tui/internal/...`.
- `.gitignore` — add `.claude/settings.local.json`.
- `Makefile` — add `demo` target.
- `README.md` — badges, install line, demo GIF, docs link.

**Untouched:** `docs/superpowers/` (internal build history), application behavior.

---

## Task 1: Module path rename

**Files:**
- Modify: `go.mod:1`
- Modify (imports): `main.go`, `internal/tree/tree.go`, `internal/tree/tree_test.go`, `internal/ui/model.go`, `internal/ui/model_test.go`, `internal/ui/styles.go`, `internal/ui/render_test.go`, `internal/ui/detailpane.go`, `internal/shell/fzf.go`, `internal/shell/fzf_test.go`

- [ ] **Step 1: Verify the baseline is green**

Run: `go test ./...`
Expected: PASS (66 tests across 5 packages).

- [ ] **Step 2: Rewrite the module directive**

Edit `go.mod` line 1: `module yaks-tui` → `module github.com/WhimsicalBees/yaks-tui`

(Casing is exact — Go module paths are case-sensitive; the org login is `WhimsicalBees`.)

- [ ] **Step 3: Rewrite every internal import**

Run this in the repo root to rewrite all import paths at once:

```bash
grep -rl "yaks-tui/internal" --include="*.go" . | xargs sed -i '' 's#yaks-tui/internal#github.com/WhimsicalBees/yaks-tui/internal#g'
```

(`sed -i ''` is the BSD/macOS form — empty backup suffix.)

- [ ] **Step 4: Verify no old paths remain**

Run: `grep -rn "\"yaks-tui/internal" --include="*.go" .`
Expected: no output (exit 1).

- [ ] **Step 5: Build and test under the new path**

Run: `go build ./... && go test ./...`
Expected: PASS (66 tests).

- [ ] **Step 6: Confirm formatting**

Run: `gofmt -l .`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add go.mod main.go internal/
git commit -m "refactor: module path github.com/WhimsicalBees/yaks-tui for go install"
```

---

## Task 2: MIT License

**Files:**
- Create: `LICENSE`

- [ ] **Step 1: Write the LICENSE file**

Create `LICENSE` with exactly this content:

```
MIT License

Copyright (c) 2026 Lena Anne Krug

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Commit**

```bash
git add LICENSE
git commit -m "docs: add MIT license"
```

---

## Task 3: AI tooling config

**Files:**
- Create: `AGENTS.md`
- Create: `CLAUDE.md` (symlink → `AGENTS.md`)
- Create: `.claude/settings.json`
- Modify: `.gitignore`

- [ ] **Step 1: Write AGENTS.md**

Create `AGENTS.md` with this content:

````markdown
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
````

- [ ] **Step 2: Create the CLAUDE.md symlink**

Run:

```bash
ln -s AGENTS.md CLAUDE.md
```

- [ ] **Step 3: Verify the symlink resolves**

Run: `cat CLAUDE.md | head -1`
Expected: `# AGENTS.md`

- [ ] **Step 4: Write the sanitized .claude/settings.json**

Create `.claude/settings.json` with this content (no `rtk`, no usernames, no absolute paths):

```json
{
  "permissions": {
    "allow": [
      "Bash(go *)",
      "Bash(git *)",
      "Bash(gofmt *)",
      "Bash(make *)",
      "Bash(yx *)",
      "Bash(staticcheck ./...)"
    ]
  }
}
```

- [ ] **Step 5: Gitignore the personal local settings**

Append to `.gitignore`:

```
.claude/settings.local.json
```

- [ ] **Step 6: Privacy gate — grep staged content for leaks**

Stage, then grep the staged blobs (not the working tree) for personal data:

```bash
git add AGENTS.md CLAUDE.md .claude/settings.json .gitignore
git diff --cached | grep -nE "lkrug|/Users/|rtk" || echo "CLEAN"
```

Expected: `CLEAN`. If anything matches, fix before committing.

- [ ] **Step 7: Commit**

```bash
git commit -m "chore: AI tooling config (AGENTS.md + CLAUDE.md symlink + sanitized settings)"
```

---

## Task 4: Diátaxis documentation

**Files:**
- Create: `docs/README.md`, `docs/tutorials/getting-started.md`, `docs/how-to/edit-context.md`, `docs/how-to/fuzzy-jump.md`, `docs/how-to/triage-workflow.md`, `docs/reference/keybindings.md`, `docs/reference/cli-preconditions.md`, `docs/reference/configuration.md`, `docs/explanation/why-a-tui.md`, `docs/explanation/architecture.md`

> Reference facts below are grounded in `internal/ui/keys.go` (key bindings) and `main.go` (preconditions). Do not invent options.

- [ ] **Step 1: Write the docs index**

Create `docs/README.md`:

```markdown
# yaks-tui documentation

Organized by the [Diátaxis](https://diataxis.fr) framework.

- **[Tutorials](tutorials/)** — learning-oriented. Start here if you're new.
  - [Getting started](tutorials/getting-started.md)
- **[How-to guides](how-to/)** — task-oriented recipes.
  - [Edit a yak's context](how-to/edit-context.md)
  - [Jump to a yak with fuzzy find](how-to/fuzzy-jump.md)
  - [Triage your yaks](how-to/triage-workflow.md)
- **[Reference](reference/)** — precise technical description.
  - [Keybindings](reference/keybindings.md)
  - [CLI preconditions](reference/cli-preconditions.md)
  - [Configuration](reference/configuration.md)
- **[Explanation](explanation/)** — background and design rationale.
  - [Why a TUI](explanation/why-a-tui.md)
  - [Architecture](explanation/architecture.md)
```

- [ ] **Step 2: Write the tutorial**

Create `docs/tutorials/getting-started.md`:

```markdown
# Getting started

This tutorial takes you from nothing to your first triage session. By the end
you'll have browsed a yak tree, opened a yak's detail, and changed its state —
all from the keyboard.

## Before you begin

You need the `yx` (yaks) binary on your PATH. If you don't have it, install it
from [yaks](https://github.com/mattwynne/yaks) first.

## 1. Get a yaks repo

yaks-tui reads an existing yaks repo. If you don't have one, make a throwaway:

    mkdir yak-demo && cd yak-demo
    git init
    echo '.yaks' >> .gitignore
    yx add "Try yaks-tui"
    yx add "Read the docs" --under "Try yaks-tui"

## 2. Build and launch

From the yaks-tui source directory:

    make build

Then, from inside your yaks repo, run the binary:

    /path/to/yaks-tui/bin/yaks-tui

You'll see two panes: the yak tree on the left, rendered detail on the right.

## 3. Move around

Press `j` and `k` (or the arrow keys) to move the cursor up and down the tree.
Watch the detail pane on the right update as you move. Press `l` to expand a yak
that has children, `h` to collapse it.

## 4. Do your first triage

Put the cursor on a yak and press `w`. Its state changes to *wip* (work in
progress) and the tree reloads with the cursor still on your yak. Try `d` to
mark one *done*.

## 5. Quit

Press `q`. You're back at your shell.

That's a full session: browse, read, triage. Next, learn how to
[edit a yak's context](../how-to/edit-context.md) or see the complete
[keybindings](../reference/keybindings.md).
```

- [ ] **Step 3: Write the edit-context how-to**

Create `docs/how-to/edit-context.md`:

```markdown
# Edit a yak's context

Edit the markdown body of a yak without leaving the TUI.

1. Move the cursor to the yak you want to edit.
2. Press `e`. The right pane becomes an editable text area pre-filled with the
   yak's current context (empty if it has none).
3. Edit the text. All normal typing and cursor keys work.
4. Press `ctrl+s` to save. The pane returns to rendered markdown showing your
   new body.

To discard your changes instead, press `esc` — nothing is written.

If the save fails (for example, `yx` returns an error), the message appears in
the status bar and you stay in the editor, so your edits aren't lost.
```

- [ ] **Step 4: Write the fuzzy-jump how-to**

Create `docs/how-to/fuzzy-jump.md`:

```markdown
# Jump to a yak with fuzzy find

When the tree is large, jump straight to a yak by name instead of scrolling.

This feature needs [`fzf`](https://github.com/junegunn/fzf) on your PATH. If
it's not installed, pressing `/` shows a friendly message instead.

1. Press `/`. The terminal hands off to fzf, listing the currently visible yaks.
2. Type to filter, use the arrow keys to highlight a match.
3. Press `enter` to select. The cursor jumps to that yak in the tree.
4. Press `esc` in fzf to cancel — you return to the tree unchanged.
```

- [ ] **Step 5: Write the triage-workflow how-to**

Create `docs/how-to/triage-workflow.md`:

```markdown
# Triage your yaks

Move yaks through their states without leaving the keyboard. Put the cursor on
a yak and press one key:

| Key | Sets state to |
|-----|---------------|
| `t` | todo |
| `w` | wip (work in progress) |
| `b` | blocked |
| `d` | done |

After each change the tree reloads and the cursor stays on the same yak (matched
by its stable id), so you can triage several in a row.

To reload the tree manually — for example after another process ran `yx sync` —
press `r`.
```

- [ ] **Step 6: Write the keybindings reference**

Create `docs/reference/keybindings.md` (grounded in `internal/ui/keys.go`):

```markdown
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
| `/` | fuzzy jump (requires fzf) |
| `r` | reload |
| `?` | toggle help |
| `q` / `ctrl+c` | quit |

## While editing context

| Key | Action |
|-----|--------|
| `ctrl+s` | save and exit the editor |
| `esc` | cancel and discard |

All other keys are text input for the editor while editing — including
`ctrl+c`, which is *not* quit in this mode. Press `esc` first, then quit.
```

- [ ] **Step 7: Write the CLI-preconditions reference**

Create `docs/reference/cli-preconditions.md` (grounded in `main.go`):

```markdown
# CLI preconditions

yaks-tui checks these at startup and exits with a message if they aren't met.

| Requirement | Why | If missing |
|-------------|-----|------------|
| `yx` on PATH | all data comes from the yaks CLI | exits: "`yx` not found on PATH", with an install link |
| A yaks repo in the current directory, with `.yaks` gitignored | yaks-tui reads the repo via `yx list` | exits: "no yaks repo here (or `.yaks` not gitignored)", with a `yx add` hint |
| `fzf` on PATH | the `/` fuzzy jump hands off to fzf | optional; `/` shows a friendly message if absent |

Markdown is rendered in-process with
[glamour](https://github.com/charmbracelet/glamour); no external renderer such
as `glow` is required.
```

- [ ] **Step 8: Write the configuration reference**

Create `docs/reference/configuration.md`:

```markdown
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
```

- [ ] **Step 9: Write the why-a-tui explanation**

Create `docs/explanation/why-a-tui.md`:

```markdown
# Why a TUI

yaks is a CLI for tracking work as a tree of nested "yaks." The CLI is great for
quick adds and scripted use, but browsing and triaging a large tree means
running `yx list`, reading IDs, and typing follow-up commands. That loop is slow
when you're working through many yaks at once.

yaks-tui exists to make browsing and triage *interactive*: move with the
keyboard, see rendered detail as you go, change state with a single key. It
deliberately **composes existing tools** rather than reimplementing them:

- It shells out to `yx` for every operation — yaks-tui is a front end, not a
  second source of truth. Anything `yx` can do to the data, it remains the
  authority on.
- It renders markdown with [glamour](https://github.com/charmbracelet/glamour),
  the same engine behind `glow`, in-process.
- It delegates fuzzy finding to [`fzf`](https://github.com/junegunn/fzf) rather
  than building a matcher.

The result is a thin, focused layer: the parts unique to "browsing a yak tree"
are ours; everything else is borrowed from tools that already do it well.
```

- [ ] **Step 10: Write the architecture explanation**

Create `docs/explanation/architecture.md`:

```markdown
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
```

- [ ] **Step 11: Verify reference docs against source**

Run: `grep -oE 'WithKeys\("[^"]+"' internal/ui/keys.go`
Expected: confirms `e`, `w`, `b`, `d`, `t`, `/`, `r`, `?`, `q` etc. match the keybindings table. Fix the table if anything differs.

- [ ] **Step 12: Commit**

```bash
git add docs/README.md docs/tutorials docs/how-to docs/reference docs/explanation
git commit -m "docs: Diátaxis documentation (tutorial, how-to, reference, explanation)"
```

---

## Task 5: VHS demo GIF

**Files:**
- Create: `docs/demo.tape`, `docs/demo.gif`
- Modify: `Makefile`

- [ ] **Step 1: Install VHS**

Run: `brew install vhs`
Expected: installs `vhs` plus `ttyd` and `ffmpeg`. Verify: `vhs --version` prints a version.

- [ ] **Step 2: Write the tape**

Create `docs/demo.tape`. It sets up a self-contained demo repo in a temp dir so
the recording never depends on a real `.yaks/`:

```tape
# yaks-tui demo. Regenerate with: make demo
Output docs/demo.gif

Set Shell "bash"
Set FontSize 18
Set Width 1200
Set Height 700
Set Padding 20

# Set up a throwaway yaks repo so the demo is self-contained.
Hide
Type "cd $(mktemp -d) && git init -q && echo '.yaks' > .gitignore" Enter
Type "yx add 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Write docs' --under 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Record demo' --under 'Ship yaks-tui v1.1'" Enter
Type "yx add 'Cut release'" Enter
Type "yaks-tui" Enter
Show

Sleep 1.5s

# Navigate the tree.
Type "j" Sleep 600ms
Type "j" Sleep 600ms
Type "l" Sleep 800ms

# Triage: mark wip.
Type "w" Sleep 1s

# Edit context inline.
Type "e" Sleep 800ms
Type "Recorded with VHS." Sleep 800ms
Ctrl+S Sleep 1.2s

# Fuzzy jump.
Type "/" Sleep 800ms
Type "Cut" Sleep 800ms
Enter Sleep 1.2s

# Quit.
Type "q" Sleep 500ms
```

(This assumes `yaks-tui` is on PATH. Step 3 ensures that.)

- [ ] **Step 3: Make the binary available to the tape**

Run: `make build && export PATH="$PWD/bin:$PATH"`
Expected: `which yaks-tui` resolves to `./bin/yaks-tui`.

- [ ] **Step 4: Add the Makefile target**

Add to `Makefile` (and add `demo` to the `.PHONY` line):

```makefile
demo:
	vhs docs/demo.tape
```

The `.PHONY` line becomes:

```makefile
.PHONY: build test lint run demo
```

- [ ] **Step 5: Generate the GIF**

Run: `make demo`
Expected: produces `docs/demo.gif`. Verify it exists and is non-empty: `ls -lh docs/demo.gif`.

- [ ] **Step 6: Commit**

```bash
git add docs/demo.tape docs/demo.gif Makefile
git commit -m "docs: VHS demo tape and generated GIF, make demo target"
```

---

## Task 6: OSS scaffolding (CI, CONTRIBUTING, CHANGELOG, README)

**Files:**
- Create: `.github/workflows/ci.yml`, `CONTRIBUTING.md`, `CHANGELOG.md`
- Modify: `README.md`

- [ ] **Step 1: Write the CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - name: Build
        run: go build ./...
      - name: Vet
        run: go vet ./...
      - name: Format check
        run: test -z "$(gofmt -l .)"
      - name: Test
        run: go test ./...
```

(The PTY and e2e tests skip when `yx`/`fzf` are absent, so CI is green on a stock runner.)

- [ ] **Step 2: Verify the Go version matches go.mod**

Run: `grep '^go ' go.mod`
Expected: `go 1.26.2`. The workflow pins `go-version: '1.26'`, which covers the
1.26.x line. If go.mod's minor version differs, update `ci.yml` to match.

- [ ] **Step 3: Write CONTRIBUTING.md**

Create `CONTRIBUTING.md`:

````markdown
# Contributing

Thanks for your interest in yaks-tui!

## Setup

You need Go and the [`yx`](https://github.com/mattwynne/yaks) binary on your
PATH. `fzf` is optional (enables fuzzy jump).

## Build and test

```bash
make build      # build the binary
go test ./...   # run the full suite
go vet ./...
gofmt -l .      # must report nothing
```

## Conventions

- Test-driven where practical; follow the table-driven test style already in
  the `*_test.go` files.
- Keep pure logic (tree/cursor math) in `internal/tree`, free of UI and I/O.
- Run `gofmt` before committing.
- See [AGENTS.md](AGENTS.md) for architecture notes and one critical gotcha
  (never query the terminal in the render loop).

## Task tracking

This repo tracks its own work with yaks, in `.yaks/`. Run `yx list` to see
what's in flight.
````

- [ ] **Step 4: Write CHANGELOG.md**

Create `CHANGELOG.md`:

```markdown
# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.1.0]

### Added
- Inline context editor: press `e` to edit a yak's markdown body in the TUI,
  `ctrl+s` to save, `esc` to cancel.

## [1.0.0]

### Added
- Two-pane TUI: yak tree on the left, rendered markdown detail on the right.
- Keyboard navigation, fold/unfold, and pane focus switching.
- Triage: set a yak's state (todo / wip / blocked / done) with a single key.
- Fuzzy jump to a yak with `/` (via fzf).
- In-process markdown rendering with glamour.
```

- [ ] **Step 5: Update the README — badges, install, demo, docs link**

In `README.md`, insert badges directly under the `# yaks-tui` title:

```markdown
![CI](https://github.com/WhimsicalBees/yaks-tui/actions/workflows/ci.yml/badge.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8.svg)
```

Add the demo GIF right after the two-pane description paragraph:

```markdown
![yaks-tui demo](docs/demo.gif)
```

Replace the `## Build & run` section body with an install option first:

```markdown
## Install

```bash
go install github.com/WhimsicalBees/yaks-tui@latest
```

Or build from source:

```bash
make build
./bin/yaks-tui      # run inside a yaks repo
```
```

Add a Documentation section and a License section before `## Scope`:

```markdown
## Documentation

Full docs live in [`docs/`](docs/README.md): a getting-started tutorial,
how-to guides, a keybinding reference, and design explanations.

## License

[MIT](LICENSE) © 2026 Lena Anne Krug
```

- [ ] **Step 6: Verify the README renders sensibly**

Run: `grep -n "demo.gif\|go install\|License\|actions/workflows" README.md`
Expected: confirms the demo image, install line, license line, and CI badge are present.

- [ ] **Step 7: Commit**

```bash
git add .github/workflows/ci.yml CONTRIBUTING.md CHANGELOG.md README.md
git commit -m "docs: CI workflow, CONTRIBUTING, CHANGELOG, README badges + install"
```

---

## Task 7: Create and push the GitHub repository

> Final step. Publishing is irreversible — confirm with the author before the push.

- [ ] **Step 1: Full verification before publishing**

Run: `go build ./... && go test ./... && test -z "$(gofmt -l .)" && echo OK`
Expected: `OK`.

- [ ] **Step 2: Privacy sweep over the whole tree about to be public**

Run: `git ls-files | xargs grep -nE "lkrug|/Users/|gho_|rtk" 2>/dev/null || echo CLEAN`
Expected: `CLEAN`. (Note: `docs/superpowers/` build history is included in the sweep. If a match appears there, decide whether to scrub or gitignore before publishing.)

- [ ] **Step 3: Confirm with the author**

Stop and confirm the repo should be created public under `WhimsicalBees`. Do not proceed without an explicit yes.

- [ ] **Step 4: Create the repo and push**

Run:

```bash
gh repo create WhimsicalBees/yaks-tui --public \
  --description "Interactive, keyboard-driven terminal UI for yaks" \
  --source . --remote origin --push
```

Expected: repo created, `origin` added, `main` pushed.

- [ ] **Step 5: Verify**

Run: `gh repo view WhimsicalBees/yaks-tui --web` (opens the published repo) and confirm CI starts on the Actions tab.

---

## Notes

- **Task order matters for Task 1.** The module rename touches every `.go`
  file; doing it first keeps all later diffs consistent. Everything else is
  order-independent except Task 7, which must be last.
- **Two install paths in the README** (go install vs. make build) are
  intentional — `go install` only works once the repo is public (Task 7), but
  the README line is correct from the moment the module path changes (Task 1).
