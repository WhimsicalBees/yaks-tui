package yaks

import (
	"context"
	"os"
	"testing"
)

// fakeRunner returns canned output/err and records the args it was called with.
type fakeRunner struct {
	out     []byte
	err     error
	gotArgs []string
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
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
