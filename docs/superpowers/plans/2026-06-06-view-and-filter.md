# View & Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add view-only tree filtering to yaks-tui: `H` hides done, `W` focuses wip/blocked, `f` does incremental name search — all ancestor-preserving and composable.

**Architecture:** A new `tree.FlattenFiltered(roots, expanded, pred)` adds predicate-based visibility with ancestor preservation to the pure tree layer. The UI model gains `hideDone`/`wipFocus`/`searching`/`search`/`query` state, composes active filters into one predicate in `rebuildRows`, and routes keys in `handleKey` (search mode mirrors the existing edit-mode routing).

**Tech Stack:** Go, Bubble Tea v1, `bubbles/textinput` (already in module cache).

---

## File Structure

- `internal/tree/tree.go` — add `Predicate` type + `FlattenFiltered`; `Flatten` becomes a nil-predicate wrapper. Pure logic, no UI.
- `internal/tree/tree_test.go` — table tests for filtered flatten + ancestor preservation.
- `internal/ui/keys.go` — add `HideDone` (`H`), `WipFocus` (`W`), `Search` (`f`) bindings; surface in help.
- `internal/ui/model.go` — new view-state fields, predicate composition in `rebuildRows`, key routing, search-mode input, View bar + empty-filtered guard.
- `internal/ui/model_test.go` — model-level tests for toggles, search routing, composition, cursor re-resolution.

Three tasks, in order: pure layer first (everything depends on it), then filter toggles, then search.

---

## Task 1: Predicate-based filtered flatten (pure layer)

**Files:**
- Modify: `internal/tree/tree.go`
- Test: `internal/tree/tree_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/tree/tree_test.go`. These cover nil-predicate parity, leaf match with ancestor preservation, deep-descendant preservation, no-match, and fold interaction:

```go
func statePred(state string) Predicate {
	return func(y *yaks.Yak) bool { return y.State == state }
}

func TestFlattenFilteredNilPredicateMatchesFlatten(t *testing.T) {
	roots := []yaks.Yak{
		{ID: "a", Name: "alpha", State: "todo", Children: []yaks.Yak{
			{ID: "b", Name: "beta", State: "wip"},
		}},
		{ID: "c", Name: "gamma", State: "done"},
	}
	exp := map[string]bool{}
	got := FlattenFiltered(roots, exp, nil)
	want := Flatten(roots, exp)
	if len(got) != len(want) {
		t.Fatalf("nil predicate: got %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Yak.ID != want[i].Yak.ID || got[i].Depth != want[i].Depth {
			t.Fatalf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFlattenFilteredLeafMatchKeepsAncestors(t *testing.T) {
	// Only "b" (wip) matches. Its parent "a" (todo) must stay so the match is
	// reachable; unrelated root "c" (done) must be hidden.
	roots := []yaks.Yak{
		{ID: "a", Name: "alpha", State: "todo", Children: []yaks.Yak{
			{ID: "b", Name: "beta", State: "wip"},
			{ID: "d", Name: "delta", State: "done"},
		}},
		{ID: "c", Name: "gamma", State: "done"},
	}
	got := FlattenFiltered(roots, map[string]bool{}, statePred("wip"))
	ids := rowIDs(got)
	want := []string{"a", "b"}
	if !equalStrings(ids, want) {
		t.Fatalf("got ids %v, want %v", ids, want)
	}
}

func TestFlattenFilteredDeepDescendantKeepsFullChain(t *testing.T) {
	// Match is two levels deep; the whole chain a→b→c stays.
	roots := []yaks.Yak{
		{ID: "a", Name: "a", State: "done", Children: []yaks.Yak{
			{ID: "b", Name: "b", State: "done", Children: []yaks.Yak{
				{ID: "c", Name: "c", State: "wip"},
			}},
		}},
	}
	got := FlattenFiltered(roots, map[string]bool{}, statePred("wip"))
	if !equalStrings(rowIDs(got), []string{"a", "b", "c"}) {
		t.Fatalf("got ids %v, want [a b c]", rowIDs(got))
	}
}

func TestFlattenFilteredNoMatchIsEmpty(t *testing.T) {
	roots := []yaks.Yak{
		{ID: "a", Name: "a", State: "todo"},
		{ID: "b", Name: "b", State: "done"},
	}
	got := FlattenFiltered(roots, map[string]bool{}, statePred("wip"))
	if len(got) != 0 {
		t.Fatalf("got %d rows, want 0", len(got))
	}
}

func TestFlattenFilteredCollapsedAncestorHidesDescendant(t *testing.T) {
	// "a" matches on its own, but is collapsed, so its matching child "b" is not
	// emitted (fold wins for display).
	roots := []yaks.Yak{
		{ID: "a", Name: "a", State: "wip", Children: []yaks.Yak{
			{ID: "b", Name: "b", State: "wip"},
		}},
	}
	got := FlattenFiltered(roots, map[string]bool{"a": false}, statePred("wip"))
	if !equalStrings(rowIDs(got), []string{"a"}) {
		t.Fatalf("got ids %v, want [a] (b hidden by collapse)", rowIDs(got))
	}
}

// helpers
func rowIDs(rows []Row) []string {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.Yak.ID
	}
	return ids
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tree/ -run TestFlattenFiltered`
Expected: FAIL — `FlattenFiltered` and `Predicate` undefined.

