# Inline Context Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Status:** Implemented on 2026-06-06 (commits `3c114c9`, `722d9b8`, `a874c93`, `7bcf28b`). This plan is retained as documentation of the build. Checkboxes are marked complete to reflect the shipped state.

**Goal:** Let the user edit a yak's context body inline in the TUI — press `e`, edit in a textarea, `ctrl+s` to save, `esc` to cancel.

**Architecture:** An `editing` flag plus a `bubbles/textarea` live on the existing `Model`. While editing, the detail pane renders the textarea and `handleKey` routes all keystrokes to it (except `ctrl+s`/`esc`). Saving pipes the body to `yx context <id>` on stdin via a new stdin-aware `Runner` method. No new screens, no `$EDITOR` subprocess.

**Tech Stack:** Go, Bubble Tea v1, `bubbles/textarea`, the `yx` CLI.

---

## File Structure

- `internal/yaks/client.go` — add `Runner.RunWithInput` (stdin pipe) and `Client.SetContext`. The `Runner` interface is the single seam between the app and the `yx` binary; stdin support belongs here.
- `internal/yaks/client_test.go` — unit test `SetContext` with a fake runner asserting piped stdin + args.
- `internal/yaks/e2e_test.go` — extend the real-`yx` round-trip; `dirRunner` gains `RunWithInput` to satisfy the interface.
- `internal/ui/model.go` — edit-mode state (`editing`, `editID`, `ta`), key routing, `enterEdit`, `saveContextCmd`, `contextSavedMsg` handling, layout sizing, View rendering. `dataSource` gains `SetContext`.
- `internal/ui/keys.go` — add the `Edit` binding (`e`) and surface it in help.
- `internal/ui/model_test.go` — stub gains `SetContext`; edit-mode behavior tests.
- `README.md` — document `e` and the edit-mode keys.

---

## Task 1: Stdin-aware Runner + SetContext

**Files:**
- Modify: `internal/yaks/client.go`
- Test: `internal/yaks/client_test.go`
- Modify: `internal/yaks/e2e_test.go` (interface satisfaction)

- [x] **Step 1: Write the failing test**

Add to `internal/yaks/client_test.go`. First extend `fakeRunner` to record stdin and satisfy the new interface method:

```go
// fakeRunner returns canned output/err and records the args it was called with.
type fakeRunner struct {
	out      []byte
	err      error
	gotArgs  []string
	gotStdin string
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.gotArgs = args
	return f.out, f.err
}

func (f *fakeRunner) RunWithInput(_ context.Context, stdin string, args ...string) ([]byte, error) {
	f.gotStdin = stdin
	f.gotArgs = args
	return f.out, f.err
}
```

Then the test:

```go
func TestClientSetContext(t *testing.T) {
	fr := &fakeRunner{out: []byte("done\n")}
	c := NewClient(fr)
	body := "# Title\n\nsome body\n"
	if err := c.SetContext(context.Background(), "deploy-app-x1y2", body); err != nil {
		t.Fatalf("SetContext: %v", err)
	}
	if fr.gotStdin != body {
		t.Fatalf("stdin = %q, want %q", fr.gotStdin, body)
	}
	want := []string{"context", "deploy-app-x1y2"}
	if len(fr.gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", fr.gotArgs, want)
	}
	for i := range want {
		if fr.gotArgs[i] != want[i] {
			t.Fatalf("args = %v, want %v", fr.gotArgs, want)
		}
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./internal/yaks/ -run TestClientSetContext`
Expected: FAIL — `c.SetContext undefined` and `RunWithInput` not in `Runner`.

- [x] **Step 3: Write minimal implementation**

In `internal/yaks/client.go`, extend the interface and refactor `ExecRunner` so both entry points share error handling:

```go
// Runner executes a `yx` invocation and returns its stdout. Behind an interface
// so tests can inject a fake instead of running the real binary.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
	// RunWithInput is like Run but pipes stdin to the command — used for
	// `yx context <id>`, which reads the new body from stdin.
	RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error)
}

// ExecRunner runs the real `yx` binary in the current working directory.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return runYx(exec.CommandContext(ctx, "yx", args...), args)
}

func (ExecRunner) RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	cmd.Stdin = strings.NewReader(stdin)
	return runYx(cmd, args)
}

// runYx executes cmd and maps failures to a friendly error. It surfaces stderr
// from yx so callers can show a useful message, falling back to wrapping the
// error itself when stderr is empty so the exit status isn't lost and the
// message is never blank. The args are joined into a readable command (e.g.
// "yx list --format json") rather than a Go slice.
func runYx(cmd *exec.Cmd, args []string) ([]byte, error) {
	out, err := cmd.Output()
	if err != nil {
		cmdline := "yx " + strings.Join(args, " ")
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s: %s", cmdline, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("%s: %w", cmdline, err)
	}
	return out, nil
}
```

