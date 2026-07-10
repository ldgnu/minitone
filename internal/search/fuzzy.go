package search

import (
	"math"
	"strings"
	"unicode"
)

type Match struct {
	Score float64
	Start int
	End   int
}

func FuzzyFind(query, text string) Match {
	if query == "" || text == "" {
		return Match{}
	}

	q := strings.ToLower(query)
	t := strings.ToLower(text)

	qRunes := []rune(q)
	tRunes := []rune(t)

	if len(qRunes) > len(tRunes) {
		return Match{}
	}

	// Fast path: exact substring.
	if idx := strings.Index(t, q); idx >= 0 {
		// Higher score for earlier matches and shorter texts.
		score := 50.0 - float64(idx)*0.5 - float64(len(tRunes)-len(qRunes))*0.3
		if score < 1 {
			score = 1
		}
		return Match{
			Score: score / float64(max(len(tRunes), 1)),
			Start: idx,
			End:   idx + len(qRunes) - 1,
		}
	}

	qi := 0
	score := 0.0
	firstMatch := -1
	lastMatch := 0
	prevMatch := -2

	for ti := 0; ti < len(tRunes) && qi < len(qRunes); ti++ {
		if qRunes[qi] == tRunes[ti] {
			if firstMatch < 0 {
				firstMatch = ti
			}
			lastMatch = ti

			if ti == prevMatch+1 {
				score += 10
			} else {
				dist := ti - prevMatch - 1
				score += math.Max(0, 5-float64(dist))
			}

			if qi == 0 && ti == 0 {
				score += 15
			}
			if ti == 0 {
				score += 5
			}
			if qi == len(qRunes)-1 && ti == len(tRunes)-1 {
				score += 15
			}
			// Word boundary bonus.
			if ti > 0 && (tRunes[ti-1] == ' ' || tRunes[ti-1] == '-' || tRunes[ti-1] == '_') {
				score += 8
			}

			prevMatch = ti
			qi++
		}
	}

	if qi < len(qRunes) {
		return Match{}
	}

	score -= float64(len(tRunes)-len(qRunes)) * 0.5
	if score < 0 {
		score = 0
	}

	return Match{
		Score: score / float64(max(len(tRunes), 1)),
		Start: firstMatch,
		End:   lastMatch,
	}
}

func HighlightRunes(text string, m Match) string {
	if m.Score == 0 {
		return text
	}
	runes := []rune(text)
	var out strings.Builder
	for i, r := range runes {
		if i >= m.Start && i <= m.End {
			out.WriteString("\033[1m")
			out.WriteRune(unicode.ToUpper(r))
			out.WriteString("\033[0m")
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
