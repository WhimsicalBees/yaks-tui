package yaks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a `yx` invocation and returns its stdout. Behind an interface
// so tests can inject a fake instead of running the real binary.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
	// RunWithInput is like Run but pipes stdin to the command — used for
	// `yx context <id>`, which reads the new body from stdin.
	RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error)
}

// ExecRunner runs the real `yx` binary in the current working directory.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	return runYx(exec.CommandContext(ctx, "yx", args...), args)
}

func (ExecRunner) RunWithInput(ctx context.Context, stdin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	cmd.Stdin = strings.NewReader(stdin)
	return runYx(cmd, args)
}

// runYx executes cmd and maps failures to a friendly error. It surfaces stderr
// from yx so callers can show a useful message, falling back to wrapping the
// error itself when stderr is empty so the exit status isn't lost and the
// message is never blank. The args are joined into a readable command (e.g.
// "yx list --format json") rather than a Go slice.
func runYx(cmd *exec.Cmd, args []string) ([]byte, error) {
	out, err := cmd.Output()
	if err != nil {
		cmdline := "yx " + strings.Join(args, " ")
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s: %s", cmdline, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("%s: %w", cmdline, err)
	}
	return out, nil
}

// Client is the only type that talks to yx.
type Client struct {
	r Runner
}

func NewClient(r Runner) *Client { return &Client{r: r} }

// List returns the full yak tree.
func (c *Client) List(ctx context.Context) ([]Yak, error) {
	out, err := c.r.Run(ctx, "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	var roots []Yak
	if err := json.Unmarshal(out, &roots); err != nil {
		return nil, fmt.Errorf("decode yx list: %w", err)
	}
	return roots, nil
}

// ValidState reports whether s is a state yx understands.
func ValidState(s string) bool {
	switch s {
	case StateTodo, StateWip, StateBlocked, StateDone:
		return true
	}
	return false
}

// SetState sets a yak's state by id (ids are stable; names can collide).
func (c *Client) SetState(ctx context.Context, id, state string) error {
	if !ValidState(state) {
		return fmt.Errorf("invalid state %q", state)
	}
	_, err := c.r.Run(ctx, "state", id, state)
	return err
}

// SetContext replaces a yak's context body by id. The content is piped to
// `yx context <id>` on stdin, which yx reads when stdin is present. An empty
// content string is valid and clears the body.
func (c *Client) SetContext(ctx context.Context, id, content string) error {
	_, err := c.r.RunWithInput(ctx, content, "context", id)
	return err
}

// BinaryAvailable reports whether `yx` is on PATH.
func BinaryAvailable() (string, error) {
	return exec.LookPath("yx")
}

// RepoInitialized returns true if `yx list` works in the current dir. Any error
// (no repo, not gitignored, etc.) returns false plus the error for messaging.
func (c *Client) RepoInitialized(ctx context.Context) (bool, error) {
	_, err := c.r.Run(ctx, "list", "--format", "json")
	if err != nil {
		return false, err
	}
	return true, nil
}
