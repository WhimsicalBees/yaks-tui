package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"yaks-tui/internal/yaks"
)

// stubClient implements the dataSource the model depends on.
type stubClient struct {
	roots    []yaks.Yak
	listErr  error
	setErr   error
	setCalls []struct{ id, state string }
}

func (s *stubClient) List(_ context.Context) ([]yaks.Yak, error) { return s.roots, s.listErr }
func (s *stubClient) SetState(_ context.Context, id, state string) error {
	s.setCalls = append(s.setCalls, struct{ id, state string }{id, state})
	return s.setErr
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
