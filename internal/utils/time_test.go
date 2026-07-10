package utils

import "testing"

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "live"},
		{-1, "live"},
		{5, "0:05"},
		{65, "1:05"},
		{3600, "60:00"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.in); got != c.want {
			t.Errorf("FormatDuration(%d)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDurationFull(t *testing.T) {
	if got := FormatDurationFull(3661); got != "1:01:01" {
		t.Fatalf("got %q", got)
	}
	if got := FormatDurationFull(61); got != "1:01" {
		t.Fatalf("got %q", got)
	}
}
