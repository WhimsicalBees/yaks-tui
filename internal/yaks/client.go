package yaks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Runner executes a `yx` invocation and returns its stdout. Behind an interface
// so tests can inject a fake instead of running the real binary.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// ExecRunner runs the real `yx` binary in the current working directory.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "yx", args...)
	out, err := cmd.Output()
	if err != nil {
		// Surface stderr from yx so callers can show a useful message. Fall back
		// to wrapping the error itself when stderr is empty, so the exit status
		// isn't lost and the message is never blank.
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("yx %v: %s", args, string(ee.Stderr))
		}
		return nil, fmt.Errorf("yx %v: %w", args, err)
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
