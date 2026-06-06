package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"yaks-tui/internal/shell"
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
type loadedMsgPreserving struct {
	roots  []yaks.Yak
	prevID string
}

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
	mdStyle  string // glamour style name, resolved once at startup (see New)
}

func New(client dataSource) Model {
	// Resolve the markdown style ONCE, here, before tea.NewProgram takes over
	// stdin. Detecting the terminal background (a stdin read) is safe at this
	// point because no event loop is competing for input; doing it later in the
	// render loop would steal the user's keystrokes. See resolveMarkdownStyle.
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	dark := false
	if isTTY {
		dark = lipgloss.HasDarkBackground()
	}
	return Model{
		client:   client,
		keys:     defaultKeys(),
		help:     help.New(),
		expanded: map[string]bool{},
		mdStyle:  resolveMarkdownStyle(isTTY, dark),
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

	case loadedMsgPreserving:
		m.roots = msg.roots
		m.rebuildRows()
		if idx := tree.IndexOfID(m.rows, msg.prevID); idx >= 0 {
			m.cursor = idx
		}
		m.cursor = tree.ClampCursor(m.cursor, len(m.rows))
		m.refreshDetail()
		return m, nil

	case errMsg:
		m.status = msg.err.Error()
		return m, nil

	case stateChangedMsg:
		m.status = ""
		return m, m.reloadPreservingCmd()

	case jumpMsg:
		if msg.id != "" {
			if idx := tree.IndexOfID(m.rows, msg.id); idx >= 0 {
				m.cursor = idx
				m.refreshDetail()
			}
		}
		return m, nil

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
		m.help.ShowAll = m.showHelp
		return m, nil
	case key.Matches(msg, m.keys.Focus):
		if m.focus == focusTree {
			m.focus = focusDetail
		} else {
			m.focus = focusTree
		}
		return m, nil
	}

	// Detail focus: forward scrolling keys to the viewport.
	if m.focus == focusDetail {
		var cmd tea.Cmd
		m.detail, cmd = m.detail.Update(msg)
		return m, cmd
	}

	// Tree focus: navigation and folding.
	switch {
	case key.Matches(msg, m.keys.Down):
		m.cursor = tree.ClampCursor(m.cursor+1, len(m.rows))
		m.refreshDetail()
	case key.Matches(msg, m.keys.Up):
		m.cursor = tree.ClampCursor(m.cursor-1, len(m.rows))
		m.refreshDetail()
	case key.Matches(msg, m.keys.Collapse):
		if id := m.selectedID(); id != "" {
			m.expanded[id] = false
			m.rebuildRows()
			m.refreshDetail()
		}
	case key.Matches(msg, m.keys.Expand):
		if id := m.selectedID(); id != "" {
			m.expanded[id] = true
			m.rebuildRows()
			m.refreshDetail()
		}
	case key.Matches(msg, m.keys.Toggle):
		if id := m.selectedID(); id != "" {
			m.expanded[id] = !current(m.expanded, id)
			m.rebuildRows()
			m.refreshDetail()
		}
	case key.Matches(msg, m.keys.Wip):
		return m, m.setStateCmd(yaks.StateWip)
	case key.Matches(msg, m.keys.Blocked):
		return m, m.setStateCmd(yaks.StateBlocked)
	case key.Matches(msg, m.keys.Done):
		return m, m.setStateCmd(yaks.StateDone)
	case key.Matches(msg, m.keys.Todo):
		return m, m.setStateCmd(yaks.StateTodo)
	case key.Matches(msg, m.keys.Reload):
		return m, m.reloadPreservingCmd()
	case key.Matches(msg, m.keys.Find):
		return m, m.findCmd()
	}
	return m, nil
}

// flatYaks returns the currently visible yaks (for fuzzy find over the open tree).
func (m Model) flatYaks() []yaks.Yak {
	ys := make([]yaks.Yak, 0, len(m.rows))
	for _, r := range m.rows {
		ys = append(ys, *r.Yak)
	}
	return ys
}

type jumpMsg struct{ id string }

// findCmd launches fzf over the visible yaks and jumps the cursor to the
// selection. It hands the terminal to fzf via tea.ExecProcess: Bubble Tea
// releases the tty (exits raw mode + alt-screen), runs the command against the
// real terminal, then restores itself and delivers the callback's message.
//
// Because ExecProcess wires the child's stdio to the terminal, we can't pipe
// candidates on stdin or capture stdout with cmd.Output(). Instead shell.FzfExec
// writes candidates to a temp file (fzf reads them via a shell stdin redirect
// while drawing its UI on /dev/tty) and captures the selection in a second temp
// file (shell stdout redirect). We read that file in the callback.
func (m Model) findCmd() tea.Cmd {
	if !shell.Available() {
		return func() tea.Msg {
			return errMsg{fmt.Errorf("fuzzy find needs `fzf` installed")}
		}
	}

	lines := shell.FzfLines(m.flatYaks())
	cmd, outPath, cleanup, err := shell.FzfExec(lines)
	if err != nil {
		cleanup()
		return func() tea.Msg { return errMsg{err} }
	}

	return tea.ExecProcess(cmd, func(runErr error) tea.Msg {
		defer cleanup()
		if runErr != nil {
			// fzf exits 130 when the user cancels (Esc/Ctrl-C); via `sh -c`
			// that code propagates. Treat any cancel as "no selection".
			if ee, ok := runErr.(*exec.ExitError); ok && ee.ExitCode() == 130 {
				return jumpMsg{""}
			}
			return errMsg{runErr}
		}
		out, readErr := os.ReadFile(outPath)
		if readErr != nil {
			return errMsg{readErr}
		}
		return jumpMsg{shell.ParseFzfSelection(string(out))}
	})
}

// current returns the expansion value for id, defaulting to true.
func current(exp map[string]bool, id string) bool {
	if v, ok := exp[id]; ok {
		return v
	}
	return true
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
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		m.detail.SetContent(subtle.Render("Select a yak"))
		return
	}
	y := *m.rows[m.cursor].Yak
	m.detail.SetContent(renderMarkdown(detailMarkdown(y), m.mdStyle, m.detail.Width))
}

func (m Model) setStateCmd(state string) tea.Cmd {
	id := m.selectedID()
	if id == "" {
		return nil
	}
	return func() tea.Msg {
		if err := m.client.SetState(context.Background(), id, state); err != nil {
			return errMsg{err}
		}
		return stateChangedMsg{}
	}
}

func (m Model) reloadPreservingCmd() tea.Cmd {
	prevID := m.selectedID()
	return func() tea.Msg {
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsgPreserving{roots: roots, prevID: prevID}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "loading…"
	}
	const minW, minH = 40, 8
	if m.width < minW || m.height < minH {
		return subtle.Render("Terminal too small — please resize (need at least 40×8).")
	}
	if len(m.rows) == 0 {
		msg := "No yaks yet.\n\nStart one with:  yx add \"my first yak\"\n\n(v1.1 will let you add them right here.)\n\nq to quit · r to reload"
		return subtle.Render(msg)
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
