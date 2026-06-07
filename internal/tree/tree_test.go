package tree

import (
	"testing"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

func sample() []yaks.Yak {
	return []yaks.Yak{{
		ID: "root", Name: "deploy app", State: "wip",
		Children: []yaks.Yak{
			{ID: "ci", Name: "set up CI", State: "blocked", Children: []yaks.Yak{
				{ID: "lint", Name: "fix linter", State: "todo"},
			}},
			{ID: "tests", Name: "write tests", State: "wip"},
		},
	}}
}

func TestFlattenAllExpanded(t *testing.T) {
	rows := Flatten(sample(), map[string]bool{}) // empty map = default expanded
	// root, ci, lint, tests => 4 rows
	if len(rows) != 4 {
		t.Fatalf("want 4 rows, got %d: %+v", len(rows), rows)
	}
	if rows[0].Yak.ID != "root" || rows[0].Depth != 0 {
		t.Errorf("row0 = %+v", rows[0])
	}
	if rows[1].Yak.ID != "ci" || rows[1].Depth != 1 || !rows[1].HasChildren {
		t.Errorf("row1 = %+v", rows[1])
	}
	if rows[2].Yak.ID != "lint" || rows[2].Depth != 2 {
		t.Errorf("row2 = %+v", rows[2])
	}
}

func TestFlattenCollapsed(t *testing.T) {
	// Collapse "ci": its child "lint" must disappear.
	rows := Flatten(sample(), map[string]bool{"ci": false})
	var ids []string
	for _, r := range rows {
		ids = append(ids, r.Yak.ID)
	}
	// root, ci, tests (no lint)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d: %v", len(rows), ids)
	}
	for _, id := range ids {
		if id == "lint" {
			t.Fatal("lint should be hidden under collapsed ci")
		}
	}
}

func TestIndexOfID(t *testing.T) {
	rows := Flatten(sample(), map[string]bool{})
	if got := IndexOfID(rows, "tests"); got != 3 {
		t.Fatalf("IndexOfID(tests) = %d, want 3", got)
	}
	if got := IndexOfID(rows, "missing"); got != -1 {
		t.Fatalf("IndexOfID(missing) = %d, want -1", got)
	}
}

func TestClampCursor(t *testing.T) {
	if got := ClampCursor(5, 4); got != 3 {
		t.Errorf("ClampCursor(5,4) = %d, want 3", got)
	}
	if got := ClampCursor(-2, 4); got != 0 {
		t.Errorf("ClampCursor(-2,4) = %d, want 0", got)
	}
	if got := ClampCursor(2, 0); got != 0 {
		t.Errorf("ClampCursor on empty = %d, want 0", got)
	}
}

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