- [ ] **Step 3: Implement FlattenFiltered**

In `internal/tree/tree.go`, add the `Predicate` type and `FlattenFiltered`, and rewrite `Flatten` as a nil-predicate wrapper. Replace the existing `Flatten` function (lines 15-40) with:

```go
// Predicate reports whether a yak matches on its own merits (not counting its
// descendants). Ancestor preservation is handled by FlattenFiltered's walk.
type Predicate func(y *yaks.Yak) bool

// Flatten walks roots depth-first, emitting a Row per visible yak. A node is
// expanded unless expanded[id] is explicitly false (default = expanded).
func Flatten(roots []yaks.Yak, expanded map[string]bool) []Row {
	return FlattenFiltered(roots, expanded, nil)
}

// FlattenFiltered is Flatten plus a visibility predicate. A node is emitted if
// it matches pred OR any descendant matches (ancestor preservation), so matches
// stay reachable in the hierarchy. Fold state still applies: the children of a
// visible-but-collapsed node are not walked for display. A nil predicate emits
// every node (identical to plain Flatten).
func FlattenFiltered(roots []yaks.Yak, expanded map[string]bool, pred Predicate) []Row {
	// subtreeMatches reports whether n or any descendant matches pred.
	var subtreeMatches func(n *yaks.Yak) bool
	subtreeMatches = func(n *yaks.Yak) bool {
		if pred == nil || pred(n) {
			return true
		}
		for i := range n.Children {
			if subtreeMatches(&n.Children[i]) {
				return true
			}
		}
		return false
	}

	var rows []Row
	var walk func(nodes []yaks.Yak, depth int)
	walk = func(nodes []yaks.Yak, depth int) {
		for i := range nodes {
			n := &nodes[i]
			if !subtreeMatches(n) {
				continue
			}
			isExpanded := true
			if v, ok := expanded[n.ID]; ok {
				isExpanded = v
			}
			rows = append(rows, Row{
				Yak:         n,
				Depth:       depth,
				HasChildren: len(n.Children) > 0,
				Expanded:    isExpanded,
			})
			if len(n.Children) > 0 && isExpanded {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(roots, 0)
	return rows
}
```

- [ ] **Step 4: Run the tree tests**

Run: `go test ./internal/tree/`
Expected: PASS — new tests plus all pre-existing `Flatten` tests (parity wrapper keeps them green).

- [ ] **Step 5: Confirm formatting + whole build**

Run: `gofmt -l internal/tree/ && go build ./...`
Expected: no gofmt output; build succeeds.

- [ ] **Step 6: Commit**

```bash
git add internal/tree/tree.go internal/tree/tree_test.go
git commit -m "feat(tree): FlattenFiltered with ancestor-preserving predicate"
```

---

## Task 2: Filter toggles (H hide-done, W wip-focus)

**Files:**
- Modify: `internal/ui/keys.go`
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/ui/model_test.go`. Uses the existing `loaded` helper and `twoYaks`-style fixtures:

```go
func parentWithStates() []yaks.Yak {
	// done parent with a wip child, and a standalone done leaf.
	return []yaks.Yak{
		{ID: "p", Name: "parent", State: "done", Children: []yaks.Yak{
			{ID: "c", Name: "child", State: "wip"},
		}},
		{ID: "z", Name: "zeta", State: "done"},
	}
}

