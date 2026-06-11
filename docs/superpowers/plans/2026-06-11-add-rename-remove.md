# Add / Rename / Remove yaks — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a user create (child/root), rename, and remove yaks from inside the TUI without dropping to the `yx` CLI.

**Architecture:** Three new id-addressed methods on `internal/yaks.Client` (behind the existing `Runner`), exposed through the model's `dataSource` interface. The UI adds one single-line input mode (add/rename, reusing the search `textinput` pattern) and a remove-confirmation gate, routed in `handleKey` with the same precedence as the existing `searching`/`editing` modes. All mutations reload the tree preserving the cursor by id.

**Tech Stack:** Go, Bubble Tea (MVU), bubbles `textinput`, the `yx` CLI.

**Spec:** `docs/superpowers/specs/2026-06-11-add-rename-remove-design.md`

---

## File structure

- `internal/yaks/client.go` — add `Add`, `Rename`, `Remove`, plus an unexported `makeID` slug/suffix helper.
- `internal/yaks/ids.go` — **new**: pure id-generation helpers (`slugify`, `randomSuffix`, `uniqueID`) so they're unit-testable without a Runner. (Keeps `client.go` focused.)
- `internal/yaks/client_test.go` — argv assertions for the three methods.
- `internal/yaks/ids_test.go` — **new**: slugify + uniqueness tests.
- `internal/yaks/e2e_test.go` — add→rename→remove round-trip.
- `internal/ui/keys.go` — four new bindings (`a`, `A`, `R`, `x`) + help wiring.
- `internal/ui/model.go` — new mode state, key routing, commands, footer rendering.
- `internal/ui/model_test.go` — routing/flow tests; extend `stubClient`.
- `internal/ui/render_test.go` — footer prompt variants.

A note on the random suffix: tests must stay deterministic. `makeID`/`uniqueID` take an injectable suffix generator (a `func() string`); production passes a crypto/math-random one, tests pass a canned sequence. This is why the id logic lives in its own file with its own seam.

---

## Task 1: Id generation helpers (pure)

**Files:**
- Create: `internal/yaks/ids.go`
- Test: `internal/yaks/ids_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package yaks

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Fix the flaky test": "fix-the-flaky-test",
		"  Trim  me  ":       "trim-me",
		"Symbols!! & stuff":  "symbols-stuff",
		"Multiple---dashes":  "multiple-dashes",
		"":                   "yak",
		"!!!":                "yak",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueIDAppendsSuffix(t *testing.T) {
	seq := []string{"aaaa"}
	i := 0
	gen := func() string { s := seq[i%len(seq)]; i++; return s }
	got := uniqueID("deploy app", map[string]bool{}, gen)
	if got != "deploy-app-aaaa" {
		t.Fatalf("uniqueID = %q, want deploy-app-aaaa", got)
	}
}

func TestUniqueIDRegeneratesOnCollision(t *testing.T) {
	seq := []string{"aaaa", "bbbb", "cccc"}
	i := 0
	gen := func() string { s := seq[i]; i++; return s }
	existing := map[string]bool{"deploy-app-aaaa": true, "deploy-app-bbbb": true}
	got := uniqueID("deploy app", existing, gen)
	if got != "deploy-app-cccc" {
		t.Fatalf("uniqueID = %q, want deploy-app-cccc", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/yaks/ -run 'TestSlugify|TestUniqueID' -v`
Expected: FAIL — `undefined: slugify` / `undefined: uniqueID`.

- [ ] **Step 3: Implement `internal/yaks/ids.go`**

```go
package yaks

import (
	"crypto/rand"
	"regexp"
	"strings"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases s, turns runs of non-alphanumerics into single dashes, and
// trims leading/trailing dashes. Falls back to "yak" when nothing survives, so
// an id is always non-empty.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "yak"
	}
	return s
}

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// randomSuffix returns a 4-char base36-ish suffix, matching yx's id style.
func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	out := make([]byte, 4)
	for i, c := range b {
		out[i] = suffixAlphabet[int(c)%len(suffixAlphabet)]
	}
	return string(out)
}

// uniqueID builds "<slug>-<suffix>" and regenerates the suffix until the result
// is absent from existing. gen supplies suffixes (injectable for tests).
func uniqueID(name string, existing map[string]bool, gen func() string) string {
	base := slugify(name)
	for {
		id := base + "-" + gen()
		if !existing[id] {
			return id
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/yaks/ -run 'TestSlugify|TestUniqueID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/yaks/ids.go internal/yaks/ids_test.go
git commit -m "feat(yaks): pure id slug/uniqueness helpers"
```

