package search

import "testing"

func TestFuzzyFindExact(t *testing.T) {
	m := FuzzyFind("radiohead", "Radiohead - Creep")
	if m.Score <= 0 {
		t.Fatalf("expected positive score, got %v", m.Score)
	}
}

func TestFuzzyFindEmpty(t *testing.T) {
	if FuzzyFind("", "x").Score != 0 {
		t.Fatal("empty query")
	}
	if FuzzyFind("x", "").Score != 0 {
		t.Fatal("empty text")
	}
}

func TestFuzzyFindNoMatch(t *testing.T) {
	m := FuzzyFind("zzzz", "hello world")
	if m.Score != 0 {
		t.Fatalf("expected 0, got %v", m.Score)
	}
}

func TestFuzzyFindPrefersPrefix(t *testing.T) {
	a := FuzzyFind("bea", "Beatles")
	b := FuzzyFind("bea", "xxxbeayyy")
	if a.Score <= 0 || b.Score <= 0 {
		t.Fatalf("both should match: %v %v", a.Score, b.Score)
	}
	// Prefix / earlier match should generally score better
	if a.Score < b.Score {
		t.Fatalf("prefix should score higher: %v vs %v", a.Score, b.Score)
	}
}

func TestFuzzyFindUnicode(t *testing.T) {
	m := FuzzyFind("café", "Café Tacvba")
	if m.Score <= 0 {
		t.Fatalf("unicode match failed: %v", m.Score)
	}
}
