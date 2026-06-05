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