Add the client method after `SetState`:

```go
// SetContext replaces a yak's context body by id. The content is piped to
// `yx context <id>` on stdin, which yx reads when stdin is present. An empty
// content string is valid and clears the body.
func (c *Client) SetContext(ctx context.Context, id, content string) error {
	_, err := c.r.RunWithInput(ctx, content, "context", id)
	return err
}
```

Add `RunWithInput` to `dirRunner` in `internal/yaks/e2e_test.go` so the package compiles:

```go
func (d dirRunner) RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	cmd.Dir = d.dir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yx %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}
```

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./internal/yaks/`
Expected: PASS (9 tests).

- [x] **Step 5: Commit**

```bash
git add internal/yaks/
git commit -m "feat(yaks): SetContext + stdin-aware Runner.RunWithInput"
```

---

## Task 2: Edit-mode model state, key routing, and save command

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/keys.go`
- Test: `internal/ui/model_test.go`

- [x] **Step 1: Write the failing tests**

In `internal/ui/model_test.go`, add `SetContext` to the stub:

```go
type stubClient struct {
	roots    []yaks.Yak
	listErr  error
	setErr   error
	setCalls []struct{ id, state string }

	ctxErr   error
	ctxCalls []struct{ id, content string }
}

func (s *stubClient) SetContext(_ context.Context, id, content string) error {
	s.ctxCalls = append(s.ctxCalls, struct{ id, content string }{id, content})
	return s.ctxErr
}
```

Add a fixture and the behavior tests:

```go
func yaksWithContext() []yaks.Yak {
	body := "existing body"
	return []yaks.Yak{
		{ID: "a", Name: "alpha", State: "todo", Context: &body},
		{ID: "b", Name: "beta", State: "wip"},
	}
}

func TestEditEntersModeAndLoadsContext(t *testing.T) {
	m := loaded(t, yaksWithContext())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	mm := m2.(Model)
	if !mm.editing {
		t.Fatal("e should enter edit mode")
	}
	if got := mm.ta.Value(); got != "existing body" {
		t.Fatalf("textarea value = %q, want existing body", got)
	}
}

func TestEditEmptyContextStartsBlank(t *testing.T) {
	m := loaded(t, yaksWithContext())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	mm := m3.(Model)
	if !mm.editing {
		t.Fatal("e should enter edit mode")
	}
	if got := mm.ta.Value(); got != "" {
		t.Fatalf("textarea value = %q, want empty", got)
	}
}

func TestEditNoSelectionIsNoOp(t *testing.T) {
	sc := &stubClient{roots: twoYaks()}
	m := New(sc)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m2.(Model).editing {
		t.Fatal("e with no selection should not enter edit mode")
	}
}

func TestEditEscCancelsWithoutSaving(t *testing.T) {
	sc := &stubClient{roots: yaksWithContext()}
	m := loaded(t, yaksWithContext())
	m.client = sc
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m3.(Model)
	if mm.editing {
		t.Fatal("esc should exit edit mode")
	}
	if len(sc.ctxCalls) != 0 {
		t.Fatalf("esc must not save, got %+v", sc.ctxCalls)
	}
}

func TestEditCtrlSSaves(t *testing.T) {
	sc := &stubClient{roots: yaksWithContext()}
	m := loaded(t, yaksWithContext())
	m.client = sc
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" more")})
	m4, cmd := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s should produce a save command")
	}
	msg := cmd()
	if _, ok := msg.(contextSavedMsg); !ok {
		t.Fatalf("expected contextSavedMsg, got %T", msg)
	}
	if len(sc.ctxCalls) != 1 || sc.ctxCalls[0].id != "a" {
		t.Fatalf("SetContext calls = %+v", sc.ctxCalls)
	}
	if sc.ctxCalls[0].content != "existing body more" {
		t.Fatalf("saved content = %q", sc.ctxCalls[0].content)
	}
	m5, _ := m4.(Model).Update(contextSavedMsg{})
	if m5.(Model).editing {
		t.Fatal("contextSavedMsg should exit edit mode")
	}
}

func TestEditSaveErrorStaysInEditMode(t *testing.T) {
	sc := &stubClient{roots: yaksWithContext(), ctxErr: errStub("save boom")}
	m := loaded(t, yaksWithContext())
	m.client = sc
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m3, cmd := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s should produce a command")
	}
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg on save failure, got %T", msg)
	}
	m4, _ := m3.(Model).Update(msg)
	if !m4.(Model).editing {
		t.Fatal("save failure must keep edit mode active")
	}
}

func TestEditKeysReachTextareaNotTriage(t *testing.T) {
	sc := &stubClient{roots: yaksWithContext()}
	m := loaded(t, yaksWithContext())
	m.client = sc
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := m3.(Model)
	if len(sc.setCalls) != 0 {
		t.Fatalf("d while editing must not trigger triage, got %+v", sc.setCalls)
	}
	if got := mm.ta.Value(); got != "existing bodyd" {
		t.Fatalf("textarea value = %q, want existing bodyd", got)
	}
}
```