func rowIDset(m Model) []string {
	ids := make([]string, len(m.rows))
	for i, r := range m.rows {
		ids[i] = r.Yak.ID
	}
	return ids
}

func TestHideDoneKeepsDoneAncestorOfActiveChild(t *testing.T) {
	m := loaded(t, parentWithStates())
	// Press capital H.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	mm := m2.(Model)
	if !mm.hideDone {
		t.Fatal("H should enable hideDone")
	}
	// parent (done) stays because child (wip) is active; zeta (done leaf) gone.
	got := rowIDset(mm)
	if len(got) != 2 || got[0] != "p" || got[1] != "c" {
		t.Fatalf("hideDone rows = %v, want [p c]", got)
	}
}

func TestHideDoneTogglesOff(t *testing.T) {
	m := loaded(t, parentWithStates())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	mm := m3.(Model)
	if mm.hideDone {
		t.Fatal("second H should disable hideDone")
	}
	if len(mm.rows) != 3 {
		t.Fatalf("after toggle off, rows = %d, want 3", len(mm.rows))
	}
}

func TestWipFocusShowsOnlyActivePlusAncestors(t *testing.T) {
	m := loaded(t, parentWithStates())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}})
	mm := m2.(Model)
	if !mm.wipFocus {
		t.Fatal("W should enable wipFocus")
	}
	got := rowIDset(mm)
	if len(got) != 2 || got[0] != "p" || got[1] != "c" {
		t.Fatalf("wipFocus rows = %v, want [p c] (parent kept as ancestor of wip child)", got)
	}
}

func TestFilterCursorReresolvesWhenSelectedHidden(t *testing.T) {
	// Cursor on the done leaf "z"; hideDone hides it; cursor must land on a valid row.
	m := loaded(t, parentWithStates())
	// move cursor down to "z" (index 2: p,c,z)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m3.(Model).selectedID() != "z" {
		t.Fatalf("precondition: selected = %q, want z", m3.(Model).selectedID())
	}
	m4, _ := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	mm := m4.(Model)
	if mm.cursor < 0 || mm.cursor >= len(mm.rows) {
		t.Fatalf("cursor %d out of bounds for %d rows", mm.cursor, len(mm.rows))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run 'TestHideDone|TestWipFocus|TestFilter'`
Expected: FAIL — `hideDone`, `wipFocus` undefined; `H`/`W` not bound.

- [ ] **Step 3: Add the key bindings**

In `internal/ui/keys.go`, add three fields to the `keyMap` struct after `Edit`:

```go
	HideDone key.Binding
	WipFocus key.Binding
	Search   key.Binding
```

Add their bindings in `defaultKeys()` after the `Edit` binding:

```go
		HideDone: key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "hide done")),
		WipFocus: key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "wip focus")),
		Search:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "search")),
```

Add to `FullHelp`'s third row (the one with Find/Reload/Help/Quit) so they show in help:

```go
		{k.HideDone, k.WipFocus, k.Search, k.Find, k.Reload, k.Help, k.Quit},
```

(`Search` is wired here but only takes effect in Task 3; binding it now keeps keys.go edited once.)

- [ ] **Step 4: Add model fields and predicate composition**

In `internal/ui/model.go`, add fields to the `Model` struct after the editing block (after `ta textarea.Model`):

```go
	hideDone bool // H: hide done yaks (done subtrees with no active descendant)
	wipFocus bool // W: show only wip/blocked yaks (+ ancestors)
```

Replace `rebuildRows` (currently lines 108-111) with a version that composes the active filters:

```go
func (m *Model) rebuildRows() {
	m.rows = tree.FlattenFiltered(m.roots, m.expanded, m.filterPredicate())
	m.cursor = tree.ClampCursor(m.cursor, len(m.rows))
}

