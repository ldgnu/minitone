package utils

import "fmt"

func FormatDuration(sec int) string {
	if sec <= 0 {
		return "live"
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func FormatDurationFull(sec int) string {
	if sec <= 0 {
		return "live"
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
