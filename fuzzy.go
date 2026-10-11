package main

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	scoreMatch       = 16
	bonusBoundary    = 10 // match at the start of the string or of a path/word segment
	bonusCamel       = 6  // match at a lower→upper transition
	bonusConsecutive = 8  // match directly after the previous match
	penaltyGapStart  = 3
	penaltyGapExtend = 1
	maxLeadPenalty   = 5
)

// fuzzyFilter returns the candidates that contain query as a subsequence,
// best match first. Matching is case-insensitive unless query contains an
// uppercase letter. An empty query matches every candidate in its given order.
func fuzzyFilter(query string, candidates []string) []string {
	if query == "" {
		return append([]string(nil), candidates...)
	}

	type scored struct {
		s     string
		score int
	}
	var matches []scored
	for _, c := range candidates {
		if score, ok := fuzzyScore(query, c); ok {
			matches = append(matches, scored{c, score})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if len(a.s) != len(b.s) {
			return len(a.s) < len(b.s)
		}
		return a.s < b.s
	})

	out := make([]string, len(matches))
	for i, m := range matches {
		out[i] = m.s
	}
	return out
}

// fuzzyScore reports whether query is a subsequence of candidate and, if so,
// the score of its best alignment. Higher is better.
func fuzzyScore(query, candidate string) (int, bool) {
	q := []rune(query)
	c := []rune(candidate)
	if len(q) == 0 {
		return 0, true
	}
	if len(q) > len(c) {
		return 0, false
	}

	caseSensitive := strings.IndexFunc(query, unicode.IsUpper) >= 0
	eq := func(a, b rune) bool {
		if caseSensitive {
			return a == b
		}
		return unicode.ToLower(a) == unicode.ToLower(b)
	}

	// best[j] is the best score for matching q[:i+1] with q[i] at c[j];
	// valid[j] is false when no such alignment exists.
	best := make([]int, len(c))
	valid := make([]bool, len(c))
	next := make([]int, len(c))
	nextValid := make([]bool, len(c))

	for i := range q {
		for j := range c {
			nextValid[j] = false
			if !eq(q[i], c[j]) {
				continue
			}
			charScore := scoreMatch + positionBonus(c, j)

			if i == 0 {
				next[j] = charScore - min(j, maxLeadPenalty)
				nextValid[j] = true
				continue
			}
			for k := i - 1; k < j; k++ {
				if !valid[k] {
					continue
				}
				s := best[k] + charScore
				if gap := j - k - 1; gap == 0 {
					s += bonusConsecutive
				} else {
					s -= penaltyGapStart + penaltyGapExtend*(gap-1)
				}
				if !nextValid[j] || s > next[j] {
					next[j] = s
					nextValid[j] = true
				}
			}
		}
		best, next = next, best
		valid, nextValid = nextValid, valid
	}

	score, ok := 0, false
	for j := range c {
		if valid[j] && (!ok || best[j] > score) {
			score, ok = best[j], true
		}
	}
	return score, ok
}

func positionBonus(c []rune, j int) int {
	if j == 0 {
		return bonusBoundary
	}
	prev, cur := c[j-1], c[j]
	switch {
	case strings.ContainsRune("/-_. ", prev):
		return bonusBoundary
	case unicode.IsLower(prev) && unicode.IsUpper(cur):
		return bonusCamel
	}
	return 0
}

// isSubsequence reports whether every rune of query appears in s, in order,
// using the same case rules as fuzzyScore.
func isSubsequence(query, s string) bool {
	_, ok := fuzzyScore(query, s)
	return ok
}

// commonPrefix returns the longest prefix shared by every string in ss.
func commonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	prefix := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, prefix) {
			_, size := utf8.DecodeLastRuneInString(prefix)
			prefix = prefix[:len(prefix)-size]
		}
	}
	return prefix
}