// filterPredicate ANDs the active view filters into one predicate, or returns
// nil when no filter is active (nil = show everything). Text search is folded
// in here in Task 3.
func (m Model) filterPredicate() tree.Predicate {
	hideDone := m.hideDone
	wipFocus := m.wipFocus
	if !hideDone && !wipFocus {
		return nil
	}
	return func(y *yaks.Yak) bool {
		if hideDone && y.State == yaks.StateDone {
			return false
		}
		if wipFocus && y.State != yaks.StateWip && y.State != yaks.StateBlocked {
			return false
		}
		return true
	}
}
```

- [ ] **Step 5: Add the toggle handling with cursor preservation**

In `handleKey`, add two cases to the tree-focus `switch` (after the `Edit` case, before the closing brace at line 254). Each preserves the cursor on the same yak when possible:

```go
	case key.Matches(msg, m.keys.HideDone):
		id := m.selectedID()
		m.hideDone = !m.hideDone
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
	case key.Matches(msg, m.keys.WipFocus):
		id := m.selectedID()
		m.wipFocus = !m.wipFocus
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
```

Add the `restoreCursor` helper near `selectedID` (after line 118):

```go
// restoreCursor puts the cursor back on the yak with the given id if it's still
// visible; otherwise leaves it clamped to a valid row (rebuildRows already
// clamped). Used after a filter change so the selection follows the yak.
func (m *Model) restoreCursor(id string) {
	if id == "" {
		return
	}
	if idx := tree.IndexOfID(m.rows, id); idx >= 0 {
		m.cursor = idx
	} else {
		m.cursor = tree.ClampCursor(m.cursor, len(m.rows))
	}
}
```

- [ ] **Step 6: Run the UI tests**

Run: `go test ./internal/ui/ -run 'TestHideDone|TestWipFocus|TestFilter'`
Expected: PASS.

- [ ] **Step 7: Run the full suite + gofmt**

Run: `gofmt -l . && go test ./...`
Expected: no gofmt output; all packages pass (existing tests unaffected — no filters active by default means nil predicate).

- [ ] **Step 8: Commit**

```bash
git add internal/ui/keys.go internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): H hide-done and W wip-focus filter toggles"
```

---

## Task 3: Incremental text search (f)

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/ui/model_test.go`:

```go
func searchYaks() []yaks.Yak {
	return []yaks.Yak{
		{ID: "a", Name: "auth login", State: "todo"},
		{ID: "b", Name: "billing", State: "wip"},
		{ID: "c", Name: "auth logout", State: "todo"},
	}
}

func TestSearchEntersModeAndFiltersLive(t *testing.T) {
	m := loaded(t, searchYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !m2.(Model).searching {
		t.Fatal("f should enter search mode")
	}
	// type "auth"
	m3 := m2
	for _, r := range "auth" {
		m3, _ = m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	got := rowIDset(m3.(Model))
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("search 'auth' rows = %v, want [a c]", got)
	}
}

func TestSearchEnterCommitsAndKeepsFilter(t *testing.T) {
	m := loaded(t, searchYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m3 := m2
	for _, r := range "billing" {
		m3, _ = m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m4, _ := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m4.(Model)
	if mm.searching {
		t.Fatal("enter should exit search input mode")
	}
	if mm.query != "billing" {
		t.Fatalf("query = %q, want billing", mm.query)
	}
	if len(mm.rows) != 1 || mm.rows[0].Yak.ID != "b" {
		t.Fatalf("committed search rows = %v, want [b]", rowIDset(mm))
	}
}

func TestSearchEscClearsFilter(t *testing.T) {
	m := loaded(t, searchYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m3 := m2
	for _, r := range "auth" {
		m3, _ = m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m4, _ := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m4.(Model)
	if mm.searching || mm.query != "" {
		t.Fatalf("esc should clear search: searching=%v query=%q", mm.searching, mm.query)
	}
	if len(mm.rows) != 3 {
		t.Fatalf("after esc, rows = %d, want 3 (full tree)", len(mm.rows))
	}
}

func TestSearchKeysAreTextNotCommands(t *testing.T) {
	// While searching, 'd' is text input, not the "done" triage key.
	sc := &stubClient{roots: searchYaks()}
	m := loaded(t, searchYaks())
	m.client = sc
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if len(sc.setCalls) != 0 {
		t.Fatalf("d while searching must not trigger triage, got %+v", sc.setCalls)
	}
	if m3.(Model).search.Value() != "d" {
		t.Fatalf("search value = %q, want d", m3.(Model).search.Value())
	}
}

func TestSearchComposesWithHideDone(t *testing.T) {
	// "auth login" todo, "auth done" done; hideDone + search "auth" → only the todo.
	roots := []yaks.Yak{
		{ID: "a", Name: "auth login", State: "todo"},
		{ID: "x", Name: "auth archived", State: "done"},
	}
	m := loaded(t, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m4 := m3
	for _, r := range "auth" {
		m4, _ = m4.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	got := rowIDset(m4.(Model))
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("hideDone+search rows = %v, want [a]", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/ -run TestSearch`
Expected: FAIL — `searching`, `search`, `query` undefined.

