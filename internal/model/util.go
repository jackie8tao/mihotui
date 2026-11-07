package model

import (
	"fmt"
)

// FormatBytes formats byte counts into human-readable strings (e.g. 1.25 MB).
func FormatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats byte rate into human-readable transfer rate (e.g. 2.40 MB/s).
func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}

// FormatDelay formats latency in ms with readable format.
func FormatDelay(delay int) string {
	if delay <= 0 {
		return "Timeout"
	}
	return fmt.Sprintf("%d ms", delay)
}