- [x] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestEdit`
Expected: FAIL — `editing`, `ta`, `contextSavedMsg` undefined.

- [x] **Step 3: Write minimal implementation**

In `internal/ui/model.go`:

Add the import `"github.com/charmbracelet/bubbles/textarea"`.

Add `SetContext` to the `dataSource` interface:

```go
type dataSource interface {
	List(ctx context.Context) ([]yaks.Yak, error)
	SetState(ctx context.Context, id, state string) error
	SetContext(ctx context.Context, id, content string) error
}
```

Add the message type alongside the others:

```go
type contextSavedMsg struct{}
```

Add the model fields:

```go
	editing bool           // true while the inline context editor is open
	editID  string         // id of the yak being edited (captured on entry)
	ta      textarea.Model // inline editor for the context body
```

Initialize the textarea in `New()` (before the `return`), and add `ta: ta` to the returned struct:

```go
	ta := textarea.New()
	ta.Prompt = ""   // no per-line prompt gutter; the body is plain markdown
	ta.CharLimit = 0 // no limit
	ta.ShowLineNumbers = false
```

Handle the saved message in `Update`, after the `stateChangedMsg` case:

```go
	case contextSavedMsg:
		// Saved successfully: leave edit mode and reload so the detail pane
		// reflects the new body (cursor preserved by id).
		m.editing = false
		m.status = ""
		return m, m.reloadPreservingCmd()
```

Route keys at the top of `handleKey`, before any global binding:

```go
	// Edit mode owns the keyboard: ctrl+s saves, esc cancels, everything else
	// (including ctrl+c) is text input for the textarea. This must come before
	// any global binding so the editor isn't interrupted by triage/quit keys.
	if m.editing {
		switch msg.Type {
		case tea.KeyCtrlS:
			return m, m.saveContextCmd()
		case tea.KeyEsc:
			m.editing = false
			return m, nil
		}
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(msg)
		return m, cmd
	}
```

Add the `Edit` case to the tree-focus switch:

```go
	case key.Matches(msg, m.keys.Edit):
		return m.enterEdit()
```

Add the two helper methods:

```go
// enterEdit opens the inline editor for the selected yak, loading its current
// context into the textarea. No selection → no-op.
func (m Model) enterEdit() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	y := *m.rows[m.cursor].Yak
	body := ""
	if y.Context != nil {
		body = *y.Context
	}
	m.editing = true
	m.editID = y.ID
	m.ta.SetValue(body)
	m.ta.CursorEnd()
	m.ta.Focus()
	return m, nil
}

// saveContextCmd writes the textarea body to the yak captured when edit mode
// was entered. On success it yields contextSavedMsg (which exits edit mode and
// reloads); on failure it yields errMsg and edit mode is left untouched, so the
// user's edits aren't lost.
func (m Model) saveContextCmd() tea.Cmd {
	id := m.editID
	content := m.ta.Value()
	return func() tea.Msg {
		if err := m.client.SetContext(context.Background(), id, content); err != nil {
			return errMsg{err}
		}
		return contextSavedMsg{}
	}
}
```

Size the textarea in `layout()`, after the viewport sizing:

```go
	m.ta.SetWidth(detailWidth)
	m.ta.SetHeight(bodyHeight)
