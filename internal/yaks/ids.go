package yaks

import (
	"crypto/rand"
	"regexp"
	"strings"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases s, turns runs of non-alphanumerics into single dashes, and
// trims leading/trailing dashes. Falls back to "yak" when nothing survives, so
// an id is always non-empty.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "yak"
	}
	return s
}

const suffixAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// randomSuffix returns a 4-char base36-ish suffix, matching yx's id style.
func randomSuffix() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	out := make([]byte, 4)
	for i, c := range b {
		out[i] = suffixAlphabet[int(c)%len(suffixAlphabet)]
	}
	return string(out)
}

// uniqueID builds "<slug>-<suffix>" and regenerates the suffix until the result
// is absent from existing. gen supplies suffixes (injectable for tests).
func uniqueID(name string, existing map[string]bool, gen func() string) string {
	base := slugify(name)
	for {
		id := base + "-" + gen()
		if !existing[id] {
			return id
		}
	}
}
