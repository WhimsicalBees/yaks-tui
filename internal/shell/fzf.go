// Package shell composes external interactive tools (fzf, later $EDITOR).
package shell

import (
	"fmt"
	"os/exec"
	"strings"

	"yaks-tui/internal/yaks"
)

const fzfSep = "\t"

// FzfLines renders one selectable line per yak: "path<TAB>id". The id is the
// last tab-delimited field so fzf can show the path while we recover the id.
func FzfLines(rows []yaks.Yak) []string {
	lines := make([]string, 0, len(rows))
	for _, y := range rows {
		label := y.FullPath
		if label == "" {
			label = y.Name
		}
		lines = append(lines, label+fzfSep+y.ID)
	}
	return lines
}

// ParseFzfSelection extracts the trailing id from a selected line.
func ParseFzfSelection(line string) string {
	line = strings.TrimRight(line, "\n")
	if line == "" {
		return ""
	}
	parts := strings.Split(line, fzfSep)
	return parts[len(parts)-1]
}

// Available reports whether fzf is on PATH.
func Available() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// Pick runs fzf over the given lines and returns the selected id, or "" if the
// user cancelled. Returns an error only for unexpected failures (not cancel).
func Pick(lines []string) (string, error) {
	if !Available() {
		return "", fmt.Errorf("fzf not installed")
	}
	cmd := exec.Command("fzf", "--with-nth=1", "--delimiter="+fzfSep, "--prompt=yak> ")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	out, err := cmd.Output()
	if err != nil {
		// fzf exits 130 on cancel — treat as no selection, not an error.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
			return "", nil
		}
		return "", err
	}
	return ParseFzfSelection(string(out)), nil
}
