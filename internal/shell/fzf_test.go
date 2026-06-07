package shell

import (
	"testing"

	"github.com/WhimsicalBees/yaks-tui/internal/yaks"
)

func TestFzfLinesAndParse(t *testing.T) {
	rows := []yaks.Yak{
		{ID: "a-1", Name: "alpha", FullPath: "alpha"},
		{ID: "b-2", Name: "beta", FullPath: "deploy/beta"},
	}
	lines := FzfLines(rows)
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	// Each line must end with a tab-delimited id we can parse back.
	id := ParseFzfSelection(lines[1])
	if id != "b-2" {
		t.Fatalf("parsed id = %q, want b-2", id)
	}
	if ParseFzfSelection("") != "" {
		t.Fatal("empty selection should parse to empty id")
	}
}

func TestFzfShellCommand(t *testing.T) {
	got := fzfShellCommand("/tmp/list", "/tmp/out")
	want := "fzf --with-nth=1 --delimiter='\t' --prompt='yak> ' < '/tmp/list' > '/tmp/out'"
	if got != want {
		t.Fatalf("fzfShellCommand:\n got %q\nwant %q", got, want)
	}
}

func TestShellSingleQuote(t *testing.T) {
	cases := map[string]string{
		"plain":     "'plain'",
		"/tmp/a b":  "'/tmp/a b'",
		"it's mine": `'it'\''s mine'`,
		"$(rm -rf)": "'$(rm -rf)'", // metachars stay literal inside single quotes
	}
	for in, want := range cases {
		if got := shellSingleQuote(in); got != want {
			t.Errorf("shellSingleQuote(%q) = %q, want %q", in, got, want)
		}
	}
}