---

## Task 2: `Client.Add`

**Files:**
- Modify: `internal/yaks/client.go`
- Test: `internal/yaks/client_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/yaks/client_test.go`:

```go
func TestClientAddRoot(t *testing.T) {
	fr := &fakeRunner{out: []byte("added\n")}
	c := NewClient(fr)
	gen := func() string { return "zzzz" }
	id, err := c.add(context.Background(), "", "deploy app", map[string]bool{}, gen)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id != "deploy-app-zzzz" {
		t.Fatalf("id = %q, want deploy-app-zzzz", id)
	}
	want := []string{"add", "deploy app", "--id", "deploy-app-zzzz"}
	assertArgs(t, fr.gotArgs, want)
}

func TestClientAddChild(t *testing.T) {
	fr := &fakeRunner{out: []byte("added\n")}
	c := NewClient(fr)
	gen := func() string { return "zzzz" }
	id, err := c.add(context.Background(), "deploy-app-x1y2", "write tests", map[string]bool{}, gen)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	want := []string{"add", "write tests", "--id", id, "--under", "deploy-app-x1y2"}
	assertArgs(t, fr.gotArgs, want)
}
```

Also add this shared helper to `client_test.go` (replaces the repeated arg-compare loops; the existing tests can keep their inline loops — do not refactor them in this task):

```go
func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args = %v, want %v", got, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/yaks/ -run TestClientAdd -v`
Expected: FAIL — `c.add undefined`.

- [ ] **Step 3: Implement in `internal/yaks/client.go`**

Add (place after `SetContext`):

```go
// Add creates a new yak named name and returns its id. When parentID is "" the
// yak is a root; otherwise it's nested under parentID. The id is generated to
// be unique against existing and passed via --id so the caller knows it without
// parsing yx output. existing is the set of ids already in the tree.
func (c *Client) Add(ctx context.Context, parentID, name string, existing map[string]bool) (string, error) {
	return c.add(ctx, parentID, name, existing, randomSuffix)
}

// add is Add with an injectable suffix generator for tests.
func (c *Client) add(ctx context.Context, parentID, name string, existing map[string]bool, gen func() string) (string, error) {
	id := uniqueID(name, existing, gen)
	args := []string{"add", name, "--id", id}
	if parentID != "" {
		args = append(args, "--under", parentID)
	}
	if _, err := c.r.Run(ctx, args...); err != nil {
		return "", err
	}
	return id, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/yaks/ -run TestClientAdd -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/yaks/client.go internal/yaks/client_test.go
git commit -m "feat(yaks): Client.Add with collision-proof --id"
```

---

## Task 3: `Client.Rename` and `Client.Remove`

**Files:**
- Modify: `internal/yaks/client.go`
- Test: `internal/yaks/client_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/yaks/client_test.go`:

```go
func TestClientRename(t *testing.T) {
	fr := &fakeRunner{out: []byte("renamed\n")}
	c := NewClient(fr)
	if err := c.Rename(context.Background(), "deploy-app-x1y2", "ship app"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	assertArgs(t, fr.gotArgs, []string{"rename", "deploy-app-x1y2", "ship app"})
}

func TestClientRemoveLeaf(t *testing.T) {
	fr := &fakeRunner{out: []byte("removed\n")}
	c := NewClient(fr)
	if err := c.Remove(context.Background(), "write-tests-hgny", false); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	assertArgs(t, fr.gotArgs, []string{"remove", "write-tests-hgny"})
}

func TestClientRemoveRecursive(t *testing.T) {
	fr := &fakeRunner{out: []byte("removed\n")}
	c := NewClient(fr)
	if err := c.Remove(context.Background(), "deploy-app-x1y2", true); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	assertArgs(t, fr.gotArgs, []string{"remove", "deploy-app-x1y2", "--recursive"})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/yaks/ -run 'TestClientRename|TestClientRemove' -v`
