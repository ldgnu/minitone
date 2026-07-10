package ui

import "testing"

func TestThemeIndex(t *testing.T) {
	if ThemeIndex("nord") < 0 {
		t.Fatal()
	}
	if themes[ThemeIndex("NORD")].Name != "nord" {
		t.Fatal("case insensitive")
	}
	if themes[ThemeIndex("")].Name != "terminal" {
		t.Fatal("default terminal (system)")
	}
	if themes[ThemeIndex("nope")].Name != "terminal" {
		t.Fatal("unknown default terminal")
	}
}

func TestRenderProgressBar(t *testing.T) {
	s := renderProgressBar(10, 0.5)
	if len([]rune(s)) != 10 {
		t.Fatalf("len %d: %q", len([]rune(s)), s)
	}
	s = renderProgressBar(5, -1)
	if len([]rune(s)) != 5 {
		t.Fatal()
	}
	s = renderProgressBar(5, 2)
	if len([]rune(s)) != 5 {
		t.Fatal()
	}
}