- [ ] **Step 3: Add the textinput import and model fields**

In `internal/ui/model.go`, add to the bubbles imports (next to `textarea`):

```go
	"github.com/charmbracelet/bubbles/textinput"
```

Add fields to the `Model` struct after `wipFocus`:

```go
	searching bool            // true while the search input line is open
	search    textinput.Model // one-line incremental name filter
	query     string          // committed search text (applied when input closed)
```

Initialize the textinput in `New()` before the `return` (next to the `ta` setup):

```go
	ti := textinput.New()
	ti.Prompt = "search: "
	ti.CharLimit = 0
```

And add `search: ti` to the returned `Model{...}` literal.

- [ ] **Step 4: Fold search into the predicate**

Update `filterPredicate` (from Task 2) so the active query also constrains
matches. Replace it with:

```go
// filterPredicate ANDs the active view filters into one predicate, or returns
// nil when none are active (nil = show everything). The text query is taken
// live from the input while searching, otherwise from the committed query.
func (m Model) filterPredicate() tree.Predicate {
	hideDone := m.hideDone
	wipFocus := m.wipFocus
	q := m.query
	if m.searching {
		q = m.search.Value()
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if !hideDone && !wipFocus && q == "" {
		return nil
	}
	return func(y *yaks.Yak) bool {
		if hideDone && y.State == yaks.StateDone {
			return false
		}
		if wipFocus && y.State != yaks.StateWip && y.State != yaks.StateBlocked {
			return false
		}
		if q != "" && !strings.Contains(strings.ToLower(y.Name), q) {
			return false
		}
		return true
	}
}
```

Add `"strings"` to the import block. (Confirmed: model.go does not currently
import it — add it to the standard-library group alongside `context`, `fmt`,
`os`, `os/exec`.)

- [ ] **Step 5: Add search-mode key routing**

In `handleKey`, add a search-mode block at the very top — before the `editing`
block — so search input owns the keyboard while active (mirrors edit mode):

```go
	// Search mode owns the keyboard: enter commits the query (filter persists),
	// esc clears it, everything else is text input for the search field.
	if m.searching {
		switch msg.Type {
		case tea.KeyEnter:
			m.query = m.search.Value()
			m.searching = false
			m.search.Blur()
			id := m.selectedID()
			m.rebuildRows()
			m.restoreCursor(id)
			m.refreshDetail()
			return m, nil
		case tea.KeyEsc:
			m.searching = false
			m.search.Blur()
			m.search.SetValue("")
			m.query = ""
			id := m.selectedID()
			m.rebuildRows()
			m.restoreCursor(id)
			m.refreshDetail()
			return m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		// Re-filter live as the query changes.
		id := m.selectedID()
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
		return m, cmd
	}
```

Add the `f` entry case to the tree-focus `switch` (next to the HideDone/WipFocus cases):

```go
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.search.SetValue("")
		m.search.Focus()
		return m, nil
```

- [ ] **Step 6: Show search input / active-filter indicator in the View bar**

In `View()`, update the bar `switch` (currently lines 444-452) so search input and
active filters are visible. Replace that `switch` with:

```go
	var bar string
	switch {
	case m.searching:
		bar = subtle.Render(m.search.View() + "  (enter to keep · esc to clear)")
	case m.editing:
		bar = subtle.Render("editing — ctrl+s save · esc cancel")
	case m.status != "":
		bar = statusErr.Render(m.status)
	default:
		if ind := m.filterIndicator(); ind != "" {
			bar = subtle.Render(ind)
		} else {
			bar = m.help.View(m.keys)
		}
	}
```

Add the `filterIndicator` helper near `View` (before it):

```go
// filterIndicator summarizes active view filters for the status bar, or "" when
// none are active.
func (m Model) filterIndicator() string {
	var parts []string
	if m.hideDone {
		parts = append(parts, "[hide-done]")
	}
	if m.wipFocus {
		parts = append(parts, "[wip-focus]")
	}
	if m.query != "" {
		parts = append(parts, "[search: "+m.query+"]")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ") + "  ·  H/W toggle · f search · esc clears search"
}
```