Expected: FAIL — `c.Rename undefined` / `c.Remove undefined`.

- [ ] **Step 3: Implement in `internal/yaks/client.go`**

Add after `add`:

```go
// Rename changes a yak's name in place (no move), addressed by id.
func (c *Client) Rename(ctx context.Context, id, newName string) error {
	_, err := c.r.Run(ctx, "rename", id, newName)
	return err
}

// Remove deletes a yak by id. A yak with children requires recursive=true
// (yx refuses otherwise).
func (c *Client) Remove(ctx context.Context, id string, recursive bool) error {
	args := []string{"remove", id}
	if recursive {
		args = append(args, "--recursive")
	}
	_, err := c.r.Run(ctx, args...)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/yaks/ -run 'TestClientRename|TestClientRemove' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/yaks/client.go internal/yaks/client_test.go
git commit -m "feat(yaks): Client.Rename and Client.Remove"
```

---

## Task 4: Keybindings

**Files:**
- Modify: `internal/ui/keys.go`

- [ ] **Step 1: Add bindings to the `keyMap` struct**

In `internal/ui/keys.go`, add four fields to `keyMap` (after `Edit`):

```go
	Add      key.Binding
	AddRoot  key.Binding
	Rename   key.Binding
	Remove   key.Binding
```

- [ ] **Step 2: Add the bindings in `defaultKeys()`**

After the `Edit:` line:

```go
		Add:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add child")),
		AddRoot:  key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "add root")),
		Rename:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "rename")),
		Remove:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
```

- [ ] **Step 3: Wire them into `FullHelp()`**

Replace the second group row so the new keys appear:

```go
	return [][]key.Binding{
		{k.Up, k.Down, k.Expand, k.Collapse, k.Toggle, k.Focus},
		{k.Wip, k.Blocked, k.Done, k.Todo, k.Edit},
		{k.Add, k.AddRoot, k.Rename, k.Remove},
		{k.HideDone, k.WipFocus, k.Search, k.Find, k.Reload, k.Help, k.Quit},
	}
```

- [ ] **Step 4: Verify it builds**

Run: `go build ./... && go vet ./internal/ui/`
Expected: no output (success).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/keys.go
git commit -m "feat(ui): keybindings for add/rename/remove"
```

---

## Task 5: Input-mode + confirm state and the `dataSource` interface

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

This task adds the state fields, the input widget initialization, and extends
the `dataSource` interface + the test `stubClient`. No key routing yet (that's
Task 6), so there's nothing to test behaviorally; the gate is that the package
still builds and the existing tests pass with the widened interface.

- [ ] **Step 1: Extend the `dataSource` interface**

In `internal/ui/model.go`, add three methods to the `dataSource` interface:

```go
type dataSource interface {
	List(ctx context.Context) ([]yaks.Yak, error)
	SetState(ctx context.Context, id, state string) error
	SetContext(ctx context.Context, id, content string) error
	Add(ctx context.Context, parentID, name string, existing map[string]bool) (string, error)
	Rename(ctx context.Context, id, newName string) error
	Remove(ctx context.Context, id string, recursive bool) error
}
```

- [ ] **Step 2: Add the mode type and model fields**

Add an `inputMode` type near the `focus` type:

```go
type inputMode int

const (
	inputNone inputMode = iota
	inputAddChild
	inputAddRoot
	inputRename
)
```

Add fields to the `Model` struct (after the `searching/search/query` block):

```go
	inputMode  inputMode       // which add/rename flow is open (inputNone = closed)
	inputParID string          // parent id for inputAddChild ("" = root)
	inputTgtID string          // target id for inputRename
	input      textinput.Model // one-line input for add/rename

	confirming bool   // remove confirmation prompt open
	removeID   string // captured target id
	removeName string // for the prompt text
	removeKids int    // child count → recursive flag + prompt wording
