package shell

import (
	"testing"

	"yaks-tui/internal/yaks"
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
