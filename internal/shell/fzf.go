// Package shell composes external interactive tools (fzf, later $EDITOR).
package shell

import (
	"fmt"
	"os"
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

// shellSingleQuote wraps s in single quotes, escaping any embedded single
// quotes, so it can be safely interpolated into an `sh -c` string.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// fzfShellCommand builds the `sh -c` argument that runs fzf reading its
// candidate list from listPath (stdin redirect) and writing the selection to
// outPath (stdout redirect). fzf draws its interactive UI on /dev/tty, so the
// stdin redirect does not break interactivity — it's the standard pipeline
// pattern (e.g. `cat list | fzf`). This is pure/testable: no I/O.
func fzfShellCommand(listPath, outPath string) string {
	return fmt.Sprintf(
		"fzf --with-nth=1 --delimiter=%s --prompt='yak> ' < %s > %s",
		shellSingleQuote(fzfSep),
		shellSingleQuote(listPath),
		shellSingleQuote(outPath),
	)
}

// FzfExec prepares an interactive fzf invocation suitable for tea.ExecProcess.
//
// Under tea.ExecProcess the child's stdin/stdout/stderr are wired to the real
// terminal, so we cannot pipe candidates via cmd.Stdin nor capture the
// selection via cmd.Output(). Instead we write the candidate list to a temp
// file and have fzf read it via a shell stdin redirect, and capture the
// selection into another temp file via a shell stdout redirect. fzf keeps its
// UI interactive by talking to /dev/tty directly.
//
// It returns the *exec.Cmd (an `sh -c` wrapper), the path to the output file
// holding the selected line after fzf exits, and a cleanup func that removes
// both temp files. cleanup is always non-nil and safe to call even on error.
func FzfExec(lines []string) (cmd *exec.Cmd, outPath string, cleanup func(), err error) {
	cleanup = func() {}

	listFile, err := os.CreateTemp("", "yaks-fzf-list-*")
	if err != nil {
		return nil, "", cleanup, err
	}
	listPath := listFile.Name()
	cleanup = func() { _ = os.Remove(listPath) }

	if _, werr := listFile.WriteString(strings.Join(lines, "\n")); werr != nil {
		_ = listFile.Close()
		cleanup()
		return nil, "", func() {}, werr
	}
	if cerr := listFile.Close(); cerr != nil {
		cleanup()
		return nil, "", func() {}, cerr
	}

	outFile, err := os.CreateTemp("", "yaks-fzf-out-*")
	if err != nil {
		cleanup()
		return nil, "", func() {}, err
	}
	outPath = outFile.Name()
	_ = outFile.Close()
	cleanup = func() {
		_ = os.Remove(listPath)
		_ = os.Remove(outPath)
	}

	cmd = exec.Command("sh", "-c", fzfShellCommand(listPath, outPath))
	return cmd, outPath, cleanup, nil
}

// Pick runs fzf over the given lines and returns the selected id, or "" if the
// user cancelled. Returns an error only for unexpected failures (not cancel).
//
// Deprecated: Pick uses inherited stdio with cmd.Output(), which does not work
// while Bubble Tea owns the terminal. The TUI uses FzfExec via tea.ExecProcess
// instead. Retained for tests and non-TUI callers.
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
