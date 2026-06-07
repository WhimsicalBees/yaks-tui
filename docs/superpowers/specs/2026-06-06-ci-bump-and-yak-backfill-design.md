# CI Action Bump + Yak Content Backfill — Design

**Date:** 2026-06-06
**Status:** Approved, ready for implementation plan

## Goal

Two small, independent pieces of cleanup:

1. **CI action bump** — upgrade the GitHub Actions in `.github/workflows/ci.yml`
   off the deprecated Node-20 versions, killing the deprecation warning on every
   run.
2. **Yak content backfill** — populate the (currently empty) context bodies of
   the completed yaks from the v1.1-editor and OSS-readiness trees, for
   historical reference.

These are unrelated; each is its own commit.

---

## 1. CI action bump

GitHub is deprecating Node 20 for Actions (forced to Node 24 starting
2026-06-16). Our workflow pins `actions/checkout@v4` and `actions/setup-go@v5`,
both of which still run on Node 20 and emit a deprecation warning.

**Change:** bump both to `@v6`, which run on Node 24 (verified against the
[checkout](https://github.com/actions/checkout/releases) and
[setup-go](https://github.com/actions/setup-go/releases) release pages — latest
majors are v6, Node-24 based).

```yaml
- uses: actions/checkout@v4   →   - uses: actions/checkout@v6
- uses: actions/setup-go@v5   →   - uses: actions/setup-go@v6
```

Nothing else in the workflow changes (`go-version: '1.26'`, the build/vet/gofmt/
test steps stay as-is). This is a public-repo change, so it is pushed to
`origin/main` and CI is watched to confirm it stays green with no deprecation
warning.

**Verification:** after pushing, the CI run on `main` passes and the
"Node.js 20 actions are deprecated" annotation no longer appears.

---

## 2. Yak content backfill

The completed yaks have empty context bodies. Backfill them so `yx show` (and
the TUI detail pane) carries a short historical record of what each yak
accomplished and where to read more.

**Scope — two trees, 14 yaks:**

*Build inline context editor v1.1* (`build-inline-context-editor-v11-y9ms`)
- `client-stdin-aware-runner-setcontext-rynw`
- `ui-edit-mode-model-state-key-routing-9vaa`
- `ui-save-command-messages-datasourcesetcontext-51ov`
- `tests-ui-routing-e2e-round-trip-4aae`
- `docs-update-readme-keys-eju8`

*Ship yaks-tui as OSS* (`ship-yaks-tui-as-oss-4e0p`)
- `module-path-rename-to-githubcomwhimsicalbeesyaks-tui-dq6x`
- `mit-license-8jn2`
- `ai-tooling-config-agentsmd-claudemd-settings-0si6`
- `diataxis-documentation-8pl1`
- `vhs-demo-gif-aiga`
- `oss-scaffolding-ci-contributing-changelog-readme-a7d0`
- `create-and-push-github-repo-ghjp`

The original v1 build tree is **out of scope** — its detail is already captured
in its own spec/plan, and it's older.

**Content depth — short summary + link.** Each yak's context is 1-3 lines: what
it accomplished, plus a pointer to the relevant spec/plan under
`docs/superpowers/`. The two parent yaks link to both their spec and plan; each
child links to the same plan (and names its task) so a reader can jump to the
detailed steps. Example shape:

```
Built the stdin-aware Runner.RunWithInput + Client.SetContext that pipes a new
body to `yx context <id>`.

Plan: docs/superpowers/plans/2026-06-06-inline-context-editor.md (Task 1)
Spec: docs/superpowers/specs/2026-06-06-inline-context-editor-design.md
```

**Mechanism.** Set context by id with stdin:
`printf '<body>' | yx context <id>`. Verified this works on *done* yaks. Note:
empty stdin is treated as "show" (it does not clear), so content must be
non-empty — not a problem here since every body has text.

**Not a code change.** This writes to `.yaks/` via `yx` only. Per the workspace
rule, `.yaks/` is never edited by hand. `.yaks/` is gitignored, so this produces
no repo commit; the record lives in the yaks event store and syncs via
`yx sync`.

**Verification:** after backfill, `yx context <id> --show` returns the expected
body for each of the 14 yaks (spot-check several, including the two parents).

---

## Out of scope

- The original v1 build tree's yaks (already documented in their spec/plan).
- Any change to application code or behavior.
- `make demo` portability (the macOS Chrome-path note is documented in the
  Makefile target; no further work planned).

## Notes

- The CI bump and the backfill are independent and can be done in either order.
  The plan does CI first (it's the public-facing fix) then the backfill.
- The backfill touches `.yaks/` only via `yx`; there is no git commit for it in
  this repo.
