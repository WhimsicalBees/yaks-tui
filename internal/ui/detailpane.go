package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour"

	"yaks-tui/internal/yaks"
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

// renderMarkdown styles markdown with glamour; on failure returns the raw source
// (graceful degradation per spec).
func renderMarkdown(md string, width int) string {
	if width < 10 {
		width = 10
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
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
