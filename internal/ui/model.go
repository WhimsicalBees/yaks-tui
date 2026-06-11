package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/WhimsicalBees/yaks-tui/internal/shell"
	"github.com/WhimsicalBees/yaks-tui/internal/tree"
	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

// dataSource is the slice of the yaks client the UI needs. Defined here (consumer
// side) so the model can be tested with a stub.
type dataSource interface {
	List(ctx context.Context) ([]yaks.Yak, error)
	SetState(ctx context.Context, id, state string) error
	SetContext(ctx context.Context, id, content string) error
	Add(ctx context.Context, parentID, name string, existing map[string]bool) (string, error)
	Rename(ctx context.Context, id, newName string) error
	Remove(ctx context.Context, id string, recursive bool) error
}

type focus int

const (
	focusTree focus = iota
	focusDetail
)

type inputMode int

const (
	inputNone inputMode = iota
	inputAddChild
	inputAddRoot
	inputRename
)

// Messages produced by async commands.
type loadedMsg struct{ roots []yaks.Yak }
type errMsg struct{ err error }
type stateChangedMsg struct{}
type contextSavedMsg struct{}
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

	editing bool           // true while the inline context editor is open
	editID  string         // id of the yak being edited (captured on entry)
	ta      textarea.Model // inline editor for the context body

	hideDone bool // H: hide done yaks (done subtrees with no active descendant)
	wipFocus bool // W: show only wip/blocked yaks (+ ancestors)

	searching bool            // true while the search input line is open
	search    textinput.Model // one-line incremental name filter
	query     string          // committed search text (applied when input closed)

	inputMode  inputMode       // which add/rename flow is open (inputNone = closed)
	inputParID string          // parent id for inputAddChild ("" = root)
	inputTgtID string          // target id for inputRename
	input      textinput.Model // one-line input for add/rename

	confirming bool   // remove confirmation prompt open
	removeID   string // captured target id
	removeName string // for the prompt text
	removeKids int    // child count → recursive flag + prompt wording
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
	ta := textarea.New()
	ta.Prompt = ""   // no per-line prompt gutter; the body is plain markdown
	ta.CharLimit = 0 // no limit
	ta.ShowLineNumbers = false
	ti := textinput.New()
	ti.Prompt = "search: "
	ti.CharLimit = 0
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 0
	return Model{
		client:   client,
		keys:     defaultKeys(),
		help:     help.New(),
		expanded: map[string]bool{},
		mdStyle:  resolveMarkdownStyle(isTTY, dark),
		ta:       ta,
		search:   ti,
		input:    in,
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
	m.rows = tree.FlattenFiltered(m.roots, m.expanded, m.filterPredicate())
	m.cursor = tree.ClampCursor(m.cursor, len(m.rows))
}

// filterPredicate ANDs the active view filters into one predicate, or returns
// nil when none are active (nil = show everything). The text query is taken
// live from the input while searching, otherwise from the committed query.
func (m Model) filterPredicate() tree.Predicate {
	hideDone := m.hideDone
	wipFocus := m.wipFocus
	q := m.query
	if m.searching {
		q = m.search.Value()
	}
	q = strings.ToLower(strings.TrimSpace(q))
	if !hideDone && !wipFocus && q == "" {
		return nil
	}
	return func(y *yaks.Yak) bool {
		if hideDone && y.State == yaks.StateDone {
			return false
		}
		if wipFocus && y.State != yaks.StateWip && y.State != yaks.StateBlocked {
			return false
		}
		if q != "" && !strings.Contains(strings.ToLower(y.Name), q) {
			return false
		}
		return true
	}
}

func (m Model) selectedID() string {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return m.rows[m.cursor].Yak.ID
	}
	return ""
}

// selectedYak returns a copy of the yak under the cursor and true, or false if
// there's no selection.
func (m Model) selectedYak() (yaks.Yak, bool) {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return *m.rows[m.cursor].Yak, true
	}
	return yaks.Yak{}, false
}

// openInput opens the single-line input in the given mode, seeded with value.
func (m *Model) openInput(mode inputMode, value string) {
	m.inputMode = mode
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.input.Focus()
}

