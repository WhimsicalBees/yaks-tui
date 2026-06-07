package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// osc11Query is the exact byte sequence glamour/termenv WRITES to the terminal
// when it asks for the background color (ESC ] 11 ; ?). The fixed code issues
// this exactly once, at startup, before the Bubble Tea event loop owns stdin.
// The buggy code (glamour.WithAutoStyle in the render loop) emits it on every
// keystroke, racing the input reader and swallowing keys. After the startup
// query is answered, navigation must produce ZERO of these.
const osc11Query = "\x1b]11;?"

// TestInputDoesNotTriggerTerminalQueryPerKeystroke runs the real binary under a
// PTY, answers the one-time startup terminal queries, then navigates with j/k
// WITHOUT answering any further queries. It asserts (a) the TUI is alive (it
// redraws in response to keys) and (b) no OSC 11 background query appears in
// response to those keystrokes — the signature of the input-lag bug (fff6ada).
func TestInputDoesNotTriggerTerminalQueryPerKeystroke(t *testing.T) {
	if testing.Short() {
		t.Skip("PTY integration test; skipped in -short")
	}
	if _, err := exec.LookPath("yx"); err != nil {
		t.Skip("yx not installed; skipping PTY test")
	}

	// 1. Build the binary to a temp path.
	dir := t.TempDir()
	bin := filepath.Join(dir, "yaks-tui")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	// 2. Seed a throwaway yaks repo. yx writes read-only (0444) files under
	//    .yaks, so the default t.TempDir RemoveAll cleanup would fail with
	//    "permission denied"; register a chmod-walk cleanup like e2e_test.go.
	repo := t.TempDir()
	t.Cleanup(func() {
		filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				os.Chmod(path, 0o700)
			}
			return nil
		})
	})
	runYx := func(args ...string) {
		t.Helper()
		c := exec.Command("yx", args...)
		c.Dir = repo
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("yx %v: %v\n%s", args, err, out)
		}
	}
	// git init + gitignore .yaks BEFORE yx add (yx refuses if .yaks not ignored).
	gitInit := exec.Command("git", "init")
	gitInit.Dir = repo
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(".yaks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runYx("add", "alpha apple")
	runYx("add", "beta banana")
	runYx("add", "gamma grape")

	// 3. Start the binary under a PTY, cwd = repo.
	cmd := exec.Command(bin)
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Skipf("cannot allocate PTY: %v", err)
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	// Window size above the "too small" guard (>=40x8); use 40 rows x 120 cols.
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120}); err != nil {
		t.Skipf("cannot set PTY size: %v", err)
	}

	// A single background goroutine owns all reads from the PTY. SetReadDeadline
	// is unreliable on a PTY master on darwin (the read blocks in a raw syscall
	// that ignores the deadline), so instead of bounding individual reads we run
	// reads on their own goroutine and bound the test by sleeping for each phase
	// window. The goroutine appends to a mutex-guarded buffer and, while "answer"
	// is enabled, replies to the startup OSC 11 background-color query and the
	// cursor-position (DSR) query so terminal detection completes promptly. It
	// exits on EOF/closed pipe (the deferred ptmx.Close in cleanup).
	var (
		mu     sync.Mutex
		buf    strings.Builder
		answer = true // reply to queries during the startup phase only
	)
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := ptmx.Read(b)
			if n > 0 {
				chunk := string(b[:n])
				mu.Lock()
				buf.WriteString(chunk)
				doAnswer := answer
				mu.Unlock()
				if doAnswer {
					if strings.Contains(chunk, osc11Query) {
						_, _ = ptmx.Write([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\"))
					}
					if strings.Contains(chunk, "\x1b[6n") {
						_, _ = ptmx.Write([]byte("\x1b[1;1R"))
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	snapshot := func() string {
		mu.Lock()
		defer mu.Unlock()
		return buf.String()
	}
	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		buf.Reset()
	}

	// 4. Let startup output flush and ANSWER the one-time queries so detection
	//    finishes. Then stop answering and clear the buffer so the next phase
	//    captures only what the keystrokes produce.
	time.Sleep(1500 * time.Millisecond)
	mu.Lock()
	answer = false
	mu.Unlock()
	reset()

	// 5. Send navigation keystrokes WITHOUT answering any further queries, then
	//    capture everything the program emits. If the render loop issues an OSC 11
	//    query per keystroke (the bug), we will SEE it here because we don't reply.
	//
	//    Send one "j" at a time with a small gap between presses. Bubble Tea
	//    throttles rendering to a frame rate and only flushes when the View output
	//    actually changes, so a single coalesced burst whose net cursor motion is
	//    zero can produce no diff at all. Discrete down-moves (cursor advances each
	//    time, and stays advanced) guarantee a visible redraw — the liveness signal
	//    we assert on — while still driving the render path the bug lived in.
	for i := 0; i < 3; i++ {
		if _, err := ptmx.Write([]byte("j")); err != nil {
			t.Fatalf("write keystroke: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(800 * time.Millisecond)
	post := snapshot()

	// 6. Quit; best-effort.
	_, _ = ptmx.Write([]byte("q"))

	// 7. Assertions on the post-keystroke output.
	//    (a) Liveness: navigation must produce SOME output (the TUI redrew). An
	//        empty buffer means the program is wedged.
	if strings.TrimSpace(post) == "" {
		t.Fatalf("no output after keystrokes — TUI appears wedged (possible input-lag regression)")
	}
	//    (b) Teeth: no OSC 11 background QUERY may appear in response to keys.
	//        The program only ever WRITES this query; we did not reply in phase 5,
	//        so any occurrence here is the program emitting it per keystroke.
	if n := strings.Count(post, osc11Query); n > 0 {
		t.Fatalf("found %d OSC 11 background query(ies) (%q) after navigation keystrokes; "+
			"the render loop must not query the terminal per keystroke (input-lag regression, see fff6ada)",
			n, osc11Query)
	}
}
