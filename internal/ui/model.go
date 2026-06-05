package ui

import (
	"context"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"yaks-tui/internal/tree"
	"yaks-tui/internal/yaks"
)

// dataSource is the slice of the yaks client the UI needs. Defined here (consumer
// side) so the model can be tested with a stub.
type dataSource interface {
	List(ctx context.Context) ([]yaks.Yak, error)
	SetState(ctx context.Context, id, state string) error
}

type focus int

const (
	focusTree focus = iota
	focusDetail
)

// Messages produced by async commands.
type loadedMsg struct{ roots []yaks.Yak }
type errMsg struct{ err error }
type stateChangedMsg struct{}

type Model struct {
	client dataSource
	keys   keyMap
	help   help.Model

	roots    []yaks.Yak
	rows     []tree.Row
	expanded map[string]bool
	cursor   int

	focus    focus
	detail   viewport.Model
	width    int
	height   int
	status   string // transient message (errors etc.)
	showHelp bool
	ready    bool
}

func New(client dataSource) Model {
	return Model{
		client:   client,
		keys:     defaultKeys(),
		help:     help.New(),
		expanded: map[string]bool{},
	}
}

func (m Model) Init() tea.Cmd { return m.loadCmd() }

// loadCmd fetches the tree asynchronously.
func (m Model) loadCmd() tea.Cmd {
	return func() tea.Msg {
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{roots}
	}
}

func (m *Model) rebuildRows() {
	m.rows = tree.Flatten(m.roots, m.expanded)
	m.cursor = tree.ClampCursor(m.cursor, len(m.rows))
}

func (m Model) selectedID() string {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return m.rows[m.cursor].Yak.ID
	}
	return ""
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case loadedMsg:
		m.roots = msg.roots
		m.rebuildRows()
		m.refreshDetail()
		return m, nil

	case errMsg:
		m.status = msg.err.Error()
		return m, nil

	case stateChangedMsg:
		// A mutation succeeded; reload preserving cursor by id.
		return m, m.reloadPreservingCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil
	}
	return m, nil
}

func (m *Model) layout() {
	// Reserve 1 line for the status/help bar; split width ~40/60.
	bodyHeight := m.height - 2
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	detailWidth := m.width*6/10 - 2
	if detailWidth < 1 {
		detailWidth = 1
	}
	if m.detail.Width == 0 {
		m.detail = viewport.New(detailWidth, bodyHeight)
	} else {
		m.detail.Width = detailWidth
		m.detail.Height = bodyHeight
	}
}

func (m *Model) refreshDetail() {
	// Filled in Task 12 (glamour render). Placeholder keeps the build green.
	if m.cursor < len(m.rows) {
		m.detail.SetContent(m.rows[m.cursor].Yak.Name)
	}
}

func (m Model) reloadPreservingCmd() tea.Cmd {
	prevID := m.selectedID()
	return func() tea.Msg {
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		_ = prevID // cursor restoration wired in Task 14
		return loadedMsg{roots}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}
	// Graceful guards added in Task 15; basic two-pane here.
	bodyHeight := m.height - 2
	treeWidth := m.width*4/10 - 2
	detailWidth := m.width*6/10 - 2

	treeBorder := blurredBorder
	detailBorder := blurredBorder
	if m.focus == focusTree {
		treeBorder = focusedBorder
	} else {
		detailBorder = focusedBorder
	}

	left := treeBorder.Width(treeWidth).Height(bodyHeight).Render(m.renderTree(treeWidth, bodyHeight))
	right := detailBorder.Width(detailWidth).Height(bodyHeight).Render(m.detail.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	bar := m.help.View(m.keys)
	if m.status != "" {
		bar = statusErr.Render(m.status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, bar)
}