```

- [ ] **Step 3: Initialize the input widget in `New`**

In `New`, after the `ti` (search) setup and before `return Model{`:

```go
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 0
```

And add `input: in,` to the returned `Model{...}` literal.

- [ ] **Step 4: Extend the test `stubClient`**

In `internal/ui/model_test.go`, add fields and methods to `stubClient`:

```go
	addErr    error
	addID     string // id to return from Add
	addCalls  []struct{ parentID, name string }

	renameErr   error
	renameCalls []struct{ id, name string }

	removeErr   error
	removeCalls []struct {
		id        string
		recursive bool
	}
```

Methods:

```go
func (s *stubClient) Add(_ context.Context, parentID, name string, _ map[string]bool) (string, error) {
	s.addCalls = append(s.addCalls, struct{ parentID, name string }{parentID, name})
	id := s.addID
	if id == "" {
		id = "new-id"
	}
	return id, s.addErr
}
func (s *stubClient) Rename(_ context.Context, id, name string) error {
	s.renameCalls = append(s.renameCalls, struct{ id, name string }{id, name})
	return s.renameErr
}
func (s *stubClient) Remove(_ context.Context, id string, recursive bool) error {
	s.removeCalls = append(s.removeCalls, struct {
		id        string
		recursive bool
	}{id, recursive})
	return s.removeErr
}
```

- [ ] **Step 5: Verify build + existing tests pass**

Run: `go build ./... && go test ./internal/ui/`
Expected: PASS (the widened interface is satisfied by `Client` and `stubClient`; no behavior changed).

- [ ] **Step 6: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): add/rename/remove mode state + dataSource methods"
```

---

## Task 6: Key routing — open the input/confirm modes

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

This wires `a`/`A`/`R`/`x` to *open* their modes. Commit happens in Task 7.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/model_test.go`:

```go
func TestAddChildOpensInput(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := m2.(Model)
	if mm.inputMode != inputAddChild {
		t.Fatalf("inputMode = %v, want inputAddChild", mm.inputMode)
	}
	if mm.inputParID != "a" { // cursor starts on first yak, id "a"
		t.Fatalf("inputParID = %q, want a", mm.inputParID)
	}
	if mm.input.Value() != "" {
		t.Fatalf("add input should start empty, got %q", mm.input.Value())
	}
}

func TestAddRootOpensInputWithNoParent(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	mm := m2.(Model)
	if mm.inputMode != inputAddRoot {
		t.Fatalf("inputMode = %v, want inputAddRoot", mm.inputMode)
	}
	if mm.inputParID != "" {
		t.Fatalf("inputParID = %q, want empty", mm.inputParID)
	}
}

func TestRenameOpensPrefilledInput(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	mm := m2.(Model)
	if mm.inputMode != inputRename {
		t.Fatalf("inputMode = %v, want inputRename", mm.inputMode)
	}
	if mm.inputTgtID != "a" {
		t.Fatalf("inputTgtID = %q, want a", mm.inputTgtID)
	}
	if mm.input.Value() != "alpha" {
		t.Fatalf("rename input = %q, want prefilled 'alpha'", mm.input.Value())
	}
}

func TestRemoveOpensConfirm(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	mm := m2.(Model)
	if !mm.confirming {
		t.Fatal("x should open the remove confirmation")
	}
	if mm.removeID != "a" || mm.removeName != "alpha" {
		t.Fatalf("removeID/Name = %q/%q, want a/alpha", mm.removeID, mm.removeName)
	}
	if mm.removeKids != 0 {
		t.Fatalf("removeKids = %d, want 0 for a leaf", mm.removeKids)
	}
}

func TestRemoveCountsChildren(t *testing.T) {
	roots := []yaks.Yak{{
		ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{{ID: "c", Name: "child", State: "todo"}},
	}}
	m := loaded(t, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	mm := m2.(Model)
	if mm.removeKids != 1 {
		t.Fatalf("removeKids = %d, want 1", mm.removeKids)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run 'TestAdd|TestRename|TestRemove' -v`
Expected: FAIL — modes never set (keys fall through to no-op).

- [ ] **Step 3: Add an open-helper and a child-count helper**

In `internal/ui/model.go`, add near `selectedID`:

```go
// selectedYak returns a copy of the yak under the cursor and true, or false if
// there's no selection.
func (m Model) selectedYak() (yaks.Yak, bool) {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return *m.rows[m.cursor].Yak, true
	}
	return yaks.Yak{}, false
}

// openInput opens the single-line input in the given mode, seeded with value.
func (m *Model) openInput(mode inputMode, value string) {
	m.inputMode = mode
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.Focus()
}
```

- [ ] **Step 4: Route the keys in the tree-focus switch**

In `handleKey`, add these cases to the tree-focus `switch` (after the `Edit` case):

```go
	case key.Matches(msg, m.keys.Add):
		if y, ok := m.selectedYak(); ok {
			m.inputParID = y.ID
			m.openInput(inputAddChild, "")
		}
		return m, nil
	case key.Matches(msg, m.keys.AddRoot):
		m.inputParID = ""
		m.openInput(inputAddRoot, "")
		return m, nil
	case key.Matches(msg, m.keys.Rename):
		if y, ok := m.selectedYak(); ok {
			m.inputTgtID = y.ID
			m.openInput(inputRename, y.Name)
		}
		return m, nil
	case key.Matches(msg, m.keys.Remove):
		if y, ok := m.selectedYak(); ok {
			m.confirming = true
			m.removeID = y.ID
			m.removeName = y.Name
			m.removeKids = len(y.Children)
		}
		return m, nil
```

Note: `AddRoot` is reachable even with an empty tree (no selection needed), which is how the very first yak gets created from the UI.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run 'TestAdd|TestRename|TestRemove' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): open add/rename input + remove confirm on keys"
```

---

## Task 7: Input + confirm handling (commit / cancel)

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

Adds the mode-owns-keyboard blocks (like `searching`/`editing`) and the commit
commands. Place both blocks at the **top of `handleKey`**, before the
`searching` block, so they own input while open.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/model_test.go`:

```go
func typeRunes(m Model, s string) Model {
	for _, r := range s {
		mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mm.(Model)
	}
	return m
}

func TestAddChildCommitCallsClient(t *testing.T) {
	stub := &stubClient{roots: twoYaks(), addID: "gamma-zzzz"}
	m := New(stub)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: twoYaks()})
	m4, _ := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := typeRunes(m4.(Model), "gamma")
	m5, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should return a command")
	}
	cmd() // execute the add command
	if len(stub.addCalls) != 1 {
		t.Fatalf("addCalls = %d, want 1", len(stub.addCalls))
	}
	if stub.addCalls[0].parentID != "a" || stub.addCalls[0].name != "gamma" {
		t.Fatalf("add called with %+v", stub.addCalls[0])
	}
	if m5.(Model).inputMode != inputNone {
		t.Fatal("inputMode should close after commit")
	}
}

