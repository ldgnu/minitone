package models

import "testing"

func TestDisplayTitle(t *testing.T) {
	s := Song{Title: "Hello", FilePath: "/x.mp3"}
	if s.DisplayTitle() != "Hello" {
		t.Fatal()
	}
	s.Title = ""
	if s.DisplayTitle() != "/x.mp3" {
		t.Fatal()
	}
}

func TestIsStream(t *testing.T) {
	if !(Song{Duration: 0}.IsStream()) {
		t.Fatal()
	}
	if (Song{Duration: 10}.IsStream()) {
		t.Fatal()
	}
}
