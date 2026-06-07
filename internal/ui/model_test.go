package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

// stubClient implements the dataSource the model depends on.
type stubClient struct {
	roots    []yaks.Yak
	listErr  error
	setErr   error
	setCalls []struct{ id, state string }

	ctxErr   error
	ctxCalls []struct{ id, content string }
}

func (s *stubClient) List(_ context.Context) ([]yaks.Yak, error) { return s.roots, s.listErr }
func (s *stubClient) SetState(_ context.Context, id, state string) error {
	s.setCalls = append(s.setCalls, struct{ id, state string }{id, state})
	return s.setErr
}
func (s *stubClient) SetContext(_ context.Context, id, content string) error {
	s.ctxCalls = append(s.ctxCalls, struct{ id, content string }{id, content})
	return s.ctxErr
}

func twoYaks() []yaks.Yak {
	return []yaks.Yak{
		{ID: "a", Name: "alpha", State: "todo"},
		{ID: "b", Name: "beta", State: "wip"},
	}
}

func TestModelHandlesLoadedMsg(t *testing.T) {
	m := New(&stubClient{roots: twoYaks()})
	// Simulate the window sizing and the async load completing.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: twoYaks()})
	mm := m3.(Model)
	if len(mm.rows) != 2 {
		t.Fatalf("want 2 rows after load, got %d", len(mm.rows))
	}
	if mm.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", mm.cursor)
	}
}

func TestModelQuit(t *testing.T) {
	m := New(&stubClient{roots: twoYaks()})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func loaded(t *testing.T, roots []yaks.Yak) Model {
	t.Helper()
	m := New(&stubClient{roots: roots})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: roots})
	return m3.(Model)
}

func TestNavigationDownUp(t *testing.T) {
	m := loaded(t, twoYaks())
	down := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	m2, _ := m.Update(down)
	if m2.(Model).cursor != 1 {
		t.Fatalf("cursor after down = %d, want 1", m2.(Model).cursor)
	}
	up := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	m3, _ := m2.Update(up)
	if m3.(Model).cursor != 0 {
		t.Fatalf("cursor after up = %d, want 0", m3.(Model).cursor)
	}
}

func TestFocusToggle(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m2.(Model).focus != focusDetail {
		t.Fatal("tab should move focus to detail")
	}
}

func TestTriageWipCallsSetState(t *testing.T) {
	sc := &stubClient{roots: twoYaks()}
	m := New(sc)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: twoYaks()})
	// cursor on "a"; press w
	m4, cmd := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	_ = m4
	if cmd == nil {
		t.Fatal("expected a command from triage key")
	}
	// Execute the command to trigger the SetState call.
	msg := cmd()
	if _, ok := msg.(stateChangedMsg); !ok {
		t.Fatalf("expected stateChangedMsg, got %T", msg)
	}
	if len(sc.setCalls) != 1 || sc.setCalls[0].id != "a" || sc.setCalls[0].state != "wip" {
		t.Fatalf("SetState calls = %+v", sc.setCalls)
	}
}

func TestReloadPreservesCursorByID(t *testing.T) {
	m := loaded(t, twoYaks())
	// move cursor to "b"
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mm := m2.(Model)
	if mm.selectedID() != "b" {
		t.Fatalf("precondition: selected = %q", mm.selectedID())
	}
	// Simulate a reload that returns the same tree; cursor should stay on "b".
	m3, _ := mm.Update(loadedMsgPreserving{roots: twoYaks(), prevID: "b"})
	if m3.(Model).selectedID() != "b" {
		t.Fatalf("cursor not preserved: selected = %q", m3.(Model).selectedID())
	}
}

func TestTriageNoSelectionIsNoOp(t *testing.T) {
	// No load happened, so there are no rows and nothing is selected. A triage
	// key must produce no command (and never call SetState).
	sc := &stubClient{roots: twoYaks()}
	m := New(sc)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd != nil {
		t.Fatal("triage with no selection should return a nil command")
	}
	if len(sc.setCalls) != 0 {
		t.Fatalf("SetState should not be called with no selection, got %+v", sc.setCalls)
	}
}

func TestReloadWhenSelectedYakVanished(t *testing.T) {
	// Cursor is on "b"; a reload returns a tree that no longer contains "b"
	// (e.g. another client removed it). The cursor must land on a valid row,
	// not dangle past the end.
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	mm := m2.(Model)
	if mm.selectedID() != "b" {
		t.Fatalf("precondition: selected = %q", mm.selectedID())
	}
	oneYak := []yaks.Yak{{ID: "a", Name: "alpha", State: "todo"}}
	m3, _ := mm.Update(loadedMsgPreserving{roots: oneYak, prevID: "b"})
	mm3 := m3.(Model)
	if mm3.cursor < 0 || mm3.cursor >= len(mm3.rows) {
		t.Fatalf("cursor %d out of bounds for %d rows", mm3.cursor, len(mm3.rows))
	}
	if mm3.selectedID() != "a" {
		t.Fatalf("cursor should fall back to a valid yak, got %q", mm3.selectedID())
	}
}

func TestTriageSetStateErrorSurfacesAndDoesNotReload(t *testing.T) {
	// When SetState fails, the command yields errMsg (not stateChangedMsg), so
	// no reload is triggered and the displayed tree stays at its pre-mutation state.
	sc := &stubClient{roots: twoYaks(), setErr: errStub("boom")}
	m := New(sc)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m3, _ := m2.Update(loadedMsg{roots: twoYaks()})
	_, cmd := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if cmd == nil {
		t.Fatal("expected a command from triage key")
	}
	msg := cmd()
	em, ok := msg.(errMsg)
	if !ok {
		t.Fatalf("expected errMsg on SetState failure, got %T", msg)
	}
	if em.err == nil {
		t.Fatal("errMsg should carry the underlying error")
	}
}

// errStub is a minimal error for exercising failure paths.
type errStub string

func (e errStub) Error() string { return string(e) }

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
	// move to "b" which has no context
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
	// no load → no rows
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
	// type into the textarea
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
	// contextSavedMsg handling should exit edit mode.
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
	// Applying the errMsg must keep us in edit mode so edits aren't lost.
	m4, _ := m3.(Model).Update(msg)
	if !m4.(Model).editing {
		t.Fatal("save failure must keep edit mode active")
	}
}

func TestEditKeysReachTextareaNotTriage(t *testing.T) {
	// While editing, a 'd' is text input, not the "done" triage key.
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
	if mm.selectedID() != "c" {
		t.Fatalf("cursor should land on a visible yak (c), got %q", mm.selectedID())
	}
}

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

func TestCollapseExpand(t *testing.T) {
	roots := []yaks.Yak{{ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{{ID: "c", Name: "child", State: "todo"}}}}
	m := loaded(t, roots)
	if len(m.rows) != 2 {
		t.Fatalf("want 2 rows initially, got %d", len(m.rows))
	}
	// cursor on parent; collapse hides child.
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if len(m2.(Model).rows) != 1 {
		t.Fatalf("after collapse want 1 row, got %d", len(m2.(Model).rows))
	}
}
