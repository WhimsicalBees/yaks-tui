package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// renderTree draws the visible rows into a box of the given inner size.
//
// Truncation runs on PLAIN text only — never on a string that already carries
// ANSI escapes. Styling (the colored state dot, the selected-row highlight) is
// applied afterward. Truncating styled text would both miscount width (escape
// bytes consume the budget) and risk slicing off a reset code, leaking the
// highlight onto following cells.
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
		// The fixed prefix (indent, fold glyph, dot, surrounding spaces) is plain;
		// only the name is variable-length, so we truncate the name to whatever
		// width remains after the prefix.
		prefix := indent + fold + " " + stateDot(row.Yak.State) + " "
		avail := width - utf8.RuneCountInString(prefix)
		name := truncate(row.Yak.Name, avail)

		// Style after truncation: color the dot, then highlight the whole row if
		// it's the cursor. The assembled visible width is already <= width.
		dot := lipgloss.NewStyle().Foreground(stateColor(row.Yak.State)).Render(stateDot(row.Yak.State))
		line := indent + fold + " " + dot + " " + name
		if i == m.cursor {
			line = selectedRow.Render(line)
		}
		b.WriteString(line)
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

// truncate hard-cuts a string to max runes, appending an ellipsis when it cuts.
// It slices on rune boundaries (not bytes); display-width of wide runes (CJK) is
// not accounted for, which is fine for yak names.
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
