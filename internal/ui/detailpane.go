package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

// detailMarkdown builds the markdown source shown in the detail pane for a yak.
func detailMarkdown(y yaks.Yak) string {
	var b strings.Builder
	// Breadcrumb: the ancestor path, derived from FullPath by dropping the yak's
	// own (last) segment. Shown only when the yak has a parent.
	if crumb := breadcrumb(y.FullPath); crumb != "" {
		fmt.Fprintf(&b, "%s\n\n", crumb)
	}
	fmt.Fprintf(&b, "# %s\n\n", y.Name)
	fmt.Fprintf(&b, "`%s`", y.State)
	if len(y.Tags) > 0 {
		fmt.Fprintf(&b, "  ·  %s", strings.Join(y.Tags, " "))
	}
	b.WriteString("\n\n")

	if y.Context != nil && strings.TrimSpace(*y.Context) != "" {
		b.WriteString(*y.Context)
		b.WriteString("\n\n")
	} else {
		b.WriteString("_No context_\n\n")
	}

	if len(y.Fields) > 0 {
		keys := make([]string, 0, len(y.Fields))
		for k := range y.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("---\n\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "## %s\n\n%s\n\n", k, y.Fields[k])
		}
	}
	return b.String()
}

// breadcrumb returns the ancestor path for a yak's full_path (everything before
// the final "/"-separated segment), or "" if the yak is a root. yaks joins path
// segments with "/", e.g. "deploy app/set up CI/fix linter" → "deploy app/set up CI".
func breadcrumb(fullPath string) string {
	i := strings.LastIndex(fullPath, "/")
	if i < 0 {
		return ""
	}
	return fullPath[:i]
}

// resolveMarkdownStyle picks a glamour style NAME without touching the terminal.
//
// glamour's WithAutoStyle resolves the style by calling termenv.HasDarkBackground,
// which writes an OSC query to the terminal and reads the response back from the
// TTY (blocking up to 5s). Done in the render loop — as it was, on every cursor
// move — that read races Bubble Tea's input reader for stdin and swallows the
// user's keystrokes, so keys need many presses to register. We instead detect
// (is-terminal, is-dark) ONCE at startup, before the event loop owns stdin, and
// pass the result here as plain booleans so this selection never does I/O.
func resolveMarkdownStyle(isTerminal, dark bool) string {
	if !isTerminal {
		return styles.NoTTYStyle
	}
	if dark {
		return styles.DarkStyle
	}
	return styles.LightStyle
}

// renderMarkdown styles markdown with glamour using an explicit, pre-resolved
// style (never WithAutoStyle — see resolveMarkdownStyle). On failure it returns
// the raw source (graceful degradation per spec).
func renderMarkdown(md, style string, width int) string {
	if width < 10 {
		width = 10
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return out
}
