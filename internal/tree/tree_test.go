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