// restoreCursor puts the cursor back on the yak with the given id if it's still
// visible. rebuildRows has already clamped the cursor to a valid row, so when
// the yak is gone we simply leave that clamped position.
func (m *Model) restoreCursor(id string) {
	if id == "" {
		return
	}
	if idx := tree.IndexOfID(m.rows, id); idx >= 0 {
		m.cursor = idx
	}
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

	case contextSavedMsg:
		// Saved successfully: leave edit mode and reload so the detail pane
		// reflects the new body (cursor preserved by id).
		m.editing = false
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
	// Remove confirmation owns the keyboard: y confirms, anything else cancels.
	if m.confirming {
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && (msg.Runes[0] == 'y' || msg.Runes[0] == 'Y') {
			cmd := m.removeCmd()
			m.confirming = false
			return m, cmd
		}
		m.confirming = false
		return m, nil
	}

	// Add/rename input owns the keyboard: enter commits (empty = cancel), esc
	// cancels, everything else is text input.
	if m.inputMode != inputNone {
		switch msg.Type {
		case tea.KeyEnter:
			name := strings.TrimSpace(m.input.Value())
			mode := m.inputMode
			m.inputMode = inputNone
			m.input.Blur()
			if name == "" {
				return m, nil // empty = no-op cancel
			}
			switch mode {
			case inputAddChild, inputAddRoot:
				return m, m.addCmd(m.inputParID, name)
			case inputRename:
				return m, m.renameCmd(m.inputTgtID, name)
			}
			return m, nil
		case tea.KeyEsc:
			m.inputMode = inputNone
			m.input.Blur()
			m.input.SetValue("")
			return m, nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// Search mode owns the keyboard: enter commits the query (filter persists),
	// esc clears it, everything else is text input for the search field.
	if m.searching {
		switch msg.Type {
		case tea.KeyEnter:
			m.query = m.search.Value()
			m.searching = false
			m.search.Blur()
			id := m.selectedID()
			m.rebuildRows()
			m.restoreCursor(id)
			m.refreshDetail()
			return m, nil
		case tea.KeyEsc:
			m.searching = false
			m.search.Blur()
			m.search.SetValue("")
			m.query = ""
			id := m.selectedID()
			m.rebuildRows()
			m.restoreCursor(id)
			m.refreshDetail()
			return m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		// Re-filter live as the query changes.
		id := m.selectedID()
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
		return m, cmd
	}

	// Edit mode owns the keyboard: ctrl+s saves, esc cancels, everything else
	// (including ctrl+c) is text input for the textarea. This must come before
	// any global binding so the editor isn't interrupted by triage/quit keys.
	if m.editing {
		switch msg.Type {
		case tea.KeyCtrlS:
			return m, m.saveContextCmd()
		case tea.KeyEsc:
			m.editing = false
			return m, nil
		}
		var cmd tea.Cmd
		m.ta, cmd = m.ta.Update(msg)
		return m, cmd
	}

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
	case key.Matches(msg, m.keys.Edit):
		return m.enterEdit()
	case key.Matches(msg, m.keys.Add):
		if y, ok := m.selectedYak(); ok {
			m.inputParID = y.ID
			m.openInput(inputAddChild, "")
		}
		return m, nil
	case key.Matches(msg, m.keys.AddRoot):
		m.inputParID = ""
		m.openInput(inputAddRoot, "")
		return m, nil
	case key.Matches(msg, m.keys.Rename):
		if y, ok := m.selectedYak(); ok {
			m.inputTgtID = y.ID
			m.openInput(inputRename, y.Name)
		}
		return m, nil
	case key.Matches(msg, m.keys.Remove):
		if y, ok := m.selectedYak(); ok {
			m.confirming = true
			m.removeID = y.ID
			m.removeName = y.Name
			m.removeKids = len(y.Children)
		}
		return m, nil
	case key.Matches(msg, m.keys.HideDone):
		id := m.selectedID()
		m.hideDone = !m.hideDone
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
	case key.Matches(msg, m.keys.WipFocus):
		id := m.selectedID()
		m.wipFocus = !m.wipFocus
		m.rebuildRows()
		m.restoreCursor(id)
		m.refreshDetail()
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		m.search.SetValue("")
		m.search.Focus()
		return m, nil
	case msg.Type == tea.KeyEsc:
		if m.hideDone || m.wipFocus || m.query != "" {
			id := m.selectedID()
			m.hideDone = false
			m.wipFocus = false
			m.query = ""
			m.search.SetValue("")
			m.rebuildRows()
			m.restoreCursor(id)
			m.refreshDetail()
		}
	}
	return m, nil
}

// enterEdit opens the inline editor for the selected yak, loading its current
// context into the textarea. No selection → no-op.
func (m Model) enterEdit() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return m, nil
	}
	y := *m.rows[m.cursor].Yak
	body := ""
	if y.Context != nil {
		body = *y.Context
	}
	m.editing = true
	m.editID = y.ID
	m.ta.SetValue(body)
	m.ta.CursorEnd()
	m.ta.Focus()
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
	m.ta.SetWidth(detailWidth)
	m.ta.SetHeight(bodyHeight)
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

// saveContextCmd writes the textarea body to the yak captured when edit mode
// was entered. On success it yields contextSavedMsg (which exits edit mode and
// reloads); on failure it yields errMsg and edit mode is left untouched, so the
// user's edits aren't lost.
func (m Model) saveContextCmd() tea.Cmd {
	id := m.editID
	content := m.ta.Value()
	return func() tea.Msg {
		if err := m.client.SetContext(context.Background(), id, content); err != nil {
			return errMsg{err}
		}
		return contextSavedMsg{}
	}
}

// existingIDs collects every id currently in the tree (for add collision checks).
func (m Model) existingIDs() map[string]bool {
	ids := map[string]bool{}
	var walk func(ys []yaks.Yak)
	walk = func(ys []yaks.Yak) {
		for i := range ys {
			ids[ys[i].ID] = true
			walk(ys[i].Children)
		}
	}
	walk(m.roots)
	return ids
}

func (m Model) addCmd(parentID, name string) tea.Cmd {
	existing := m.existingIDs()
	return func() tea.Msg {
		id, err := m.client.Add(context.Background(), parentID, name, existing)
		if err != nil {
			return errMsg{fmt.Errorf("couldn't create yak: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsgPreserving{roots: roots, prevID: id}
	}
}

func (m Model) renameCmd(id, name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.Rename(context.Background(), id, name); err != nil {
			return errMsg{fmt.Errorf("rename failed: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return loadedMsgPreserving{roots: roots, prevID: id}
	}
}

func (m Model) removeCmd() tea.Cmd {
	id := m.removeID
	recursive := m.removeKids > 0
	return func() tea.Msg {
		if err := m.client.Remove(context.Background(), id, recursive); err != nil {
			return errMsg{fmt.Errorf("remove failed: %w", err)}
		}
		roots, err := m.client.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		// prevID is the removed yak; IndexOfID won't find it, so the cursor
		// stays clamped near where it was — the intended "fall to neighbor".
		return loadedMsgPreserving{roots: roots, prevID: id}
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
		var msg string
		if m.hideDone || m.wipFocus || m.query != "" || m.searching {
			msg = "No yaks match the current view.\n\nPress esc to clear search, or H / W to clear filters."
		} else {
			msg = "No yaks yet.\n\nStart one with:  yx add \"my first yak\"\n\nq to quit · r to reload"
		}
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

	// While editing, the right pane shows the textarea and takes focus styling
	// regardless of the underlying focus field.
	rightContent := m.detail.View()
	if m.editing {
		rightContent = m.ta.View()
		detailBorder = focusedBorder
		treeBorder = blurredBorder
	}

	left := treeBorder.Width(treeWidth).Height(bodyHeight).Render(m.renderTree(treeWidth, bodyHeight))
	right := detailBorder.Width(detailWidth).Height(bodyHeight).Render(rightContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	var bar string
	switch {
	case m.searching:
		bar = subtle.Render(m.search.View() + "  (enter to keep · esc to clear)")
	case m.editing:
		bar = subtle.Render("editing — ctrl+s save · esc cancel")
	case m.status != "":
		bar = statusErr.Render(m.status)
	default:
		if ind := m.filterIndicator(); ind != "" {
			bar = subtle.Render(ind)
		} else {
			bar = m.help.View(m.keys)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, bar)
}

// filterIndicator summarizes active view filters for the status bar, or "" when
// none are active.
func (m Model) filterIndicator() string {
	var parts []string
	if m.hideDone {
		parts = append(parts, "[hide-done]")
	}
	if m.wipFocus {
		parts = append(parts, "[wip-focus]")
	}
	if m.query != "" {
		parts = append(parts, "[search: "+m.query+"]")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ") + "  ·  esc clears · H/W toggle · f search"
}
