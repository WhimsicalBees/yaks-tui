package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTree draws the visible rows into a box of the given inner size.
func (m Model) renderTree(width, height int) string {
	var b strings.Builder
	start, end := windowBounds(m.cursor, len(m.rows), height)
	for i := start; i < end; i++ {
		row := m.rows[i]
		indent := strings.Repeat("  ", row.Depth)
		fold := " "
		if row.HasChildren {
			if row.Expanded {
				fold = "▾"
			} else {
				fold = "▸"
			}
		}
		dot := lipgloss.NewStyle().Foreground(stateColor(row.Yak.State)).Render(stateDot(row.Yak.State))
		line := indent + fold + " " + dot + " " + row.Yak.Name
		if i == m.cursor {
			line = selectedRow.Render(line)
		}
		b.WriteString(truncate(line, width))
		b.WriteString("\n")
	}
	return b.String()
}

// windowBounds returns a [start,end) slice window of size <= height keeping cursor visible.
func windowBounds(cursor, n, height int) (int, int) {
	if height <= 0 || n == 0 {
		return 0, 0
	}
	if n <= height {
		return 0, n
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > n {
		end = n
		start = end - height
	}
	return start, end
}

// truncate hard-cuts a string to max display width (rune-naive; fine for ASCII names).
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
