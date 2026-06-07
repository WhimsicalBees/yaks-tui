package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

// stateDot returns a glyph representing a yak's state.
func stateDot(state string) string {
	switch state {
	case yaks.StateTodo:
		return "◌"
	case yaks.StateWip:
		return "●"
	case yaks.StateBlocked:
		return "▲"
	case yaks.StateDone:
		return "✓"
	default:
		return "?"
	}
}

// stateColor maps a state to a Lip Gloss color.
func stateColor(state string) lipgloss.Color {
	switch state {
	case yaks.StateWip:
		return lipgloss.Color("39") // blue
	case yaks.StateBlocked:
		return lipgloss.Color("203") // red
	case yaks.StateDone:
		return lipgloss.Color("78") // green
	default:
		return lipgloss.Color("245") // grey (todo/unknown)
	}
}

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39"))
	blurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	selectedRow = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Bold(true)
	statusErr = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	subtle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
