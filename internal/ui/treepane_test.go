package ui

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWindowBounds(t *testing.T) {
	cases := []struct {
		name              string
		cursor, n, height int
		wantStart         int
		wantEnd           int
	}{
		{"empty n", 0, 0, 10, 0, 0},
		{"zero height", 0, 5, 0, 0, 0},
		{"negative height", 0, 5, -3, 0, 0},
		{"n less than height", 2, 3, 10, 0, 3},
		{"n equals height", 4, 5, 5, 0, 5},
		// n > height cases below.
		{"cursor at start clamps", 0, 100, 10, 0, 10},
		{"cursor near start clamps", 2, 100, 10, 0, 10},
		{"cursor in middle centers", 50, 100, 10, 45, 55},
		{"cursor near end clamps", 98, 100, 10, 90, 100},
		{"cursor at end clamps", 99, 100, 10, 90, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := windowBounds(c.cursor, c.n, c.height)
			if start != c.wantStart || end != c.wantEnd {
				t.Fatalf("windowBounds(%d,%d,%d) = (%d,%d); want (%d,%d)",
					c.cursor, c.n, c.height, start, end, c.wantStart, c.wantEnd)
			}
			// Invariants for the n > height cases (where a real window is
			// computed). Gate on height > 0 so the degenerate height<=0 case
			// (which short-circuits to (0,0)) isn't treated as a real window.
			if c.height > 0 && c.n > c.height {
				if start < 0 {
					t.Errorf("start = %d; must be >= 0", start)
				}
				if end > c.n {
					t.Errorf("end = %d; must be <= n (%d)", end, c.n)
				}
				if end-start > c.height {
					t.Errorf("window size = %d; must be <= height (%d)", end-start, c.height)
				}
				// When n > height the window is always exactly height rows.
				if end-start != c.height {
					t.Errorf("window size = %d; want exactly height (%d)", end-start, c.height)
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"max zero", "hello", 0, ""},
		{"max negative", "hello", -5, ""},
		{"shorter than max", "hi", 10, "hi"},
		{"exactly max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hell…"},
		{"max one", "hello", 1, "h"},
		{"empty string", "", 5, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncate(c.s, c.max)
			if got != c.want {
				t.Fatalf("truncate(%q,%d) = %q; want %q", c.s, c.max, got, c.want)
			}
		})
	}

	// When truncation occurs, result must contain the ellipsis and have
	// rune-length exactly == max.
	t.Run("truncated result has ellipsis and exact rune length", func(t *testing.T) {
		got := truncate("hello world", 5)
		if !strings.Contains(got, "…") {
			t.Errorf("truncate result %q should contain ellipsis", got)
		}
		if n := utf8.RuneCountInString(got); n != 5 {
			t.Errorf("truncate result %q has rune length %d; want 5", got, n)
		}
	})

	// Multi-byte runes: truncation must slice on rune boundaries and never
	// produce invalid UTF-8 (would corrupt bytes if it sliced the []byte).
	t.Run("multibyte rune string slices on rune boundaries", func(t *testing.T) {
		s := "héllo wörld" // contains multi-byte runes é and ö
		got := truncate(s, 5)
		if !utf8.ValidString(got) {
			t.Errorf("truncate(%q,5) = %q; result is not valid UTF-8", s, got)
		}
		if n := utf8.RuneCountInString(got); n != 5 {
			t.Errorf("truncate(%q,5) rune length = %d; want 5", s, n)
		}
		if !strings.HasPrefix(got, "héll") {
			t.Errorf("truncate(%q,5) = %q; want prefix %q", s, got, "héll")
		}
	})

	t.Run("emoji string slices on rune boundaries", func(t *testing.T) {
		s := "a😀b😀c😀d" // emoji are 4-byte runes
		got := truncate(s, 4)
		if !utf8.ValidString(got) {
			t.Errorf("truncate(%q,4) = %q; result is not valid UTF-8", s, got)
		}
		if n := utf8.RuneCountInString(got); n != 4 {
			t.Errorf("truncate(%q,4) rune length = %d; want 4", s, n)
		}
	})
}
