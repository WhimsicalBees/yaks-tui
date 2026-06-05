package ui

import (
	"testing"

	"yaks-tui/internal/yaks"
)

func TestStateDot(t *testing.T) {
	cases := map[string]string{
		yaks.StateTodo:    "◌",
		yaks.StateWip:     "●",
		yaks.StateBlocked: "▲",
		yaks.StateDone:    "✓",
		"weird":           "?",
	}
	for state, want := range cases {
		if got := stateDot(state); got != want {
			t.Errorf("stateDot(%q) = %q, want %q", state, got, want)
		}
	}
}

func TestRenderTreeContainsYaksAndCursor(t *testing.T) {
	m := New(&stubClient{roots: twoYaks()})
	mm := m
	mm.width, mm.height = 80, 24
	mm.layout()
	mm.roots = twoYaks()
	mm.rebuildRows()
	out := mm.renderTree(38, 22)
	if !contains(out, "alpha") || !contains(out, "beta") {
		t.Fatalf("tree missing yak names:\n%s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
