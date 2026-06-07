package yaks

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_ListAndSetState(t *testing.T) {
	if _, err := exec.LookPath("yx"); err != nil {
		t.Skip("yx not installed; skipping e2e")
	}
	// Not t.TempDir(): yx writes read-only (0444) state files under .yaks, which
	// makes the automatic RemoveAll cleanup fail with "permission denied". We own
	// the cleanup here and chmod the tree writable first.
	dir, err := os.MkdirTemp("", "yaks-e2e")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err == nil {
				os.Chmod(path, 0o700)
			}
			return nil
		})
		os.RemoveAll(dir)
	})
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".yaks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("yx", "add", "alpha")

	// Build a client whose ExecRunner runs in dir.
	c := NewClient(dirRunner{dir})
	roots, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(roots) != 1 || roots[0].Name != "alpha" {
		t.Fatalf("unexpected roots: %+v", roots)
	}
	id := roots[0].ID
	if err := c.SetState(context.Background(), id, StateWip); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	roots, err = c.List(context.Background())
	if err != nil {
		t.Fatalf("List after SetState: %v", err)
	}
	if roots[0].State != StateWip {
		t.Fatalf("state = %q, want wip", roots[0].State)
	}
}

// dirRunner runs yx in a fixed directory (test-only).
type dirRunner struct{ dir string }

func (d dirRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	cmd.Dir = d.dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yx %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func (d dirRunner) RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	cmd.Dir = d.dir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yx %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}