- [ ] **Step 7: Fix the empty-state guard to distinguish "no yaks" from "filtered out"**

In `View()`, the guard at line 414 (`if len(m.rows) == 0`) currently assumes an
empty tree means no yaks exist. With filters, it can mean "everything filtered
out." Replace that block with:

```go
	if len(m.rows) == 0 {
		var msg string
		if m.hideDone || m.wipFocus || m.query != "" || m.searching {
			msg = "No yaks match the current view.\n\nPress esc to clear search, or H / W to clear filters."
		} else {
			msg = "No yaks yet.\n\nStart one with:  yx add \"my first yak\"\n\nq to quit · r to reload"
		}
		return subtle.Render(msg)
	}
```

(Also removes the stale "(v1.1 will let you add them right here.)" line, which is now shipped.)

- [ ] **Step 8: Settle deps if needed**

Run: `go build ./... 2>&1`
If it reports a missing go.sum entry for `textinput`'s deps, run `go mod tidy`.
Expected: build succeeds (textinput is in the bubbles module already cached, so
no new download is likely).

- [ ] **Step 9: Run search tests**

Run: `go test ./internal/ui/ -run TestSearch`
Expected: PASS.

- [ ] **Step 10: Run the full suite + gofmt**

Run: `gofmt -l . && go vet ./... && go test ./...`
Expected: no gofmt output; vet clean; all packages pass.

- [ ] **Step 11: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go go.mod go.sum
git commit -m "feat(ui): incremental text search (f), composes with filters"
```

---

## Task 4: Documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/reference/keybindings.md`
- Modify: `docs/how-to/` (new file)

- [ ] **Step 1: Update the README keys table**

In `README.md`, add these rows to the Keys table after the `e` row:

```markdown
| `H` | hide done yaks |
| `W` | focus wip / blocked |
| `f` | search by name |
```

- [ ] **Step 2: Update the keybindings reference**

In `docs/reference/keybindings.md`, add the same three keys to the main table,
and add a short "Filtering & search" subsection noting that `H` and `W` are
toggles, `f` opens an incremental search where `enter` keeps the filter and
`esc` clears it, and that the filters compose.

- [ ] **Step 3: Write a how-to for filtering**

Create `docs/how-to/filter-and-search.md`:

```markdown
# Filter and search the tree

Narrow what the tree pane shows without changing any yak.

## Hide completed work

Press `H` to hide done yaks. A done yak that still has an active (todo/wip/
blocked) descendant stays visible, so you don't lose the path to live work.
Press `H` again to show everything.

## Focus on what's active

Press `W` to show only `wip` and `blocked` yaks (plus their parents). This is
the "what am I working on right now" view. Press `W` again to clear it.

## Search by name

Press `f` and type — the tree narrows to yaks whose name contains what you typed
(case-insensitive), keeping their parents visible. Then:

- `enter` keeps the filter applied so you can navigate and triage the narrowed
  set.
- `esc` clears the search and restores the full tree.

The filters and search stack: with `H` on and a search active, you see only
matching, non-done yaks. The status bar shows which filters are active.
```

- [ ] **Step 4: Commit**

```bash
git add README.md docs/reference/keybindings.md docs/how-to/filter-and-search.md
git commit -m "docs: document H/W filters and f search"
```

---

## Self-review notes

- **Spec coverage:** FlattenFiltered + ancestor preservation (Task 1); H/W
  toggles + composition + cursor re-resolution (Task 2); search mode, live
  filter, commit/cancel, routing, indicator, empty-filtered guard (Task 3);
  docs (Task 4). All spec sections map to a task.
- **Type consistency:** `Predicate`, `FlattenFiltered`, `filterPredicate`,
  `restoreCursor`, `filterIndicator`, and fields `hideDone`/`wipFocus`/
  `searching`/`search`/`query` are used identically across tasks.
- **Order:** pure layer first; `restoreCursor` is introduced in Task 2 and
  reused in Task 3; `filterPredicate` is created in Task 2 and extended in Task
  3 (the engineer replaces the whole function — full text given both times per
  the no-"similar-to" rule).
- **No new yx calls;** read-only round, so no client/interface changes and the
  existing stubClient needs no new methods.
