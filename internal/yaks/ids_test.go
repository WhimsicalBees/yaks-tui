package yaks

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Fix the flaky test": "fix-the-flaky-test",
		"  Trim  me  ":       "trim-me",
		"Symbols!! & stuff":  "symbols-stuff",
		"Multiple---dashes":  "multiple-dashes",
		"":                   "yak",
		"!!!":                "yak",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueIDAppendsSuffix(t *testing.T) {
	seq := []string{"aaaa"}
	i := 0
	gen := func() string { s := seq[i%len(seq)]; i++; return s }
	got := uniqueID("deploy app", map[string]bool{}, gen)
	if got != "deploy-app-aaaa" {
		t.Fatalf("uniqueID = %q, want deploy-app-aaaa", got)
	}
}

func TestUniqueIDRegeneratesOnCollision(t *testing.T) {
	seq := []string{"aaaa", "bbbb", "cccc"}
	i := 0
	gen := func() string { s := seq[i]; i++; return s }
	existing := map[string]bool{"deploy-app-aaaa": true, "deploy-app-bbbb": true}
	got := uniqueID("deploy app", existing, gen)
	if got != "deploy-app-cccc" {
		t.Fatalf("uniqueID = %q, want deploy-app-cccc", got)
	}
}