func TestAddEmptyIsNoopCancel(t *testing.T) {
	stub := &stubClient{roots: twoYaks()}
	m := loadedWith(t, stub, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m3, cmd := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty add should not call the client")
	}
	if m3.(Model).inputMode != inputNone {
		t.Fatal("empty add should close the input")
	}
	if len(stub.addCalls) != 0 {
		t.Fatalf("addCalls = %d, want 0", len(stub.addCalls))
	}
}

func TestInputEscCancels(t *testing.T) {
	stub := &stubClient{roots: twoYaks()}
	m := loadedWith(t, stub, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m3.(Model).inputMode != inputNone {
		t.Fatal("esc should close the input")
	}
	if len(stub.renameCalls) != 0 {
		t.Fatal("esc must not rename")
	}
}

func TestRenameCommitCallsClient(t *testing.T) {
	stub := &stubClient{roots: twoYaks()}
	m := loadedWith(t, stub, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	// clear prefilled "alpha", type "ship"
	mm := m2.(Model)
	mm.input.SetValue("ship")
	m3, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should return a command")
	}
	cmd()
	if len(stub.renameCalls) != 1 || stub.renameCalls[0].id != "a" || stub.renameCalls[0].name != "ship" {
		t.Fatalf("rename calls = %+v", stub.renameCalls)
	}
	if m3.(Model).inputMode != inputNone {
		t.Fatal("inputMode should close after rename")
	}
}

func TestRemoveConfirmYesCallsClient(t *testing.T) {
	stub := &stubClient{roots: twoYaks()}
	m := loadedWith(t, stub, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m3, cmd := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("y should return a remove command")
	}
	cmd()
	if len(stub.removeCalls) != 1 || stub.removeCalls[0].id != "a" || stub.removeCalls[0].recursive {
		t.Fatalf("remove calls = %+v", stub.removeCalls)
	}
	if m3.(Model).confirming {
		t.Fatal("confirm should close after y")
	}
}

func TestRemoveConfirmNoCancels(t *testing.T) {
	stub := &stubClient{roots: twoYaks()}
	m := loadedWith(t, stub, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m3.(Model).confirming {
		t.Fatal("n should cancel the confirm")
	}
	if len(stub.removeCalls) != 0 {
		t.Fatal("n must not remove")
	}
}

func TestRemoveRecursiveWhenChildren(t *testing.T) {
	roots := []yaks.Yak{{
		ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{{ID: "c", Name: "child", State: "todo"}},
	}}
	stub := &stubClient{roots: roots}
	m := loadedWith(t, stub, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	_, cmd := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	cmd()
	if len(stub.removeCalls) != 1 || !stub.removeCalls[0].recursive {
		t.Fatalf("remove calls = %+v, want recursive", stub.removeCalls)
	}
}
```

Add this helper to `model_test.go` (a `loaded` variant that takes a specific stub):

```go
func loadedWith(t *testing.T, stub *stubClient, roots []yaks.Yak) Model {
	t.Helper()
	m := New(stub)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: roots})
	return m3.(Model)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run 'TestAddChildCommit|TestAddEmpty|TestInputEsc|TestRenameCommit|TestRemoveConfirm|TestRemoveRecursive' -v`
Expected: FAIL — input keys aren't handled; commit commands don't exist.

- [ ] **Step 3: Add the confirm + input blocks at the top of `handleKey`**

Insert at the very start of `handleKey` (before the `if m.searching` block):

```go
	// Remove confirmation owns the keyboard: y confirms, anything else cancels.
	if m.confirming {
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'y' || msg.Runes[0] == 'Y') {
			cmd := m.removeCmd()
			m.confirming = false
			return m, cmd
		}
		m.confirming = false
		return m, nil
	}

	// Add/rename input owns the keyboard: enter commits (empty = cancel), esc
	// cancels, everything else is text input.
	if m.inputMode != inputNone {
		switch msg.Type {
		case tea.KeyEnter:
			name := strings.TrimSpace(m.input.Value())
			mode := m.inputMode
			m.inputMode = inputNone
			m.input.Blur()
			if name == "" {
				return m, nil // empty = no-op cancel
			}
			switch mode {
			case inputAddChild, inputAddRoot:
				return m, m.addCmd(m.inputParID, name)
			case inputRename:
				return m, m.renameCmd(m.inputTgtID, name)
			}
			return m, nil
		case tea.KeyEsc:
			m.inputMode = inputNone
			m.input.Blur()
			m.input.SetValue("")
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
```

- [ ] **Step 4: Add the commit commands**

Add near `setStateCmd` in `model.go`:

```go
// existingIDs collects every id currently in the tree (for add collision checks).
func (m Model) existingIDs() map[string]bool {
	ids := map[string]bool{}
	var walk func(ys []yaks.Yak)
	walk = func(ys []yaks.Yak) {
		for i := range ys {
			ids[ys[i].ID] = true
			walk(ys[i].Children)
		}
	}
	walk(m.roots)
	return ids
}

func (m Model) addCmd(parentID, name string) tea.Cmd {
	existing := m.existingIDs()
	return func() tea.Msg {
		id, err := m.client.Add(context.Background(), parentID, name, existing)
		if err != nil {
			return errMsg{fmt.Errorf("couldn't create yak: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsgPreserving{roots: roots, prevID: id}
	}
}

func (m Model) renameCmd(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.Rename(context.Background(), id, name); err != nil {
			return errMsg{fmt.Errorf("rename failed: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsgPreserving{roots: roots, prevID: id}
	}
}

func (m Model) removeCmd() tea.Cmd {
	id := m.removeID
	recursive := m.removeKids > 0
	return func() tea.Msg {
		if err := m.client.Remove(context.Background(), id, recursive); err != nil {
			return errMsg{fmt.Errorf("remove failed: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		// prevID is the removed yak; IndexOfID won't find it, so the cursor
		// stays clamped near where it was — the intended "fall to neighbor".
		return loadedMsgPreserving{roots: roots, prevID: id}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run 'TestAddChildCommit|TestAddEmpty|TestInputEsc|TestRenameCommit|TestRemoveConfirm|TestRemoveRecursive' -v`
Expected: PASS.

- [ ] **Step 6: Run the full ui suite (catch routing regressions)**

Run: `go test ./internal/ui/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): commit/cancel handling for add/rename/remove"
```

---

## Task 8: Footer rendering for the input + confirm prompts

**Files:**
- Modify: `internal/ui/model.go` (the `View` footer `switch`)
- Test: `internal/ui/render_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/render_test.go`:

```go
func TestFooterAddChildPrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	out := m2.(Model).View()
	if !contains(out, "add child of") || !contains(out, "alpha") {
		t.Fatalf("footer missing add-child prompt:\n%s", out)
	}
}

func TestFooterAddRootPrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	out := m2.(Model).View()
	if !contains(out, "add root") {
		t.Fatalf("footer missing add-root prompt:\n%s", out)
	}
}

func TestFooterRenamePrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	out := m2.(Model).View()
	if !contains(out, "rename") {
		t.Fatalf("footer missing rename prompt:\n%s", out)
	}
}

func TestFooterConfirmLeaf(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	out := m2.(Model).View()
	if !contains(out, `remove "alpha"?`) {
		t.Fatalf("footer missing leaf confirm:\n%s", out)
	}
}

func TestFooterConfirmSubtree(t *testing.T) {
	roots := []yaks.Yak{{
		ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{{ID: "c", Name: "child", State: "todo"}},
	}}
	m := loaded(t, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	out := m2.(Model).View()
	if !contains(out, "and its 1 children") {
		t.Fatalf("footer missing subtree confirm:\n%s", out)
	}
}
```

`render_test.go` needs the bubbletea import; add `tea "github.com/charmbracelet/bubbletea"` to its import block if not present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestFooter -v`
Expected: FAIL — footer doesn't render these prompts yet.

- [ ] **Step 3: Add the footer cases in `View`**

In `View`, extend the `var bar string` `switch` — add these cases **before** `case m.searching:` (confirm and input take precedence over the search/status/help bar, matching the routing order). `parentName` is added in Step 4:

```go
	case m.confirming:
		prompt := fmt.Sprintf("remove %q? (y/n)", m.removeName)
		if m.removeKids > 0 {
			prompt = fmt.Sprintf("remove %q and its %d children? (y/n)", m.removeName, m.removeKids)
		}
		bar = statusErr.Render(prompt)
	case m.inputMode == inputAddChild:
		bar = subtle.Render(fmt.Sprintf("add child of %q: ", m.parentName()) + m.input.View())
	case m.inputMode == inputAddRoot:
		bar = subtle.Render("add root: " + m.input.View())
	case m.inputMode == inputRename:
		bar = subtle.Render("rename: " + m.input.View())
```

- [ ] **Step 4: Add the `parentName` helper**

Add to `model.go`:

```go
// parentName returns the display name of inputParID, or "" if not found. Used
// only to label the add-child prompt.
func (m Model) parentName() string {
	if m.inputParID == "" {
		return ""
	}
	var found string
	var walk func(ys []yaks.Yak)
	walk = func(ys []yaks.Yak) {
		for i := range ys {
			if ys[i].ID == m.inputParID {
				found = ys[i].Name
				return
			}
			walk(ys[i].Children)
		}
	}
	walk(m.roots)
	return found
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/ -run TestFooter -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/model.go internal/ui/render_test.go
git commit -m "feat(ui): footer prompts for add/rename/remove"
```

---

## Task 9: e2e round-trip against real `yx`

**Files:**
- Modify: `internal/yaks/e2e_test.go`

- [ ] **Step 1: Write the test**

Append to `internal/yaks/e2e_test.go`:

```go
func TestE2E_AddRenameRemove(t *testing.T) {
	if _, err := exec.LookPath("yx"); err != nil {
		t.Skip("yx not installed; skipping e2e")
	}
	dir, err := os.MkdirTemp("", "yaks-e2e-crud")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				os.Chmod(path, 0o700)
			}
			return nil
		})
		os.RemoveAll(dir)
	})
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".yaks\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewClient(dirRunner{dir})
	ctx := context.Background()

	// Add a root, then a child under it.
	rootID, err := c.Add(ctx, "", "deploy app", map[string]bool{})
	if err != nil {
		t.Fatalf("Add root: %v", err)
	}
	childID, err := c.Add(ctx, rootID, "write tests", map[string]bool{rootID: true})
	if err != nil {
		t.Fatalf("Add child: %v", err)
	}

	// Rename the child.
	if err := c.Rename(ctx, childID, "write more tests"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	roots, err := c.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(roots) != 1 || len(roots[0].Children) != 1 || roots[0].Children[0].Name != "write more tests" {
		t.Fatalf("unexpected tree after rename: %+v", roots)
	}

	// Removing the parent without recursive must fail (it has a child).
	if err := c.Remove(ctx, rootID, false); err == nil {
		t.Fatal("expected non-recursive remove of a parent to fail")
	}
	// Recursive remove succeeds and empties the tree.
	if err := c.Remove(ctx, rootID, true); err != nil {
		t.Fatalf("recursive Remove: %v", err)
	}
	roots, err = c.List(ctx)
	if err != nil {
		t.Fatalf("List after remove: %v", err)
	}
	if len(roots) != 0 {
		t.Fatalf("tree not empty after recursive remove: %+v", roots)
	}
}
```

- [ ] **Step 2: Run the test (or confirm skip)**

Run: `go test ./internal/yaks/ -run TestE2E_AddRenameRemove -v`
Expected: PASS if `yx` is installed; SKIP otherwise. If it fails on the
non-recursive-remove assertion, check whether your `yx` version actually
refuses — adjust the assertion to match real behavior and note it.

- [ ] **Step 3: Commit**

```bash
git add internal/yaks/e2e_test.go
git commit -m "test(yaks): e2e add/rename/remove round-trip"
```

---

## Task 10: Docs + full verification

**Files:**
- Modify: `README.md` (key reference)
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Update the README key list**

Find the keybinding section in `README.md` and add rows for the new keys:

```
| `a` | add a child yak under the cursor |
| `A` | add a root yak |
| `R` | rename the selected yak |
| `x` | remove the selected yak (confirms; recursive if it has children) |
```

(Match the existing table's exact column format — open `README.md` and follow
the surrounding rows.)

- [ ] **Step 2: Update CHANGELOG**

Add under an Unreleased/next section in `CHANGELOG.md`:

```
- Add, rename, and remove yaks from inside the TUI (`a`/`A`/`R`/`x`).
```

- [ ] **Step 3: Full verification suite**

Run each; all must pass clean:

```bash
gofmt -l .          # must print nothing
go vet ./...        # must be silent
go test ./...       # full suite green (PTY/e2e skip without yx/fzf)
go build -o bin/yaks-tui .
```

Expected: `gofmt -l .` prints nothing; `go vet` silent; tests PASS; build succeeds.

- [ ] **Step 4: Commit**

```bash
git add README.md CHANGELOG.md
git commit -m "docs: document add/rename/remove keys"
```

---

## Self-review notes

- **Spec coverage:** Client `Add`/`Rename`/`Remove` (Tasks 2–3), collision-proof
  `--id` (Task 1–2), `a/A/R/x` keys (Task 4), input+confirm state (Task 5),
  open routing (Task 6), commit/cancel + recursive-from-child-count (Task 7),
  footer prompt variants incl. silent success via existing `status` (Task 8),
  e2e (Task 9), docs (Task 10). All spec sections map to a task.
- **Cursor-falls-to-neighbor on remove:** achieved for free — `removeCmd`
  passes the removed id as `prevID`; `IndexOfID` misses, the cursor stays
  clamped at its prior index (the next sibling shifts into that slot).
- **Type consistency:** `inputMode`/`inputNone/inputAddChild/inputAddRoot/inputRename`,
  `addCmd/renameCmd/removeCmd`, `selectedYak`, `openInput`, `existingIDs`,
  `parentName` are defined once and referenced consistently.
- **No placeholders:** every code step shows complete code; the one transient
  placeholder in Task 8 Step 3 is explicitly called out and replaced in the
  same step.
