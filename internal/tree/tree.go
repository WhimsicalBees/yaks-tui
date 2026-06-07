// Package tree turns the nested yak structure into a flat, navigable list of
// rows given an expansion state. Pure logic — no I/O, no UI.
package tree

import "github.com/WhimsicalBees/yaks-tui/internal/yaks"

// Row is one visible line in the tree pane.
type Row struct {
	Yak         *yaks.Yak
	Depth       int
	HasChildren bool
	Expanded    bool
}

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

// IndexOfID returns the row index whose yak has the given id, or -1.
func IndexOfID(rows []Row, id string) int {
	for i := range rows {
		if rows[i].Yak.ID == id {
			return i
		}
	}
	return -1
}

// ClampCursor keeps a cursor within [0, n-1]; returns 0 when n == 0.
func ClampCursor(cursor, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor > n-1 {
		return n - 1
	}
	return cursor
}
