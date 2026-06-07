# CI Action Bump + Yak Content Backfill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bump the CI actions off deprecated Node 20, and backfill historical context onto the completed v1.1-editor and OSS-readiness yaks.

**Architecture:** Two independent tasks. Task 1 edits one workflow file and pushes to the public repo. Task 2 writes context bodies to 14 yaks via `yx` (no repo commit — `.yaks/` is gitignored).

**Tech Stack:** GitHub Actions, the `yx` CLI.

---

## File Structure

- `.github/workflows/ci.yml` — the only file modified (Task 1).
- `.yaks/` — written via `yx context` only, never by hand (Task 2). Gitignored, so no repo commit.

---

## Task 1: Bump CI actions to v6

**Files:**
- Modify: `.github/workflows/ci.yml:12-13`

- [ ] **Step 1: Confirm current pinned versions**

Run: `grep -nE "uses:" .github/workflows/ci.yml`
Expected: shows `actions/checkout@v4` (line 12) and `actions/setup-go@v5` (line 13).

- [ ] **Step 2: Bump checkout to v6**

Change the line `      - uses: actions/checkout@v4` to:

```yaml
      - uses: actions/checkout@v6
```

- [ ] **Step 3: Bump setup-go to v6**

Change the line `      - uses: actions/setup-go@v5` to:

```yaml
      - uses: actions/setup-go@v6
```

- [ ] **Step 4: Verify the edit**

Run: `grep -nE "uses:|go-version:" .github/workflows/ci.yml`
Expected: `actions/checkout@v6`, `actions/setup-go@v6`, and `go-version: '1.26'` unchanged.

- [ ] **Step 5: Commit and push**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: bump checkout and setup-go to v6 (Node 24)"
git push origin main
```

- [ ] **Step 6: Watch CI and confirm green with no deprecation warning**

```bash
gh run watch "$(gh run list --repo WhimsicalBees/yaks-tui --limit 1 --json databaseId --jq '.[0].databaseId')" --repo WhimsicalBees/yaks-tui --exit-status
```

Expected: the run completes successfully. Then confirm the deprecation annotation is gone:

```bash
gh run view "$(gh run list --repo WhimsicalBees/yaks-tui --limit 1 --json databaseId --jq '.[0].databaseId')" --repo WhimsicalBees/yaks-tui --log 2>/dev/null | grep -i "Node.js 20 actions are deprecated" || echo "NO DEPRECATION WARNING"
```

Expected: `NO DEPRECATION WARNING`.

---

## Task 2: Backfill yak context

> Each step pipes a body to `yx context <id>` on stdin. Empty stdin is treated as
> "show" (does not write), so every body below is non-empty. These yaks are
> `done`; setting context on a done yak works (verified). `.yaks/` is gitignored
> — there is no git commit for this task.

- [ ] **Step 1: Backfill the v1.1 editor parent**

```bash
printf 'Inline context editor for yaks-tui: press e to edit a yak'\''s markdown body in the TUI, ctrl+s to save, esc to cancel.\n\nSpec: docs/superpowers/specs/2026-06-06-inline-context-editor-design.md\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md\n' | yx context build-inline-context-editor-v11-y9ms
```

- [ ] **Step 2: Backfill the v1.1 editor children**

```bash
printf 'Added the stdin-aware Runner.RunWithInput plus Client.SetContext, piping a new body to `yx context <id>`.\n\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Task 1)\n' | yx context client-stdin-aware-runner-setcontext-rynw

printf 'Added edit-mode state (editing flag, textarea, editID) and key routing: ctrl+s saves, esc cancels, all else goes to the textarea.\n\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Task 2)\n' | yx context ui-edit-mode-model-state-key-routing-9vaa

printf 'Added saveContextCmd, the contextSavedMsg flow, and SetContext on the dataSource interface. Committed with the UI state task.\n\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Task 2)\n' | yx context ui-save-command-messages-datasourcesetcontext-51ov

printf 'UI routing table tests plus an e2e round-trip of SetContext against real yx.\n\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Tasks 2-3)\n' | yx context tests-ui-routing-e2e-round-trip-4aae

printf 'Documented the e edit key and the ctrl+s / esc edit-mode keys in the README.\n\nPlan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Task 4)\n' | yx context docs-update-readme-keys-eju8
```

- [ ] **Step 3: Backfill the OSS-readiness parent**

```bash
printf 'Published yaks-tui as open source: module path, MIT license, Diataxis docs, demo GIF, AI config, CI/CONTRIBUTING/CHANGELOG, and the public GitHub repo at WhimsicalBees/yaks-tui.\n\nSpec: docs/superpowers/specs/2026-06-06-oss-readiness-design.md\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md\n' | yx context ship-yaks-tui-as-oss-4e0p
```

- [ ] **Step 4: Backfill the OSS-readiness children**

```bash
printf 'Renamed the Go module to github.com/WhimsicalBees/yaks-tui (case-sensitive) and updated all internal imports, so go install works.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 1)\n' | yx context module-path-rename-to-githubcomwhimsicalbeesyaks-tui-dq6x

printf 'Added the MIT LICENSE file, (c) 2026 Lena Anne Krug.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 2)\n' | yx context mit-license-8jn2

printf 'Added committed AI-agent config: AGENTS.md (single source of truth), CLAUDE.md symlink, sanitized .claude/settings.json, with a privacy gate. OpenCode reads AGENTS.md natively.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 3)\n' | yx context ai-tooling-config-agentsmd-claudemd-settings-0si6

printf 'Wrote the four-quadrant Diataxis docs (tutorial, how-tos, reference, explanation), with reference pages verified against keys.go and main.go.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 4)\n' | yx context diataxis-documentation-8pl1

printf 'Wrote a self-contained VHS tape and generated docs/demo.gif, plus a make demo target (points rod at system Chrome on macOS).\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 5)\n' | yx context vhs-demo-gif-aiga

printf 'Added CI (build/vet/gofmt/test), CONTRIBUTING.md, CHANGELOG.md, and README badges/install/demo.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 6)\n' | yx context oss-scaffolding-ci-contributing-changelog-readme-a7d0

printf 'Merged to main, ran the privacy sweep (scrubbed one absolute path), and created + pushed the public repo via gh. CI green on first run.\n\nPlan: docs/superpowers/plans/2026-06-06-oss-readiness.md (Task 7)\n' | yx context create-and-push-github-repo-ghjp
```

- [ ] **Step 5: Spot-check the backfill**

```bash
for id in build-inline-context-editor-v11-y9ms ship-yaks-tui-as-oss-4e0p mit-license-8jn2 vhs-demo-gif-aiga create-and-push-github-repo-ghjp; do
  echo "=== $id ==="
  yx context "$id" --show
done
```

Expected: each prints its summary plus the spec/plan link(s). No empties.

- [ ] **Step 6: Sync the yaks**

```bash
yx sync
```

Expected: completes without conflict, so the backfilled context propagates to the shared event store.

---

## Self-review notes

- **No git commit for Task 2.** `.yaks/` is gitignored; the record lives in the
  yaks event store. This is expected, not an omission.
- **IDs are exact**, taken from `yx list --format json` at planning time. If any
  `yx context <id>` reports "not found," re-list to get the current id rather
  than guessing.
- **Apostrophe escaping:** Step 1 of Task 2 uses `'\''` inside the single-quoted
  printf to render "yak's". Keep it as written.
