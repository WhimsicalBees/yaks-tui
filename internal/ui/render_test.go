package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
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

func TestDetailMarkdownIncludesNameAndContext(t *testing.T) {
	ctx := "Hello **world**"
	y := yaks.Yak{ID: "a", Name: "alpha", State: "wip", Context: &ctx, Tags: []string{"@bug"}}
	md := detailMarkdown(y)
	if !contains(md, "alpha") {
		t.Errorf("detail md missing name:\n%s", md)
	}
	if !contains(md, "Hello") {
		t.Errorf("detail md missing context:\n%s", md)
	}
	if !contains(md, "bug") {
		t.Errorf("detail md missing tag:\n%s", md)
	}
}

func TestDetailMarkdownNoContext(t *testing.T) {
	y := yaks.Yak{ID: "a", Name: "empty", State: "todo"}
	md := detailMarkdown(y)
	if !contains(md, "empty") {
		t.Errorf("missing name:\n%s", md)
	}
	if !contains(md, "_No context_") {
		t.Errorf("expected no-context placeholder:\n%s", md)
	}
}

func TestViewTooSmall(t *testing.T) {
	m := loaded(t, twoYaks())
	m.width, m.height = 10, 3
	m.layout()
	out := m.View()
	if !contains(out, "too small") {
		t.Fatalf("expected resize hint, got:\n%s", out)
	}
}

func TestViewEmptyRepo(t *testing.T) {
	m := loaded(t, []yaks.Yak{})
	out := m.View()
	if !contains(out, "No yaks yet") {
		t.Fatalf("expected empty-state hint, got:\n%s", out)
	}
}

// TestRenderTreeCursorRowAnsiSafe guards against truncating an already-styled
// row: when the selected yak's name overflows the pane width, the highlighted
// row must still be valid ANSI (every escape sequence terminated, ending in a
// reset) so the highlight doesn't bleed onto following cells. This only
// manifests under a real color profile, so we force one for the test.
func TestRenderTreeCursorRowAnsiSafe(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	defer lipgloss.SetColorProfile(prev)

	long := strings.Repeat("verylongname", 10) // far wider than the pane
	roots := []yaks.Yak{{ID: "a", Name: long, State: yaks.StateWip}}
	m := loaded(t, roots)

	out := m.renderTree(20, 5)
	// The styled output must end its (single) content line with a reset before
	// the newline — i.e. no dangling open style. lipgloss uses "\x1b[0m".
	trimmed := strings.TrimRight(out, "\n")
	if strings.Contains(trimmed, "\x1b[") && !strings.HasSuffix(trimmed, "\x1b[0m") {
		t.Fatalf("cursor row not reset-terminated (highlight would bleed):\n%q", out)
	}
}

func TestBreadcrumb(t *testing.T) {
	cases := map[string]string{
		"deploy app":                      "", // root → no breadcrumb
		"deploy app/set up CI":            "deploy app",
		"deploy app/set up CI/fix linter": "deploy app/set up CI",
		"":                                "",
	}
	for in, want := range cases {
		if got := breadcrumb(in); got != want {
			t.Errorf("breadcrumb(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetailMarkdownShowsBreadcrumb(t *testing.T) {
	y := yaks.Yak{ID: "c", Name: "fix linter", State: "todo",
		FullPath: "deploy app/set up CI/fix linter"}
	md := detailMarkdown(y)
	if !contains(md, "deploy app/set up CI") {
		t.Errorf("detail md missing breadcrumb:\n%s", md)
	}
}

// TestResolveMarkdownStyle pins the style selection that lets us avoid glamour's
// WithAutoStyle — which queries the terminal background by reading stdin and, in
// the render loop, races Bubble Tea's input reader (the input-lag bug). The
// terminal-vs-dark detection is done once at startup and passed in here as plain
// booleans, so this mapping never touches the terminal.
func TestResolveMarkdownStyle(t *testing.T) {
	cases := []struct {
		isTerminal, dark bool
		want             string
	}{
		{false, false, "notty"},
		{false, true, "notty"}, // not a terminal: dark is irrelevant
		{true, true, "dark"},
		{true, false, "light"},
	}
	for _, c := range cases {
		if got := resolveMarkdownStyle(c.isTerminal, c.dark); got != c.want {
			t.Errorf("resolveMarkdownStyle(term=%v, dark=%v) = %q, want %q",
				c.isTerminal, c.dark, got, c.want)
		}
	}
}

func TestFooterAddChildPrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	out := m2.(Model).View()
	if !contains(out, "add child of") || !contains(out, "alpha") {
		t.Fatalf("footer missing add-child prompt:\n%s", out)
	}
}

func TestFooterAddRootPrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	out := m2.(Model).View()
	if !contains(out, "add root") {
		t.Fatalf("footer missing add-root prompt:\n%s", out)
	}
}

func TestFooterRenamePrompt(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	out := m2.(Model).View()
	if !contains(out, "rename") {
		t.Fatalf("footer missing rename prompt:\n%s", out)
	}
}

func TestFooterConfirmLeaf(t *testing.T) {
	m := loaded(t, twoYaks())
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	out := m2.(Model).View()
	if !contains(out, `remove "alpha"?`) {
		t.Fatalf("footer missing leaf confirm:\n%s", out)
	}
}

func TestFooterConfirmSingleChild(t *testing.T) {
	roots := []yaks.Yak{{
		ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{{ID: "c", Name: "child", State: "todo"}},
	}}
	m := loaded(t, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	out := m2.(Model).View()
	if !contains(out, "and its 1 child?") {
		t.Fatalf("single-child confirm should be grammatical:\n%s", out)
	}
}

func TestFooterConfirmSubtree(t *testing.T) {
	roots := []yaks.Yak{{
		ID: "p", Name: "parent", State: "todo",
		Children: []yaks.Yak{
			{ID: "c1", Name: "child1", State: "todo"},
			{ID: "c2", Name: "child2", State: "todo"},
		},
	}}
	m := loaded(t, roots)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	out := m2.(Model).View()
	if !contains(out, "and its 2 children") {
		t.Fatalf("footer missing subtree confirm:\n%s", out)
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