```

Render the textarea while editing, in `View()` — replace the right-pane/bar block:

```go
	// While editing, the right pane shows the textarea and takes focus styling
	// regardless of the underlying focus field.
	rightContent := m.detail.View()
	if m.editing {
		rightContent = m.ta.View()
		detailBorder = focusedBorder
		treeBorder = blurredBorder
	}

	left := treeBorder.Width(treeWidth).Height(bodyHeight).Render(m.renderTree(treeWidth, bodyHeight))
	right := detailBorder.Width(detailWidth).Height(bodyHeight).Render(rightContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var bar string
	switch {
	case m.editing:
		bar = subtle.Render("editing — ctrl+s save · esc cancel")
	case m.status != "":
		bar = statusErr.Render(m.status)
	default:
		bar = m.help.View(m.keys)
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, bar)
```

In `internal/ui/keys.go`, add the `Edit` field to `keyMap`, bind it, and surface it in help:

```go
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
```

Add `k.Edit` to `ShortHelp` (after `k.Done`) and to the second row of `FullHelp`.

- [x] **Step 4: Settle deps, then run tests**

`textarea` pulls transitive deps (`atotto/clipboard`, `MakeNowJust/heredoc`) not yet in `go.sum`:

Run: `go mod tidy`
Run: `go test ./internal/ui/`
Expected: PASS (49 tests).

- [x] **Step 5: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat(ui): inline context editor (e to edit, ctrl+s save, esc cancel)"
```

---

## Task 3: End-to-end round-trip against real yx

**Files:**
- Modify: `internal/yaks/e2e_test.go`

- [x] **Step 1: Extend the e2e test**

In `TestE2E_ListAndSetState` (which already adds a yak and flips its state), append a context round-trip after the state assertion:

```go
	// Round-trip the context body through SetContext → List.
	body := "# Notes\n\nremember the milk\n"
	if err := c.SetContext(context.Background(), id, body); err != nil {
		t.Fatalf("SetContext: %v", err)
	}
	roots, err = c.List(context.Background())
	if err != nil {
		t.Fatalf("List after SetContext: %v", err)
	}
	if roots[0].Context == nil {
		t.Fatal("context is nil after SetContext")
	}
	if got := strings.TrimSpace(*roots[0].Context); got != strings.TrimSpace(body) {
		t.Fatalf("context = %q, want %q", got, body)
	}
```

(`strings` is already imported by this file.)

- [x] **Step 2: Run the e2e test**

Run: `go test ./internal/yaks/ -run TestE2E -v`
Expected: PASS (skips with "yx not installed" if the binary is absent).

- [x] **Step 3: Run the full suite**

Run: `go test ./...`
Expected: PASS (66 tests across 5 packages).

- [x] **Step 4: Commit**

```bash
git add internal/yaks/e2e_test.go
git commit -m "test(yaks): e2e round-trip for SetContext against real yx"
```

---

## Task 4: Documentation

**Files:**
- Modify: `README.md`

- [x] **Step 1: Update the keys table and scope note**

Add to the keys table after the triage row:

```markdown
| `e` | edit context body inline |
```

Add below the table:

```markdown
While editing: `ctrl+s` saves, `esc` cancels.
```

Update the Scope section to note v1.1 adds inline editing of the context body, with other fields/add/move/sync still planned.

- [x] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: README — inline context editor key (e) and v1.1 scope"
```

---

## Manual verification (gate before calling v1.1 done)

Model logic is unit-tested, but textarea rendering, sizing inside the bordered
pane, and the save/cancel feel only show up in a real terminal. Run:

```bash
go run . .
```

Then: press `e` on a yak, edit the body, `ctrl+s` to save (detail pane should
refresh with the new body), re-enter and `esc` to confirm cancel discards.

**Known behavior:** while editing, `ctrl+c` is text input (swallowed by the
textarea), not quit — `esc` first, then quit. This follows the spec's
"everything else passes through to the textarea" rule.

---

## Notes / deviations from the spec

- **Two UI tasks merged.** The spec split "model state + key routing" and "save
  command + messages" into separate yaks. They share the same compile unit and
  are mutually dependent (routing intercepts `ctrl+s`, which needs the save
  command to exist), so they were implemented and committed together. Both yaks
  were closed at that commit.
- **`runYx` extraction.** Adding `RunWithInput` would have duplicated
  `ExecRunner`'s error-mapping. The shared logic was pulled into a `runYx`
  helper rather than copy-pasted — DRY, and a behavior-preserving refactor.
- **`editID` field.** The plan captures the yak id when edit mode is entered
  (not at save time) so a reload or external change between entering and saving
  can't redirect the write to a different yak.
