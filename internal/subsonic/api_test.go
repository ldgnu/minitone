package subsonic

import "testing"

func TestStrIntHelpers(t *testing.T) {
	if strVal("hi") != "hi" {
		t.Fatal()
	}
	if strVal(1) != "" {
		t.Fatal()
	}
	if intVal(float64(42)) != 42 {
		t.Fatal()
	}
	if int64Val(float64(99)) != 99 {
		t.Fatal()
	}
}

func TestParseSongs(t *testing.T) {
	raw := []any{
		map[string]any{
			"id":       "1",
			"title":    "Song",
			"artist":   "Art",
			"duration": float64(120),
		},
	}
	songs := parseSongs(raw)
	if len(songs) != 1 || songs[0].Title != "Song" || songs[0].Duration != 120 {
		t.Fatalf("%+v", songs)
	}
}

func TestParseSongField(t *testing.T) {
	if len(parseSongField(nil)) != 0 {
		t.Fatal()
	}
	one := parseSongField(map[string]any{"id": "1", "title": "A"})
	if len(one) != 1 || one[0].Title != "A" {
		t.Fatal()
	}
}

func TestNewClientBaseURL(t *testing.T) {
	c := NewClient("http://localhost:4533", "u", "p")
	if c.baseURL != "http://localhost:4533/rest" {
		t.Fatalf("base %q", c.baseURL)
	}
	c2 := NewClient("http://localhost:4533/rest/", "u", "p")
	if c2.baseURL != "http://localhost:4533/rest" {
		t.Fatalf("base %q", c2.baseURL)
	}
}
