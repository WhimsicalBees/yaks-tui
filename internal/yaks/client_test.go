package yaks

import (
	"context"
	"os"
	"testing"
)

// fakeRunner returns canned output/err and records the args it was called with.
type fakeRunner struct {
	out      []byte
	err      error
	gotArgs  []string
	gotStdin string
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.gotArgs = args
	return f.out, f.err
}

func (f *fakeRunner) RunWithInput(_ context.Context, stdin string, args ...string) ([]byte, error) {
	f.gotStdin = stdin
	f.gotArgs = args
	return f.out, f.err
}

func TestClientList(t *testing.T) {
	data, err := os.ReadFile("testdata/list.json")
	if err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{out: data}
	c := NewClient(fr)

	roots, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(roots) != 1 || roots[0].Name != "deploy app" {
		t.Fatalf("unexpected roots: %+v", roots)
	}
	// Verify we invoked the right yx command.
	want := []string{"list", "--format", "json"}
	if len(fr.gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", fr.gotArgs, want)
	}
	for i := range want {
		if fr.gotArgs[i] != want[i] {
			t.Fatalf("args = %v, want %v", fr.gotArgs, want)
		}
	}
}

func TestClientSetState(t *testing.T) {
	fr := &fakeRunner{out: []byte("Set 'x' state to wip\n")}
	c := NewClient(fr)
	if err := c.SetState(context.Background(), "write-tests-hgny", StateWip); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	want := []string{"state", "write-tests-hgny", "wip"}
	if len(fr.gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", fr.gotArgs, want)
	}
	for i := range want {
		if fr.gotArgs[i] != want[i] {
			t.Fatalf("args = %v, want %v", fr.gotArgs, want)
		}
	}
}

func TestClientSetStateRejectsBadState(t *testing.T) {
	fr := &fakeRunner{}
	c := NewClient(fr)
	if err := c.SetState(context.Background(), "id", "frobnicate"); err == nil {
		t.Fatal("expected error for invalid state")
	}
	if fr.gotArgs != nil {
		t.Fatal("should not have called yx for invalid state")
	}
}

func TestClientSetContext(t *testing.T) {
	fr := &fakeRunner{out: []byte("done\n")}
	c := NewClient(fr)
	body := "# Title\n\nsome body\n"
	if err := c.SetContext(context.Background(), "deploy-app-x1y2", body); err != nil {
		t.Fatalf("SetContext: %v", err)
	}
	if fr.gotStdin != body {
		t.Fatalf("stdin = %q, want %q", fr.gotStdin, body)
	}
	want := []string{"context", "deploy-app-x1y2"}
	if len(fr.gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", fr.gotArgs, want)
	}
	for i := range want {
		if fr.gotArgs[i] != want[i] {
			t.Fatalf("args = %v, want %v", fr.gotArgs, want)
		}
	}
}

func TestBinaryAvailable(t *testing.T) {
	// LookPath-based; just assert the helper exists and returns a bool+err shape.
	_, err := BinaryAvailable()
	_ = err // may be nil or not depending on environment; we only test it runs
}

func TestRepoInitializedTrueWhenListSucceeds(t *testing.T) {
	fr := &fakeRunner{out: []byte("[]")}
	c := NewClient(fr)
	ok, err := c.RepoInitialized(context.Background())
	if err != nil {
		t.Fatalf("RepoInitialized: %v", err)
	}
	if !ok {
		t.Fatal("want initialized=true when list succeeds")
	}
}

func TestRepoInitializedFalseOnGitignoreError(t *testing.T) {
	fr := &fakeRunner{err: errString(".yaks is not gitignored. Fix with: echo '.yaks' >> .gitignore")}
	c := NewClient(fr)
	ok, _ := c.RepoInitialized(context.Background())
	if ok {
		t.Fatal("want initialized=false when yx errors")
	}
}

func TestClientAddRoot(t *testing.T) {
	fr := &fakeRunner{out: []byte("added\n")}
	c := NewClient(fr)
	gen := func() string { return "zzzz" }
	id, err := c.add(context.Background(), "", "deploy app", map[string]bool{}, gen)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if id != "deploy-app-zzzz" {
		t.Fatalf("id = %q, want deploy-app-zzzz", id)
	}
	want := []string{"add", "deploy app", "--id", "deploy-app-zzzz"}
	assertArgs(t, fr.gotArgs, want)
}

func TestClientAddChild(t *testing.T) {
	fr := &fakeRunner{out: []byte("added\n")}
	c := NewClient(fr)
	gen := func() string { return "zzzz" }
	id, err := c.add(context.Background(), "deploy-app-x1y2", "write tests", map[string]bool{}, gen)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	want := []string{"add", "write tests", "--id", id, "--under", "deploy-app-x1y2"}
	assertArgs(t, fr.gotArgs, want)
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("args = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args = %v, want %v", got, want)
		}
	}
}

// errString is a tiny helper to make an error from a string.
type errString string

func (e errString) Error() string { return string(e) }
