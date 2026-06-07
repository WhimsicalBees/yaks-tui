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

// Flatten walks roots depth-first, emitting a Row per visible yak. A node is
// expanded unless expanded[id] is explicitly false (default = expanded).
func Flatten(roots []yaks.Yak, expanded map[string]bool) []Row {
	var rows []Row
	var walk func(nodes []yaks.Yak, depth int)
	walk = func(nodes []yaks.Yak, depth int) {
		for i := range nodes {
			n := &nodes[i]
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
